import { onBeforeUnmount, onMounted } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'

/**
 * Shared shell behaviour for the full-screen "settings-style" modals (Settings /
 * AgentEditor / OrganizationSettings / KnowledgeBaseEditor):
 *   - Esc closes (ignored while a TDesign dialog / drawer / popup is stacked on top, so
 *     those close first)
 *   - clicking the mask closes
 *   - unsaved changes prompt for confirmation first
 *
 *   const shell = useModalShell({
 *     visible: () => props.visible,
 *     close: () => emit('update:visible', false),
 *   })
 */
export interface ModalShellOptions {
  visible: () => boolean
  close: () => void
  snapshot?: () => unknown
  ignoreEscape?: () => boolean
}

function serialize(value: unknown): string {
  try {
    return JSON.stringify(value ?? null)
  } catch {
    return ''
  }
}

function hasOpenTDesignOverlay(): boolean {
  if (typeof document === 'undefined') return false
  const dialogCtx = document.querySelector<HTMLElement>('.t-dialog__ctx')
  if (dialogCtx && dialogCtx.style.display !== 'none') return true
  if (document.querySelector('.t-drawer--open')) return true
  const popups = document.querySelectorAll<HTMLElement>('.t-popup')
  for (const popup of popups) {
    if (popup.style.display !== 'none') return true
  }
  return false
}

export function useModalShell(options: ModalShellOptions) {
  let cleanSnapshot = serialize(options.snapshot?.())
  let confirming = false

  const markClean = () => {
    cleanSnapshot = serialize(options.snapshot?.())
  }

  const isDirty = () => {
    if (!options.snapshot) return false
    return serialize(options.snapshot()) !== cleanSnapshot
  }

  const requestClose = () => {
    if (!options.visible() || confirming) return
    if (!isDirty()) {
      options.close()
      return
    }
    confirming = true
    const dialog = DialogPlugin.confirm({
      header: 'Unsaved changes',
      body: 'Your changes will be lost if you close now. Close anyway?',
      theme: 'warning',
      confirmBtn: { content: 'Discard changes', theme: 'danger' },
      cancelBtn: 'Keep editing',
      onConfirm: () => {
        confirming = false
        dialog.destroy()
        options.close()
      },
      onClose: () => {
        confirming = false
        dialog.destroy()
      },
    })
  }

  const onKeydown = (event: KeyboardEvent) => {
    if (event.key !== 'Escape' || !options.visible()) return
    if (options.ignoreEscape?.()) return
    if (hasOpenTDesignOverlay()) return
    requestClose()
  }

  onMounted(() => window.addEventListener('keydown', onKeydown))
  onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))

  return { requestClose, markClean, isDirty }
}
