import { DialogPlugin } from 'tdesign-vue-next'

interface ConfirmDeleteOptions {
  title?: string
  body: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => Promise<void> | void
}

export function useConfirmDelete() {

  return (opts: ConfirmDeleteOptions) => {
    const dialog = DialogPlugin.confirm({
      header: opts.title || ('Confirm Delete' as string),
      body: opts.body,
      confirmBtn: { content: opts.confirmText || ('Delete' as string), theme: 'danger' },
      cancelBtn: opts.cancelText || ('Cancel' as string),
      theme: 'danger',
      onConfirm: async () => {
        try {
          await opts.onConfirm()
        } finally {
          dialog.hide()
        }
      }
    })
    return dialog
  }
}
