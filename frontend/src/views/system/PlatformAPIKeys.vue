<template>
  <div class="platform-api-keys">
    <header class="section-header">
      <h2>{{ 'Platform API Keys' }}</h2>
      <p class="section-description">{{ 'Create platform credentials for cross-workspace automation. Use X-Tenant-ID for workspace APIs.' }}</p>
    </header>

    <t-alert theme="warning" :message="'Platform API keys can target any workspace. Grant only required capabilities; plaintext is shown once.'" class="security-alert">
      <template #operation>
        <t-button size="small" variant="outline" @click="openCreate">
          <template #icon><t-icon name="add" /></template>
          {{ 'Create platform API key' }}
        </t-button>
      </template>
    </t-alert>

    <section class="keys-section">
      <div v-if="loading" class="keys-state">
        <t-loading size="small" />
        <span>{{ 'Loading…' }}</span>
      </div>
      <div v-else-if="keys.length === 0" class="keys-state keys-state--empty">
        <span>{{ 'No platform API keys' }}</span>
        <t-button size="small" variant="outline" @click="openCreate">
          <template #icon><t-icon name="add" /></template>
          {{ 'Create platform API key' }}
        </t-button>
      </div>
      <div v-else class="api-key-table-wrap">
        <table class="api-key-table">
          <thead>
            <tr>
              <th>{{ 'Name' }}</th>
              <th>{{ 'Key' }}</th>
              <th>{{ 'Capabilities' }}</th>
              <th>{{ 'Last used' }}</th>
              <th>{{ 'Created' }}</th>
              <th class="api-key-table__actions-heading">{{ 'Actions' }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="key in keys" :key="key.id">
              <td>
                <span class="api-key-name">{{ key.name }}</span>
              </td>
              <td>
                <code class="api-key-fingerprint">{{ key.api_key }}</code>
              </td>
              <td class="api-key-table__capability-cell">
                <div class="api-key-capability-inline">
                  <span
                    v-for="chip in visibleCapabilityChips(key)"
                    :key="chip.id"
                    class="api-key-capability-chip"
                  >
                    {{ chip.label }}
                  </span>
                  <t-popup
                    v-if="hiddenCapabilityCount(key) > 0"
                    trigger="click"
                    placement="bottom-left"
                    destroy-on-close
                    overlay-class-name="platform-api-key-capability-popup-overlay"
                  >
                    <button
                      type="button"
                      class="api-key-capability-chip api-key-capability-chip--more"
                      :aria-label="'View all capabilities'"
                    >
                      {{ `+${hiddenCapabilityCount(key)}` }}
                    </button>
                    <template #content>
                      <div class="api-key-capability-popup">
                        <div class="api-key-capability-popup__title">{{ 'Capabilities' }}</div>
                        <div
                          v-for="group in capabilityGroupsForKey(key)"
                          :key="group.key"
                          class="api-key-capability-block"
                        >
                          <div class="api-key-capability-block__title">{{ group.label }}</div>
                          <div class="api-key-capability-block__chips">
                            <span
                              v-for="label in group.labels"
                              :key="label"
                              class="api-key-capability-chip"
                            >
                              {{ label }}
                            </span>
                          </div>
                        </div>
                      </div>
                    </template>
                  </t-popup>
                </div>
              </td>
              <td>
                <span class="api-key-meta">{{ formatDate(key.last_used_at) }}</span>
              </td>
              <td>
                <time class="api-key-date" :datetime="key.created_at">{{ formatDate(key.created_at) }}</time>
              </td>
              <td>
                <div class="api-key-table__actions">
                  <t-popconfirm
                    :content="`Delete “${key.name}”? Automation using this key will stop immediately.`"
                    :confirm-btn="{ content: 'Delete', theme: 'danger' }"
                    :cancel-btn="{ content: 'Cancel' }"
                    placement="bottom-right"
                    @confirm="deleteKey(key)"
                  >
                    <t-button
                      shape="square"
                      variant="text"
                      theme="danger"
                      :title="'Delete'"
                      @click.stop
                    >
                      <t-icon name="delete" />
                    </t-button>
                  </t-popconfirm>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <SettingDrawer
      :visible="drawerVisible"
      class="api-key-create-drawer"
      :title="'Create platform API key'"
      :description="'Platform keys are not bound to a workspace; capabilities still limit every operation.'"
      icon="secured"
      width="560px"
      :min-width="480"
      :max-width="920"
      storage-key="setting-drawer:width:platform-api-key-create"
      :close-on-overlay-click="false"
      :confirm-text="'Create platform API key'"
      :confirm-loading="creating"
      @update:visible="drawerVisible = $event"
      @confirm="createKey"
    >
      <div class="api-key-dialog">
        <div class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Name' }}</label>
          </div>
          <t-input v-model="form.name" :placeholder="'For example: central operations automation'" />
        </div>

        <div class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Capabilities' }}</label>
          </div>
          <p class="scope-hint">{{ 'Workspace capabilities apply to the X-Tenant-ID target; system capabilities apply to control-plane APIs.' }}</p>
          <div class="api-key-capability-list">
            <div
              v-for="group in PLATFORM_API_KEY_CAPABILITY_GROUPS"
              :key="group.key"
              class="api-key-capability-group"
            >
              <div class="api-key-capability-group__header">
                <span>{{ group.labelKey }}</span>
                <t-button
                  size="small"
                  variant="text"
                  @click="toggleGroup(group.capabilities.map(item => item.value))"
                >
                  {{
                    groupSelected(group.capabilities.map(item => item.value))
                      ? 'Clear'
                      : 'Select all'
                  }}
                </t-button>
              </div>
              <div class="api-key-capability-group__items">
                <div
                  v-for="item in group.capabilities"
                  :key="item.value"
                  class="api-key-capability-item"
                >
                  <t-checkbox v-model="selected[item.value]">{{ item.labelKey }}</t-checkbox>
                  <p class="scope-hint">{{ item.hintKey }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </SettingDrawer>

    <t-dialog
      v-model:visible="tokenVisible"
      :header="'Platform API key created'"
      :confirm-btn="{ content: 'Copy key', theme: 'primary' }"
      :cancel-btn="null"
      :close-on-overlay-click="false"
      @confirm="copyToken"
    >
      <p>{{ 'Copy and store this key now. The full value will not be shown again.' }}</p>
      <t-textarea :value="createdToken" readonly autosize />
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { copyWithToast } from '@/utils/clipboard'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import type { TenantAPIKey, TenantAPIKeyCapability } from '@/api/tenant'
import { createPlatformAPIKey, deletePlatformAPIKey, listPlatformAPIKeys } from '@/api/system'
import {
  PLATFORM_API_KEY_CAPABILITY_GROUPS,
  SYSTEM_API_KEY_CAPABILITIES,
  TENANT_API_KEY_CAPABILITIES,
} from '@/config/apiKeyCapabilities'

const allCapabilities = [...SYSTEM_API_KEY_CAPABILITIES, ...TENANT_API_KEY_CAPABILITIES]
const keys = ref<TenantAPIKey[]>([])
const loading = ref(false)
const creating = ref(false)
const drawerVisible = ref(false)
const tokenVisible = ref(false)
const createdToken = ref('')
const form = reactive({ name: '' })
const selected = reactive<Record<TenantAPIKeyCapability, boolean>>(
  allCapabilities.reduce((result, capability) => {
    result[capability] = false
    return result
  }, {} as Record<TenantAPIKeyCapability, boolean>),
)

const VISIBLE_CAPABILITY_CHIP_COUNT = 4

type CapabilityChipView = {
  id: TenantAPIKeyCapability
  label: string
}

function capabilityChipsForKey(key: TenantAPIKey): CapabilityChipView[] {
  const chips: CapabilityChipView[] = []
  for (const group of PLATFORM_API_KEY_CAPABILITY_GROUPS) {
    for (const item of group.capabilities) {
      if (key.capabilities?.includes(item.value)) {
        chips.push({ id: item.value, label: item.labelKey })
      }
    }
  }
  return chips
}

function visibleCapabilityChips(key: TenantAPIKey) {
  return capabilityChipsForKey(key).slice(0, VISIBLE_CAPABILITY_CHIP_COUNT)
}

function hiddenCapabilityCount(key: TenantAPIKey) {
  const total = capabilityChipsForKey(key).length
  return Math.max(0, total - VISIBLE_CAPABILITY_CHIP_COUNT)
}

function capabilityGroupsForKey(key: TenantAPIKey) {
  return PLATFORM_API_KEY_CAPABILITY_GROUPS
    .map(group => {
      const labels = group.capabilities
        .filter(item => key.capabilities?.includes(item.value))
        .map(item => item.labelKey)
      if (labels.length === 0) return null
      return { key: group.key, label: group.labelKey, labels }
    })
    .filter((group): group is { key: string; label: string; labels: string[] } => group !== null)
}

function formatDate(value?: string) {
  if (!value) return 'Never'
  const date = new Date(value)
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function resetForm() {
  form.name = ''
  allCapabilities.forEach(capability => { selected[capability] = false })
}

function openCreate() {
  resetForm()
  drawerVisible.value = true
}

function groupSelected(capabilities: TenantAPIKeyCapability[]) {
  return capabilities.every(capability => selected[capability])
}

function toggleGroup(capabilities: TenantAPIKeyCapability[]) {
  const next = !groupSelected(capabilities)
  capabilities.forEach(capability => { selected[capability] = next })
}

async function reload() {
  loading.value = true
  try {
    const response = await listPlatformAPIKeys()
    keys.value = response.data ?? []
  } catch (error: any) {
    MessagePlugin.error(error?.message || 'Failed to load platform API keys')
  } finally {
    loading.value = false
  }
}

async function createKey() {
  const capabilities = allCapabilities.filter(capability => selected[capability])
  if (!form.name.trim()) {
    MessagePlugin.warning('Enter a name')
    return
  }
  if (capabilities.length === 0) {
    MessagePlugin.warning('Select at least one capability')
    return
  }
  creating.value = true
  try {
    const response = await createPlatformAPIKey({ name: form.name.trim(), capabilities })
    createdToken.value = response.data?.token ?? ''
    drawerVisible.value = false
    tokenVisible.value = true
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || 'Failed to create platform API key')
  } finally {
    creating.value = false
  }
}

async function deleteKey(key: TenantAPIKey) {
  try {
    await deletePlatformAPIKey(key.id)
    MessagePlugin.success('Platform API key deleted')
    await reload()
  } catch (error: any) {
    MessagePlugin.error(error?.message || 'Failed to delete platform API key')
  }
}

async function copyToken() {
  const ok = await copyWithToast(createdToken.value, 'Key copied')
  if (ok) tokenVisible.value = false
}

onMounted(reload)
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.platform-api-keys {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.section-header h2 {
  margin: 0 0 8px;
  font-size: var(--app-text-3xl);
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.security-alert {
  margin-bottom: 20px;
}

.keys-section {
  border: 1px solid var(--td-component-border);
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-container);
  overflow: hidden;
}

.keys-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 120px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
}

.keys-state--empty {
  flex-direction: column;
  gap: 12px;
}

.api-key-table-wrap {
  width: 100%;
  overflow-x: auto;
}

.api-key-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.api-key-table th,
.api-key-table td {
  padding: 13px 14px;
  border-bottom: 1px solid var(--td-component-stroke);
  text-align: left;
  vertical-align: middle;
}

.api-key-table th {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  font-weight: 500;
  line-height: 1.4;
}

.api-key-table td {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.45;
}

.api-key-table th:nth-child(1),
.api-key-table td:nth-child(1) {
  width: 11%;
}

.api-key-table th:nth-child(2),
.api-key-table td:nth-child(2) {
  width: 17%;
}

.api-key-table th:nth-child(3),
.api-key-table td:nth-child(3) {
  width: auto;
}

.api-key-table th:nth-child(4),
.api-key-table td:nth-child(4) {
  width: 80px;
}

.api-key-table th:nth-child(5),
.api-key-table td:nth-child(5) {
  width: 128px;
}

.api-key-table th:nth-child(6),
.api-key-table td:nth-child(6) {
  width: 52px;
}

.api-key-table__capability-cell {
  vertical-align: middle;
}

.api-key-capability-inline {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 5px;
}

.api-key-capability-chip--more {
  border: 1px dashed color-mix(in srgb, var(--td-success-color) 35%, transparent);
  background: transparent;
  cursor: pointer;
}

.api-key-capability-popup {
  width: 320px;
  max-width: min(360px, 88vw);
  max-height: 360px;
  overflow: auto;
  padding: 12px 14px;
}

.api-key-capability-popup__title {
  margin-bottom: 10px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
  line-height: 1.4;
}

.api-key-table tbody tr:last-child td {
  border-bottom: none;
}

.api-key-table__actions-heading {
  text-align: right !important;
}

.api-key-table__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.api-key-capability-block + .api-key-capability-block {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--td-component-stroke);
}

.api-key-capability-block__title {
  margin-bottom: 6px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-sm);
  font-weight: 600;
  line-height: 1.4;
}

.api-key-capability-block__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.api-key-capability-chip {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: var(--app-radius-sm);
  background: color-mix(in srgb, var(--td-success-color) 10%, var(--td-bg-color-container));
  color: var(--td-success-color);
  font-size: var(--app-text-sm);
  font-weight: 500;
  line-height: 20px;
  white-space: nowrap;
}

.api-key-name {
  display: block;
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-fingerprint {
  display: inline-block;
  max-width: 100%;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-sm);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: top;
  white-space: nowrap;
}

.api-key-meta {
  display: block;
  min-width: 0;
  font-size: var(--app-text-sm);
  line-height: 1.4;
  white-space: nowrap;
}

.api-key-date {
  display: block;
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-sm);
  line-height: 1.4;
  white-space: nowrap;
  color: var(--td-text-color-secondary);
}

.api-key-dialog {
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.api-key-dialog-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 0 16px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.api-key-dialog-row:first-child {
  padding-top: 0;
}

.api-key-dialog-row:last-child {
  border-bottom: none;
}

.api-key-dialog-row__label label {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  font-weight: 600;
  line-height: 1.45;
}

.api-key-dialog-row__label label::before {
  content: '';
  flex-shrink: 0;
  width: 3px;
  height: 14px;
  border-radius: 2px;
  background: var(--td-brand-color);
}

.api-key-dialog-row :deep(.t-input) {
  border-radius: var(--app-radius-xs);
  background-color: var(--td-bg-color-secondarycontainer);
  border-color: transparent;
  box-shadow: none !important;
}

.api-key-dialog-row :deep(.t-input:hover),
.api-key-dialog-row :deep(.t-input.t-is-focused) {
  border-color: var(--td-component-border);
  background-color: var(--td-bg-color-container);
}

.scope-hint {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 18px;
}

.api-key-capability-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.api-key-capability-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 0 2px;
}

.api-key-capability-group + .api-key-capability-group {
  border-top: 1px solid var(--td-component-stroke);
}

.api-key-capability-group__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 24px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.api-key-capability-group__items {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.api-key-capability-item .scope-hint {
  margin: 2px 0 0 24px;
}
</style>
