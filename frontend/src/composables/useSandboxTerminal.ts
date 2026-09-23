import { onUnmounted, ref, type Ref } from 'vue'
import { post } from '@/utils/request'
import { readStoredPtyId, writeStoredPtyId } from '@/utils/sandboxPtyId'

export type SandboxTerminalStatus =
  | 'paused'
  | 'connecting'
  | 'ready'
  | 'needs_provision'
  | 'exited'
  | 'no_sandbox'
  | 'unsupported'
  | 'idle'
  | 'unauthorized'
  | 'error'

export type SandboxTerminalControlFrame = {
  type: string
  code?: string
  message?: string
  pty_id?: number
  backend?: string
  exit_code?: number | null
  cols?: number
  rows?: number
}

const RECONNECT_BASE_DELAY_MS = 1000
const RECONNECT_MAX_DELAY_MS = 30000
const APP_PING_INTERVAL_MS = 25000
const PENDING_OUTPUT_MAX_BYTES = 1024 * 1024
// The server caps a single input frame at 4 KiB (terminalMaxInputBytes); gorilla closes
// the connection outright when that is exceeded, and xterm delivers a paste to onData in
// one piece, so pasting a script would drop the connection. Slice by bytes before
// sending: a PTY is a byte stream, so a cut in the middle of a multi-byte character is
// still reassembled verbatim and in order.
const INPUT_FRAME_MAX_BYTES = 2048

function resolveWsBase(): string {
  const base = (import.meta.env.BASE_URL || '/').replace(/\/+$/, '')
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.host}${base}`
}

async function mintTerminalTicket(sessionId: string): Promise<string> {
  const res = await post<{ success?: boolean; data?: { ticket?: string } }>(
    `/api/v1/sessions/${encodeURIComponent(sessionId)}/sandbox/terminal-ticket`,
    {},
  )
  const ticket = res?.data?.ticket
  if (!ticket) {
    throw new Error('missing terminal ticket')
  }
  return ticket
}

/**
 * provisionAttempted decides what SANDBOX_NOT_BOUND means: without an intent to create it
 * is merely "there is no sandbox yet, shall we make one"; with one, a failure means there
 * really is nowhere to create it.
 */
function statusFromErrorCode(
  code: string | undefined,
  provisionAttempted: boolean,
): SandboxTerminalStatus {
  if (code === 'SANDBOX_NOT_BOUND') {
    return provisionAttempted ? 'no_sandbox' : 'needs_provision'
  }
  if (code === 'SANDBOX_PAUSED') return 'paused'
  if (code === 'TERMINAL_UNSUPPORTED') return 'unsupported'
  if (code === 'IDLE_DISCONNECTED') return 'idle'
  if (code === 'AUTH_REVOKED') return 'unauthorized'
  return 'error'
}

export type SandboxTerminalSession = {
  status: Ref<SandboxTerminalStatus>
  /**
   * Injected by SandboxTerminal.vue: writes PTY output into xterm.
   * Binary frames that arrive before the handler is registered are queued first, so the
   * bash prompt is not dropped before xterm mounts. Pass null to start buffering again on
   * unmount.
   */
  onOutput: (handler: ((data: Uint8Array) => void) | null) => void
  /**
   * Connects and opens the terminal; idempotent while a connection is in flight.
   *
   * provision means "this is an explicit user action, the backend may create or wake a
   * sandbox". Only a button click should pass true: creating/waking a sandbox is real,
   * billable infrastructure. The automatic connect when the panel opens never passes it -
   * a running sandbox is attached directly, while a paused or not-yet-created one falls
   * back to asking the user to confirm.
   */
  connect: (options?: { provision?: boolean; cols?: number; rows?: number }) => void
  sendInput: (data: string) => void
  resize: (cols: number, rows: number) => void
  dispose: () => void
}

/**
 * All three arguments must be live refs (`toRef(props, ...)`), never a snapshot such as
 * `ref(props.x)`: every openSocket re-reads them, so that after the user switches agent
 * (including to one shared from another space) the next connection creates the sandbox
 * from the new agent's sandbox config.
 */
export function useSandboxTerminal(
  sessionId: Ref<string>,
  agentId: Ref<string | undefined>,
  agentSourceTenantId: Ref<string | number | null | undefined> = ref(undefined),
): SandboxTerminalSession {
  const status = ref<SandboxTerminalStatus>('connecting')

  let ws: WebSocket | null = null
  let opening = false
  let outputHandler: ((data: Uint8Array) => void) | null = null
  let pendingOutput: Uint8Array[] = []
  let pendingOutputBytes = 0
  let disposed = false
  let reconnectAttempt = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let pingTimer: ReturnType<typeof setInterval> | null = null
  let pendingResize: { cols: number; rows: number } | null = null
  let pendingGeometry: { cols: number; rows: number } | null = null
  let lastPid: number | null = readStoredPtyId(sessionId.value)
  let allowProvision = false

  const textEncoder = new TextEncoder()

  function rememberPid(next: number | null) {
    lastPid = next
    writeStoredPtyId(sessionId.value, next)
  }

  function clearReconnectTimer() {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function clearPingTimer() {
    if (pingTimer !== null) {
      clearInterval(pingTimer)
      pingTimer = null
    }
  }

  function scheduleReconnect() {
    if (disposed) return
    clearReconnectTimer()
    allowProvision = false
    const delay = Math.min(
      RECONNECT_BASE_DELAY_MS * 2 ** reconnectAttempt,
      RECONNECT_MAX_DELAY_MS,
    )
    reconnectAttempt += 1
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      void openSocket()
    }, delay)
  }

  async function openSocket() {
    if (disposed || opening || ws) return
    const sid = sessionId.value
    if (!sid) return

    opening = true
    status.value = 'connecting'
    try {
      const ticket = await mintTerminalTicket(sid)
      if (disposed) return
      const query = new URLSearchParams({ ticket })
      if (allowProvision) {
        query.set('provision', '1')
        const agent = agentId.value
        if (agent && agent !== 'builtin-quick-answer') {
          query.set('agent_id', agent)
        }
        const sourceTenant = agentSourceTenantId.value
        if (sourceTenant != null && String(sourceTenant).trim() !== '') {
          query.set('agent_source_tenant_id', String(sourceTenant).trim())
        }
      }
      if (lastPid && lastPid > 0) {
        query.set('pty_id', String(lastPid))
      }
      if (pendingGeometry) {
        query.set('cols', String(pendingGeometry.cols))
        query.set('rows', String(pendingGeometry.rows))
      }
      const socket = new WebSocket(
        `${resolveWsBase()}/api/v1/sessions/${encodeURIComponent(sid)}/sandbox/terminal?${query.toString()}`,
      )
      if (disposed) {
        socket.close()
        return
      }
      ws = socket
    } catch {
      ws = null
      if (disposed) return
      status.value = 'error'
      scheduleReconnect()
      return
    } finally {
      opening = false
    }
    if (!ws) return
    ws.binaryType = 'arraybuffer'

    ws.onopen = () => {
      clearPingTimer()
      pingTimer = setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }))
        }
      }, APP_PING_INTERVAL_MS)
    }

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        handleControlFrame(event.data)
        return
      }
      const data =
        event.data instanceof ArrayBuffer ? new Uint8Array(event.data) : new Uint8Array(0)
      deliverOutput(data)
    }

    ws.onclose = (event) => {
      clearPingTimer()
      ws = null
      if (disposed) return
      const reason = typeof event.reason === 'string' ? event.reason : ''
      if (
        event.code === 1008
        || reason === 'SANDBOX_NOT_BOUND'
        || reason === 'SANDBOX_PAUSED'
        || reason === 'TERMINAL_UNSUPPORTED'
        || reason === 'IDLE_DISCONNECTED'
        || reason === 'AUTH_REVOKED'
      ) {
        if (reason) {
          status.value = statusFromErrorCode(reason, allowProvision)
        } else if (status.value === 'ready' || status.value === 'connecting') {
          status.value = 'error'
        }
        if (status.value !== 'idle' && status.value !== 'unauthorized' && status.value !== 'paused') {
          rememberPid(null)
        }
        return
      }
      if (
        status.value !== 'needs_provision'
        && status.value !== 'paused'
        && status.value !== 'no_sandbox'
        && status.value !== 'unsupported'
        && status.value !== 'exited'
        && status.value !== 'idle'
        && status.value !== 'unauthorized'
      ) {
        status.value = 'error'
        scheduleReconnect()
      }
    }

    ws.onerror = () => {
    }
  }

  function handleControlFrame(raw: string) {
    let frame: SandboxTerminalControlFrame
    try {
      frame = JSON.parse(raw)
    } catch {
      return
    }
    switch (frame.type) {
      case 'ready':
        status.value = 'ready'
        rememberPid(typeof frame.pty_id === 'number' ? frame.pty_id : null)
        reconnectAttempt = 0
        // The provisioning intent ends here. It only ever explains SANDBOX_NOT_BOUND:
        // before connecting it means "there is no sandbox yet, shall we create one";
        // receiving it after connecting always means "the sandbox was reclaimed". Without
        // resetting it, a reclaim is misreported as no_sandbox ("the agent has no backend
        // configured") and the message has nothing to do with the real cause.
        allowProvision = false
        if (pendingResize) {
          sendResize(pendingResize.cols, pendingResize.rows)
          pendingResize = null
        }
        break
      case 'exited': {
        status.value = 'exited'
        rememberPid(null)
        break
      }
      case 'error': {
        status.value = statusFromErrorCode(frame.code, allowProvision)
        if (status.value !== 'idle' && status.value !== 'unauthorized' && status.value !== 'paused') {
          rememberPid(null)
        }
        break
      }
      default:
        break
    }
  }

  function deliverOutput(data: Uint8Array) {
    if (data.length === 0) return
    if (outputHandler) {
      outputHandler(data)
      return
    }
    pendingOutput.push(data)
    pendingOutputBytes += data.length
    while (pendingOutputBytes > PENDING_OUTPUT_MAX_BYTES && pendingOutput.length > 0) {
      const dropped = pendingOutput.shift()
      if (dropped) pendingOutputBytes -= dropped.length
    }
  }

  function sendRaw(frame: SandboxTerminalControlFrame) {
    ws?.send(JSON.stringify(frame))
  }

  function sendResize(cols: number, rows: number) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      pendingResize = { cols, rows }
      return
    }
    sendRaw({ type: 'resize', cols, rows })
  }

  const session: SandboxTerminalSession = {
    status,
    onOutput(handler) {
      outputHandler = handler
      if (!handler) return
      const queued = pendingOutput
      pendingOutput = []
      pendingOutputBytes = 0
      for (const chunk of queued) {
        handler(chunk)
      }
    },
    connect(options) {
      if (disposed || ws || opening) return
      clearReconnectTimer()
      allowProvision = options?.provision === true
      reconnectAttempt = 0
      const cols = options?.cols
      const rows = options?.rows
      pendingGeometry =
        cols && rows && cols > 0 && rows > 0 ? { cols, rows } : null
      void openSocket()
    },
    sendInput(data) {
      if (!ws || ws.readyState !== WebSocket.OPEN) return
      const bytes = textEncoder.encode(data)
      for (let offset = 0; offset < bytes.length; offset += INPUT_FRAME_MAX_BYTES) {
        ws.send(bytes.subarray(offset, offset + INPUT_FRAME_MAX_BYTES))
      }
    },
    resize: sendResize,
    dispose() {
      disposed = true
      clearReconnectTimer()
      clearPingTimer()
      if (ws) {
        const socket = ws
        ws = null
        socket.onclose = null
        socket.onerror = null
        socket.onmessage = null
        socket.close()
      }
    },
  }

  onUnmounted(() => session.dispose())

  return session
}
