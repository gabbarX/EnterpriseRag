<!--
  ResetPasswordDialog: SystemAdmin "reset user password" in-place popup
  for the Access row.

  The trigger button goes in the default slot. The parent owns visibility
  via v-model:visible and relays the `announced` event into its sr-only
  live region.
-->
<template>
  <t-popup :visible="visible && active" trigger="click" placement="left-top" destroy-on-close
    overlay-class-name="wk-popover system-admin-action-popup-overlay" @visible-change="onVisibleChange">
    <span class="system-admin-action-popup-anchor">
      <slot />
    </span>
    <template #content>
      <div class="system-admin-action-popup-inner" @click.stop>
        <div class="system-admin-action-popup-title">
          {{ 'Reset another user\'s password' }}
        </div>
        <p class="system-admin-action-popup-hint">
          {{ 'This is a high-risk action. Verify the user email carefully. You cannot reset your own password here.' }}
        </p>
        <t-form ref="formRef" :data="form" :rules="rules" label-align="top" class="system-admin-action-popup-form">
          <t-form-item :label="'User email'" name="email">
            <t-input v-model="form.email" type="text" clearable autocomplete="off" :disabled="submitting"
              :placeholder="'Enter the email of the user to reset'" />
          </t-form-item>
          <t-form-item :label="'New password'" name="newPassword">
            <t-input v-model="form.newPassword" type="password" autocomplete="new-password" :disabled="submitting"
              :placeholder="'Enter new password'">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </t-form-item>
          <t-form-item :label="'Confirm new password'" name="confirmPassword">
            <t-input v-model="form.confirmPassword" type="password" autocomplete="new-password" :disabled="submitting"
              :placeholder="'Enter the new password again'" @enter="submit">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </t-form-item>
        </t-form>
        <div class="system-admin-action-popup-footer">
          <t-button variant="outline" :disabled="submitting" @click="emit('update:visible', false)">
            {{ 'Cancel' }}
          </t-button>
          <t-button theme="danger" :loading="submitting" @click="submit">
            {{ 'Confirm reset' }}
          </t-button>
        </div>
      </div>
    </template>
  </t-popup>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { resetUserPassword } from '@/api/system'
import { getAuthConfig } from '@/api/auth'
import { newPasswordRules } from '@/utils/passwordPolicy'

// Hide the teleported popup on other tabs without discarding a pending reset.
const props = defineProps<{ visible: boolean; active: boolean }>()
const emit = defineEmits<{
  'update:visible': [boolean]
  announced: [string]
}>()


const submitting = ref(false)
const formRef = ref<FormInstanceFunctions>()
const form = reactive({
  email: '',
  newPassword: '',
  confirmPassword: '',
})
const complexPasswordEnabled = ref(false)

const loadAuthConfig = async () => {
  try {
    const resp = await getAuthConfig()
    complexPasswordEnabled.value = !!resp.complex_password_enabled
  } catch (err: any) {
    const msg = err?.message || 'Failed to load system settings'
    MessagePlugin.error(msg)
    complexPasswordEnabled.value = false
  }
}

const rules = computed<Record<string, FormRule[]>>(() => ({
  email: [
    { required: true, message: 'Enter email address', type: 'error' },
    { email: true, message: 'Enter correct email format', type: 'error' }
  ],
  newPassword: newPasswordRules(complexPasswordEnabled.value),
  confirmPassword: [
    { required: true, message: 'Confirm password', trigger: 'blur' },
    {
      validator: (value: string) => value === form.newPassword,
      message: 'Entered passwords do not match',
      trigger: 'blur',
    },
  ],
}))

function resetFormFields() {
  form.email = ''
  form.newPassword = ''
  form.confirmPassword = ''
  formRef.value?.clearValidate?.()
}

// destroy-on-close remounts the form DOM on every open, so the fields
// start empty regardless; the explicit reset + clearValidate here just
// guarantees the reactive state and any stale inline errors are gone too
// (mirrors the pre-refactor behaviour in SystemSettings).
watch(() => props.visible, async (visible) => {
  if (!visible) {
    resetFormFields()
    return
  }
  resetFormFields()
  await loadAuthConfig()
  await nextTick()
  formRef.value?.clearValidate?.()
})

function onVisibleChange(visible: boolean) {
  // Never let an outside click or Esc collapse the popup while a reset
  // roundtrip is in flight, so the write can't be orphaned. Emitting
  // nothing keeps the parent's v-model:visible true and the popup open.
  if (!visible && submitting.value) return
  emit('update:visible', visible)
}

async function submit() {
  if (submitting.value) return
  submitting.value = true
  try {
    const valid = await formRef.value?.validate?.()
    if (valid !== true) return
    await resetUserPassword({
      email: form.email.trim(),
      new_password: form.newPassword,
    })
    const success = 'Password reset; the user\'s existing sessions were revoked'
    emit('announced', success)
    MessagePlugin.success(success)
    emit('update:visible', false)
  } catch (err: any) {
    const msg = err?.message || 'Failed to reset password'
    emit('announced', msg)
    MessagePlugin.error(msg)
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="less">
@import './systemAdminDialog.less';
</style>
