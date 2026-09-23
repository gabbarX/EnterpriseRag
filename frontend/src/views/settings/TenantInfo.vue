<template>
  <div class="tenant-info">
    <div class="section-header">
      <h2>{{ 'Workspace Information' }}</h2>
      <p class="section-description">{{ 'View detailed configuration for the workspace' }}</p>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="loading-inline">
      <t-loading size="small" />
      <span>{{ 'Loading information...' }}</span>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">{{ 'Retry' }}</t-button>
        </template>
      </t-alert>
    </div>

    <div v-else class="tenant-info-body">
      <div class="settings-group">
        <!-- Tenant ID -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Workspace ID' }}</label>
            <p class="desc">{{ 'Unique identifier of your workspace' }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ tenantInfo?.id || '-' }}</span>
          </div>
        </div>

        <!-- Tenant name -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Workspace Name' }}</label>
            <p class="desc">{{ 'Name of your workspace' }}</p>
          </div>
          <div class="setting-control">
            <template v-if="!editing">
              <span class="info-value">{{ tenantInfo?.name || '-' }}</span>
              <t-button v-if="canEditTenant" theme="default" variant="text" shape="square" size="small"
                class="edit-btn" :title="'Edit name'" :aria-label="'Edit name'"
                @click="startEditName">
                <template #icon>
                  <t-icon name="edit" />
                </template>
              </t-button>
            </template>
            <div v-else class="inline-edit">
              <t-input v-model="editName" :placeholder="'Enter the new workspace name'" :maxlength="64"
                :disabled="saving" autofocus class="inline-edit-input" @enter="saveTenantName"
                @keydown="onEditKeydown" />
              <t-button theme="primary" size="small" :loading="saving" :disabled="!canSubmit" @click="saveTenantName">
                {{ 'Save' }}
              </t-button>
              <t-button theme="default" variant="outline" size="small" :disabled="saving" @click="cancelEditName">
                {{ 'Cancel' }}
              </t-button>
            </div>
          </div>
        </div>

        <!-- Tenant description -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Workspace Description' }}</label>
            <p class="desc">{{ 'Detailed description of the workspace' }}</p>
          </div>
          <div class="setting-control">
            <template v-if="!editingDescription">
              <span class="info-value description-value" :class="{ 'is-empty': !tenantInfo?.description }">
                {{ tenantInfo?.description || 'Not set' }}
              </span>
              <t-button v-if="canEditTenant" theme="default" variant="text" shape="square" size="small"
                class="edit-btn" :title="'Edit description'"
                :aria-label="'Edit description'" @click="startEditDescription">
                <template #icon>
                  <t-icon name="edit" />
                </template>
              </t-button>
            </template>
            <div v-else class="inline-edit inline-edit-description">
              <t-textarea v-model="editDescription"
                :placeholder="'Enter the new workspace description'" :maxlength="512"
                :autosize="{ minRows: 2, maxRows: 6 }" :disabled="savingDescription" autofocus
                class="inline-edit-textarea" @keydown="onEditDescriptionKeydown" />
              <div class="inline-edit-actions">
                <t-button theme="primary" size="small" :loading="savingDescription"
                  :disabled="!canSubmitDescription" @click="saveTenantDescription">
                  {{ 'Save' }}
                </t-button>
                <t-button theme="default" variant="outline" size="small" :disabled="savingDescription"
                  @click="cancelEditDescription">
                  {{ 'Cancel' }}
                </t-button>
              </div>
            </div>
          </div>
        </div>

        <!-- Tenant business -->
        <div v-if="tenantInfo?.business" class="setting-row">
          <div class="setting-info">
            <label>{{ 'Workspace Business' }}</label>
            <p class="desc">{{ 'Business domain that the workspace belongs to' }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ tenantInfo.business }}</span>
          </div>
        </div>

        <!-- Tenant status -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Workspace Status' }}</label>
            <p class="desc">{{ 'Current operational status of the workspace' }}</p>
          </div>
          <div class="setting-control">
            <t-tag :theme="getStatusTheme(tenantInfo?.status)" variant="light" size="small">
              {{ getStatusText(tenantInfo?.status) }}
            </t-tag>
          </div>
        </div>

        <!-- Tenant creation time -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Workspace Creation Time' }}</label>
            <p class="desc">{{ 'Time when the workspace was created' }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ formatDate(tenantInfo?.created_at) }}</span>
          </div>
        </div>

        <!-- Storage quota -->
        <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
          <div class="setting-info">
            <label>{{ 'Storage Quota' }}</label>
            <p class="desc">{{ 'Total storage capacity allocated to the workspace' }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ formatBytes(tenantInfo.storage_quota) }}</span>
          </div>
        </div>

        <!-- Used storage -->
        <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
          <div class="setting-info">
            <label>{{ 'Used Storage' }}</label>
            <p class="desc">{{ 'Storage space that has been used' }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ formatBytes(tenantInfo.storage_used || 0) }}</span>
          </div>
        </div>

        <!-- Storage usage -->
        <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
          <div class="setting-info">
            <label>{{ 'Storage Usage' }}</label>
            <p class="desc">{{ 'Percentage of storage capacity used' }}</p>
          </div>
          <div class="setting-control">
            <div class="usage-control">
              <span class="usage-text">{{ getUsagePercentage() }}%</span>
              <!-- t-progress: theme picks the shape (line/plump/circle); the colour comes from status. -->
              <t-progress :percentage="getUsagePercentage()" :show-info="false" size="small"
                :status="getUsagePercentage() > 80 ? 'warning' : 'success'" style="flex: 1;" />
            </div>
          </div>
        </div>

      </div>

      <aside v-if="showLeaveDangerZone" class="leave-space-panel" :aria-label="'Leave this workspace'">
        <div class="leave-space-panel-inner">
          <div class="leave-space-panel-text">
            <div class="leave-space-panel-title">{{ 'Leave this workspace' }}</div>
            <p class="leave-space-panel-desc">{{ 'Ends your membership in this workspace. You will lose access to its knowledge bases and agents. You can be invited again later.' }}</p>
          </div>
          <div class="leave-space-panel-action">
            <t-button theme="danger" variant="outline" size="medium" @click="confirmLeaveTenant">
              {{ 'Leave workspace' }}
            </t-button>
          </div>
        </div>
      </aside>

      <aside v-if="showDeleteDangerZone" class="leave-space-panel delete-space-panel"
        :aria-label="'Delete this workspace'">
        <div class="leave-space-panel-inner">
          <div class="leave-space-panel-text">
            <div class="leave-space-panel-title">{{ 'Delete this workspace' }}</div>
            <p class="leave-space-panel-desc">{{ 'Delete the whole workspace and its configuration. Members will no longer be able to access its knowledge bases, agents, or API key.' }}</p>
          </div>
          <div class="leave-space-panel-action">
            <t-button theme="danger" size="medium" @click="confirmDeleteTenant">
              {{ 'Delete workspace' }}
            </t-button>
          </div>
        </div>
      </aside>

    </div>

    <t-dialog v-model:visible="deleteTenantVisible" :header="'Delete this workspace?'"
      :confirm-btn="{
        content: 'Delete workspace',
        theme: 'danger',
        disabled: deleteConfirmName.trim() !== (tenantInfo?.name || ''),
        loading: deletingTenant,
      }" :cancel-btn="'Cancel'" :close-on-overlay-click="!deletingTenant"
      :close-btn="!deletingTenant" @confirm="deleteCurrentTenant">
      <div class="delete-tenant-confirm">
        <p class="delete-tenant-confirm-body">
          {{ `This will delete \u0022${tenantInfo?.name || ''}\u0022 and make its knowledge bases, agents, members, and API key unavailable. This action cannot be undone.` }}
        </p>
        <p class="delete-tenant-confirm-hint">
          {{ `Type the workspace name \u0022${tenantInfo?.name || ''}\u0022 to confirm deletion.` }}
        </p>
        <t-input v-model="deleteConfirmName" :placeholder="tenantInfo?.name || ''" :disabled="deletingTenant"
          clearable />
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { getCurrentUser, type TenantInfo } from '@/api/auth'
import { deleteTenant as deleteTenantApi, updateTenant as updateTenantApi } from '@/api/tenant'
import {
  leaveTenant,
  fetchAllTenantMembers,
  type TenantMember,
  type TenantRole,
} from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import { useRoleLabel, useHomeTenant } from '@/composables/useRoleLabel'
import {
  navigateAfterTenantSwitch,
  persistLastActiveTenantPreference,
  stashTenantSwitchToast,
} from '@/utils/tenantSwitch'

const { formatRole } = useRoleLabel()
const { homeTenantId } = useHomeTenant()
const authStore = useAuthStore()

// Reactive state
const tenantInfo = ref<TenantInfo | null>(null)
const loading = ref(true)
const error = ref('')

// Only an owner may rename the workspace, mirroring the g.Owner() guard in the
// backend router. The server is always the final authority on permissions; this
// only decides whether the UI shows the entry point.
const canEditTenant = computed(() => authStore.hasRole('owner'))

/** Matches TenantMembers.vue: the last remaining Owner is not offered "leave", so the UI stays aligned with the server's last-owner rule. */
const activeTenantNumericId = computed(() => Number(authStore.currentTenantId ?? 0))

const leaveMembersSnap = ref<TenantMember[]>([])
const leaveGateReady = ref(false)
const leaveGateLoading = ref(false)

const currentTenantRole = computed<TenantRole | ''>(() => (authStore.currentTenantRole || '') as TenantRole | '')

const canLeaveSpace = computed(() => {
  const r = currentTenantRole.value
  if (!r || !tenantInfo.value?.id) return false
  if (r !== 'owner') return true
  return leaveMembersSnap.value.filter((m) => m.role === 'owner').length > 1
})

const showLeaveDangerZone = computed(() => {
  if (loading.value || error.value || !tenantInfo.value) return false
  if (!leaveGateReady.value || leaveGateLoading.value) return false
  if (!currentTenantRole.value) return false
  if (Number(tenantInfo.value.id) !== activeTenantNumericId.value) return false
  return canLeaveSpace.value
})

const showDeleteDangerZone = computed(() => {
  if (loading.value || error.value || !tenantInfo.value) return false
  if (Number(tenantInfo.value.id) !== activeTenantNumericId.value) return false
  return authStore.hasRole('owner')
})

async function evaluateLeaveGate(): Promise<void> {
  leaveGateReady.value = false
  leaveMembersSnap.value = []
  leaveGateLoading.value = false

  const infoId = tenantInfo.value?.id != null ? Number(tenantInfo.value.id) : 0
  if (!infoId || !activeTenantNumericId.value || infoId !== activeTenantNumericId.value) {
    leaveGateReady.value = true
    return
  }

  const role = currentTenantRole.value
  if (!role) {
    leaveGateReady.value = true
    return
  }
  if (role !== 'owner') {
    leaveGateReady.value = true
    return
  }

  leaveGateLoading.value = true
  try {
    leaveMembersSnap.value = await fetchAllTenantMembers(infoId)
  } finally {
    leaveGateLoading.value = false
    leaveGateReady.value = true
  }
}

function confirmLeaveTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  if (!tid) return

  const dlg = DialogPlugin.confirm({
    header: 'Leave this workspace?',
    body: 'You will lose access to all knowledge bases and agents in this workspace. You can be re-invited later.',
    confirmBtn: { content: 'Leave', theme: 'danger' },
    cancelBtn: 'Cancel',
    onConfirm: async () => {
      try {
        const resp = await leaveTenant(tid)
        if (resp.success) {
          MessagePlugin.success('You have left the workspace')
          authStore.logout()
          window.location.href = '/login'
        } else {
          MessagePlugin.error(resp.message || 'Something went wrong. Please try again.')
        }
      } catch (err: any) {
        const status = err?.status
        if (status === 409) {
          MessagePlugin.error('Cannot demote, remove, or leave as the last Owner. Promote another member to Owner first.')
        } else {
          MessagePlugin.error(err?.message || 'Something went wrong. Please try again.')
        }
      } finally {
        dlg.destroy()
      }
    },
    onClose: () => dlg.destroy(),
  })
}

function confirmDeleteTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  const tenantName = tenantInfo.value?.name || ''
  if (!tid || !tenantName) return
  deleteConfirmName.value = ''
  deleteTenantVisible.value = true
}

async function deleteCurrentTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  const tenantName = tenantInfo.value?.name || ''
  if (!tid || !tenantName) return
  if (deleteConfirmName.value.trim() !== tenantName) {
    MessagePlugin.warning('Workspace name does not match')
    return
  }
  try {
    deletingTenant.value = true
    const resp = await deleteTenantApi(tid)
    if (resp.success) {
      MessagePlugin.success('Workspace deleted')
      authStore.setMemberships(
        (authStore.memberships ?? []).filter((m) => m.tenant_id !== tid),
      )
      await authStore.refreshFromAuthMe()
      const next =
        authStore.memberships.find((m) => m.tenant_id === homeTenantId.value) ??
        authStore.memberships[0]
      if (next) {
        const switchingToHome =
          homeTenantId.value !== null && homeTenantId.value === next.tenant_id
        const name = next.tenant_name?.trim() || `#${next.tenant_id}`
        authStore.setSelectedTenant(next.tenant_id, name)
        stashTenantSwitchToast({
          name,
          role: formatRole(next.role) || undefined,
          roleEnum: next.role || undefined,
        })
        const persist = persistLastActiveTenantPreference(
          switchingToHome ? null : next.tenant_id,
        )
        await Promise.race([persist, new Promise((r) => setTimeout(r, 400))])
        navigateAfterTenantSwitch()
        return
      }
      authStore.logout()
      window.location.href = '/login'
    } else {
      MessagePlugin.error(resp.message || 'Failed to delete workspace')
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to delete workspace')
  } finally {
    deletingTenant.value = false
    deleteConfirmName.value = ''
    deleteTenantVisible.value = false
  }
}

watch(
  [() => tenantInfo.value?.id, () => authStore.currentTenantId, () => authStore.currentTenantRole],
  () => {
    if (!loading.value && tenantInfo.value && !error.value) {
      void evaluateLeaveGate()
    }
  },
)

const editing = ref(false)
const editName = ref('')
const saving = ref(false)
const deleteConfirmName = ref('')
const deleteTenantVisible = ref(false)
const deletingTenant = ref(false)
const editNameTrimmed = computed(() => editName.value.trim())
// The backend has no unique index and no duplicate-name check on the tenant name,
// so there is no "already exists" validation here; the service only rejects an
// empty name on create, so a non-empty guard on the client is enough.
const canSubmit = computed(
  () => !saving.value && !!editNameTrimmed.value && editNameTrimmed.value !== tenantInfo.value?.name,
)

const startEditName = () => {
  editName.value = tenantInfo.value?.name || ''
  editing.value = true
}

const cancelEditName = () => {
  if (saving.value) return
  editing.value = false
  editName.value = ''
}

// t-input does not bubble Escape, so handle it manually (symmetric with enter).
const onEditKeydown = (_value: any, ctx: { e: KeyboardEvent }) => {
  if (ctx?.e?.key === 'Escape') {
    cancelEditName()
  }
}

// Description may be empty (it is an optional field), so the submit condition only
// requires that the value changed, not that it is non-empty.
const editingDescription = ref(false)
const editDescription = ref('')
const savingDescription = ref(false)
const editDescriptionTrimmed = computed(() => editDescription.value.trim())
const canSubmitDescription = computed(
  () => !savingDescription.value && editDescriptionTrimmed.value !== (tenantInfo.value?.description || ''),
)

const startEditDescription = () => {
  editDescription.value = tenantInfo.value?.description || ''
  editingDescription.value = true
}

const cancelEditDescription = () => {
  if (savingDescription.value) return
  editingDescription.value = false
  editDescription.value = ''
}

const onEditDescriptionKeydown = (_value: any, ctx: { e: KeyboardEvent }) => {
  const e = ctx?.e
  if (!e) return
  if (e.key === 'Escape') {
    cancelEditDescription()
    return
  }
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    void saveTenantDescription()
  }
}

const saveTenantDescription = async () => {
  if (!tenantInfo.value?.id) return
  const newDesc = editDescriptionTrimmed.value
  if (newDesc === (tenantInfo.value.description || '')) {
    editingDescription.value = false
    return
  }

  try {
    savingDescription.value = true
    const resp = await updateTenantApi(Number(tenantInfo.value.id), { description: newDesc })
    if (resp.success) {
      // Echo locally instead of waiting for an /auth/me round trip. Unlike the
      // name, the description is not shown in the workspace switcher, so there is
      // nothing to sync into authStore.tenant / memberships.
      if (tenantInfo.value) {
        tenantInfo.value = { ...tenantInfo.value, description: newDesc }
      }
      MessagePlugin.success('Workspace description updated')
      editingDescription.value = false
    } else {
      MessagePlugin.error(resp.message || 'Failed to update workspace description')
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to update workspace description')
  } finally {
    savingDescription.value = false
  }
}

const saveTenantName = async () => {
  const newName = editNameTrimmed.value
  if (!newName) {
    MessagePlugin.warning('Workspace name cannot be empty')
    return
  }
  if (!tenantInfo.value?.id) return
  if (newName === tenantInfo.value.name) {
    editing.value = false
    return
  }

  try {
    saving.value = true
    const resp = await updateTenantApi(Number(tenantInfo.value.id), { name: newName })
    if (resp.success) {
      // Echo locally instead of waiting for an /auth/me round trip, and refresh
      // the cached tenant in the auth store so the workspace switcher picks up the
      // new name.
      if (tenantInfo.value) {
        tenantInfo.value = { ...tenantInfo.value, name: newName }
      }
      if (authStore.tenant && String(authStore.tenant.id) === String(tenantInfo.value?.id)) {
        authStore.setTenant({ ...authStore.tenant, name: newName })
      }
      // tenant_name in memberships is what the workspace switcher reads, so sync it too.
      if (authStore.memberships?.length) {
        const next = authStore.memberships.map((m) =>
          String(m.tenant_id) === String(tenantInfo.value?.id)
            ? { ...m, tenant_name: newName }
            : m,
        )
        authStore.setMemberships(next)
      }
      MessagePlugin.success('Workspace name updated')
      editing.value = false
    } else {
      MessagePlugin.error(resp.message || 'Failed to update workspace name')
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to update workspace name')
  } finally {
    saving.value = false
  }
}

// Methods
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''

    const userResponse = await getCurrentUser()

    const data = userResponse?.data as { tenant?: TenantInfo } | undefined
    if ((userResponse as any).success && data?.tenant) {
      tenantInfo.value = data.tenant
    } else {
      error.value = userResponse.message || 'Failed to fetch workspace information'
    }
  } catch (err: any) {
    error.value = err?.message || 'Network error, please try again later'
  } finally {
    loading.value = false
  }
  // Must run after loading=false, otherwise the loading guard inside
  // showLeaveDangerZone hides the leave entry; in some environments the role also
  // hydrates slightly after /auth/me returns.
  if (tenantInfo.value && !error.value) {
    await evaluateLeaveGate()
  }
}

const getStatusText = (status: string | undefined) => {
  switch (status) {
    case 'active':
      return 'Active'
    case 'inactive':
      return 'Not activated'
    case 'suspended':
      return 'Suspended'
    default:
      return 'Unknown'
  }
}

const getStatusTheme = (status: string | undefined) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'inactive':
      return 'warning'
    case 'suspended':
      return 'danger'
    default:
      return 'default'
  }
}

const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return 'Unknown'

  try {
    const date = new Date(dateStr)
    const formatter = new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
    return formatter.format(date)
  } catch {
    return 'Format error'
  }
}

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'

  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getUsagePercentage = () => {
  if (!tenantInfo.value?.storage_quota || tenantInfo.value.storage_quota === 0) {
    return 0
  }

  const used = tenantInfo.value.storage_used || 0
  const percentage = (used / tenantInfo.value.storage_quota) * 100
  return Math.min(Math.round(percentage * 100) / 100, 100)
}

// Lifecycle
onMounted(() => {
  loadInfo()
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.tenant-info {
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

.tenant-info-body {
  display: flex;
  flex-direction: column;
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
  flex: 0 0 auto;
  width: max-content;
  min-width: 140px;
  max-width: 40%;
  padding-right: 24px;

  label {
    font-size: var(--app-text-lg);
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  .setting-control();
  flex: 1 1 auto;
  min-width: 0;

  .info-value {
    font-size: var(--app-text-base);
    color: var(--td-text-color-primary);
    text-align: right;
    /* anywhere rather than break-word: long unbroken strings still wrap instead
       of stretching the whole row. */
    overflow-wrap: anywhere;
    min-width: 0;
  }

  .edit-btn {
    flex-shrink: 0;
  }
}

.inline-edit {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  justify-content: flex-end;
}

.inline-edit-input {
  max-width: 220px;
  flex: 1;
}

.inline-edit-description {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  width: 100%;
  max-width: 360px;
}

.inline-edit-textarea {
  width: 100%;
}

.inline-edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.description-value {
  white-space: pre-wrap;
  word-break: break-word;

  &.is-empty {
    color: var(--td-text-color-placeholder);
  }
}

.leave-space-panel {
  margin-top: 4px;
}

.delete-space-panel {
  margin-top: 12px;
}

.leave-space-panel-inner {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 18px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--td-component-stroke);
  background-color: var(--td-bg-color-secondarycontainer);
  box-sizing: border-box;
}

.leave-space-panel-text {
  flex: 1;
  min-width: 0;
  max-width: min(65%, 28rem);
  padding-right: 8px;
}

.leave-space-panel-title {
  font-size: var(--app-text-lg);
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;
  margin-bottom: 4px;
}

.leave-space-panel-desc {
  margin: 0;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.leave-space-panel-action {
  flex-shrink: 0;
}

@media (max-width: 560px) {
  .leave-space-panel-inner {
    flex-direction: column;
    align-items: stretch;
  }

  .leave-space-panel-text {
    max-width: none;
    padding-right: 0;
  }

  .leave-space-panel-action {
    display: flex;
    justify-content: flex-end;
  }
}

.usage-control {
  //   width: 100%;
  //   display: flex;
  //   align-items: center;
  //   gap: 12px;

  .usage-text {
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--td-text-color-primary);
    min-width: 50px;
    text-align: right;
  }
}

.delete-tenant-confirm-body {
  margin: 0 0 10px;
  color: var(--td-text-color-primary);
  line-height: 1.6;
}

.delete-tenant-confirm-hint {
  margin: 0 0 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}
</style>
