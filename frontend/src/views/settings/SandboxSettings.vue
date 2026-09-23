<template>
  <div class="sandbox-settings">
    <div class="section-header">
      <div class="section-header__top">
        <div>
          <div class="section-header__titlewrap">
            <h2>{{ 'Sandbox Config' }}</h2>
            <t-popup placement="bottom-start" trigger="hover" :overlay-inner-style="{ maxWidth: '380px' }">
              <button type="button" class="hint-trigger"
                :aria-label="'What is a sandbox?'">
                <t-icon name="info-circle" size="16px" />
              </button>
              <template #content>
                <div class="hint-popover">
                  <p class="hint-popover__title">{{ 'What is a sandbox?' }}</p>
                  <p class="hint-popover__text">{{ 'A sandbox is the isolated environment where agent skill scripts run. A workspace can have several configs (Docker, E2B, CubeSandbox); each agent picks one. Skills are installed on the Skill Management page into the selected config\'s image. Scripts stay disabled until a config is selected.' }}</p>
                </div>
              </template>
            </t-popup>
          </div>
          <p class="section-description">{{ 'Configure isolated runtimes for agent skill scripts. Each agent picks one workspace config.' }}</p>
        </div>
        <div class="header-actions">
          <a class="header-action-link" :href="sandboxGuideUrl" target="_blank" rel="noopener noreferrer">
            <t-icon name="help-circle" />
            {{ 'Cluster setup guide' }}
          </a>
        </div>
      </div>
    </div>

    <!--
      Workspace-wide kill switch. It used to be a text button next to the type
      tabs, where it read as a filter action; as a labelled switch row it is
      clearly a persistent workspace setting instead.
    -->
    <div class="setting-row">
      <div class="setting-info">
        <label>{{ 'Allow skill scripts to run in sandboxes' }}</label>
        <p class="desc">{{ 'When off, agents in this workspace can only read skill content. Remote sandboxes already running are released once their sessions end.' }}</p>
      </div>
      <div class="setting-control">
        <!--
          Only switching execution off needs a confirmation, and binding :value
          one-way means the switch cannot flip until the change is accepted.
        -->
        <t-popconfirm v-if="!workspaceScriptsDisabled" theme="warning"
          :content="'All agents in this workspace — will no longer run skill scripts in a sandbox. They can still read skill content. Existing remote sandboxes are not destroyed automatically; end or delete the related sessions to release them. Continue?'"
          :confirm-btn="{ content: 'Disable sandbox execution', theme: 'danger' }"
          :cancel-btn="{ content: 'Cancel' }" placement="left"
          @confirm="setScriptsDisabled(true)">
          <t-switch :value="true" :loading="policySaving" />
        </t-popconfirm>
        <t-switch v-else :value="false" :loading="policySaving" @change="setScriptsDisabled(false)" />
      </div>
    </div>

    <div class="sandbox-tabs-row">
      <t-tabs v-model="activeType" class="sandbox-type-tabs">
        <t-tab-panel value="all" :label="`${'All'}(${records.length})`" />
        <t-tab-panel v-for="type in backendTypes" :key="type" :value="type"
          :label="`${backendLabel(type)}(${countByType(type)})`" />
      </t-tabs>
    </div>

    <t-loading :loading="loading" size="small" class="sandbox-list-loading">
      <t-alert
        v-if="!loading && dockerTabDisabled && filteredRecords.length > 0"
        theme="warning"
        class="sandbox-docker-banner"
        :message="'Docker sandbox is not enabled on this deployment'"
      >
        <template #description>
          <p>{{ 'A local docker.sock is equivalent to root on the host. For a single-machine private install, a system admin can enable it under Settings → System settings → Network security.' }}</p>
        </template>
      </t-alert>
      <div v-if="!loading && dockerTabDisabled && filteredRecords.length === 0" class="sandbox-docker-disabled">
        <t-empty :description="'Docker sandbox is not enabled on this deployment'" />
        <p class="sandbox-empty-hint">{{ 'A local docker.sock is equivalent to root on the host. For a single-machine private install, a system admin can enable it under Settings → System settings → Network security.' }}</p>
      </div>
      <div v-else-if="!loading" class="sandbox-grid">
        <div v-for="record in filteredRecords" :key="record.id" class="sandbox-card"
          :class="[`sandbox-card--${record.sandbox_type}`, { 'sandbox-card--clickable': !isLegacyRecord(record) }]"
          :role="isLegacyRecord(record) ? undefined : 'button'"
          :tabindex="isLegacyRecord(record) ? undefined : 0"
          @click="openCard(record)" @keydown.enter="openCard(record)">
          <SandboxBackendBadge :type="record.sandbox_type" />
          <div class="sandbox-card__body">
            <div class="sandbox-card__header">
              <h3 class="sandbox-card__title" :title="record.name">{{ record.name }}</h3>
              <t-tag v-if="isLegacyRecord(record)" theme="warning" variant="light" size="small">
                {{ 'Deprecated' }}
              </t-tag>
              <div class="sandbox-card__actions" @click.stop>
                <t-dropdown
                  :options="cardMenu(record)"
                  placement="bottom-right"
                  attach="body"
                  trigger="click"
                  @click="(data: any) => onMenuAction(data.value, record)"
                >
                  <t-button variant="text" shape="square" size="small" class="sandbox-card__more">
                    <t-icon name="ellipsis" />
                  </t-button>
                </t-dropdown>
              </div>
            </div>
            <div class="sandbox-card__subtitle">
              <span class="sandbox-card__type">{{ backendLabel(record.sandbox_type) }}</span>
              <template v-if="record.description">
                <span class="sandbox-card__sep">·</span>
                <span class="sandbox-card__desc" :title="record.description">{{ record.description }}</span>
              </template>
            </div>
            <div v-if="targetSummary(record)" class="sandbox-card__url" :title="targetSummary(record)">
              {{ targetSummary(record) }}
            </div>
            <ul v-if="cardWarnings[record.id]?.length" class="sandbox-card__warnings">
              <li v-for="item in cardWarnings[record.id]" :key="item.key">
                <t-icon name="error-circle" size="12px" />
                <span>{{ item.text }}</span>
              </li>
            </ul>
          </div>
        </div>
        <button v-if="canCreateOnTab" type="button" class="sandbox-card sandbox-card--add" @click="openCreate">
          <span class="sandbox-card--add__icon" aria-hidden="true"><t-icon name="add" /></span>
          <span class="sandbox-card--add__label">{{ 'Add sandbox' }}</span>
        </button>
      </div>
      <p v-if="!loading && !dockerTabDisabled && records.length === 0" class="sandbox-empty-hint">
        {{ 'No sandbox yet. Agents without a workspace configuration will not run skill scripts.' }}
      </p>
    </t-loading>

    <SandboxConfigEditorDrawer v-model:visible="showEditor" :record="editingRecord"
      :preset-type="createPresetType"
      @saved="load" />

    <!--
      Same SettingDrawer chrome as the skill editor. The list is chat sessions
      that still hold a live sandbox for this config — their titles often look
      like skill names, so the section label has to say "session".
    -->
    <SettingDrawer
      v-model:visible="showInventory"
      :title="'Running instances'"
      :description="inventorySubtitle"
      width="400px"
      :resizable="false"
      storage-key="setting-drawer:width:sandbox-inventory"
      :hide-footer="inventoryNotice !== 'unverifiable'"
    >
      <t-loading :loading="inventoryLoading" size="small">
        <div v-if="inventoryNotice === 'blocked'" class="inventory-banner">
          <t-alert theme="warning"
            :message="`This config still owns ${inventory?.sandbox_count ?? 0} running or paused sandbox(es), so identity fields cannot be changed yet.`">
            <template #description>
              <p>{{ 'End or delete those sessions (deleting a session destroys its sandbox), or create a second config and point the agents at it.' }}</p>
            </template>
          </t-alert>
        </div>
        <div v-else-if="inventory?.unverifiable" class="inventory-banner">
          <t-alert theme="warning" :message="'The backend is unreachable, so the sandbox count is unknown — that is not the same as none.'" />
        </div>

        <section class="setting-drawer__section inventory-section">
          <h4 class="setting-drawer__section-title">{{ 'Sessions' }}</h4>
          <ul v-if="inventorySessions.length" class="inventory-list">
            <li v-for="id in inventorySessions" :key="id">
              <button type="button" class="inventory-row" @click="openSession(id)">
                <span class="inventory-row__text">
                  <span class="inventory-row__title" :title="id">{{ sessionTitle(id) }}</span>
                  <span class="inventory-row__meta">{{ 'Conversation' }}</span>
                </span>
                <t-icon name="chevron-right" size="16px" />
              </button>
            </li>
          </ul>
          <p v-else-if="!inventoryLoading" class="inventory-empty">
            {{ 'No occupying sessions' }}
          </p>
        </section>

        <section v-if="inventoryAgents.length" class="setting-drawer__section inventory-section">
          <h4 class="setting-drawer__section-title">{{ 'Linked agents' }}</h4>
          <p class="inventory-agents">{{ inventoryAgents.join(', ') }}</p>
        </section>
      </t-loading>

      <!--
        Force delete only surfaces where its justification is on screen: the
        occupancy could not be verified, so the admin decides with the
        unverifiable warning above the button.
      -->
      <template v-if="inventoryNotice === 'unverifiable' && inventoryRecord" #footer-right>
        <t-popconfirm theme="warning" :content="'The backend cannot be reached to verify whether sandboxes remain. If it is gone for good you can force-delete this config; if it is only temporarily unreachable, forcing it leaves any remaining sandboxes with nobody to reclaim them. Force delete anyway?'"
          :confirm-btn="{ content: 'Force delete', theme: 'danger' }"
          :cancel-btn="{ content: 'Cancel' }" placement="top"
          @confirm="forceRemove(inventoryRecord)">
          <t-button theme="danger">
            {{ 'Force delete' }}
          </t-button>
        </t-popconfirm>
      </template>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import SandboxConfigEditorDrawer from '@/components/SandboxConfigEditorDrawer.vue'
import SandboxBackendBadge from '@/components/settings/SandboxBackendBadge.vue'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import { useConfirmDelete } from '@/components/settings/useConfirmDelete'
import { getSession } from '@/api/chat/index'
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities'
import {
  deleteSandboxConfig,
  getSandboxConfigInventory,
  isNamedSandboxBackend,
  listSandboxConfigs,
  NAMED_SANDBOX_BACKEND_TYPES,
  parseSandboxConflict,
  setSandboxWorkspacePolicy,
  type SandboxConfigRecord,
  type SandboxInventory,
} from '@/api/system'

const SETTINGS_SANDBOX_BACKENDS_LABELS: Record<string, string> = {
  disabled: 'Disabled',
  local: 'Local process',
  docker: 'Docker',
  cube: 'CubeSandbox',
  e2b: 'E2B',
}

const router = useRouter()
const confirmDelete = useConfirmDelete()
const deploymentCapabilities = useDeploymentCapabilitiesStore()
const dockerBackendEnabled = computed(() =>
  deploymentCapabilities.isSupported('settings.sandbox.docker'),
)

const sandboxGuideUrl = 'https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/README.md'

const backendTypes = [...NAMED_SANDBOX_BACKEND_TYPES]

const dockerTabDisabled = computed(() =>
  activeType.value === 'docker' && !dockerBackendEnabled.value,
)

const canCreateOnTab = computed(() => !dockerTabDisabled.value)

const createPresetType = computed(() => {
  if (activeType.value === 'all') return ''
  if (dockerTabDisabled.value) return ''
  return activeType.value
})

const loading = ref(false)
const policySaving = ref(false)
const workspaceScriptsDisabled = ref(false)
const records = ref<SandboxConfigRecord[]>([])
const activeType = ref<string>('all')

const showEditor = ref(false)
const editingRecord = ref<SandboxConfigRecord | null>(null)

const showInventory = ref(false)
const inventoryLoading = ref(false)
const inventory = ref<SandboxInventory | null>(null)
const inventoryRecord = ref<SandboxConfigRecord | null>(null)
// Why the drawer is open: a plain look, or a delete the server refused.
const inventoryNotice = ref<'' | 'blocked' | 'unverifiable'>('')
// Agent names per config, filled in when a delete confirmation opens.
const deleteAgents = ref<Record<string, string[]>>({})

const backendLabel = (value: string) => (SETTINGS_SANDBOX_BACKENDS_LABELS[value] ?? '')

const isLegacyRecord = (record: SandboxConfigRecord) => !isNamedSandboxBackend(record.sandbox_type)

const filteredRecords = computed(() => {
  const base = activeType.value === 'all'
    ? records.value
    : records.value.filter((r) => r.sandbox_type === activeType.value)
  return base
})

const countByType = (type: string) =>
  records.value.filter((r) => r.sandbox_type === type && isNamedSandboxBackend(r.sandbox_type)).length

type CardMenuOption = { content: string; value: string; theme?: 'error' }

// Card click opens the connection editor. Skills live on their own settings page.
const cardMenu = (record: SandboxConfigRecord): CardMenuOption[] => {
  if (isLegacyRecord(record)) {
    return [{ content: 'Delete', value: 'delete', theme: 'error' }]
  }
  const options: CardMenuOption[] = [
    { content: 'Edit', value: 'edit' },
  ]
  if (record.sandbox_type === 'cube' || record.sandbox_type === 'e2b') {
    options.push({ content: 'Running instances', value: 'inventory' })
  }
  options.push({ content: 'Delete', value: 'delete', theme: 'error' })
  return options
}

const inventorySubtitle = computed(() => {
  const record = inventoryRecord.value
  if (!record) return ''
  return `${record.name} · ${backendLabel(record.sandbox_type)}`
})

const inventorySessions = computed(() => inventory.value?.session_ids || [])
const inventoryAgents = computed(() => inventory.value?.agent_names || [])

const sessionTitles = ref<Record<string, string>>({})

function sessionTitle(id: string) {
  const title = sessionTitles.value[id]
  if (title) return title
  return 'Untitled session'
}

async function loadSessionTitles(ids: string[]) {
  const unique = [...new Set(ids.filter(Boolean))]
  if (!unique.length) {
    sessionTitles.value = {}
    return
  }
  const next: Record<string, string> = {}
  await Promise.all(unique.map(async (id) => {
    try {
      const res: any = await getSession(id)
      next[id] = String(res?.data?.title || '').trim()
    } catch {
      next[id] = ''
    }
  }))
  sessionTitles.value = next
}

function openSession(id: string) {
  showInventory.value = false
  router.push(`/platform/chat/${encodeURIComponent(id)}`)
}

// The endpoint host is what tells two configs of the same backend apart at a
// glance, which is the whole point of allowing several of them.
function endpointHost(record: SandboxConfigRecord): string {
  const raw = record.config?.e2b?.api_url || record.config?.cube?.api_url || ''
  if (!raw) return ''
  try {
    return new URL(raw).host
  } catch {
    return raw
  }
}

// Agent references never block deletion, but the admin has to see which agents
// will start failing — otherwise the breakage is discovered mid-conversation.
// The lookup happens when the confirmation opens, so the warning appears in the
// same popup without every card probing its backend on page load.
function deleteConfirmText(record: SandboxConfigRecord): string {
  const agents = deleteAgents.value[record.id]
  if (agents?.length) {
    return `Delete sandbox "${record.name}"? ${`Agents using this config: ${agents.join(', ')}` + ' '}They will fail on their next skill execution.`
  }
  return `Delete sandbox "${record.name}"?`
}

async function onDeleteConfirmOpen(visible: boolean, record: SandboxConfigRecord) {
  if (!visible || deleteAgents.value[record.id]) return
  try {
    const res = await getSandboxConfigInventory(record.id)
    deleteAgents.value = { ...deleteAgents.value, [record.id]: res?.data?.agent_names || [] }
  } catch {
    // An unreachable backend must not stop the admin from trying to delete.
  }
}

function openCreate() {
  if (!canCreateOnTab.value) return
  editingRecord.value = null
  showEditor.value = true
}

function openEdit(record: SandboxConfigRecord) {
  if (isLegacyRecord(record)) return
  editingRecord.value = record
  showEditor.value = true
}

function openCard(record: SandboxConfigRecord) {
  openEdit(record)
}

// What this config actually points at: the remote host or the container image.
// Two configs of the same backend are told apart by it.
function targetSummary(record: SandboxConfigRecord): string {
  if (record.sandbox_type === 'docker') {
    return record.config?.docker?.image || ''
  }
  return endpointHost(record)
}

interface CardWarning {
  key: string
  text: string
}

const REMOTE_BACKENDS = new Set(['cube', 'e2b'])

// Cards only surface blockers: healthy defaults (template present, timeout,
// TTL, env count) belong in the editor, not on every tile. Nothing here
// probes a provider; liveness stays behind the card menu.
const cardWarnings = computed<Record<string, CardWarning[]>>(() => {
  const map: Record<string, CardWarning[]> = {}
  for (const record of records.value) {
    map[record.id] = buildCardWarnings(record)
  }
  return map
})

function buildCardWarnings(record: SandboxConfigRecord): CardWarning[] {
  const warnings: CardWarning[] = []
  const config = record.config || {}
  const remote = config.cube || config.e2b

  if (REMOTE_BACKENDS.has(record.sandbox_type)) {
    if (!remote?.template_id?.trim()) {
      warnings.push({
        key: 'template',
        text: 'Template not configured',
      })
    }
    // Cube API keys are optional; only E2B fails at runtime without one.
    if (record.sandbox_type === 'e2b' && !remote?.api_key?.trim()) {
      warnings.push({
        key: 'credential',
        text: 'API key missing',
      })
    }
  }
  if (record.sandbox_type === 'docker' && !dockerBackendEnabled.value) {
    warnings.push({
      key: 'docker-disabled',
      text: 'Docker sandbox is disabled on this deployment; this config will not create containers',
    })
  }
  if (record.sandbox_type === 'docker' && !config.docker?.image?.trim()) {
    warnings.push({
      key: 'image',
      text: 'Image not configured',
    })
  }
  return warnings
}

async function load() {
  loading.value = true
  try {
    const res = await listSandboxConfigs()
    records.value = res?.data || []
    workspaceScriptsDisabled.value = res?.workspace_scripts_disabled === true
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to load sandbox configuration')
  } finally {
    loading.value = false
  }
}

async function setScriptsDisabled(disabled: boolean) {
  policySaving.value = true
  try {
    const res = await setSandboxWorkspacePolicy(disabled)
    workspaceScriptsDisabled.value = res?.workspace_scripts_disabled === true
    MessagePlugin.success(
      disabled ? 'Sandbox execution disabled for this workspace' : 'Sandbox execution restored for this workspace',
    )
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to update sandbox execution policy')
  } finally {
    policySaving.value = false
  }
}

async function onMenuAction(action: string, record: SandboxConfigRecord) {
  if (action === 'edit') {
    openEdit(record)
    return
  }
  if (action === 'inventory') {
    await openInventory(record)
    return
  }
  if (action === 'delete') {
    void confirmRemove(record)
  }
}

async function confirmRemove(record: SandboxConfigRecord) {
  await onDeleteConfirmOpen(true, record)
  confirmDelete({
    body: deleteConfirmText(record),
    onConfirm: () => removeRecord(record),
  })
}

async function openInventory(record: SandboxConfigRecord) {
  inventoryRecord.value = record
  inventoryNotice.value = ''
  showInventory.value = true
  inventoryLoading.value = true
  inventory.value = null
  sessionTitles.value = {}
  try {
    const res = await getSandboxConfigInventory(record.id)
    inventory.value = res?.data || null
    await loadSessionTitles(inventory.value?.session_ids || [])
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to load sandbox usage')
    showInventory.value = false
  } finally {
    inventoryLoading.value = false
  }
}

async function removeRecord(record: SandboxConfigRecord, force = false) {
  try {
    await deleteSandboxConfig(record.id, force)
    MessagePlugin.success('Deleted')
    showInventory.value = false
    await load()
  } catch (e: any) {
    const refusal = parseSandboxConflict(e)
    // Both refusals are about what the config still occupies, so both land in
    // the occupancy drawer where that state is spelled out. Counted sandboxes
    // are never overridable — forcing the row away would leave paused instances
    // nobody can reach, billing indefinitely — so only the unverifiable case
    // offers a force option there.
    if (refusal?.code === 'sandboxes_still_live') {
      showRefusal(record, 'blocked', refusal.inventory)
      return
    }
    if (refusal?.code === 'sandbox_inventory_unverifiable') {
      showRefusal(record, 'unverifiable', refusal.inventory)
      return
    }
    MessagePlugin.error(e?.message || 'Failed to delete')
  }
}

function showRefusal(
  record: SandboxConfigRecord,
  notice: 'blocked' | 'unverifiable',
  inv?: SandboxInventory,
) {
  inventoryRecord.value = record
  inventoryNotice.value = notice
  inventory.value = inv || { sandbox_count: 0, unverifiable: notice === 'unverifiable' }
  inventoryLoading.value = false
  showInventory.value = true
  void loadSessionTitles(inventory.value.session_ids || [])
}

async function forceRemove(record: SandboxConfigRecord) {
  await removeRecord(record, true)
}

onMounted(() => {
  void deploymentCapabilities.ensureLoaded()
  load()
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/provider-card.less';

@import (reference) '@/components/css/settings-section.less';

.sandbox-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.defaults-trigger {
  --td-bg-color-container-hover: transparent;
  flex-shrink: 0;
  padding-left: 0;
  padding-right: 0;
  font-weight: 600;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 18px;
  flex-shrink: 0;
}

.header-action-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--td-brand-color);
  font-size: var(--app-text-base);
  font-weight: 600;
  text-decoration: none;

  &:hover {
    color: var(--td-brand-color-hover);
  }
}

.section-header__titlewrap {
  display: flex;
  align-items: center;
  gap: 6px;

  h2 {
    margin-bottom: 0;
  }
}

.hint-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  border: none;
  border-radius: var(--app-radius-xs);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: help;
  line-height: 1;

  &:hover,
  &:focus-visible {
    color: var(--td-brand-color);
  }
}

.hint-popover {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-width: 340px;
}

.hint-popover__title {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.hint-popover__text {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.55;
}

.sandbox-list-loading {
  min-height: 120px;
}

.setting-row {
  .setting-row();
  gap: 24px;
}

.setting-info {
  .setting-info();
}

.setting-control {
  .setting-control();
}

.sandbox-tabs-row {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

.sandbox-type-tabs {
  flex: 1;
  min-width: 0;
  margin-bottom: 0;

  :deep(.t-tabs__nav-item) {
    font-size: var(--app-text-md);
  }

  :deep(.t-tabs__nav-item-wrapper) {
    padding: 0 12px;
    margin: 0;
  }

  :deep(.t-tabs__operations) {
    display: none;
  }

  :deep(.t-tabs__nav-scroll) {
    overflow-x: auto;
    scrollbar-width: none;

    &::-webkit-scrollbar {
      display: none;
    }
  }

  :deep(.t-tabs__content) {
    display: none;
  }
}

.sandbox-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;

  .sandbox-card--add {
    width: 100%;
    height: 100%;
  }
}

.sandbox-card {
  .provider-card();

  &--clickable {
    .provider-card-interactive();


  }

  &--add {
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    min-height: 68px;
    border-style: dashed;
    background: transparent;
    color: var(--td-text-color-placeholder);
    cursor: pointer;
    font: inherit;
    text-align: center;

    &:hover,
    &:focus-visible {
      color: var(--td-brand-color);
      border-color: var(--td-brand-color);
      box-shadow: none;
    }



    &__icon {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 32px;
      height: 32px;
      border-radius: var(--app-radius-md);
      background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
      color: var(--td-brand-color);
      font-size: var(--app-text-2xl);
    }

    &__label {
      font-size: var(--app-text-md);
      font-weight: 500;
      line-height: 1.4;
    }
  }
}

.sandbox-card__body {
  .provider-card-body();
}

.sandbox-card__header {
  .provider-card-header();
}

.sandbox-card__title {
  .provider-card-title();
}

.sandbox-card__subtitle {
  .provider-card-subtitle();
}

.sandbox-card__type {
  font-weight: 500;
}

.sandbox-card__sep {
  color: var(--td-text-color-placeholder);
}

.sandbox-card__desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.sandbox-card__url {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: var(--app-text-xs);
  line-height: 1.4;
  color: var(--td-text-color-placeholder);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.sandbox-card__warnings {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  margin: 2px 0 0;
  padding: 0;
  list-style: none;

  li {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--td-warning-color-7);
    font-size: var(--app-text-sm);
    line-height: 1.4;
  }
}

.sandbox-card__actions {
  flex-shrink: 0;
}

.sandbox-card__more {
  .provider-card-more();
}

.sandbox-card:hover .sandbox-card__more,
.sandbox-card:focus-within .sandbox-card__more,
.sandbox-card__actions:focus-within .sandbox-card__more {
  opacity: 1;
}

.sandbox-docker-banner {
  margin-bottom: 12px;
}

.sandbox-docker-disabled {
  padding: 48px 16px 24px;
  text-align: center;
}

.sandbox-empty-hint {
  margin: 16px 0 0;
  font-size: var(--app-text-md);
  color: var(--td-text-color-placeholder);
}

.sandbox-docker-disabled .sandbox-empty-hint {
  margin-top: 8px;
}

.inventory-banner {
  margin-bottom: 4px;
}

.inventory-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.inventory-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  margin: 0;
  padding: 6px 8px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;

  &:hover,
  &:focus-visible {
    background: var(--td-bg-color-container-hover);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}

.inventory-row__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.inventory-row__title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  line-height: 1.4;
}

.inventory-row__meta {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.4;
}

.inventory-row .t-icon {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
}

.inventory-empty,
.inventory-agents {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.5;
}
</style>

<style lang="less">
.setting-drawer .inventory-section.setting-drawer__section {
  padding: 0;
  gap: 8px;
  border-bottom: none;
  animation: none;
}
</style>
