const LOCAL_BROWSER_LABELS: Record<string, string> = {
  pipOpen: 'Pop out preview',
  pipReturn: 'Return to conversation',
  pipFailed: 'Could not open the floating window. Please try again.',
  captureScreenshot: 'Capture screenshot',
  navigationIncomplete: 'Navigation did not reach the requested loading phase. Check the current page.',
  noEntries: 'No entries returned.',
  stopping: 'Ending task…',
  elapsedSeconds: '{seconds} s',
  searchInstructionsTitle: 'Browser search instructions',
  searchInstructionsDescription: 'Set your preferred search engine and search URL.',
  searchInstructionsHint: 'Applies to your next request. Leave empty to use the default.',
  searchInstructionsReset: 'Restore default',
  searchInstructionsSaved: 'Saved',
  sourceHint: 'Use the local browser to look up and interact with pages this turn, alongside web search, knowledge bases and other tools.',
  pressKey: 'Press key',
  hoverPage: 'Hover over element',
  scrollPage: 'Scroll page',
  focusElement: 'Focus element',
  blurElement: 'Remove focus',
  selectOption: 'Select option',
  closeTab: 'Close tab',
  runScript: 'Run page script',
  readConsole: 'Read console',
  readNetwork: 'Inspect network requests',
  resizeWindow: 'Resize window',
  emulateDevice: 'Emulate device',
  actionPending: 'Operating browser…',
  actionRecorded: 'Browser action recorded',
  untitledTab: 'Untitled tab',
  contentTruncated: 'Only part of the page content is shown.',
  controlScope: 'Controls this conversation’s browser only',
  pauseHint: 'Interrupt the browser action and keep the pages for resuming. The conversation continues.',
  stopHint: 'Close tabs created by this task and return borrowed tabs. Keep the browser open and paired.',
  openPage: 'Open page',
  switchPage: 'Switch page',
  readPage: 'Read page',
  listTabs: 'List tabs',
  clickPage: 'Click element',
  fillPage: 'Fill content',
  waitPage: 'Wait for page',
  openTab: 'Create task tab',
  switchTab: 'Switch task tab',
  authorizeTab: 'Request tab permission',
  returnTab: 'Return tab',
  needHelp: 'Needs your input',
  browserAction: 'Browser action',
  actionFailed: 'Incomplete',
  actionCompleted: 'Action completed',
  commandBusy: 'The previous browser command is still running. Wait before continuing.',
  invalidArguments: 'Browser tool arguments are invalid or incomplete. The agent must correct them before proceeding.',
  commandInterrupted: 'The browser action was interrupted. Check the page before resuming from the preview.',
  actionFailedHint: 'The browser action did not finish. Check the page and try again.',
  previewStale: 'Preview has not updated',
  previewIdle: 'Last preview retained',
  previewLive: 'Preview syncing',
  previewLoading: 'Fetching preview',
  borrowHint: 'Switch to the page being borrowed and choose Allow or Deny in the BrowserSkill confirmation. Approval continues automatically. Continue operation only resumes a paused task; it does not approve borrowing.',
  helpHint: 'Open the browser from the preview, complete the requested step, then confirm completion in the browser help overlay.',
  settingsTitle: 'Browser connection',
  settingsDescription: 'Pair BrowserSkill with your local Chrome to operate real web pages from conversations.',
  openSettings: 'Open browser settings',
  settingsHint: 'Connect BrowserSkill in personal settings to use your local browser here.',
  unavailable: 'Local browsing is not enabled on this server. Contact your administrator.',
  source: 'Browser source',
  sandbox: 'Sandbox browser',
  local: 'Local browser',
  paused: 'Paused',
  connected: 'Connected',
  disconnected: 'Disconnected',
  resume: 'Resume browser',
  pause: 'Pause browser',
  start: 'Start task',
  stop: 'End browser task',
  pairHint: 'Paste the pairing link into Remote connection in the extension and confirm the server.',
  copyPairing: 'Copy pairing link',
  copied: 'Copied',
  windowHint: 'Tasks run in a labeled Chrome tab group. Existing tabs require your permission.',
  preview: 'Local browser task preview',
  waiting: 'Waiting for a task page',
  startHint: 'A browser request creates labeled task tabs in the background.',
  revoke: 'Revoke device access',
  revokeConfirm: 'You will need to pair again before using the local browser.',
  failed: 'Operation failed. Please retry.',
  productDescription: 'Run browser tasks in your Chrome',
  offline: 'Offline',
  notPaired: 'Not paired',
  lastSeen: 'Last connected',
  readyHint: 'Ready. Return to the conversation and describe your browser task.',
  reconnectHint: 'Authorization is saved. Keep Chrome and BrowserSkill open to reconnect automatically.',
  replaceDevice: 'Change browser',
  installExtension: 'Install BrowserSkill',
  installHint: 'Download the extension built for this server and install it in Chrome.',
  downloadExtension: 'Download extension',
  pairBrowser: 'Connect this browser',
  packageUnavailable: 'Ask your administrator for the matching extension package; a download is not configured yet.',
  manualCopy: 'Select and copy the link below. It expires in 5 minutes and can be used once.',
  pairingReady: 'Link copied. Paste it in the extension within 5 minutes. Single use only.',
  copyAgain: 'Copy again',
  usageTitle: 'How to use',
  usageStep1Title: 'Install the extension',
  usageStep1Text: 'Unzip the package, enable Developer mode on the Chrome extensions page, and choose Load unpacked.',
  usageStep2Title: 'Pair this browser',
  usageStep2Text: 'Copy the pairing link and paste it into Remote connection in the extension. Pair once for all conversations in this space.',
  usageStep3Title: 'Describe the task in chat',
  usageStep3Text: 'Turn on the local browser in the input bar and describe the web task. It runs in a labeled tab group.',
  usageStep4Title: 'Preview, locate, and resume',
  usageStep4Text: 'A small preview appears in the conversation. Click it to find the task tab. Interrupted tasks stay paused after reconnect — resume them from the preview. Existing tabs require your permission.',
  running: 'Running',
  locateWindow: 'Show browser',
  reconnectShort: 'Waiting to reconnect',
}

export type BrowserToolEvent = { arguments?: unknown; output?: unknown; error?: unknown; pending?: boolean; success?: boolean; tool_data?: unknown }
const methodKeys: Record<string, string> = {
  navigate: 'openPage', navigate_back: 'switchPage', navigate_forward: 'switchPage', reload: 'openPage',
  screenshot: 'captureScreenshot', stop: 'stop',
  observe: 'readPage', snapshot: 'readPage', get_html: 'readPage', tab_list: 'listTabs',
  click: 'clickPage', fill: 'fillPage', press: 'pressKey', wait_ms: 'waitPage', wait_for_navigation: 'waitPage',
  hover: 'hoverPage', wheel: 'scrollPage', scroll_to: 'scrollPage', focus: 'focusElement', blur: 'blurElement',
  select: 'selectOption', tab_close: 'closeTab', evaluate: 'runScript', console: 'readConsole', network: 'readNetwork',
  window_resize: 'resizeWindow', emulate: 'emulateDevice',
  tab_create: 'openTab', tab_select: 'switchTab', tab_borrow: 'authorizeTab', tab_return: 'returnTab',
  request_help: 'needHelp',
}
function record(value: unknown): Record<string, any> {
  if (typeof value === 'string') {
    try { value = JSON.parse(value) } catch { return {} }
  }
  return value && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, any> : {}
}
function text(value: unknown): string { return typeof value === 'string' ? value : '' }
// Show the destination without credentials or query/fragment tokens. Never make
// page-provided URLs clickable or fetch them while rendering a tool result.
export function browserPageAddress(value: unknown): string {
  try {
    const url = new URL(text(value))
    return ['https:', 'http:'].includes(url.protocol) ? `${url.origin}${url.pathname}` : ''
  } catch { return '' }
}
function sourceLocation(entry: Record<string, any>): string {
  const address = browserPageAddress(entry.url)
  return address ? address + (Number.isInteger(entry.line) ? ':' + entry.line : '') + (Number.isInteger(entry.column) ? ':' + entry.column : '') : ''
}
const navigationMethods = new Set(['navigate', 'navigate_back', 'navigate_forward', 'reload', 'wait_for_navigation'])
export function browserToolIncomplete(event: BrowserToolEvent): boolean {
  return event.success === false || navigationMethods.has(record(event.arguments).method) && record(event.output).reached === 'timeout'
}
export function browserActionLabel(method: string): string {
  return (LOCAL_BROWSER_LABELS[methodKeys[method] || 'browserAction'] ?? '')
}
export function browserToolTitle(event: BrowserToolEvent): string {
  const args = record(event.arguments)
  const label = browserActionLabel(args.method)
  let target = ''
  if (args.method === 'navigate' || args.method === 'tab_create') {
    try { target = new URL(browserPageAddress(args.url)).host } catch { /* No valid destination yet. */ }
  } else if (args.method === 'press') {
    target = text(args.key).slice(0, 40)
  }
  return `${'Local browser'} · ${label}${target ? ` · ${target}` : ''}${event.pending ? '…' : browserToolIncomplete(event) ? ` · ${'Incomplete'}` : ''}`
}
export function browserToolSummary(event: BrowserToolEvent): string {
  const failure = event.error || event.output
  const raw = typeof failure === 'string' ? failure : text(record(failure).message)
  if (event.pending) return 'Operating browser…'
  if (navigationMethods.has(record(event.arguments).method) && record(event.output).reached === 'timeout') return 'Navigation did not reach the requested loading phase. Check the current page.'
  if (event.success !== false) return (event.success === true ? 'Action completed' : 'Browser action recorded')
  if (/unfinished command|preview.*busy/i.test(raw)) return 'The previous browser command is still running. Wait before continuing.'
  if (/Parameter validation failed|Invalid browser arguments|invalid_params|duration_ms/i.test(raw)) return 'Browser tool arguments are invalid or incomplete. The agent must correct them before proceeding.'
  if (/paused|interrupted|timed out|timeout/i.test(raw)) return 'The browser action was interrupted. Check the page before resuming from the preview.'
  if (/disconnected|offline|connect BrowserSkill|not paired/i.test(raw)) return 'Authorization is saved. Keep Chrome and BrowserSkill open to reconnect automatically.'
  return 'The browser action did not finish. Check the page and try again.'
}
export function browserToolContent(event: BrowserToolEvent) {
  const output = { ...record(event.tool_data), ...record(event.output) }
  const args = record(event.arguments)
  let content = text(output.text) || text(output.html)
  const entries = Array.isArray(output.entries) ? output.entries : []
  if (args.method === 'console') {
    content = entries.slice(0, 100).map(record).map(entry => {
      const location = sourceLocation(entry)
      const stack = Array.isArray(entry.stack_trace) ? entry.stack_trace.slice(0, 20).map(record).map(frame =>
        `  ${text(frame.function_name)} ${sourceLocation(frame)}`).join('\n') : ''
      return `[${text(entry.level) || text(entry.kind)}] ${text(entry.text)}${location ? '\n' + location : ''}${stack ? '\n' + stack : ''}`
    }).join('\n\n')
  } else if (args.method === 'network') {
    content = entries.slice(0, 100).map(record).map(entry =>
      [text(entry.method), typeof entry.status === 'number' ? String(entry.status) : '', browserPageAddress(entry.url), text(entry.status_text), text(entry.error_text)].filter(Boolean).join(' ')
    ).join('\n')
  } else if (args.method === 'evaluate' && Object.hasOwn(output, 'value')) {
    content = typeof output.value === 'string' && output.value !== '' ? output.value : JSON.stringify(output.value, null, 2) ?? ''
  }
  const resultError = record(output.error)
  const error = text(output.error_text) || text(resultError.message) || text(resultError.text) || text(event.error) || text(record(event.error).message)
  const diagnostics = args.method === 'console' || args.method === 'network'
  const image = text(output.image_base64)
  const format = text(output.format)
  return {
    error: error.slice(0, 4000),
    recoveryHint: text(output.recovery_hint).slice(0, 4000),
    empty: diagnostics && entries.length === 0 && !error && !event.pending && !browserToolIncomplete(event),
    title: text(output.title).slice(0, 300),
    address: browserPageAddress(output.final_url || output.url || args.url),
    text: content.slice(0, 12000),
    truncated: output.truncated === true || content.length > 12000 || diagnostics && (entries.length > 100 || entries.some(entry => record(entry).truncated === true || Array.isArray(record(entry).stack_trace) && record(entry).stack_trace.length > 20)),
    image: ['png', 'jpeg'].includes(format) && image.length <= 8 * 1024 * 1024 && /^[A-Za-z0-9+/\r\n]+={0,2}$/.test(image)
      ? `data:image/${format};base64,${image}` : '',
    tabs: Array.isArray(output.tabs) ? output.tabs.map(record).map(tab => ({
      title: text(tab.title).slice(0, 300), address: browserPageAddress(tab.url),
    })) : [],
    prompt: args.method === 'request_help' ? text(args.prompt) : '',
  }
}
