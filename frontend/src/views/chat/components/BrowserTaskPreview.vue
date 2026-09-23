<template>
  <Teleport :to="pipTarget || 'body'" :disabled="!pipTarget">
  <aside v-if="status.selected" ref="previewElement" class="browser-task-preview" :class="{ 'is-dragging': dragging, 'is-pip': !!pipTarget, 'needs-help': status.needs_help }" :style="pipTarget ? {} : positionStyle" :aria-label="'Local browser task preview'">
    <div class="preview-heading" @pointerdown="!pipTarget && startDrag($event)" @pointermove="moveDrag" @pointerup="stopDrag" @pointercancel="stopDrag" @lostpointercapture="stopDrag"><BrowserIcon class="preview-browser-icon" width="20" height="20" /><strong>{{ 'Local browser' }}</strong><span>{{ (!status.connected ? 'Offline' : status.stopping ? 'Ending task…' : status.paused ? 'Paused' : status.needs_help ? 'Needs your input' : status.task_id ? 'Connected' : 'Waiting for a task page') }}</span>
      <button v-if="pipSupported" class="preview-popout" :disabled="pipOpening" :title="(pipTarget ? 'Return to conversation' : 'Pop out preview')" :aria-label="(pipTarget ? 'Return to conversation' : 'Pop out preview')" @pointerdown.stop @click="togglePictureInPicture"><t-icon :name="pipTarget ? 'fullscreen-exit' : 'fullscreen'" size="16px" /></button>
    </div>
    <section v-if="status.needs_help" class="preview-handoff" role="status" aria-live="polite">
      <strong>{{ 'Needs your input' }}</strong>
      <p v-if="status.help_prompt" class="handoff-prompt">{{ status.help_prompt }}</p>
      <p>{{ (status.action === 'tab_borrow' ? 'Switch to the page being borrowed and choose Allow or Deny in the BrowserSkill confirmation. Approval continues automatically. Continue operation only resumes a paused task; it does not approve borrowing.' : 'Open the browser from the preview, complete the requested step, then confirm completion in the browser help overlay.') }}</p>
      <t-button v-if="status.action !== 'tab_borrow'" class="handoff-locate" theme="default" variant="outline" size="small" :disabled="!status.connected || !status.task_id || busy || status.stopping" @click="act('focus')"><t-icon name="jump" />{{ 'Show browser' }}</t-button>
    </section>
    <button v-if="!(status.needs_help && status.action === 'tab_borrow')" class="preview-image" :disabled="!status.connected || !status.task_id || busy || status.stopping" :aria-label="'Show browser'" @click="act('focus')">
      <img v-if="preview" :src="preview" :alt="'Local browser task preview'" />
      <span v-else>{{ (status.connected ? 'Waiting for a task page' : 'Waiting to reconnect') }}</span>
      <span v-if="status.connected && status.task_id" class="locate"><t-icon name="jump" size="13px" />{{ 'Show browser' }}</span>
    </button>
    <p v-if="status.action" class="preview-progress">{{ browserActionLabel(status.action) }} · {{ `${Math.floor((status.action_elapsed_ms || 0) / 1000)} s` }}</p>
    <p v-if="browserPageAddress(status.page_url)" class="preview-address">{{ browserPageAddress(status.page_url) }}</p>
    <p v-if="status.last_error" class="preview-error">{{ status.last_error }}</p>
    <p class="preview-sync" role="status">{{ (!status.connected ? 'Waiting to reconnect' : status.idle ? 'Last preview retained' : previewStale ? 'Preview has not updated' : preview ? 'Preview syncing' : 'Fetching preview') }}</p>
    <p v-if="pipFailed" class="preview-error" role="alert">{{ 'Could not open the floating window. Please try again.' }}</p>
    <p v-if="error" class="preview-error" role="alert">{{ error }}</p>
    <p class="preview-scope">{{ 'Controls this conversation’s browser only' }}</p>
    <div class="preview-actions">
      <t-button v-if="status.connected" :title="(status.paused ? 'Resume browser' : 'Interrupt the browser action and keep the pages for resuming. The conversation continues.')" size="small" theme="default" variant="text" :disabled="busy || status.stopping" @click="act(status.paused ? 'resume' : 'pause')">{{ (status.paused ? 'Resume browser' : 'Pause browser') }}</t-button>
      <t-button v-else size="small" theme="default" variant="text" @click="openBrowserSettings">{{ 'Browser connection' }}</t-button>
      <t-button :title="'Close tabs created by this task and return borrowed tabs. Keep the browser open and paired.'" size="small" theme="default" variant="text" :disabled="busy || status.stopping" @click="act('stop')">{{ 'End browser task' }}</t-button>
    </div>
  </aside>
  </Teleport>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { get, post } from '@/utils/request'
import { useUIStore } from '@/stores/ui'
import { browserActionLabel, browserPageAddress } from '@/utils/browserToolDisplay'
import BrowserIcon from '@/components/icons/BrowserIcon.vue'
import { useFloatingPreviewDrag } from '@/composables/useFloatingPreviewDrag'
import { useDocumentPictureInPicture } from '@/composables/useDocumentPictureInPicture'
const props = defineProps<{ sessionId: string }>()
const uiStore = useUIStore()
const status = ref({ enabled: false, selected: false, connected: false, paused: false, idle: false, needs_help: false, help_prompt: '', task_id: '', action: '', action_elapsed_ms: 0, page_url: '', last_error: '', stopping: false })
const busy = ref(false), error = ref(''), preview = ref(''), previewStale = ref(false)
const { supported: pipSupported, target: pipTarget, opening: pipOpening, open: openPiP, close: closePiP } = useDocumentPictureInPicture(computed(() => status.value.selected), () => 'Local browser task preview')
const previewElement = ref<HTMLElement | null>(null)
const { positionStyle, dragging, startDrag, moveDrag, stopDrag } = useFloatingPreviewDrag(computed(() => pipTarget.value ? null : previewElement.value))
const pipFailed = ref(false)
async function togglePictureInPicture() {
  pipFailed.value = false
  if (pipTarget.value) { closePiP(); window.focus(); return }
  try { await openPiP() }
  catch { if (alive) pipFailed.value = true }
}
function openBrowserSettings() {
  closePiP()
  window.focus()
  uiStore.openSettings('browserconnection')
}

let lastFrame = 0
const endpoint = `/api/v1/sessions/${encodeURIComponent(props.sessionId)}/local-browser`
const controller = new AbortController()
let alive = true, cancelTimer: (() => void) | undefined, revision = 0
async function act(action: string) {
  if (busy.value) return
  busy.value = true; error.value = ''; revision++
  try {
    const result = await post<{ data: typeof status.value }>(endpoint, { action }, { timeout: 45000, signal: controller.signal })
    if (alive) { status.value = result.data; if (action === 'stop') preview.value = '' }
  } catch (e: any) { if (alive) error.value = e?.message || 'Operation failed. Please retry.' }
  finally { if (alive) busy.value = false }
}
let polling = false
async function poll() {
  if (polling || !alive) return
  polling = true
  try {
    if (!busy.value && (!document.hidden || pipTarget.value)) {
      const current = revision
      const result = await get<{ data: typeof status.value }>(endpoint, { signal: controller.signal })
      if (!alive || current !== revision) return
      status.value = result.data
      if (!status.value.connected || !status.value.task_id) preview.value = ''
      if (alive && !busy.value && status.value.selected && status.value.connected && status.value.task_id && (!status.value.idle || !preview.value)) {
        const image = await post<{ data: { image_base64: string; format: string; captured_at: string } }>(endpoint, { action: 'preview' }, { timeout: 10000, signal: controller.signal })
        if (alive && revision === current && image.data.image_base64 && ['png', 'jpeg'].includes(image.data.format)) { preview.value = `data:image/${image.data.format};base64,${image.data.image_base64}`; lastFrame = Date.parse(image.data.captured_at) || Date.now(); previewStale.value = false }
      }
    }
  } catch { /* Background polling does not replace explicit action errors. */ }
  finally {
    polling = false
    if (alive) {
      previewStale.value = !!preview.value && Date.now() - lastFrame > 5000
      // Schedule in the visible PiP document when the conversation is in the background.
      const timerWindow = pipTarget.value?.ownerDocument.defaultView || window
      const timer = timerWindow.setTimeout(poll, !status.value.enabled ? 30000 : status.value.selected && !status.value.idle ? 1000 : 5000)
      cancelTimer = () => timerWindow.clearTimeout(timer)
    }
  }
}
const onVisible = () => { cancelTimer?.(); if (!document.hidden || pipTarget.value) void poll() }
watch(pipTarget, onVisible, { flush: 'post' })
onMounted(() => { document.addEventListener('visibilitychange', onVisible); void poll() })
onBeforeUnmount(() => { alive = false; document.removeEventListener('visibilitychange', onVisible); revision++; cancelTimer?.(); controller.abort(); preview.value = '' })
</script>
<style scoped>
.browser-task-preview { position: absolute; right: 20px; bottom: 16px; width: 240px; max-width: calc(100% - 40px); max-height: calc(100% - 24px); box-sizing: border-box; z-index: 5; overflow: auto; background: var(--td-bg-color-container); border: 1px solid var(--td-component-border); border-radius: var(--app-radius-xl); box-shadow: 0 4px 20px #0000000d; }
.preview-browser-icon { flex-shrink: 0; color: var(--td-text-color-secondary); }
.preview-heading { display: flex; align-items: center; gap: 7px; padding: 10px 12px; cursor: grab; touch-action: none; user-select: none; position: sticky; top: 0; z-index: 1; background: var(--td-bg-color-container); }.is-dragging .preview-heading { cursor: grabbing; }.preview-heading strong { font-size: var(--app-text-sm); font-weight: 500; flex: 1; }.preview-heading span { color: var(--td-text-color-secondary); font-size: var(--app-text-xs); }
.preview-image { width: 100%; height: 126px; border: 0; border-block: 1px solid var(--td-component-border); background: var(--td-bg-color-secondarycontainer); color: var(--td-text-color-secondary); display: grid; place-items: center; position: relative; cursor: pointer; padding: 0; font-size: var(--app-text-sm); }.preview-image:disabled { cursor: default; }.preview-image img { width: 100%; height: 100%; object-fit: contain; object-position: top; }.preview-image:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: -2px; }.locate { position: absolute; display: inline-flex; align-items: center; gap: 5px; bottom: 8px; padding: 4px 8px; border-radius: 5px; background: var(--td-bg-color-container); color: var(--td-text-color-primary); box-shadow: 0 1px 5px #00000014; }.preview-image:hover .locate { color: var(--td-brand-color); }
.preview-progress, .preview-address, .preview-sync, .preview-help, .preview-scope { font-size: var(--app-text-xs); line-height: 1.5; margin: 6px 12px; color: var(--td-text-color-secondary); }
.preview-address { overflow-wrap: anywhere; }
.preview-help { color: var(--td-brand-color); }
.preview-actions { display: flex; flex-wrap: wrap; gap: 4px; justify-content: space-between; padding: 5px 8px; }.preview-error { margin: 8px 12px; color: var(--td-error-color); font-size: var(--app-text-sm); line-height: 1.5; }
@media(max-width:720px){.browser-task-preview { right: 12px; bottom: 12px; width: 200px; }.preview-image { height: 100px; }}
.preview-popout { display: grid; place-items: center; flex-shrink: 0; padding: 3px; border: 0; border-radius: var(--app-radius-xs); background: transparent; color: var(--td-text-color-secondary); cursor: pointer; }
.preview-popout:hover { background: var(--td-bg-color-secondarycontainer); color: var(--td-brand-color); }
.preview-popout:focus-visible { outline: 2px solid var(--td-brand-color); }
.preview-popout:disabled { opacity: .5; cursor: wait; }
.browser-task-preview.is-pip { position: static; width: 100%; max-width: none; min-height: 100%; max-height: none; border: 0; border-radius: 0; box-shadow: none; }
.is-pip .preview-heading { cursor: default; }
.is-pip .preview-image { height: clamp(126px, 45vh, 480px); }
.browser-task-preview.needs-help { width: 420px; }
.preview-handoff { display: flex; flex-direction: column; align-items: stretch; gap: 8px; padding: 12px; border-top: 1px solid var(--td-component-border); background: var(--td-bg-color-secondarycontainer); overflow-wrap: anywhere; }
.preview-handoff strong { font-size: var(--app-text-md); font-weight: 600; color: var(--td-text-color-primary); }
.preview-handoff p { font-size: var(--app-text-sm); line-height: 1.6; margin: 0; color: var(--td-text-color-secondary); white-space: pre-wrap; }
.preview-handoff .handoff-prompt { max-height: 22vh; overflow-y: auto; color: var(--td-text-color-primary); }
.preview-handoff .handoff-locate { align-self: flex-end; flex-shrink: 0; max-width: 100%; margin-top: 2px; }
.needs-help .preview-image { height: clamp(140px, 30vh, 260px); }
.needs-help .preview-heading { flex-wrap: wrap; }
.browser-task-preview.is-pip.needs-help { width: 100%; }
@media(max-width:480px) { .browser-task-preview.needs-help { width: calc(100% - 24px); max-width: calc(100% - 24px); } .browser-task-preview.is-pip.needs-help { width: 100%; max-width: none; } }
</style>
