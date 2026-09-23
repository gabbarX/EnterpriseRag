<!--
  CreateUserDialog: SystemAdmin "Create user" in-place popup (Access row).

  Anchors to the row action via the default slot, matching TenantMembers
  invite and UserProfile password popups. Form (username + email +
  auto-generate toggle) and, when the server mints a one-time password, a
  reveal view that must be acknowledged before the popup can close. Esc is
  captured on window so the parent Settings modal cannot unmount this and
  discard the password. The idempotent retry path closes with its own
  success toast instead.

  Visibility is owned by the parent via v-model:visible. Every user-visible
  message is also emitted via `announced` so the parent's sr-only live
  region can relay it. Shared popup chrome comes from systemAdminDialog.less.
-->
<template>
  <t-popup :visible="visible" trigger="click" placement="left-top" :destroy-on-close="!locked"
    overlay-class-name="wk-popover system-admin-action-popup-overlay" @visible-change="onVisibleChange">
    <span class="system-admin-action-popup-anchor">
      <slot />
    </span>
    <template #content>
      <div class="system-admin-action-popup-inner" @click.stop>
        <div class="system-admin-action-popup-title">
          {{ createUserSuccess
            ? 'User created'
            : 'Create a new user' }}
        </div>
        <p class="system-admin-action-popup-hint">
          {{ createUserSuccess
            ? 'A random password was generated for this user. It will only be shown once.'
            : 'This is a high-risk action. Verify the username and email carefully — they are how the new user will sign in.' }}
        </p>
        <template v-if="createUserSuccess">
          <div class="create-user-reveal">
            <div class="create-user-reveal-item">
              <span class="create-user-reveal-label">
                {{ 'Username' }}
              </span>
              <span class="create-user-reveal-value">{{ createUserSuccess.username }}</span>
            </div>
            <div class="create-user-reveal-item">
              <span class="create-user-reveal-label">
                {{ 'Email' }}
              </span>
              <span class="create-user-reveal-value">{{ createUserSuccess.email }}</span>
            </div>
            <div class="create-user-reveal-item">
              <span class="create-user-reveal-label">
                {{ 'Generated password' }}
              </span>
              <pre
                class="create-user-reveal-value create-user-reveal-value--mono">{{ createUserSuccess.generatedPassword }}</pre>
            </div>
          </div>
          <div class="system-admin-action-popup-footer">
            <t-button theme="primary" variant="outline" @click="copyAccountDetails">
              {{ 'Copy account details' }}
            </t-button>
            <t-button theme="primary" @click="acknowledge">
              {{ 'I have saved the password' }}
            </t-button>
          </div>
        </template>
        <template v-else>
          <t-form ref="formRef" :data="form" :rules="rules" label-align="top" class="system-admin-action-popup-form">
            <t-form-item :label="'Username'" name="username">
              <t-input v-model="form.username" type="text" clearable autocomplete="off" :disabled="submitting"
                :placeholder="'2-50 characters'" />
            </t-form-item>
            <t-form-item :label="'Email'" name="email">
              <t-input v-model="form.email" type="text" clearable autocomplete="off" :disabled="submitting"
                :placeholder="'Enter a valid email address'" />
            </t-form-item>
            <t-form-item name="autoGenerate">
              <t-checkbox v-model="form.autoGenerate" :disabled="submitting">
                {{ 'Auto-generate a random password' }}
              </t-checkbox>
            </t-form-item>
            <template v-if="!form.autoGenerate">
              <t-form-item :label="'New password'" name="newPassword">
                <t-input v-model="form.newPassword" type="password" autocomplete="new-password" :disabled="submitting"
                  :placeholder="'8-32 characters, including letters and numbers'">
                  <template #prefix-icon><t-icon name="lock-on" /></template>
                </t-input>
              </t-form-item>
              <t-form-item :label="'Confirm new password'" name="confirmPassword">
                <t-input v-model="form.confirmPassword" type="password" autocomplete="new-password"
                  :disabled="submitting" :placeholder="'Enter the new password again'"
                  @enter="submit">
                  <template #prefix-icon><t-icon name="lock-on" /></template>
                </t-input>
              </t-form-item>
            </template>
          </t-form>
          <div class="system-admin-action-popup-footer">
            <t-button variant="outline" :disabled="submitting" @click="emit('update:visible', false)">
              {{ 'Cancel' }}
            </t-button>
            <t-button theme="primary" :loading="submitting" @click="submit">
              {{ 'Create user' }}
            </t-button>
          </div>
        </template>
      </div>
    </template>
  </t-popup>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { createSystemUser, type CreateSystemUserRequest } from '@/api/system'
import { getAuthConfig } from '@/api/auth'
import { copyWithToast } from '@/utils/clipboard'
import { newPasswordRules } from '@/utils/passwordPolicy'
import {
  applyCreateUserResponse,
  captureLockedEscape,
  formatCreateUserCredentials,
  resolveCreateUserView,
  shouldAcceptCreateUserSubmit,
  type CreateUserReveal,
} from './createUserDialogState'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{
  'update:visible': [boolean]
  announced: [string]
}>()


const submitting = ref(false)
const dismissing = ref(false)
const formRef = ref<FormInstanceFunctions>()
const form = reactive({
  username: '',
  email: '',
  autoGenerate: true,
  newPassword: '',
  confirmPassword: '',
})
const createUserSuccess = ref<CreateUserReveal | null>(null)
const locked = computed(() => submitting.value || createUserSuccess.value !== null)
const complexPasswordEnabled = ref(false)

const loadAuthConfig = async () => {
  const resp = await getAuthConfig()
  complexPasswordEnabled.value = !!resp.complex_password_enabled
  if (!resp.success) {
    MessagePlugin.warning('Failed to load system settings')
  }
}

const rules = computed<Record<string, FormRule[]>>(() => ({
  username: [
    { required: true, message: 'Enter a username', trigger: 'blur' },
    { min: 2, max: 50, message: 'Username must be 2-50 characters', trigger: 'blur' },
  ],
  email: [
    { required: true, message: 'Enter an email address', trigger: 'blur' },
    { email: true, message: 'Enter a valid email address', trigger: 'blur' },
  ],
  newPassword: newPasswordRules(complexPasswordEnabled.value),
  confirmPassword: [
    { required: true, message: 'Enter the new password again', trigger: 'blur' },
    {
      validator: (value: string) => value === form.newPassword,
      message: 'The passwords do not match',
      trigger: 'blur',
    },
  ],
}))

watch(() => props.visible, async (visible) => {
  if (!visible) {
    createUserSuccess.value = null
    dismissing.value = false
    resetFormFields()
    return
  }
  dismissing.value = false
  createUserSuccess.value = null
  resetFormFields()
  await loadAuthConfig()
  await nextTick()
  formRef.value?.clearValidate?.()
})

function onVisibleChange(visible: boolean) {
  if (!visible && locked.value && !dismissing.value) return
  emit('update:visible', visible)
}

function onWindowEscCapture(e: KeyboardEvent) {
  captureLockedEscape(props.visible, locked.value && !dismissing.value, e)
}

onMounted(() => {
  window.addEventListener('keydown', onWindowEscCapture, true)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onWindowEscCapture, true)
})

function resetFormFields() {
  form.username = ''
  form.email = ''
  form.autoGenerate = true
  form.newPassword = ''
  form.confirmPassword = ''
  formRef.value?.clearValidate?.()
}

function announce(msg: string, kind: 'success' | 'error' = 'success') {
  emit('announced', msg)
  if (kind === 'error') {
    MessagePlugin.error(msg)
  } else {
    MessagePlugin.success(msg)
  }
}

async function submit() {
  if (!shouldAcceptCreateUserSubmit(submitting.value, createUserSuccess.value !== null)) return
  submitting.value = true
  try {
    const valid = await formRef.value?.validate?.()
    if (valid !== true) return

    const payload: CreateSystemUserRequest = {
      username: form.username.trim(),
      email: form.email.trim(),
    }
    if (!form.autoGenerate) {
      payload.password = form.newPassword
    }
    const response = await createSystemUser(payload)
    const view = resolveCreateUserView(
      response,
      { username: payload.username, email: payload.email },
      form.autoGenerate,
    )
    const next = applyCreateUserResponse(createUserSuccess.value, view)
    createUserSuccess.value = next.success
    if (next.notice === 'reveal') {
      emit('announced', 'User created')
    } else if (next.notice === 'created') {
      announce('User created')
    } else if (next.notice === 'idempotent') {
      announce('User already existed; no changes made')
    } else if (next.notice === 'missingPassword') {
      announce('User was created but the generated password was not returned. Reset the password to recover access.', 'error')
    }
    if (next.close) {
      dismissing.value = true
      emit('update:visible', false)
    }
  } catch (err: any) {
    const msg = err?.message || 'Failed to create user'
    announce(msg, 'error')
  } finally {
    submitting.value = false
  }
}

async function copyAccountDetails() {
  if (!createUserSuccess.value) return
  await copyWithToast(
    formatCreateUserCredentials(createUserSuccess.value, {
      username: 'Username',
      email: 'Email',
      password: 'Generated password',
    }),
    'Account details copied to clipboard',
  )
}

function acknowledge() {
  if (!createUserSuccess.value) return
  dismissing.value = true
  MessagePlugin.success('User created')
  emit('update:visible', false)
}
</script>

<style lang="less">
@import './systemAdminDialog.less';

.create-user-reveal {
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 2px 0 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-page);
  overflow: hidden;
}

.create-user-reveal-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child {
    border-bottom: none;
  }
}

.create-user-reveal-label {
  font-size: var(--app-text-sm);
  line-height: 16px;
  color: var(--td-text-color-secondary);
}

.create-user-reveal-value {
  font-size: var(--app-text-base);
  line-height: 20px;
  color: var(--td-text-color-primary);
  word-break: break-all;
}

.create-user-reveal-value--mono {
  margin: 0;
  font-family: var(--td-font-family-mono);
  font-size: var(--app-text-sm);
  line-height: 18px;
  white-space: pre-wrap;
  user-select: all;
}
</style>
