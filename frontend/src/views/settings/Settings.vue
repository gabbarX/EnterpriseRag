<template>
  <SettingsModalShell :visible="visible" :title="'Settings'" @close="modalShell.requestClose">
    <template #nav>
      <template v-for="group in navGroups" :key="group.key">
        <div class="nav-group-title">{{ group.label }}</div>
        <template v-for="item in group.items" :key="item.key">
          <div :class="['nav-item', {
            'active': currentSection === item.key,
            'has-submenu': item.children && item.children.length > 0,
            'expanded': expandedMenus.includes(item.key)
          }]" @click="handleNavClick(item)">
            <svg v-if="item.key === 'websearch'" width="17" height="17" viewBox="0 0 18 18" fill="none"
              xmlns="http://www.w3.org/2000/svg" class="nav-icon">
              <circle cx="9" cy="9" r="7" stroke="currentColor" stroke-width="1.2" fill="none" />
              <path d="M 9 2 A 3.5 7 0 0 0 9 16" stroke="currentColor" stroke-width="1.2" fill="none" />
              <path d="M 9 2 A 3.5 7 0 0 1 9 16" stroke="currentColor" stroke-width="1.2" fill="none" />
              <line x1="2.94" y1="5.5" x2="15.06" y2="5.5" stroke="currentColor" stroke-width="1.2"
                stroke-linecap="round" />
              <line x1="2.94" y1="12.5" x2="15.06" y2="12.5" stroke="currentColor" stroke-width="1.2"
                stroke-linecap="round" />
            </svg>
            <svg v-else-if="item.key === 'sandbox'" width="17" height="17" viewBox="0 0 18 18" fill="none"
              xmlns="http://www.w3.org/2000/svg" class="nav-icon">
              <rect x="2.5" y="3" width="13" height="12" rx="2" stroke="currentColor" stroke-width="1.2"
                fill="none" />
              <path d="M2.5 6.5h13" stroke="currentColor" stroke-width="1.2" />
              <path d="M5.5 10h4M5.5 12.5h2.5" stroke="currentColor" stroke-width="1.2"
                stroke-linecap="round" />
            </svg>
            <BrowserIcon v-else-if="item.key === 'browserconnection'" class="nav-icon" width="17" height="17" />
            <span v-else-if="item.emoji" class="nav-icon nav-icon-emoji">{{ item.emoji }}</span>
            <t-icon v-else :name="item.icon" class="nav-icon" />
            <span class="nav-label">{{ item.label }}</span>
            <t-icon v-if="item.children && item.children.length > 0"
              :name="expandedMenus.includes(item.key) ? 'chevron-down' : 'chevron-right'"
              class="expand-icon" />
          </div>

          <Transition name="submenu">
            <div v-if="item.children && expandedMenus.includes(item.key)" class="submenu">
              <div v-for="(child, childIndex) in item.children" :key="childIndex"
                :class="['submenu-item', { 'active': currentSubSection === child.key }]"
                @click.stop="handleSubMenuClick(item.key, child.key)">
                <span class="submenu-label">{{ child.label }}</span>
              </div>
            </div>
          </Transition>
        </template>
      </template>
    </template>
    <div class="content-wrapper" :class="{
      'content-wrapper--wide': currentSection === 'members',
      'content-wrapper--full': SYSTEM_ADMIN_SECTIONS.has(currentSection) || isIntegrationSection(currentSection),
    }">
      <!-- Fallback for a section the current role cannot see (deep link, or a
           role downgrade after switching workspace). Normal navigation is
           filtered by navItems, but the watch(navItems) fallback only fires a
           moment later, so this branch covers that gap and old URLs. -->
      <div v-if="!canSeeSection(currentSection)" class="section role-denied">
        <div class="role-denied-icon">
          <t-icon name="lock-on" size="48px" />
        </div>
        <div class="role-denied-title">{{ 'Insufficient permissions' }}</div>
        <div class="role-denied-desc">{{ 'Your role can\'t access this settings page. Ask an admin of this workspace to grant the required role.' }}</div>
      </div>
      <template v-else>
        <div v-if="currentSection === 'general'" class="section">
          <GeneralSettings />
        </div>

        <div v-if="currentSection === 'ollama'" class="section">
          <OllamaSettings />
        </div>

        <div v-if="currentSection === 'models'" class="section">
          <ModelSettings />
        </div>

        <div v-if="currentSection === 'websearch'" class="section">
          <WebSearchSettings />
        </div>

        <div v-if="currentSection === 'chathistory'" class="section">
          <ChatHistorySettings />
        </div>

        <div v-if="currentSection === 'memory'" class="section">
          <MemoryWorkspaceSettings />
        </div>

        <div v-if="currentSection === 'mymemory'" class="section">
          <MemorySettings />
        </div>

        <div v-if="currentSection === 'envvars'" class="section">
          <EnvVarSettings />
        </div>

        <div v-if="currentSection === 'vectorstore'" class="section">
          <VectorStoreSettings />
        </div>

        <div v-if="currentSection === 'parser'" class="section">
          <ParserEngineSettings />
        </div>

        <div v-if="currentSection === 'storage'" class="section">
          <StorageBackendSettings />
        </div>

        <div v-if="currentSection === 'sandbox'" class="section">
          <SandboxSettings />
        </div>

        <div v-if="currentSection === 'skills'" class="section">
          <SkillSettings :initial-sandbox-id="currentSubSection" />
        </div>

        <div v-if="currentSection === 'system'" class="section">
          <SystemInfo />
        </div>

        <div v-if="currentSection === 'system-global'" class="section">
          <SystemSettings />
        </div>

        <div v-if="currentSection === 'runtime-queues'" class="section">
          <RuntimeQueues />
        </div>

        <div v-if="currentSection === 'platform-api-keys'" class="section">
          <PlatformAPIKeys />
        </div>

        <div v-if="currentSection === 'system-audit-log'" class="section">
          <SystemAuditLog />
        </div>

        <div v-if="currentSection === 'userprofile'" class="section">
          <UserProfile />
        </div>

        <div v-if="currentSection === 'browserconnection'" class="section"><BrowserConnectionSettings /></div>

        <div v-if="currentSection === 'tenant'" class="section">
          <TenantInfo />
        </div>

        <div v-if="currentSection === 'members'" class="section">
          <TenantMembers />
        </div>

        <div v-if="isIntegrationSection(currentSection)" class="section">
          <IntegrationSettingsSection :tab="integrationTabFromSection(currentSection)" />
        </div>

        <div v-if="currentSection === 'mcp'" class="section">
          <McpSettings />
        </div>
      </template>
    </div>
  </SettingsModalShell>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { LocationQueryRaw } from 'vue-router'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities'
import { MessagePlugin } from 'tdesign-vue-next'
import { useModalShell } from '@/composables/useModalShell'
import SettingsModalShell from '@/components/SettingsModalShell.vue'
import SystemInfo from './SystemInfo.vue'
import TenantInfo from './TenantInfo.vue'
import UserProfile from './UserProfile.vue'
import GeneralSettings from './GeneralSettings.vue'
import BrowserConnectionSettings from './BrowserConnectionSettings.vue'
import BrowserIcon from '@/components/icons/BrowserIcon.vue'
import ModelSettings from './ModelSettings.vue'
import OllamaSettings from './OllamaSettings.vue'
import McpSettings from './McpSettings.vue'
import WebSearchSettings from './WebSearchSettings.vue'
import ChatHistorySettings from './ChatHistorySettings.vue'
import MemorySettings from './MemorySettings.vue'
import EnvVarSettings from './EnvVarSettings.vue'
import MemoryWorkspaceSettings from './MemoryWorkspaceSettings.vue'
import VectorStoreSettings from './VectorStoreSettings.vue'
import ParserEngineSettings from './ParserEngineSettings.vue'
import StorageBackendSettings from './StorageBackendSettings.vue'
import SandboxSettings from './SandboxSettings.vue'
import SkillSettings from './SkillSettings.vue'
import TenantMembers from './TenantMembers.vue'
import SystemSettings from '@/views/system/SystemSettings.vue'
import RuntimeQueues from '@/views/system/RuntimeQueues.vue'
import PlatformAPIKeys from '@/views/system/PlatformAPIKeys.vue'
import SystemAuditLog from '@/views/system/SystemAuditLog.vue'
import IntegrationSettingsSection from '@/views/integrations/IntegrationSettingsSection.vue'
import {
  INTEGRATION_PREVIEW_ITEMS,
  INTEGRATION_TAB_CAPABILITY,
  INTEGRATION_TAB_MIN_ROLE,
} from '@/config/integrations'
import {
  SETTINGS_SECTION_MIN_ROLE,
  SYSTEM_ADMIN_SETTINGS_SECTIONS,
} from '@/config/settingsAccess'
import { SETTINGS_SECTION_CAPABILITY } from '@/config/deploymentCapabilities'
import { SKILL_ICON } from '@/types/mention'
import {
  buildSettingsRouteQuery,
  integrationSectionKey,
  integrationTabFromSection,
  isIntegrationSection,
  normalizeSettingsSection as normalizeSettingsSectionFromQuery,
  settingsQueryUnchanged,
} from '@/config/settingsRoute'

const INTEGRATIONS_TABS_LABELS: Record<string, string> = {
  im: 'IM Integration',
  embed: 'Web Embed',
  api: 'API Integration',
  chrome: 'Chrome Extension',
  cli: 'CLI',
  claw: 'Claw Skill',
  mcpserver: 'MCP Server',
}

const route = useRoute()
const router = useRouter()
const uiStore = useUIStore()
const authStore = useAuthStore()
const deploymentCapabilities = useDeploymentCapabilitiesStore()

const currentSection = ref<string>('general')
const currentSubSection = ref<string>('')
const expandedMenus = ref<string[]>([])

type NavItem = {
  key: string
  icon: string
  label: string
  emoji?: string
  children?: Array<{ key: string; label: string }>
}

type NavGroup = {
  key: string
  label: string
  items: NavItem[]
}

// The minimum visible role for each settings sub-nav entry comes from
// settingsAccess.ts and mirrors the guard matrix in internal/router/router.go.
// The baseline is "the lowest role required by at least one meaningful write on
// that page": infrastructure config (model writes, Ollama downloads, web-search
// writes, parser/storage/vector/mcp CRUD, sandbox connect, skill installs,
// chat-history config) is admin; read-only pages (general / system info /
// tenant info / members roster) stay visible to viewer; resetting the API key is
// owner-only. Re-check the matching route group in router.go before editing it.
//
// Notes:
// - The only switch on the chathistory page (PUT /tenants/kv/chat-history-config)
//   is g.Admin() on the server. Showing viewers and contributors an entry that
//   403s on save is a poor experience, so the entry itself is admin.
// - The models list is readable by viewer; the add / edit / delete buttons inside
//   ModelSettings.vue are gated separately with hasRole('admin'), so keeping the
//   entry at viewer is reasonable (contributors can browse the model list too).
const SYSTEM_ADMIN_SECTIONS = SYSTEM_ADMIN_SETTINGS_SECTIONS

const normalizeSettingsSection = (section: string) => {
  return normalizeSettingsSectionFromQuery(section, route.query.tab as string | undefined)
}

const syncSettingsRoute = (sectionKey: string) => {
  if (route.path !== '/platform/settings') return
  const query = buildSettingsRouteQuery(sectionKey, route.query)
  if (settingsQueryUnchanged(route.query, query)) return
  void router.replace({
    path: '/platform/settings',
    query: query as LocationQueryRaw,
  })
}

const isSectionSupported = (key: string): boolean => {
  if (isIntegrationSection(key)) {
    return deploymentCapabilities.isSupported(
      INTEGRATION_TAB_CAPABILITY[integrationTabFromSection(key)],
    )
  }
  return deploymentCapabilities.isSupported(SETTINGS_SECTION_CAPABILITY[key])
}

const canSeeSection = (key: string): boolean => {
  if (isIntegrationSection(key)) {
    const min = INTEGRATION_TAB_MIN_ROLE[integrationTabFromSection(key)]
    if (!min) return true
    if (authStore.canAccessAllTenants) return true
    return authStore.hasRole(min)
  }
  if (SYSTEM_ADMIN_SECTIONS.has(key)) {
    return authStore.isSystemAdmin
  }
  const min = SETTINGS_SECTION_MIN_ROLE[key] ?? 'viewer'
  // canAccessAllTenants (superuser) must bypass here exactly as it does in the
  // router, otherwise a cross-tenant admin cannot see the entries they are
  // allowed to operate (see canManage in TenantMembers.vue).
  if (authStore.canAccessAllTenants) return true
  return authStore.hasRole(min)
}

const navItems = computed(() => {
  // Always go through the SETTINGS_SECTION_MIN_ROLE table so ad-hoc
  // isAdmin/isOwner checks do not spread across files. The server still enforces
  // g.Viewer/Admin/Owner on every route; this only decides whether the UI shows
  // the entry. Update settingsAccess.ts and the backend route together.
  const integrationItems: NavItem[] = INTEGRATION_PREVIEW_ITEMS.map((item) => ({
    key: integrationSectionKey(item.key),
    icon: item.icon.type === 'icon' ? item.icon.name : 'integration',
    emoji: item.icon.type === 'emoji' ? item.icon.value : undefined,
    label: (INTEGRATIONS_TABS_LABELS[item.key] ?? ''),
  }))
  const all: NavItem[] = [
    { key: 'general', icon: 'setting', label: 'General Settings' },
    { key: 'ollama', icon: 'server', label: 'Ollama' },
    { key: 'models', icon: 'control-platform', label: 'Model Management' },
    { key: 'websearch', icon: 'search', label: 'Web Search' },
    { key: 'chathistory', icon: 'chat', label: 'Message Management' },
    { key: 'memory', icon: 'bulletpoint', label: 'Long-term memory' },
    { key: 'vectorstore', icon: 'data-base', label: 'Vector DB Engine' },
    { key: 'parser', icon: 'file-search', label: 'Parser Engine' },
    { key: 'storage', icon: 'cloud', label: 'Storage Engine' },
    { key: 'sandbox', icon: 'code', label: 'Sandbox Config' },
    { key: 'skills', icon: SKILL_ICON, label: 'Skill Management' },
    { key: 'mcp', icon: 'tools', label: 'MCP Service' },
    { key: 'system', icon: 'info-circle', label: 'Version Info' },
    { key: 'system-global', icon: 'server', label: 'System Settings' },
    { key: 'runtime-queues', icon: 'queue', label: 'Task Queues' },
    { key: 'platform-api-keys', icon: 'secured', label: 'Platform API Keys' },
    { key: 'system-audit-log', icon: 'history', label: 'Audit log' },
    { key: 'userprofile', icon: 'user', label: 'User Profile' },
    { key: 'browserconnection', icon: 'laptop', label: 'Browser connection' },
    { key: 'mymemory', icon: 'bookmark', label: 'My memory' },
    { key: 'envvars', icon: 'key', label: 'Sandbox secrets' },
    { key: 'tenant', icon: 'user-circle', label: 'Workspace Info' },
    { key: 'members', icon: 'usergroup', label: 'Members' },
    ...integrationItems,
  ]
  // An empty currentTenantRole means membership has not loaded yet. Rendering the
  // full viewer nav and then dropping entries once the role arrives is jarring,
  // so hold off rendering instead.
  if (!authStore.currentTenantRole && !authStore.canAccessAllTenants) {
    return [] as NavItem[]
  }
  return all.filter((it) => canSeeSection(it.key) && isSectionSupported(it.key))
})

const navGroups = computed<NavGroup[]>(() => {
  const itemMap = new Map(navItems.value.map((item) => [item.key, item]))
  const pickItems = (keys: string[]) => keys.map((key) => itemMap.get(key)).filter(Boolean) as NavItem[]
  return [
    {
      key: 'account',
      label: 'Account',
      items: pickItems(['general', 'userprofile', 'browserconnection', 'mymemory', 'envvars']),
    },
    {
      key: 'workspace',
      label: 'Workspace',
      items: pickItems(['tenant', 'members', 'chathistory', 'memory']),
    },
    {
      key: 'models_runtime',
      label: 'Models',
      items: pickItems(['models', 'ollama']),
    },
    {
      key: 'integrations',
      label: 'Publish & Integrations',
      items: pickItems(INTEGRATION_PREVIEW_ITEMS.map((item) => integrationSectionKey(item.key))),
    },
    {
      key: 'data_extensions',
      label: 'Data & Extensions',
      items: pickItems([
        'vectorstore',
        'parser',
        'storage',
        'sandbox',
        'skills',
        'websearch',
        'mcp',
      ]),
    },
    {
      key: 'system_administration',
      label: 'System Administration',
      items: pickItems(['system-global', 'runtime-queues', 'platform-api-keys', 'system-audit-log']),
    },
    {
      key: 'platform',
      label: 'Platform',
      items: pickItems(['system']),
    },
  ].filter((group) => group.items.length > 0)
})

const handleNavClick = (item: any) => {
  if (item.children && item.children.length > 0) {
    const index = expandedMenus.value.indexOf(item.key)
    if (index > -1) {
      expandedMenus.value.splice(index, 1)
    } else {
      expandedMenus.value.push(item.key)
    }
    currentSubSection.value = item.children[0].key
  } else {
    currentSubSection.value = ''
  }

  // Also sync the URL to ?section=<navKey>: without it the query stays put when
  // arriving from another section and the route watcher pulls the content back.
  currentSection.value = item.key
  syncSettingsRoute(item.key)
}

const handleSubMenuClick = (parentKey: string, childKey: string) => {
  currentSection.value = parentKey
  currentSubSection.value = childKey

  setTimeout(() => {
    const element = document.querySelector(`[data-model-type="${childKey}"]`)
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }, 100)
}

const visible = computed(() => {
  return route.path === '/platform/settings' || uiStore.showSettingsModal
})

const handleClose = () => {
  // Blur before unmount so TDesign textarea autosize won't run on a detached node.
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur()
  }
  uiStore.closeSettings()
  if (route.path === '/platform/settings') {
    const sec = route.query.section
    if (sec === 'system-global' || sec === 'runtime-queues' || sec === 'platform-api-keys' || sec === 'system-audit-log') {
      router.push('/platform/knowledge-bases')
    } else {
      router.back()
    }
  }
}

watch(() => uiStore.settingsInitialSection, (section) => {
  if (section && visible.value) {
    const normalizedSection = normalizeSettingsSection(section)
    if (deploymentCapabilities.loaded && !isSectionSupported(normalizedSection)) {
      MessagePlugin.warning('This feature is not supported by the current deployment. You have been returned to an available page.')
      currentSection.value = navItems.value[0]?.key || 'general'
      currentSubSection.value = ''
      return
    }
    currentSection.value = normalizedSection
    syncSettingsRoute(normalizedSection)
    const navItem = (navItems.value as any[]).find((item) => item.key === normalizedSection)
    if (navItem && navItem.children && navItem.children.length > 0) {
      if (!expandedMenus.value.includes(section)) {
        expandedMenus.value.push(section)
      }
      currentSubSection.value = uiStore.settingsInitialSubSection || navItem.children[0].key
      if (uiStore.settingsInitialSubSection) {
        setTimeout(() => {
          const element = document.querySelector(`[data-model-type="${uiStore.settingsInitialSubSection}"]`)
          if (element) {
            element.scrollIntoView({ behavior: 'smooth', block: 'start' })
          }
        }, 300)
      }
    } else if (normalizedSection === 'skills') {
      // Sandbox config id from the agent editor / sandbox cards. Skills has
      // no nav children, so this is the only way to preselect the target image.
      currentSubSection.value = uiStore.settingsInitialSubSection || ''
    } else {
      currentSubSection.value = ''
    }
  }
}, { immediate: true })

watch(() => uiStore.settingsInitialSubSection, (sub) => {
  if (uiStore.settingsInitialSection === 'skills' && visible.value) {
    currentSubSection.value = sub || ''
  }
})

watch(
  () => [visible.value, route.path, route.query.section, deploymentCapabilities.loaded] as const,
  ([isVisible, path, section, capabilitiesLoaded]) => {
    if (!isVisible || path !== '/platform/settings') return
    if (typeof section !== 'string') {
      syncSettingsRoute(currentSection.value || 'general')
      return
    }
    const normalizedSection = normalizeSettingsSectionFromQuery(
      section,
      typeof route.query.tab === 'string' ? route.query.tab : undefined,
    )
    if (capabilitiesLoaded && !isSectionSupported(normalizedSection)) {
      MessagePlugin.warning('This feature is not supported by the current deployment. You have been returned to an available page.')
      const fallback = navItems.value[0]?.key || 'general'
      currentSection.value = fallback
      currentSubSection.value = ''
      syncSettingsRoute(fallback)
      return
    }
    currentSection.value = normalizedSection
    currentSubSection.value = normalizedSection === 'skills'
      ? (uiStore.settingsInitialSubSection || '')
      : ''
    syncSettingsRoute(normalizedSection)
  },
  { immediate: true },
)

// Switching workspace can change the role, so a previously visible admin-only
// panel may disappear. Fall back to the first visible entry in that case.
watch(navItems, (items) => {
  if (!items.some((item) => item.key === currentSection.value)) {
    const fallback = items[0]?.key || 'general'
    currentSection.value = fallback
    currentSubSection.value = ''
    syncSettingsRoute(fallback)
  }
})

const modalShell = useModalShell({
  visible: () => visible.value,
  close: handleClose,
})

const handleSettingsNav = (e: CustomEvent) => {
  const { section, subsection } = e.detail
  if (section) {
    const normalizedSection = normalizeSettingsSection(section)
    if (deploymentCapabilities.loaded && !isSectionSupported(normalizedSection)) {
      MessagePlugin.warning('This feature is not supported by the current deployment. You have been returned to an available page.')
      currentSection.value = navItems.value[0]?.key || 'general'
      currentSubSection.value = ''
      return
    }
    currentSection.value = normalizedSection
    syncSettingsRoute(normalizedSection)
    const navItem = (navItems.value as any[]).find((item: any) => item.key === normalizedSection)
    if (navItem && navItem.children && navItem.children.length > 0) {
      if (!expandedMenus.value.includes(section)) {
        expandedMenus.value.push(section)
      }
      currentSubSection.value = subsection || navItem.children[0].key
    }
  }
}

onMounted(() => {
  window.addEventListener('settings-nav', handleSettingsNav as EventListener)
})

watch(currentSection, () => {
  if (document.activeElement instanceof HTMLElement) {
    document.activeElement.blur()
  }
})

onUnmounted(() => {
  window.removeEventListener('settings-nav', handleSettingsNav as EventListener)
})
</script>

<style lang="less" scoped>
.expand-icon {
  margin-left: 4px;
  font-size: var(--app-text-base);
  transition: transform var(--app-motion-base) ease;
}

.submenu {
  margin-left: 28px;
  margin-bottom: 3px;
  overflow: hidden;
}

.submenu-item {
  padding: 5px 12px;
  margin-bottom: 2px;
  border-radius: var(--app-radius-xs);
  cursor: pointer;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  transition: all var(--app-motion-base) ease;
  user-select: none;

  &:hover {
    background-color: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }

  &.active {
    background-color: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
    font-weight: 500;
  }
}

.submenu-label {
  display: block;
}

.submenu-enter-active,
.submenu-leave-active {
  transition: all var(--app-motion-base) ease;
}

.submenu-enter-from {
  opacity: 0;
  max-height: 0;
}

.submenu-enter-to {
  opacity: 1;
  max-height: 300px;
}

.submenu-leave-from {
  opacity: 1;
  max-height: 300px;
}

.submenu-leave-to {
  opacity: 0;
  max-height: 0;
}

.content-wrapper {
  // Bumped from 600 to 760 when the modal grew from 900→1080 (see
  // .settings-modal). Without this, single-column panes (General,
  // Tenant, API key, …) leave a wide right-hand gutter inside the
  // wider modal. 760 keeps comfortable reading-width on long
  // descriptions without the form fields stretching to the full
  // panel width — which would look stranger than a small gutter.
  max-width: 760px;
  padding: 40px 48px;

  /* Member and audit tables have many columns; 600px pushes the actions column
     against the edge, so fill the right-hand content column instead. */
  &--wide {
    max-width: none;
    width: 100%;
    padding: 32px 36px 40px;
    box-sizing: border-box;
  }

  &--full {
    max-width: none;
    width: 100%;
    padding: 30px 34px 40px;
    box-sizing: border-box;
  }
}

.section {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.role-denied {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 64px 24px;
  gap: 12px;
  min-height: 240px;

  .role-denied-icon {
    color: var(--td-text-color-placeholder);
  }

  .role-denied-title {
    font-size: var(--app-text-xl);
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  .role-denied-desc {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    max-width: 360px;
    line-height: 1.6;
  }
}
</style>
