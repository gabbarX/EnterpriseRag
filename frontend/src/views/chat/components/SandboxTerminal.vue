<template>
    <div ref="containerRef" class="sandbox-terminal" :class="{ 'is-dark': isDarkTheme }"
        @mousedown="focusTerminal">
        <div v-if="status !== 'ready'" class="sandbox-terminal__overlay">
            <div class="sandbox-terminal__overlay-card">
                <t-icon v-if="status === 'connecting'" name="loading" size="24px"
                    class="sandbox-terminal__spinner" />
                <t-icon
                    v-else-if="status === 'paused' || status === 'needs_provision' || status === 'no_sandbox'"
                    name="terminal" size="28px" />
                <t-icon v-else-if="status === 'unsupported'" name="error-circle" size="28px" />
                <t-icon v-else-if="status === 'idle'" name="time" size="28px" />
                <t-icon v-else name="cloud" size="28px" />
                <p class="sandbox-terminal__overlay-text">{{ statusText }}</p>
                <t-button v-if="actionLabel" size="small"
                    :theme="status === 'needs_provision' ? 'primary' : 'default'" variant="outline"
                    @click.stop="start">
                    {{ actionLabel }}
                </t-button>
            </div>
        </div>
        <div ref="terminalHost" class="sandbox-terminal__host"></div>
    </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import { useSandboxTerminal, type SandboxTerminalStatus } from '@/composables/useSandboxTerminal';
import { useTheme } from '@/composables/useTheme';
import { createPtyEchoPredictor } from '@/utils/ptyEchoPredictor';
import {
    PTY_PROMPT_NUDGE_DELAY_MS,
    xtermBufferLooksEmpty,
} from '@/utils/ptyPromptNudge';

const props = defineProps<{
    sessionId: string;
    /** Agent selected for this session; on first connect the backend creates the sandbox from its configuration. */
    agentId?: string;
    /** Source workspace of a shared agent, matching agent_source_tenant_id on the chat request. */
    agentSourceTenantId?: string | number | null;
}>();


const containerRef = ref<HTMLElement | null>(null);
const terminalHost = ref<HTMLElement | null>(null);

// The theme must derive from useTheme's shared ref, not from the `theme-mode` DOM
// attribute: a DOM attribute is not a reactive source, so such a computed would have
// no dependencies, be evaluated once and cached forever. The watch below would then
// never fire and the terminal colours would stay as they were at first mount.
const { currentTheme } = useTheme();
const systemPrefersDark = ref(prefersDarkQuery()?.matches === true);
const isDarkTheme = computed(() =>
    currentTheme.value === 'system' ? systemPrefersDark.value : currentTheme.value === 'dark',
);

function prefersDarkQuery(): MediaQueryList | null {
    if (typeof window === 'undefined' || !window.matchMedia) return null;
    return window.matchMedia('(prefers-color-scheme: dark)');
}

// In `system` mode the terminal must also follow live OS changes. useTheme's global
// listener only writes the DOM attribute and does not expose the resolved light/dark
// value, so subscribe separately here.
const prefersDark = prefersDarkQuery();
const onSystemThemeChange = (event: MediaQueryListEvent) => {
    systemPrefersDark.value = event.matches;
};
prefersDark?.addEventListener('change', onSystemThemeChange);

// ANSI palette: ls --color uses the usual dircolors mapping (dir=blue,
// exec=green, link=cyan). Only the green slots stay EnterpriseRag brand so
// user@host (01;32) matches the product color; path (01;34) stays blue
// like directories.
function xtermTheme(dark: boolean) {
    return dark
        ? {
            background: '#1a1a1a',
            foreground: '#e6e6e6',
            cursor: '#e6e6e6',
            cursorAccent: '#1a1a1a',
            selectionBackground: '#3a3a3a',
            red: '#c64751',
            brightRed: '#de6670',
            green: '#06b04d',
            brightGreen: '#07c05f',
            yellow: '#c4a000',
            brightYellow: '#fce94f',
            blue: '#3465a4',
            brightBlue: '#729fcf',
            magenta: '#75507b',
            brightMagenta: '#ad7fa8',
            cyan: '#06989a',
            brightCyan: '#34e2e2',
        }
        : {
            background: '#ffffff',
            foreground: '#242424',
            cursor: '#242424',
            cursorAccent: '#ffffff',
            selectionBackground: '#d0d7de',
            red: '#e34d59',
            brightRed: '#f36d78',
            green: '#06b04d',
            brightGreen: '#07c05f',
            yellow: '#c4a000',
            brightYellow: '#c4a000',
            blue: '#3465a4',
            brightBlue: '#729fcf',
            magenta: '#75507b',
            brightMagenta: '#ad7fa8',
            cyan: '#06989a',
            brightCyan: '#34e2e2',
        };
}

// toRef, not ref(props.x): the latter is a snapshot, so reconnecting after an agent
// switch would still create the sandbox from the old agent's configuration. The
// parent's :key rebuild covers sessionId, but it does not cover agentId.
const terminal = useSandboxTerminal(
    toRef(props, 'sessionId'),
    toRef(props, 'agentId'),
    toRef(props, 'agentSourceTenantId'),
);
const { status } = terminal;

const statusText = computed(() => {
    switch (status.value as SandboxTerminalStatus) {
        case 'paused':
            return 'This conversation\'s sandbox is paused. Starting the terminal resumes it.';
        case 'connecting':
            return 'Connecting to sandbox…';
        case 'needs_provision':
            return 'This conversation has no running sandbox. Creating one starts a new sandbox, billed according to your workspace configuration.';
        case 'no_sandbox':
            return 'No sandbox yet, and the current agent has no sandbox backend configured, so there is nowhere to create one. Switch to an agent with a sandbox configured, or send a message that runs code.';
        case 'unsupported':
            return 'The current sandbox backend does not support interactive terminals';
        case 'exited':
            return 'Terminal session ended';
        case 'idle':
            return 'The terminal disconnected after being idle. The sandbox will pause on its own TTL. You can reconnect.';
        case 'unauthorized':
            return 'Your session is no longer valid, so the terminal was disconnected. Sign in again, then reconnect.';
        default:
            return 'Connection lost';
    }
});

// Overlay button: a paused sandbox offers "Start terminal", one that does not exist
// yet offers "Create and start". Only a click may create or wake a sandbox; the
// lookup performed when the panel opens does neither.
const actionLabel = computed(() => {
    switch (status.value as SandboxTerminalStatus) {
        case 'paused':
            return 'Start terminal';
        case 'needs_provision':
            return 'Create and start';
        case 'error':
        case 'exited':
        case 'idle':
        case 'unauthorized':
        // no_sandbox means "the agent has no sandbox backend configured, so there is
        // nowhere to create one". Once the user configures it they should be able to
        // retry in place instead of having to close and reopen the panel, which would
        // otherwise be a dead end with no action at all.
        case 'no_sandbox':
            return 'Reconnect';
        default:
            return '';
    }
});

// The xterm instance is mounted once, only after `ready`; the overlay sits on top of
// it to show status.
let xterm: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let echo: ReturnType<typeof createPtyEchoPredictor> | null = null;
let resizeObserver: ResizeObserver | null = null;
let resizeDebounce: ReturnType<typeof setTimeout> | null = null;
let promptNudgeTimer: ReturnType<typeof setTimeout> | null = null;
let unmounted = false;

function writeToXterm(chunk: string | Uint8Array) {
    xterm?.write(chunk);
}

function focusTerminal() {
    if (status.value !== 'ready') return;
    xterm?.focus();
}

function mountTerminal() {
    if (!terminalHost.value || xterm) return;
    xterm = new Terminal({
        cursorBlink: true,
        cursorStyle: 'bar',
        cursorInactiveStyle: 'outline',
        fontSize: 13,
        fontFamily: "'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace",
        theme: xtermTheme(isDarkTheme.value),
        scrollback: 5000,
    });
    echo = createPtyEchoPredictor(writeToXterm);
    fitAddon = new FitAddon();
    xterm.loadAddon(fitAddon);
    xterm.open(terminalHost.value);
    xterm.onData((data) => {
        echo?.onLocal(data);
        terminal.sendInput(data);
    });

    resizeObserver = new ResizeObserver(() => {
        if (resizeDebounce) clearTimeout(resizeDebounce);
        resizeDebounce = setTimeout(() => applyFit(), 100);
    });
    resizeObserver.observe(containerRef.value || terminalHost.value);
    // Fit BEFORE flushing buffered PTY bytes. FitAddon.fit() calls
    // _renderService.clear() when the default 80x24 becomes the panel size,
    // which would wipe a prompt painted a moment earlier and leave only the
    // cursor until the next keystroke.
    applyFit();
    terminal.onOutput((data) => echo?.onRemote(data));
    schedulePromptNudge();
    void nextTick(() => {
        requestAnimationFrame(() => fitAndFocus());
    });
}

function unmountTerminal() {
    terminal.onOutput(null);
    echo = null;
    resizeObserver?.disconnect();
    resizeObserver = null;
    if (resizeDebounce) clearTimeout(resizeDebounce);
    resizeDebounce = null;
    if (promptNudgeTimer) clearTimeout(promptNudgeTimer);
    promptNudgeTimer = null;
    xterm?.dispose();
    xterm = null;
    fitAddon = null;
}

// The only entry point that may create or wake a sandbox. provision: true marks the
// user's explicit confirmation. Mounting only performs a lookup: a running sandbox is
// attached to directly, while a paused or not-yet-created one waits on the overlay for
// a click.
function start() {
    unmountTerminal();
    const { cols, rows } = estimatePtySize();
    terminal.connect({ provision: true, cols, rows });
}

function connectLookup() {
    const { cols, rows } = estimatePtySize();
    terminal.connect({ provision: false, cols, rows });
}

onMounted(() => {
    connectLookup();
});

watch(isDarkTheme, (dark) => {
    if (xterm) xterm.options.theme = xtermTheme(dark);
});

// Mount xterm once ready; leaving the ready state resets the instance
// (a reconnect means a new PTY).
watch(status, (next, prev) => {
    if (next === 'ready' && prev !== 'ready') {
        requestAnimationFrame(() => {
            // The component can be unmounted between `ready` and the next frame
            // (panel closed / session switched). Without this check we would create a
            // Terminal on a destroyed component and observe a detached node, and that
            // instance could not even be reclaimed by the watch.
            if (unmounted) return;
            mountTerminal();
        });
    } else if (prev === 'ready' && next !== 'ready') {
        unmountTerminal();
    }
});

// watch(status) only covers "status leaves ready"; it does not cover the component
// itself being destroyed. Closing the panel goes through SandboxSidePanel's v-if and
// switching sessions goes through the :key rebuild, and on both paths status never
// changes, so the watch never fires. Without this hook every panel open/close would
// leak one xterm instance (renderer plus 5000 lines of scrollback) and one
// ResizeObserver.
onBeforeUnmount(() => {
    unmounted = true;
    prefersDark?.removeEventListener('change', onSystemThemeChange);
    unmountTerminal();
});

function estimatePtySize() {
    const el = containerRef.value;
    const width = Math.max(0, (el?.clientWidth ?? 0) - 16);
    const height = Math.max(0, (el?.clientHeight ?? 0) - 16);
    return {
        cols: Math.max(20, Math.floor(width / 8) || 80),
        rows: Math.max(8, Math.floor(height / 17) || 24),
    };
}

function xtermVisibleBufferEmpty(): boolean {
    if (!xterm) return true;
    const buf = xterm.buffer.active;
    const origin = buf.viewportY;
    return xtermBufferLooksEmpty(
        (row) => buf.getLine(origin + row)?.translateToString(true),
        xterm.rows,
    );
}

// Pty.Connect does not replay a prompt bash already printed. A same-size
// resize is a no-op; flipping rows by 1 sends SIGWINCH so readline (and
// TUIs) redraw. Do not inject Ctrl-L or Enter: that would go to whatever
// is running in a reattached PTY.
function schedulePromptNudge() {
    if (promptNudgeTimer) clearTimeout(promptNudgeTimer);
    promptNudgeTimer = setTimeout(() => {
        promptNudgeTimer = null;
        if (unmounted || !xterm || status.value !== 'ready') return;
        if (!xtermVisibleBufferEmpty()) return;
        const cols = Math.max(2, xterm.cols);
        const rows = Math.max(2, xterm.rows);
        terminal.resize(cols, rows - 1);
        terminal.resize(cols, rows);
    }, PTY_PROMPT_NUDGE_DELAY_MS);
}

// While v-show hides the terminal the container is 0x0. FitAddon can still compute
// something like 2x1 and push it to the PTY, and on returning to the terminal tab bash
// would still be stuck at that size, which looks like a failed connection. So do
// nothing unless the container is big enough.
function containerHasPtySize() {
    const el = containerRef.value;
    return !!el && el.clientWidth >= 20 && el.clientHeight >= 20;
}

function applyFit() {
    if (!xterm || !containerHasPtySize()) return;
    try {
        fitAddon?.fit();
    } catch {
        // fit throws when the container has zero size; ignoring it is fine.
        return;
    }
    if (xterm.cols < 2 || xterm.rows < 2) return;
    terminal.resize(xterm.cols, xterm.rows);
    xterm.refresh(0, xterm.rows - 1);
}

function fitAndFocus() {
    // Right after v-show reveals the terminal, layout may still be pending inside
    // nextTick, so wait two frames before fitting.
    requestAnimationFrame(() => {
        requestAnimationFrame(() => {
            if (unmounted) return;
            applyFit();
            xterm?.focus();
        });
    });
}

defineExpose({
    focus: fitAndFocus,
});
</script>

<style scoped lang="less">
.sandbox-terminal {
    position: relative;
    height: 100%;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--td-bg-color-container);
    border-radius: var(--app-radius-md);
    overflow: hidden;
    cursor: text;
}

.sandbox-terminal__host {
    flex: 1;
    min-height: 0;
    padding: 8px;
    cursor: text;

    :deep(.xterm) {
        height: 100%;
        cursor: text;
    }

    :deep(.xterm-viewport) {
        overflow-y: auto;
    }

    :deep(.xterm-helper-textarea) {
        // xterm reads the keyboard through a hidden textarea; it must be focusable or
        // the cursor never blinks.
        pointer-events: auto;
    }
}

.sandbox-terminal__overlay {
    position: absolute;
    inset: 0;
    z-index: 2;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--td-bg-color-container);
    cursor: default;
}

.sandbox-terminal__overlay-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
    max-width: 280px;
    padding: 20px;
    text-align: center;
    color: var(--td-text-color-placeholder);
}

.sandbox-terminal__overlay-text {
    margin: 0;
    font-size: var(--app-text-md);
    line-height: 1.6;
    white-space: pre-line;
}

.sandbox-terminal__spinner {
    animation: wk-spin 0.9s linear infinite;
}

</style>
