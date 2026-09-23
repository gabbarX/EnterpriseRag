<template>
  <div class="user-profile">
    <div class="section-header">
      <h2>{{ 'User Profile' }}</h2>
      <p class="section-description">{{ 'View your account info (user ID, username, email, registration time) and change your password.' }}</p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-inline">
      <t-loading size="small" />
      <span>{{ 'Loading information...' }}</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">{{ 'Retry' }}</t-button>
        </template>
      </t-alert>
    </div>

    <!-- Content -->
    <div v-else class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'User ID' }}</label>
          <p class="desc">{{ 'Your unique user identifier' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ userInfo?.id || '-' }}</span>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Username' }}</label>
          <p class="desc">{{ 'Your login username' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ userInfo?.username || '-' }}</span>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Email' }}</label>
          <p class="desc">{{ 'Your registered email address' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ userInfo?.email || '-' }}</span>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Registration Time' }}</label>
          <p class="desc">{{ 'Time when the account was created' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ formatDate(userInfo?.created_at) }}</span>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Change password' }}</label>
          <p class="desc">
            {{ oidcOnlyLogin
              ? 'Your account was created via OIDC sign-in and has no known local password yet.'
              : 'Verify your current password, then set a new one. All signed-in sessions will be revoked and you will need to sign in again.' }}
          </p>
        </div>
        <div class="setting-control">
          <template v-if="oidcOnlyLogin">
            <span class="info-value info-value--muted">—</span>
          </template>
          <template v-else>
            <span class="info-value password-mask" aria-hidden="true">••••••••</span>
            <t-popup
              v-model="passwordPopupVisible"
              trigger="click"
              placement="bottom-end"
              destroy-on-close
              overlay-class-name="wk-popover user-profile-password-popup-overlay"
            >
              <t-button
                theme="default"
                variant="text"
                shape="square"
                size="small"
                class="edit-btn"
                :title="'Change password'"
                :aria-label="'Change password'"
              >
                <template #icon>
                  <t-icon name="edit" />
                </template>
              </t-button>
              <template #content>
                <div class="password-popup-inner" @click.stop>
                  <div class="password-popup-title">{{ 'Change password' }}</div>
                  <p class="password-popup-hint">{{ 'Verify your current password, then set a new one. All signed-in sessions will be revoked and you will need to sign in again.' }}</p>
                  <t-form
                    ref="passwordFormRef"
                    :data="passwordForm"
                    :rules="passwordRules"
                    label-align="top"
                    class="password-popup-form"
                    @submit.prevent
                  >
                    <t-form-item :label="'Current password'" name="oldPassword">
                      <t-input
                        v-model="passwordForm.oldPassword"
                        type="password"
                        autocomplete="current-password"
                        :disabled="passwordSubmitting"
                        :placeholder="'Enter your current password'"
                      />
                    </t-form-item>
                    <t-form-item :label="'New password'" name="newPassword">
                      <t-input
                        v-model="passwordForm.newPassword"
                        type="password"
                        autocomplete="new-password"
                        :disabled="passwordSubmitting"
                        :placeholder="'Enter new password'"
                      />
                    </t-form-item>
                    <t-form-item :label="'Confirm new password'" name="confirmPassword">
                      <t-input
                        v-model="passwordForm.confirmPassword"
                        type="password"
                        autocomplete="new-password"
                        :disabled="passwordSubmitting"
                        :placeholder="'Enter the new password again'"
                        @enter="submitPasswordChange"
                      />
                    </t-form-item>
                  </t-form>
                  <div class="password-popup-footer">
                    <t-button
                      variant="outline"
                      :disabled="passwordSubmitting"
                      @click="closePasswordPopup"
                    >
                      {{ 'Cancel' }}
                    </t-button>
                    <t-button
                      theme="primary"
                      :loading="passwordSubmitting"
                      @click="submitPasswordChange"
                    >
                      {{ 'Update password' }}
                    </t-button>
                  </div>
                </div>
              </template>
            </t-popup>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import {
  getCurrentUser,
  getAuthConfig,
  changePassword,
  logout as logoutApi,
  type UserInfo,
} from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { newPasswordRules } from '@/utils/passwordPolicy'

const router = useRouter()
const authStore = useAuthStore()

const userInfo = ref<UserInfo | null>(null)
const complexPasswordEnabled = ref(false)
const loading = ref(true)
const error = ref('')

const passwordPopupVisible = ref(false)
const passwordFormRef = ref<FormInstanceFunctions | null>(null)
const passwordSubmitting = ref(false)
const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

const loadPasswordPolicy = async () => {
  try {
    const resp = await getAuthConfig()
    complexPasswordEnabled.value = !!resp.complex_password_enabled
  } catch {
    complexPasswordEnabled.value = false
  }
}

const oidcOnlyLogin = computed(
  () => userInfo.value?.preferences?.oidc_only_login === true,
)

watch(passwordPopupVisible, (open) => {
  if (!open) {
    resetPasswordForm()
    return
  }
  resetPasswordForm()
  void loadPasswordPolicy()
})

const passwordRules = computed<Record<string, FormRule[]>>(() => ({
  oldPassword: [
    { required: true, message: 'Enter your current password', type: 'error' },
  ],
  newPassword: newPasswordRules(complexPasswordEnabled.value, [
    {
      validator: (val: string) => val !== passwordForm.oldPassword,
      message: 'New password must differ from your current password',
      type: 'error',
    },
  ]),
  confirmPassword: [
    { required: true, message: 'Confirm password', type: 'error' },
    {
      validator: (val: string) => val === passwordForm.newPassword,
      message: 'Entered passwords do not match',
      type: 'error',
      trigger: 'blur',
    },
  ],
}))

const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''
    const resp = await getCurrentUser()
    if ((resp as any).success && resp.data) {
      userInfo.value = resp.data.user
    } else {
      error.value = resp.message || 'Failed to fetch workspace information'
    }
  } catch (err: any) {
    error.value = err?.message || 'Network error, please try again later'
  } finally {
    loading.value = false
  }
}

const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return 'Unknown'
  try {
    const d = new Date(dateStr)
    const fmt = new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
    return fmt.format(d)
  } catch {
    return 'Format error'
  }
}

const resetPasswordForm = () => {
  passwordForm.oldPassword = ''
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  passwordFormRef.value?.clearValidate?.()
}

const closePasswordPopup = () => {
  if (passwordSubmitting.value) return
  passwordPopupVisible.value = false
  resetPasswordForm()
}

const submitPasswordChange = async () => {
  if (passwordSubmitting.value) return
  const result = await passwordFormRef.value?.validate?.()
  if (result !== true) return

  passwordSubmitting.value = true
  try {
    const resp = await changePassword({
      old_password: passwordForm.oldPassword,
      new_password: passwordForm.newPassword,
    })
    if (!resp.success) {
      MessagePlugin.error(resp.message || 'Failed to change password. Check that your current password is correct.')
      return
    }

    passwordPopupVisible.value = false
    MessagePlugin.success('Password updated. Please sign in with your new password.')
    resetPasswordForm()

    // Backend revokes all sessions on success; mirror that locally and
    // force a fresh login with the new credential.
    try {
      await logoutApi()
    } catch {
      /* ignore — local cleanup still proceeds */
    }
    authStore.logout()
    router.push('/login')
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to change password. Check that your current password is correct.')
  } finally {
    passwordSubmitting.value = false
  }
}

onMounted(loadInfo)
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.user-profile {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.loading-inline {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 40px 0;
  justify-content: center;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-base);
}

.error-inline {
  padding: 20px 0;
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  .setting-row();
}

.setting-info {
  .setting-info();
}

.setting-control {
  .setting-control();

  .info-value {
    font-size: var(--app-text-base);
    color: var(--td-text-color-primary);
    text-align: right;
    word-break: break-word;
  }

  .info-value--muted {
    color: var(--td-text-color-placeholder);
  }

  .edit-btn {
    flex-shrink: 0;
  }
}

.password-mask {
  letter-spacing: 0.12em;
  color: var(--td-text-color-secondary);
}

.password-popup-inner {
  max-width: 100%;
}

.password-popup-title {
  font-size: var(--app-text-lg);
  font-weight: 600;
  color: var(--td-text-color-primary);
  margin: 0 0 8px;
  line-height: 1.35;
}

.password-popup-hint {
  margin: 0 0 12px;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.password-popup-form {
  :deep(.t-form__item) {
    margin-bottom: 14px;

    &:last-child {
      margin-bottom: 4px;
    }
  }
}

.password-popup-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}
</style>

<style lang="less">
/* t-popup mounts onto body, so this must be global; the z-index has to sit above the settings full-screen overlay (2000). */
.user-profile-password-popup-overlay {
  z-index: 3050 !important;

}

:root[theme-mode='dark'] .user-profile-password-popup-overlay .t-popup__content {
  background: rgba(36, 36, 36, 0.92) !important;
  border-color: rgba(255, 255, 255, 0.08) !important;
  box-shadow:
    0 0 0 0.5px rgba(255, 255, 255, 0.05),
    0 2px 4px rgba(0, 0, 0, 0.12),
    0 8px 32px rgba(0, 0, 0, 0.28) !important;
}
</style>
