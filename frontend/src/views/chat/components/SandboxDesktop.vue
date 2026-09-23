<template>
  <div class="sandbox-desktop">
    <div ref="screenRef" class="sandbox-desktop__screen"
         @mousedown="onUserActivity" @keydown="onUserActivity" />

    <div v-if="status !== 'connected'" class="sandbox-desktop__overlay">
      <div class="sandbox-desktop__overlay-card">
        <t-icon v-if="status === 'starting'" name="loading" size="24px"
                class="sandbox-desktop__spinner" />
        <t-icon v-else-if="status === 'unsupported'" name="error-circle" size="28px" />
        <t-icon v-else-if="status === 'busy' || status === 'rebuilt'" name="info-circle" size="28px" />
        <t-icon v-else-if="status === 'paused' || (status === 'idle' && hasConnected)" name="time" size="28px" />
        <t-icon v-else name="desktop" size="28px" />
        <p class="sandbox-desktop__overlay-text">{{ statusText }}</p>
        <t-button v-if="actionLabel" size="small"
                  :theme="status === 'idle' || status === 'needs_provision' || status === 'paused' ? 'primary' : 'default'"
                  variant="outline" @click.stop="start">
          {{ actionLabel }}
        </t-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, toRef, watch } from 'vue'
import { useSandboxDesktop } from '@/composables/useSandboxDesktop'

const props = defineProps<{
  sessionId: string
  agentId?: string
  agentSourceTenantId?: string | number | null
}>()

const screenRef = ref<HTMLElement | null>(null)

const desktop = useSandboxDesktop(
  toRef(props, 'sessionId'),
  toRef(props, 'agentId'),
  toRef(props, 'agentSourceTenantId'),
)
const status = desktop.status

// Composable uses 'starting' on first paint (lookup in flight) and 'idle'
// for IDLE_DISCONNECTED. :key="sessionId" remounts this component, so the
// flag resets per session.
const hasConnected = ref(false)
watch(status, (next) => {
  if (next === 'connected') hasConnected.value = true
})

const statusText = computed(() => {
  switch (status.value) {
    case 'starting': return 'Connecting to the desktop… (3-8 seconds on first use)'
    case 'unsupported': return 'This sandbox config has no desktop. Pick a desktop-image template in the workspace sandbox settings, on a Cube or E2B backend.'
    case 'busy': return 'This conversation already has a desktop open. Only one connection is allowed at a time, otherwise two people share one keyboard and mouse.'
    case 'start_failed': return 'The desktop failed to start. You can retry.'
    case 'rebuilt': return 'The sandbox was rebuilt for a skill update, so the previous desktop and any unsaved work are gone. Reconnecting gives you a fresh desktop.'
    case 'needs_provision': return 'This conversation has no running sandbox. Create and connect starts a new sandbox, billed according to your workspace configuration.'
    case 'paused': return 'This conversation\'s sandbox is paused. Connecting the desktop resumes it.'
    case 'unauthorized': return 'Your session is no longer valid, so the terminal was disconnected. Sign in again, then reconnect.'
    case 'disconnected': return 'Desktop disconnected'
    case 'error': return 'Desktop disconnected'
    case 'idle':
      return hasConnected.value
        ? 'The desktop disconnected after being idle. The sandbox will pause on its own TTL. You can reconnect.'
        : 'The desktop is not connected yet. Connecting it attaches to this conversation\'s sandbox.'
    default: return 'The desktop is not connected yet. Connecting it attaches to this conversation\'s sandbox.'
  }
})

// `unsupported` is a configuration problem, so retrying can never help and no
// action button is offered for it.
const actionLabel = computed(() => {
  switch (status.value) {
    case 'idle':
      return hasConnected.value
        ? 'Reconnect'
        : 'Connect desktop'
    case 'paused': return 'Connect desktop'
    case 'needs_provision': return 'Create and connect'
    case 'start_failed':
    case 'rebuilt':
    case 'busy':
    case 'disconnected':
    case 'error': return 'Reconnect'
    default: return ''
  }
})

function start() {
  if (!screenRef.value) return
  desktop.connect(screenRef.value, { provision: true })
}

function connectLookup() {
  if (!screenRef.value) {
    void nextTick(() => {
      if (screenRef.value) desktop.connect(screenRef.value, { provision: false })
    })
    return
  }
  desktop.connect(screenRef.value, { provision: false })
}

onMounted(() => {
  connectLookup()
})

// Fallback activity signal: it only matters when backend opcode parsing degrades.
// The composable already debounces it to 30 seconds.
function onUserActivity() {
  desktop.reportActivity()
}

onBeforeUnmount(() => desktop.dispose())

defineExpose({ start })
</script>

<style scoped lang="less">
.sandbox-desktop {
  position: relative;
  width: 100%;
  height: 100%;
  background: #1e1e1e;
  overflow: hidden;
}

.sandbox-desktop__screen {
  width: 100%;
  height: 100%;
}

.sandbox-desktop__overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(30, 30, 30, 0.92);
}

.sandbox-desktop__overlay-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  max-width: 320px;
  padding: 24px;
  text-align: center;
  color: #d8d8d8;
}

.sandbox-desktop__overlay-text {
  margin: 0;
  font-size: var(--app-text-md);
  line-height: 1.6;
}

.sandbox-desktop__spinner {
  animation: wk-spin 1s linear infinite;
}

</style>
