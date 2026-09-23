<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createDataSource,
  updateDataSource,
  triggerSync,
  validateConnection,
  validateCredentials,
  listResources,
  resolveResourceAncestors,
  deleteDataSource,
  putDataSourceCredentials,
  deleteDataSourceCredentials,
  type DataSource,
  type Resource,
} from '@/api/datasource'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import DataSourceTypeIcon from './DataSourceTypeIcon.vue'
import { getDatasourceIconUrl } from './datasourceIcons'

const DATASOURCE_PREREQ_BAR_TEXT: Record<string, string> = {
}

const DATASOURCE_PREREQ_STEP1_BRIEF: Record<string, string> = {
}

const DATASOURCE_PREREQ_STEP1_DESC: Record<string, string> = {
}

const DATASOURCE_PREREQ_STEP2_BRIEF: Record<string, string> = {
}

const DATASOURCE_PREREQ_STEP3_BRIEF: Record<string, string> = {
}

const DATASOURCE_PREREQ_STEP3_DESC: Record<string, string> = {
}

const DATASOURCE_PREREQ_OPEN_CONSOLE: Record<string, string> = {
}

const DATASOURCE_NO_RESOURCES_DESC: Record<string, string> = {
  notion: 'The app needs Notion page access permissions to fetch content',
}

const DATASOURCE_GUIDE_STEP1: Record<string, string> = {
  notion: 'Open the page or database you want to sync in Notion',
}

const DATASOURCE_GUIDE_STEP2: Record<string, string> = {
  notion: 'Click the "···" menu at the top right, select "Connect to" or "Add connections"',
}

const DATASOURCE_GUIDE_STEP3: Record<string, string> = {
  notion: 'Search and select your Integration app, then come back and click Retry',
}

const DATASOURCE_CONNECTOR_LABELS: Record<string, string> = {
  notion: 'Notion',
  confluence: 'Confluence',
  rss: 'RSS / Atom Feed',
  gitlab: 'GitLab',
}

const DATASOURCE_CONNECTOR_DESC_LABELS: Record<string, string> = {
  notion: 'Sync pages and databases from Notion',
  confluence: 'Sync spaces and pages from Confluence as Markdown',
  rss: 'Sync articles from RSS / Atom feeds',
  gitlab: 'Sync files from GitLab projects',
}

const DATASOURCE_PREREQ_STEP2_DESC_LABELS: Record<string, string> = {
}

const props = defineProps<{
  kbId: string
  dataSource: DataSource | null
}>()

const visible = defineModel<boolean>('visible', { default: false })
const emit = defineEmits<{ saved: [] }>()

const isEdit = computed(() => !!props.dataSource)
const step = ref(0)
const submitting = ref(false)

// In edit mode the credential "configured?" flag travels on the main
// DataSource response (DataSource.credentials.credentials.configured —
// server-side dto.DataSourceResponse.Credentials). True iff a credential
// map is currently stored server-side.
const credentialsConfigured = ref(false)

// "Replace credentials" mode toggle in edit. Defaults to false: a configured
// connector shows a small "Credentials configured ✓" line with Replace /
// Remove actions. Toggling Replace reveals the credential inputs so the
// user can type a new set. Untoggling discards anything typed.
const replaceCredentialsMode = ref(false)

// Whether the credential input section is interactive right now. In create
// mode it's always shown; in edit mode only when the user opted in to
// Replace, OR when nothing is configured yet (degenerate case where the
// data source row exists with no credentials stored).
const credentialsInputVisible = computed(() => {
  if (!isEdit.value) return true
  if (!credentialsConfigured.value) return true
  return replaceCredentialsMode.value
})

function refreshCredentialsStatus() {
  // Re-derive from whatever the parent passed in props.dataSource. Called
  // when the dialog opens or props.dataSource is swapped; the parent is
  // expected to re-fetch the data source list after credential mutations
  // so the new metadata flows in here automatically.
  if (!isEdit.value || !props.dataSource) {
    credentialsConfigured.value = false
    return
  }
  credentialsConfigured.value =
    props.dataSource.credentials?.credentials?.configured === true
}

// Single-click remove with toast feedback. Mirrors the CredentialResource
// component's UX: the secret is irrecoverable client-side either way, so a
// modal confirm just adds friction. The danger-themed button is the deterrent.
const pendingRemoveCredentials = ref(false)
const removingCredentials = ref(false)

function requestRemoveCredentials() {
  pendingRemoveCredentials.value = true
}

function cancelPendingRemoveCredentials() {
  pendingRemoveCredentials.value = false
}

async function confirmRemoveCredentials() {
  if (!props.dataSource?.id) return
  removingCredentials.value = true
  try {
    await deleteDataSourceCredentials(props.dataSource.id)
    credentialsConfigured.value = false
    replaceCredentialsMode.value = false
    pendingRemoveCredentials.value = false
    form.value.config.credentials = {}
    MessagePlugin.success('Credential removed')
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to remove credential')
  } finally {
    removingCredentials.value = false
  }
}

function cancelReplaceCredentials() {
  replaceCredentialsMode.value = false
  pendingRemoveCredentials.value = false
  form.value.config.credentials = {}
  rssAuthHeaders.value = []
  testResult.value = credentialsConfigured.value ? 'success' : ''
  testErrorMsg.value = ''
}

interface CustomHeaderItem {
  key: string
  value: string
}

const rssAuthHeaders = ref<CustomHeaderItem[]>([])

function serializeAuthHeaders(items: CustomHeaderItem[]): string {
  return items
    .filter(h => h.key.trim())
    .map(h => `${h.key.trim()}: ${h.value}`)
    .join('\n')
}

function syncRssAuthHeadersToCredentials() {
  if (form.value.type !== 'rss') return
  const serialized = serializeAuthHeaders(rssAuthHeaders.value)
  if (serialized) {
    form.value.config.credentials.auth_headers = serialized
  } else {
    delete form.value.config.credentials.auth_headers
  }
}

// Feed URLs may still live in credentials on older rows (not returned by the
// API). The backend copies them into settings on read; fall back to the
// selected feed resource IDs when settings are still empty.
function hydrateRssFeedUrlsFromConfig(config: { settings?: Record<string, any>; resource_ids?: string[] }) {
  const settings = config.settings || {}
  if (String(settings.feed_urls || '').trim()) {
    return { ...settings }
  }
  const ids = config.resource_ids || []
  if (ids.length === 0) {
    return { ...settings }
  }
  return { ...settings, feed_urls: ids.join('\n') }
}

function addRssAuthHeader() {
  rssAuthHeaders.value.push({ key: '', value: '' })
}

function removeRssAuthHeader(idx: number) {
  rssAuthHeaders.value.splice(idx, 1)
}

function needsConnectionTest(): boolean {
  return !(isEdit.value && credentialsConfigured.value && !replaceCredentialsMode.value)
}

function hydrateConfluenceCredentialsFromSettings() {
  if (form.value.type !== 'confluence') return
  const settings = form.value.config.settings || {}
  const creds = form.value.config.credentials || {}
  form.value.config.credentials = {
    ...creds,
    edition: creds.edition || settings.edition || 'server',
    base_url: creds.base_url || settings.base_url || '',
    username: creds.username || settings.username || '',
  }
}

function syncConfluencePublicFieldsToSettings() {
  if (form.value.type !== 'confluence') return
  const creds = form.value.config.credentials || {}
  form.value.config.settings = {
    ...(form.value.config.settings || {}),
    ...(creds.edition ? { edition: creds.edition } : {}),
    ...(creds.base_url ? { base_url: creds.base_url } : {}),
    ...(creds.username ? { username: creds.username } : {}),
  }
}

function enterReplaceCredentials() {
  pendingRemoveCredentials.value = false
  replaceCredentialsMode.value = true
  hydrateConfluenceCredentialsFromSettings()
  testResult.value = ''
  testErrorMsg.value = ''
}

// Form data
const form = ref({
  name: '',
  type: '',
  config: {
    credentials: {} as Record<string, any>,
    resource_ids: [] as string[],
    settings: {} as Record<string, any>,
  },
  sync_schedule: '0 0 */6 * * *',
  sync_mode: 'incremental' as 'incremental' | 'full',
  conflict_strategy: 'overwrite' as 'overwrite' | 'skip',
  sync_deletions: true,
})

// Step 2: Resources
const resources = ref<Resource[]>([])
const loadingResources = ref(false)
const selectedResourceIds = ref<string[]>([])
const expandedResourceIds = ref(new Set<string>())
// Lazy loading: parents whose children have already been fetched, and parents
// currently being fetched. Used to load hierarchical sources (e.g. Confluence)
// one level at a time instead of traversing the whole tree up front (#1672).
const loadedChildrenIds = ref(new Set<string>())
const loadingChildrenIds = ref(new Set<string>())
// True when the initial listing already returned the whole tree (connectors like
// Notion populate parent_id on the first call). In that case expanding a node
// never needs an extra request.
const treeFullyLoaded = ref(false)

// Drive root input: the Drive connectors have no "list spaces" API, so
// the user must supply a root folder_token. We collect it here, write it into
// form.config.resource_ids as the single root, then loadResources lists its
// children. See ADR-0004.
const isGitLabConnector = (type: string) => type === 'gitlab'

interface GitLabProjectInput { project_id: string; ref: string; pathsText: string }
const gitlabProjects = ref<GitLabProjectInput[]>([])
function syncGitLabProjectsToSettings() {
  if (!isGitLabConnector(form.value.type)) return
  form.value.config.settings.projects = gitlabProjects.value
    .filter(project => project.project_id.trim())
    .map(project => ({
      project_id: project.project_id.trim(), ref: project.ref.trim(),
      paths: project.pathsText.split(/[\n,]/).map(path => path.trim()).filter(Boolean),
    }))
}
function addGitLabProject() { gitlabProjects.value.push({ project_id: '', ref: '', pathsText: '' }) }
function removeGitLabProject(index: number) { gitlabProjects.value.splice(index, 1); syncGitLabProjectsToSettings() }

// Shared children/parent indexes — used by tree rendering and selection logic
const childrenMap = computed(() => {
  const map = new Map<string, Resource[]>()
  for (const r of resources.value) {
    if (r.parent_id) {
      const siblings = map.get(r.parent_id)
      if (siblings) siblings.push(r)
      else map.set(r.parent_id, [r])
    }
  }
  return map
})

const parentMap = computed(() => {
  const map = new Map<string, string>()
  for (const r of resources.value) {
    if (r.parent_id) map.set(r.external_id, r.parent_id)
  }
  return map
})

// `selectedResourceIds` is a MINIMAL COVER SET: only the roots of fully-selected
// subtrees. Sending this to the backend gives "sync these IDs and all descendants"
// semantics — including any pages added later under a selected parent.
type CheckState = 'checked' | 'indeterminate' | 'unchecked'

const checkStates = computed(() => {
  const states = new Map<string, CheckState>()
  const cover = new Set(selectedResourceIds.value)

  // Single post-order walk: a node is `checked` if itself or any ancestor is
  // in the cover set; otherwise `indeterminate` if any descendant is checked;
  // otherwise `unchecked`. Returns whether the subtree contains a checked node.
  function walk(node: Resource, ancestorChecked: boolean): boolean {
    const selfChecked = ancestorChecked || cover.has(node.external_id)
    let descendantChecked = false
    for (const c of childrenMap.value.get(node.external_id) || []) {
      if (walk(c, selfChecked)) descendantChecked = true
    }
    if (selfChecked) states.set(node.external_id, 'checked')
    else states.set(node.external_id, descendantChecked ? 'indeterminate' : 'unchecked')
    return selfChecked || descendantChecked
  }
  for (const r of resources.value) {
    if (!r.parent_id) walk(r, false)
  }
  return states
})

function toggleExpand(id: string) {
  const next = new Set(expandedResourceIds.value)
  if (next.has(id)) {
    next.delete(id)
    expandedResourceIds.value = next
    return
  }
  next.add(id)
  expandedResourceIds.value = next
  void ensureChildrenLoaded(id)
}

// ensureChildrenLoaded fetches the direct children of a node on demand. It is a
// no-op when the connector already delivered the whole tree in one call (e.g.
// Notion) or when this node's children have already been fetched.
async function ensureChildrenLoaded(id: string) {
  if (!tempDsId.value) return
  if (loadedChildrenIds.value.has(id) || loadingChildrenIds.value.has(id)) return
  if (treeFullyLoaded.value) {
    loadedChildrenIds.value = new Set(loadedChildrenIds.value).add(id)
    return
  }

  loadingChildrenIds.value = new Set(loadingChildrenIds.value).add(id)
  try {
    const res = await listResources(tempDsId.value, id)
    const children: Resource[] = res?.data || res || []
    if (children.length > 0) {
      const existing = new Set(resources.value.map(r => r.external_id))
      const merged = resources.value.slice()
      for (const c of children) {
        if (!existing.has(c.external_id)) merged.push(c)
      }
      resources.value = merged
    }
    loadedChildrenIds.value = new Set(loadedChildrenIds.value).add(id)
  } catch (e: any) {
    MessagePlugin.error(e?.message || e?.error || 'Failed to load resources')
    // Collapse again so the user can retry the expand.
    const next = new Set(expandedResourceIds.value)
    next.delete(id)
    expandedResourceIds.value = next
  } finally {
    const s = new Set(loadingChildrenIds.value)
    s.delete(id)
    loadingChildrenIds.value = s
  }
}

const visibleTree = computed(() => {
  const roots = resources.value.filter(r => !r.parent_id)
  const result: { resource: Resource; depth: number }[] = []
  function walk(items: Resource[], depth: number) {
    for (const r of items) {
      result.push({ resource: r, depth })
      if (r.has_children && expandedResourceIds.value.has(r.external_id)) {
        walk(childrenMap.value.get(r.external_id) || [], depth + 1)
      }
    }
  }
  walk(roots, 0)
  return result
})

// Connection test
const testing = ref(false)
const testResult = ref<'success' | 'error' | ''>('')
const testErrorMsg = ref('')

// Collapsible prereq in Step 1
const prereqExpanded = ref(false)


// Temp data source for resource listing
const tempDsId = ref('')

// Schedule presets
const schedulePresets = computed(() => [
  { label: 'Every 30 min', value: '0 */30 * * * *' },
  { label: 'Every hour', value: '0 0 * * * *' },
  { label: 'Every 6 hours', value: '0 0 */6 * * *' },
  { label: 'Every 12 hours', value: '0 0 */12 * * *' },
  { label: 'Daily', value: '0 0 2 * * *' },
])

// --- Connector definitions ---
interface ConnectorDef {
  type: string
  available: boolean
  docUrl: string
  permissionDocUrl: string
  permissionPageUrl: string
  requiredPermissions: string[]
  fields: {
    key: string
    labelKey: string
    placeholder: string
    secret?: boolean
    optional?: boolean
    hintKey?: string
    multiline?: boolean
    fieldType?: 'custom_headers'
  }[]
}

const connectorDefs = computed<ConnectorDef[]>(() => [
  {
    type: 'notion',
    available: true,
    docUrl: 'https://www.notion.so/my-integrations',
    permissionDocUrl: '',
    permissionPageUrl: '',
    requiredPermissions: [],
    fields: [
      { key: 'api_key', labelKey: 'Integration Token', placeholder: 'ntn_xxxx', secret: true },
    ],
  },
  {
    type: 'confluence',
    available: true,
    docUrl: 'https://developer.atlassian.com/cloud/confluence/rest/',
    permissionDocUrl: 'https://developer.atlassian.com/cloud/confluence/rest/',
    permissionPageUrl: 'https://id.atlassian.com/manage-profile/security/api-tokens',
    requiredPermissions: [],
    fields: [
      { key: 'base_url', labelKey: 'Confluence URL', placeholder: 'https://confluence.example.com or https://team.atlassian.net/wiki' },
      { key: 'username', labelKey: 'Username or email', placeholder: 'name or email' },
      { key: 'password', labelKey: 'Server/DC password', placeholder: 'Server/DC password', secret: true },
      { key: 'api_token', labelKey: 'Cloud API token', placeholder: 'Cloud API token', secret: true },
    ],
  },
  {
    type: 'rss',
    available: true,
    docUrl: '',
    permissionDocUrl: '',
    permissionPageUrl: '',
    requiredPermissions: [],
    fields: [
      { key: 'auth_headers', labelKey: 'Custom headers (optional)', placeholder: '', optional: true, hintKey: 'For private feeds. One per line in "Name: Value" form, e.g. Authorization: Bearer xxxx', fieldType: 'custom_headers' },
    ],
  },
  {
    type: 'gitlab', available: true, docUrl: '', permissionDocUrl: '', permissionPageUrl: '', requiredPermissions: [],
    fields: [
      { key: 'base_url', labelKey: 'GitLab URL', placeholder: 'https://gitlab.example.com' },
      { key: 'access_token', labelKey: 'Personal access token', placeholder: '', secret: true },
    ],
  },
])


const currentDef = computed(() => connectorDefs.value.find(d => d.type === form.value.type))

const displayedCredentialFields = computed(() => {
  const fields = currentDef.value?.fields || []
  if (form.value.type !== "confluence") return fields

  return fields.filter((field) => {
    if (field.key === "password") return form.value.config.credentials.edition !== "cloud"
    if (field.key === "api_token") return form.value.config.credentials.edition === "cloud"
    return true
  })
})

// --- Drawer lifecycle ---
watch(visible, async (v) => {
  if (!v) {
    if (!isEdit.value && tempDsId.value) {
      try {
        await deleteDataSource(tempDsId.value)
      } catch {
        // Ignore cleanup errors
      }
      tempDsId.value = ''
    }
    return
  }
  step.value = isEdit.value ? 1 : 0
  testResult.value = ''
  testErrorMsg.value = ''
  tempDsId.value = ''
  prereqExpanded.value = false
  pendingRemoveCredentials.value = false
  resources.value = []
  selectedResourceIds.value = []
  expandedResourceIds.value = new Set()
  loadedChildrenIds.value = new Set()
  loadingChildrenIds.value = new Set()
  treeFullyLoaded.value = false
  rssAuthHeaders.value = []
  gitlabProjects.value = []

  if (isEdit.value && props.dataSource) {
    // Reset edit/replace toggle every open so an aborted replace doesn't
    // carry over. credentialsConfigured will be refreshed from the
    // /credentials subresource (run separately below).
    replaceCredentialsMode.value = false
    credentialsConfigured.value = false
    refreshCredentialsStatus()
    testResult.value = credentialsConfigured.value ? 'success' : ''
    const editConfig = props.dataSource.config || {}
    form.value = {
      name: props.dataSource.name,
      type: props.dataSource.type,
      config: {
        credentials: {},
        resource_ids: editConfig.resource_ids || [],
        settings: props.dataSource.type === 'rss'
          ? hydrateRssFeedUrlsFromConfig(editConfig)
          : (editConfig.settings || {}),
      },
      sync_schedule: props.dataSource.sync_schedule,
      sync_mode: props.dataSource.sync_mode,
      conflict_strategy: props.dataSource.conflict_strategy,
      sync_deletions: props.dataSource.sync_deletions,
    }
    selectedResourceIds.value = form.value.config?.resource_ids || []
    if (isGitLabConnector(form.value.type)) {
      const savedProjects = Array.isArray(form.value.config.settings.projects) ? form.value.config.settings.projects : []
      gitlabProjects.value = savedProjects.map((project: any) => ({
        project_id: String(project.project_id || ''), ref: String(project.ref || ''),
        pathsText: Array.isArray(project.paths) ? project.paths.join('\n') : '',
      }))
    }
    tempDsId.value = props.dataSource.id
  } else {
    replaceCredentialsMode.value = false
    credentialsConfigured.value = false
    form.value = {
      name: '',
      type: '',
      config: { credentials: {}, resource_ids: [], settings: {} },
      sync_schedule: '0 0 */6 * * *',
      sync_mode: 'incremental',
      conflict_strategy: 'overwrite',
      sync_deletions: true,
    }
  }
})

watch(
  () => form.value.config.credentials,
  () => {
    if (needsConnectionTest()) {
      testResult.value = ''
      testErrorMsg.value = ''
    }
  },
  { deep: true },
)

watch(
  rssAuthHeaders,
  () => {
    syncRssAuthHeadersToCredentials()
    if (needsConnectionTest()) {
      testResult.value = ''
      testErrorMsg.value = ''
    }
  },
  { deep: true },
)

watch(
  () => form.value.config.settings.feed_urls,
  () => {
    if (needsConnectionTest()) {
      testResult.value = ''
      testErrorMsg.value = ''
    }
  },
)

function selectType(def: ConnectorDef) {
  if (!def.available) return
  form.value.type = def.type
  form.value.name = (DATASOURCE_CONNECTOR_LABELS[def.type] ?? '')
  form.value.config.credentials = def.type === "confluence" ? { edition: "server" } : {}
  if (def.type === 'confluence') {
    form.value.config.settings = { ...form.value.config.settings, edition: 'server' }
  }
  if (isGitLabConnector(def.type)) addGitLabProject()
  rssAuthHeaders.value = []
  step.value = 1
}

// --- Test connection ---
async function testConnection() {
  syncRssAuthHeadersToCredentials()
  syncConfluencePublicFieldsToSettings()
  if (!validateRssFeedUrls()) return
  if (!isEdit.value || !credentialsConfigured.value || replaceCredentialsMode.value) {
    const fields = displayedCredentialFields.value
    for (const f of fields) {
      if (f.optional || f.fieldType === 'custom_headers') continue
      if (!form.value.config.credentials[f.key]) {
        MessagePlugin.warning(`${f.labelKey} ${'is required'}`)
        return
      }
    }
  }

  testing.value = true
  testResult.value = ''
  testErrorMsg.value = ''
  try {
    // The main update endpoint ignores credentials. Only use the saved
    // connection when keeping its credentials; test replacements directly
    // without persisting them until the user saves the data source.
    if (isEdit.value && tempDsId.value && !needsConnectionTest()) {
      await updateDataSource(tempDsId.value, {
        ...form.value,
        knowledge_base_id: props.kbId,
      } as any)
      await validateConnection(tempDsId.value)
    } else {
      const creds = { ...form.value.config.credentials }
      if (form.value.type === 'rss') {
        // validate-credentials is credentials-only; feed URLs live in settings.
        creds.feed_urls = form.value.config.settings.feed_urls
      }
      await validateCredentials(form.value.type, creds)
    }
    testResult.value = 'success'
    MessagePlugin.success('Connection successful')
  } catch (e: any) {
    testResult.value = 'error'
    testErrorMsg.value = e?.message || e?.error || ''
    MessagePlugin.error('Connection failed')
  }
  testing.value = false
}

// --- Load resources ---
async function loadResources() {
  loadingResources.value = true
  try {
    syncConfluencePublicFieldsToSettings()
    if (!tempDsId.value) {
      const res = await createDataSource({
        ...form.value,
        knowledge_base_id: props.kbId,
        status: 'paused',
      } as any)
      const created = res?.data || res
      tempDsId.value = created.id
    } else if (!isEdit.value) {
      await updateDataSource(tempDsId.value, {
        ...form.value,
        knowledge_base_id: props.kbId,
      } as any)
    }

    const res = await listResources(tempDsId.value)
    resources.value = res?.data || res || []
    // Any parent that already arrived with children (connectors returning the
    // full tree, e.g. Notion) needs no further lazy fetch.
    const parentsWithChildren = new Set<string>()
    for (const r of resources.value) {
      if (r.parent_id) parentsWithChildren.add(r.parent_id)
    }
    loadedChildrenIds.value = parentsWithChildren
    loadingChildrenIds.value = new Set<string>()
    // If any resource already has a parent, the connector returned the whole tree
    // up front, so per-node lazy fetching is unnecessary.
    treeFullyLoaded.value = parentsWithChildren.size > 0
    // Auto-expand top-level nodes whose children are already loaded; lazy nodes
    // (children not yet fetched) stay collapsed until the user expands them.
    expandedResourceIds.value = new Set(
      resources.value
        .filter(r => !r.parent_id && r.has_children && parentsWithChildren.has(r.external_id))
        .map(r => r.external_id),
    )
    // When editing a lazily-loaded source, reveal pre-existing selections that
    // live below the (not-yet-loaded) tree so they are visible and checked.
    if (isEdit.value && !treeFullyLoaded.value) {
      const loaded = new Set(resources.value.map(r => r.external_id))
      const hidden = selectedResourceIds.value.filter(id => !loaded.has(id))
      if (hidden.length > 0) void revealExistingSelections(hidden)
    }
  } catch (e: any) {
    MessagePlugin.error(e?.message || e?.error || 'Failed to load resources')
  }
  loadingResources.value = false
}

// revealExistingSelections asks the backend which ancestors must be expanded to
// surface the current (possibly deeply nested) selection, then loads each level
// so the saved selection becomes visible and correctly checked in the tree.
async function revealExistingSelections(hiddenIds: string[]) {
  if (!tempDsId.value || hiddenIds.length === 0) return
  try {
    const res = await resolveResourceAncestors(tempDsId.value, hiddenIds)
    const ancestors: string[] = res?.data?.ancestors || res?.ancestors || []
    if (ancestors.length === 0) return
    const expanded = new Set(expandedResourceIds.value)
    for (const id of ancestors) expanded.add(id)
    expandedResourceIds.value = expanded
    // Load each ancestor level (children include the next ancestor / the
    // selection itself); calls are independent and dedup on merge.
    await Promise.all(ancestors.map(id => ensureChildrenLoaded(id)))
  } catch (e: any) {
    MessagePlugin.error(e?.message || e?.error || 'Failed to load resources')
  }
}

function getDescendantIds(id: string): string[] {
  const ids: string[] = []
  const children = childrenMap.value.get(id) || []
  for (const c of children) {
    ids.push(c.external_id)
    ids.push(...getDescendantIds(c.external_id))
  }
  return ids
}

function getAncestorChain(id: string): string[] {
  const chain = [id]
  for (let p = parentMap.value.get(id); p; p = parentMap.value.get(p)) {
    chain.push(p)
  }
  return chain
}

function isCovered(id: string, cover: Set<string>): boolean {
  for (let cur: string | undefined = id; cur; cur = parentMap.value.get(cur)) {
    if (cover.has(cur)) return true
  }
  return false
}

function checkResource(id: string, cover: Set<string>) {
  if (isCovered(id, cover)) return
  const descendants = new Set(getDescendantIds(id))
  for (const d of [...cover]) {
    if (descendants.has(d)) cover.delete(d)
  }
  cover.add(id)
}

// Removes id from the cover set. If id is covered transitively (an ancestor is
// in the cover set), the ancestor is replaced with explicit entries for each
// sibling along the path so the rest of the subtree stays selected.
function uncheckResource(id: string, cover: Set<string>) {
  const chain = getAncestorChain(id) // [id, parent, ..., root]
  let highestIdx = -1
  for (let i = chain.length - 1; i >= 0; i--) {
    if (cover.has(chain[i])) { highestIdx = i; break }
  }
  if (highestIdx > 0) {
    cover.delete(chain[highestIdx])
    for (let i = highestIdx; i > 0; i--) {
      const parent = chain[i]
      const next = chain[i - 1]
      for (const sib of childrenMap.value.get(parent) || []) {
        if (sib.external_id !== next) cover.add(sib.external_id)
      }
    }
  }
  cover.delete(id)
  const descendants = new Set(getDescendantIds(id))
  for (const d of [...cover]) {
    if (descendants.has(d)) cover.delete(d)
  }
}

function toggleResource(id: string) {
  const cover = new Set(selectedResourceIds.value)
  if ((checkStates.value.get(id) || 'unchecked') === 'unchecked') {
    checkResource(id, cover)
  } else {
    uncheckResource(id, cover)
  }
  selectedResourceIds.value = [...cover]
}

function validateRssFeedUrls(): boolean {
  if (form.value.type !== 'rss') return true
  if (!String(form.value.config.settings.feed_urls || '').trim()) {
    MessagePlugin.warning(`${'Feed URLs'} ${'is required'}`)
    return false
  }
  return true
}

function validateStep1Fields(): boolean {
  syncRssAuthHeadersToCredentials()
  if (!validateRssFeedUrls()) return false
  if (isEdit.value && credentialsConfigured.value && !replaceCredentialsMode.value) {
    return true
  }

  const fields = displayedCredentialFields.value
  for (const f of fields) {
    if (f.optional || f.fieldType === 'custom_headers') continue
    if (!form.value.config.credentials[f.key]) {
      MessagePlugin.warning(`${f.labelKey} ${'is required'}`)
      return false
    }
  }
  return true
}

async function nextStep() {
  if (step.value === 1) {
    if (!validateStep1Fields()) return
    if (needsConnectionTest() && testResult.value !== 'success') {
      await testConnection()
      if ((testResult.value as string) !== 'success') return
    }
  }
  if (step.value === 2 && isGitLabConnector(form.value.type)) {
    syncGitLabProjectsToSettings()
    if (!gitlabProjects.value.some(project => project.project_id.trim())) {
      MessagePlugin.warning('Add at least one GitLab project')
      return
    }
  }
  step.value++
  if (step.value === 2) {
    if (isGitLabConnector(form.value.type)) return
    loadResources()
  }
}

function prevStep() {
  step.value--
}

// Build the config payload for Create / Update requests.
//
// Create mode: credentials flow inline so the initial data source row
// already carries them.
//
// Edit mode: credentials NEVER flow through the main PUT — they go via the
// /credentials subresource, committed before the main submit (see
// commitCredentialsIfNeeded). Sending an empty map keeps the backend
// validator happy.
function buildConfigPayload(): Record<string, unknown> {
  syncGitLabProjectsToSettings()
  syncConfluencePublicFieldsToSettings()
  return {
    credentials: isEdit.value ? {} : { ...form.value.config.credentials },
    resource_ids: form.value.config.resource_ids,
    settings: form.value.config.settings,
  }
}

// In edit mode, when the user opted in to Replace credentials and typed at
// least one value, commit it to /credentials before the main PUT. Aborts
// the whole submit on failure so we don't leave the row partially saved.
async function commitCredentialsIfNeeded(dsId: string): Promise<boolean> {
  if (!isEdit.value || !replaceCredentialsMode.value) return true
  syncRssAuthHeadersToCredentials()
  syncConfluencePublicFieldsToSettings()
  const filled = Object.entries(form.value.config.credentials).filter(
    ([, v]) => typeof v === 'string' ? v !== '' : v != null,
  )
  if (filled.length === 0) return true
  try {
    await putDataSourceCredentials(dsId, Object.fromEntries(filled))
    credentialsConfigured.value = true
    replaceCredentialsMode.value = false
    form.value.config.credentials = {}
    rssAuthHeaders.value = []
    return true
  } catch (e: any) {
    MessagePlugin.error(e?.message || e?.error || 'Failed to save credential')
    return false
  }
}

// --- Final submit ---
async function handleSubmit() {
  form.value.config.resource_ids = selectedResourceIds.value
  submitting.value = true
  try {
    let dataSourceId = tempDsId.value

    if (tempDsId.value) {
      // Commit credential replacement BEFORE the main PUT so a validation
      // failure on credentials doesn't leave us with an updated row that
      // still points at the old broken token.
      const credsOk = await commitCredentialsIfNeeded(tempDsId.value)
      if (!credsOk) {
        submitting.value = false
        return
      }
      await updateDataSource(tempDsId.value, {
        ...form.value,
        config: buildConfigPayload(),
        knowledge_base_id: props.kbId,
        status: 'active',
      } as any)
    } else {
      const res = await createDataSource({
        ...form.value,
        config: buildConfigPayload(),
        knowledge_base_id: props.kbId,
        status: 'active',
      } as any)
      const created = res?.data || res
      dataSourceId = created.id
      tempDsId.value = created.id
    }

    if (isEdit.value) {
      MessagePlugin.warning('Data source updated. Editing does not start a sync automatically; click Sync to import changes.')
    } else {
      try {
        await triggerSync(dataSourceId)
        MessagePlugin.success('Data source created and sync task submitted')
      } catch (e: any) {
        MessagePlugin.warning(e?.message || e?.error || 'Data source created, but failed to trigger sync')
      }
    }

    emit('saved')
    // Clear before close — otherwise the visible watcher treats the just-saved
    // row as an abandoned temp draft and DELETEs it (loadResources creates the
    // row early at step 2 with tempDsId).
    tempDsId.value = ''
    visible.value = false
  } catch (e: any) {
    MessagePlugin.error(e?.message || e?.error || 'Failed to save')
  }
  submitting.value = false
}

function handleClose() {
  visible.value = false
}

async function handleDrawerConfirm() {
  if (step.value === 1 || step.value === 2) {
    await nextStep()
  } else if (step.value === 3) {
    handleSubmit()
  }
}

const selectedResourceCount = computed(() => {
  let count = 0
  for (const state of checkStates.value.values()) {
    if (state === 'checked') count++
  }
  return count
})

const hasExpandableNodes = computed(() => resources.value.some(r => r.has_children))

function resourceIconName(r: Resource): string {
  if (r.has_children) return 'folder'
  switch (r.type) {
    case 'wiki_space':
      return 'root-list'
    case 'book':
      return 'book'
    case 'doc_category':
      return 'folder-open'
    default:
      return 'file'
  }
}

function expandAllNodes() {
  const expandable = resources.value.filter(r => r.has_children)
  expandedResourceIds.value = new Set(expandable.map(r => r.external_id))
  // Lazily load children of any expanded node that hasn't been fetched yet.
  for (const r of expandable) {
    void ensureChildrenLoaded(r.external_id)
  }
}

function collapseAllNodes() {
  expandedResourceIds.value = new Set()
}

const resourceTypeLabelMap: Record<string, string> = {
  wiki_space: 'Wiki Space',
  doc_category: 'Document Tag',
}

function resourceTypeLabel(type: string): string {
  const key = resourceTypeLabelMap[type]
  if (key) return key
  return ''
}

function shouldShowResourceType(type: string): boolean {
  return !!resourceTypeLabelMap[type]
}

function resourceRowState(id: string): CheckState {
  return checkStates.value.get(id) || 'unchecked'
}

const stepTitles = computed(() => [
  'Select Type',
  'Credentials',
  'Resources',
  'Strategy',
])

const drawerTitle = computed(() =>
  isEdit.value ? 'Edit Data Source' : 'Add Data Source',
)

const drawerDescription = computed(() => stepTitles.value[step.value] ?? '')

const drawerConfirmText = computed(() => {
  if (step.value === 3) {
    return isEdit.value ? 'Save' : 'Create & Sync Now'
  }
  if (step.value >= 1) return 'Next'
  return 'Save'
})
</script>

<template>
  <SettingDrawer
    v-model:visible="visible"
    :title="drawerTitle"
    :description="drawerDescription"
    :class="[form.type ? `datasource-editor-drawer datasource-editor-drawer--${form.type}` : 'datasource-editor-drawer', { 'ds-fixed-step': step === 2 && !isGitLabConnector(form.type) }]"
    :hide-footer="step === 0"
    :confirm-text="drawerConfirmText"
    :confirm-loading="submitting || (step === 1 && testing)"
    storage-key="setting-drawer:width:datasource-editor"
    width="640px"
    @confirm="handleDrawerConfirm"
    @cancel="handleClose"
  >
    <template v-if="form.type && getDatasourceIconUrl(form.type)" #headerIcon>
      <img
        :src="getDatasourceIconUrl(form.type)"
        :alt="form.type"
        class="datasource-header-icon__img"
      >
    </template>

    <template v-if="step === 1" #footer-left>
      <t-button v-if="!isEdit" variant="outline" @click="step = 0">
        {{ 'Back' }}
      </t-button>
      <t-button variant="outline" :loading="testing" @click="testConnection">
        <template #icon>
          <t-icon
            v-if="!testing && testResult === 'success'"
            name="check-circle-filled"
            class="status-icon available"
          />
          <t-icon
            v-else-if="!testing && testResult === 'error'"
            name="close-circle-filled"
            class="status-icon unavailable"
          />
        </template>
        {{ testing ? 'Testing...' : 'Test Connection' }}
      </t-button>
      <span
        v-if="testResult"
        :class="['footer-test-message', testResult === 'success' ? 'success' : 'error']"
        :title="testResult === 'error' ? testErrorMsg : 'Connected'"
      >
        {{
          testResult === 'success'
            ? 'Connected'
            : (testErrorMsg || 'Connection failed')
        }}
      </span>
    </template>

    <template v-else-if="step === 2 || step === 3" #footer-left>
      <t-button variant="outline" @click="prevStep">
        {{ 'Back' }}
      </t-button>
    </template>

    <!-- Step indicator -->
    <div class="ds-steps">
      <div
        v-for="(title, i) in stepTitles"
        :key="i"
        :class="['ds-step', { active: step === i, done: step > i }]"
      >
        <span class="ds-step-num">
          <t-icon v-if="step > i" name="check" class="ds-step-check" />
          <template v-else>{{ i + 1 }}</template>
        </span>
        <span class="ds-step-title">{{ title }}</span>
      </div>
    </div>

    <!-- Step 0: Select connector type -->
    <section v-if="step === 0" class="setting-drawer__section">
      <h4 class="setting-drawer__section-title">{{ 'Select Type' }}</h4>
      <div class="ds-type-grid">
        <button
          v-for="def in connectorDefs"
          :key="def.type"
          type="button"
          :class="['ds-type-card', { disabled: !def.available }]"
          :disabled="!def.available"
          @click="selectType(def)"
        >
          <div class="ds-type-header">
            <DataSourceTypeIcon :type="def.type" :size="20" />
            <span class="ds-type-name">{{ (DATASOURCE_CONNECTOR_LABELS[def.type] ?? '') }}</span>
            <span v-if="!def.available" class="ds-type-soon">{{ 'Coming soon' }}</span>
          </div>
          <div class="ds-type-desc">{{ (DATASOURCE_CONNECTOR_DESC_LABELS[def.type] ?? '') }}</div>
        </button>
      </div>
    </section>

    <!-- Step 1: Credentials -->
    <template v-if="step === 1">
      <div
        v-if="currentDef && currentDef.requiredPermissions.length > 0"
        class="ds-setup-guide ds-setup-guide--standalone"
      >
        <button
          type="button"
          class="ds-setup-guide__toggle"
          :aria-expanded="prereqExpanded"
          @click="prereqExpanded = !prereqExpanded"
        >
          <t-icon name="info-circle-filled" size="15px" class="ds-setup-guide__icon" />
          <span class="ds-setup-guide__summary">
            {{ (DATASOURCE_PREREQ_BAR_TEXT[form.type] ?? 'First time? Click to see the setup guide') }}
          </span>
          <t-icon
            :name="prereqExpanded ? 'chevron-up' : 'chevron-down'"
            size="14px"
            class="ds-setup-guide__chevron"
          />
        </button>
        <div v-if="prereqExpanded" class="ds-setup-guide__body">
          <ol class="ds-setup-steps">
            <li class="ds-setup-step">
              <span class="ds-setup-step__title">{{ (DATASOURCE_PREREQ_STEP1_BRIEF[form.type] ?? 'Add \u0022Bot\u0022 capability to your app') }}</span>
              <span class="ds-setup-step__desc">{{ (DATASOURCE_PREREQ_STEP1_DESC[form.type] ?? 'Open Platform > Add App Capability > Bot > create version and publish') }}</span>
            </li>
            <li class="ds-setup-step">
              <span class="ds-setup-step__title">{{ (DATASOURCE_PREREQ_STEP2_BRIEF[form.type] ?? 'Grant API permissions') }}</span>
              <span class="ds-setup-step__desc">
                <template v-if="!(DATASOURCE_PREREQ_STEP2_DESC_LABELS[form.type] ?? '')">
                  <code
                    v-for="perm in currentDef.requiredPermissions"
                    :key="perm"
                    class="ds-perm-tag"
                  >{{ perm }}</code>
                </template>
                <template v-else>{{ (DATASOURCE_PREREQ_STEP2_DESC_LABELS[form.type] ?? '') }}</template>
              </span>
            </li>
            <li class="ds-setup-step">
              <span class="ds-setup-step__title">{{ (DATASOURCE_PREREQ_STEP3_BRIEF[form.type] ?? 'Add app to wiki via group chat') }}</span>
              <span class="ds-setup-step__desc">{{ (DATASOURCE_PREREQ_STEP3_DESC[form.type] ?? 'Create group chat > add app as bot > add group chat as wiki member') }}</span>
            </li>
          </ol>
          <a
            v-if="currentDef.permissionPageUrl"
            :href="currentDef.permissionPageUrl"
            target="_blank"
            rel="noopener"
            class="doc-link ds-setup-guide__link"
          >
            {{ (DATASOURCE_PREREQ_OPEN_CONSOLE[form.type] ?? 'Open the provider console') }}
            <t-icon name="link" class="link-icon" />
          </a>
        </div>
      </div>

      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Basic information' }}</h4>

        <div v-if="currentDef?.docUrl" class="inline-alert">
          <t-icon name="info-circle-filled" class="inline-alert__icon" />
          <span class="inline-alert__text">{{ 'Get credentials at:' }}</span>
          <a
            :href="currentDef.docUrl"
            target="_blank"
            rel="noopener"
            class="inline-alert__action doc-link"
          >
            {{ 'Open documentation' }}
            <t-icon name="link" class="link-icon" />
          </a>
        </div>

        <div class="form-item">
          <label class="form-label required">{{ 'Name' }}</label>
          <t-input v-model="form.name" :placeholder="'Enter data source name'" />
        </div>
      </section>

      <section v-if="form.type === 'rss'" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Feed URLs' }}</h4>
        <div class="form-item">
          <label class="form-label required">{{ 'Feed URLs' }}</label>
          <t-textarea
            v-model="form.config.settings.feed_urls"
            placeholder="https://example.com/feed.xml"
            :autosize="{ minRows: 2, maxRows: 6 }"
            autocomplete="off"
            spellcheck="false"
          />
          <p class="form-desc">{{ 'One RSS / Atom feed URL per line; multiple feeds are supported.' }}</p>
        </div>
      </section>

      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'credentials' }}</h4>

        <div v-if="isEdit && credentialsConfigured && !replaceCredentialsMode" class="form-item">
          <div
            class="credential-faux-input"
            :class="{ 'is-confirm-remove': pendingRemoveCredentials }"
            :title="pendingRemoveCredentials ? '' : 'Configured'"
          >
            <template v-if="pendingRemoveCredentials">
              <t-icon name="error-circle-filled" class="credential-status-icon warn" />
              <span class="credential-faux-text danger">{{ 'Remove this credential? This cannot be undone.' }}</span>
              <div class="credential-actions">
                <t-button size="small" variant="text" @click="cancelPendingRemoveCredentials">
                  {{ 'Cancel' }}
                </t-button>
                <span class="action-divider" />
                <t-button
                  size="small"
                  variant="text"
                  theme="danger"
                  :loading="removingCredentials"
                  @click="confirmRemoveCredentials"
                >
                  {{ 'Confirm remove' }}
                </t-button>
              </div>
            </template>
            <template v-else>
              <t-icon name="check-circle-filled" class="credential-status-icon success" />
              <span class="credential-faux-text">{{ 'Configured' }}</span>
              <div class="credential-actions">
                <t-button size="small" variant="text" @click="enterReplaceCredentials">
                  {{ 'Replace' }}
                </t-button>
                <span class="action-divider" />
                <t-button size="small" variant="text" theme="danger" @click="requestRemoveCredentials">
                  {{ 'Remove' }}
                </t-button>
              </div>
            </template>
          </div>
        </div>

        <div
          v-else-if="isEdit && !credentialsConfigured && !replaceCredentialsMode"
          class="form-item"
        >
          <div
            class="credential-faux-input is-empty"
            @click="enterReplaceCredentials"
          >
            <t-icon name="lock-on" class="credential-status-icon muted" />
            <span class="credential-faux-text muted">{{ 'Not configured' }}</span>
            <div class="credential-actions">
              <t-button size="small" variant="text" theme="primary" @click.stop="enterReplaceCredentials">
                {{ 'Configure' }}
              </t-button>
            </div>
          </div>
        </div>

        <template v-else-if="credentialsInputVisible">
          <div v-if="form.type === 'confluence'" class="form-item">
            <label class="form-label">{{ 'Edition' }}</label>
            <t-select v-model="form.config.credentials.edition">
              <t-option value="server" :label="'Server / Data Center'" />
              <t-option value="cloud" :label="'Cloud'" />
            </t-select>
          </div>
          <div
            v-for="field in displayedCredentialFields"
            :key="field.key"
            class="form-item"
          >
            <template v-if="field.fieldType === 'custom_headers'">
              <div class="custom-headers-header">
                <label class="form-label" style="margin-bottom: 0;">{{ field.labelKey }}</label>
                <t-button variant="text" size="small" theme="primary" @click="addRssAuthHeader">
                  <template #icon><t-icon name="add" /></template>
                  {{ 'Add Header' }}
                </t-button>
              </div>
              <p v-if="field.hintKey" class="form-desc custom-headers-desc">{{ field.hintKey }}</p>
              <div v-if="rssAuthHeaders.length > 0" class="custom-headers-list">
                <div v-for="(item, idx) in rssAuthHeaders" :key="idx" class="custom-header-row">
                  <t-input
                    v-model="item.key"
                    :placeholder="'Header name'"
                    class="custom-header-key"
                    autocomplete="off"
                    spellcheck="false"
                  />
                  <t-input
                    v-model="item.value"
                    :placeholder="'Header value'"
                    class="custom-header-value"
                    autocomplete="off"
                    spellcheck="false"
                  />
                  <t-button
                    variant="text"
                    shape="square"
                    size="small"
                    class="custom-header-remove"
                    :aria-label="'Delete'"
                    @click="removeRssAuthHeader(idx)"
                  >
                    <t-icon name="close" />
                  </t-button>
                </div>
              </div>
            </template>
            <template v-else>
              <label class="form-label" :class="{ required: !field.optional }">
                {{ field.labelKey }}
              </label>
              <t-textarea
                v-if="field.multiline"
                v-model="form.config.credentials[field.key]"
                :placeholder="field.placeholder || 'Enter value'"
                :autosize="{ minRows: 2, maxRows: 6 }"
                autocomplete="off"
                spellcheck="false"
              />
              <t-input
                v-else
                v-model="form.config.credentials[field.key]"
                :placeholder="field.placeholder || 'Enter value'"
                :type="field.secret ? 'password' : 'text'"
                autocomplete="off"
                spellcheck="false"
              >
                <template v-if="field.secret" #prefix-icon><t-icon name="lock-on" /></template>
              </t-input>
              <p v-if="field.hintKey" class="form-desc">{{ field.hintKey }}</p>
            </template>
          </div>
          <div v-if="isEdit && replaceCredentialsMode" class="credential-edit-actions">
            <t-button size="small" variant="text" @click="cancelReplaceCredentials">
              {{ 'Cancel' }}
            </t-button>
          </div>
        </template>
      </section>
    </template>

    <!-- Step 2: Select resources -->
    <section v-if="step === 2" class="setting-drawer__section ds-resource-section">
      <template v-if="isGitLabConnector(form.type)">
        <h4 class="setting-drawer__section-title">{{ 'GitLab projects' }}</h4>
        <p class="ds-resource-hint">{{ 'Enter a project ID or namespace path (for example group/project), with optional branch and directories.' }}</p>
        <div class="gitlab-project-list">
          <div v-for="(project, index) in gitlabProjects" :key="index" class="gitlab-project-row">
            <div class="gitlab-project-row__header">
              <strong>{{ 'Project' }} {{ index + 1 }}</strong>
              <t-button variant="text" size="small" theme="danger" @click="removeGitLabProject(index)"><t-icon name="delete" /></t-button>
            </div>
            <label class="form-label required">{{ 'Project ID' }}</label>
            <t-input v-model="project.project_id" :placeholder="'For example: 12345 or group/project'" />
            <label class="form-label">{{ 'Branch' }}</label>
            <t-input v-model="project.ref" :placeholder="'Leave empty to use the default branch'" />
            <label class="form-label">{{ 'Directories' }}</label>
            <t-textarea v-model="project.pathsText" :placeholder="'One directory per line; leave empty to sync the whole project'" :autosize="{ minRows: 2, maxRows: 5 }" />
          </div>
          <t-button variant="outline" @click="addGitLabProject"><template #icon><t-icon name="add" /></template>{{ 'Add project' }}</t-button>
        </div>
      </template>
      <template v-else>
      <h4 class="setting-drawer__section-title">{{ 'Resources' }}</h4>
      <p class="ds-resource-hint">{{ 'Select the spaces or folders to sync' }}</p>

      <div v-if="loadingResources" class="ds-loading-center"><t-loading /></div>
      <div v-else-if="resources.length > 0" class="resource-picker">
        <div class="resource-picker__toolbar">
          <span class="resource-picker__count">
            {{ `${selectedResourceCount} selected` }}
          </span>
          <div v-if="hasExpandableNodes" class="resource-picker__actions">
            <button type="button" class="resource-picker__action" @click="expandAllNodes">
              {{ 'Expand children' }}
            </button>
            <span class="resource-picker__action-sep" aria-hidden="true">·</span>
            <button type="button" class="resource-picker__action" @click="collapseAllNodes">
              {{ 'Collapse children' }}
            </button>
          </div>
        </div>
        <div class="resource-picker__list" role="tree">
          <div
            v-for="{ resource: r, depth } in visibleTree"
            :key="r.external_id"
            class="resource-picker__row"
            :class="{
              'is-checked': resourceRowState(r.external_id) === 'checked',
              'is-indeterminate': resourceRowState(r.external_id) === 'indeterminate',
            }"
            :style="{ '--depth': depth }"
            role="treeitem"
            :aria-expanded="r.has_children ? expandedResourceIds.has(r.external_id) : undefined"
            @click="toggleResource(r.external_id)"
          >
            <button
              v-if="r.has_children"
              type="button"
              class="resource-picker__expand"
              :aria-label="expandedResourceIds.has(r.external_id)
                ? 'Collapse children'
                : 'Expand children'"
              @click.stop="toggleExpand(r.external_id)"
            >
              <t-loading v-if="loadingChildrenIds.has(r.external_id)" size="12px" />
              <t-icon
                v-else
                :name="expandedResourceIds.has(r.external_id) ? 'chevron-down' : 'chevron-right'"
                size="12px"
              />
            </button>
            <span v-else class="resource-picker__expand-spacer" aria-hidden="true" />
            <span
              class="resource-picker__check"
              :class="{
                'is-checked': resourceRowState(r.external_id) === 'checked',
                'is-indeterminate': resourceRowState(r.external_id) === 'indeterminate',
              }"
              aria-hidden="true"
            >
              <svg
                v-if="resourceRowState(r.external_id) === 'checked'"
                width="10"
                height="10"
                viewBox="0 0 12 12"
                fill="none"
              >
                <path
                  d="M10 3L4.5 8.5L2 6"
                  stroke="#fff"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="resource-picker__icon" aria-hidden="true">
              <t-icon :name="resourceIconName(r)" size="16px" />
            </span>
            <span class="resource-picker__label">
              <span class="resource-picker__name" :title="r.name || 'Untitled'">
                {{ r.name || 'Untitled' }}
              </span>
              <span
                v-if="shouldShowResourceType(r.type)"
                class="resource-picker__type"
              >{{ resourceTypeLabel(r.type) }}</span>
            </span>
          </div>
        </div>
      </div>
      <div v-else class="ds-resource-empty">
        <t-icon name="info-circle" size="32px" style="color: var(--td-warning-color); margin-bottom: 8px;" />
        <p class="ds-empty-title">{{ 'No wiki spaces found' }}</p>
        <p class="ds-empty-desc">{{ (DATASOURCE_NO_RESOURCES_DESC[form.type] ?? 'The app needs wiki access via a group chat to fetch content') }}</p>
        <div class="ds-guide-steps">
          <div class="ds-guide-step">
            <span class="ds-guide-num">1</span>
            <span>{{ (DATASOURCE_GUIDE_STEP1[form.type] ?? 'Create an integration or application in the provider console') }}</span>
          </div>
          <div class="ds-guide-step">
            <span class="ds-guide-num">2</span>
            <span>{{ (DATASOURCE_GUIDE_STEP2[form.type] ?? 'Open wiki \u0022Settings\u0022 > \u0022Member Settings\u0022 > \u0022Add Member\u0022, search for the group chat and add it') }}</span>
          </div>
          <div class="ds-guide-step">
            <span class="ds-guide-num">3</span>
            <span>{{ (DATASOURCE_GUIDE_STEP3[form.type] ?? 'Ensure the group chat role is at least \u0022Can Read\u0022, then come back and click Retry') }}</span>
          </div>
        </div>
        <div class="ds-empty-actions">
          <button type="button" class="ds-empty-retry" @click="loadResources">
            {{ 'Retry' }}
          </button>
          <a
            v-if="currentDef?.permissionDocUrl"
            :href="currentDef.permissionDocUrl"
            target="_blank"
            rel="noopener"
            class="doc-link"
          >
            {{ 'View the provider permission docs' }}
            <t-icon name="link" class="link-icon" />
          </a>
        </div>
      </div>
      </template>
    </section>

    <!-- Step 3: Sync strategy -->
    <template v-if="step === 3">
      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Sync schedule' }}</h4>
        <t-select v-model="form.sync_schedule">
          <t-option v-for="p in schedulePresets" :key="p.value" :value="p.value" :label="p.label" />
        </t-select>
      </section>

      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Sync mode' }}</h4>
        <div class="form-item form-item--flat">
          <div class="option-group" role="radiogroup" :aria-label="'Sync mode'">
            <button
              type="button"
              class="option-pill"
              :class="{ 'is-active': form.sync_mode === 'incremental' }"
              role="radio"
              :aria-checked="form.sync_mode === 'incremental'"
              @click="form.sync_mode = 'incremental'"
            >
              {{ 'Incremental' }}
            </button>
            <button
              type="button"
              class="option-pill"
              :class="{ 'is-active': form.sync_mode === 'full' }"
              role="radio"
              :aria-checked="form.sync_mode === 'full'"
              @click="form.sync_mode = 'full'"
            >
              {{ 'Full' }}
            </button>
          </div>
        </div>

        <div class="form-item form-item--flat">
          <label class="form-label">{{ 'Conflict strategy' }}</label>
          <div class="option-group" role="radiogroup" :aria-label="'Conflict strategy'">
            <button
              type="button"
              class="option-pill"
              :class="{ 'is-active': form.conflict_strategy === 'overwrite' }"
              role="radio"
              :aria-checked="form.conflict_strategy === 'overwrite'"
              @click="form.conflict_strategy = 'overwrite'"
            >
              {{ 'Overwrite' }}
            </button>
            <button
              type="button"
              class="option-pill"
              :class="{ 'is-active': form.conflict_strategy === 'skip' }"
              role="radio"
              :aria-checked="form.conflict_strategy === 'skip'"
              @click="form.conflict_strategy = 'skip'"
            >
              {{ 'Skip existing' }}
            </button>
          </div>
        </div>

        <div class="form-item form-item--flat">
          <t-checkbox v-model="form.sync_deletions">{{ 'Sync deletions (remove knowledge when deleted at source)' }}</t-checkbox>
        </div>
      </section>
    </template>
  </SettingDrawer>
</template>

<style scoped lang="less">
@import './datasource-surface.less';
.ds-steps {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  border-bottom: 1px solid var(--td-component-stroke);
  padding-bottom: 14px;
}

.ds-step {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
  font-size: var(--app-text-md);
  color: var(--td-text-color-placeholder);
}

.ds-step-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ds-step.active {
  color: var(--td-brand-color);
  font-weight: 500;
}

.ds-step.done {
  color: var(--td-text-color-secondary);
  font-weight: 500;
}

.ds-step-num {
  flex-shrink: 0;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--app-text-sm);
  font-weight: 600;
  border: 1px solid var(--td-component-stroke);
  color: var(--td-text-color-placeholder);
  background: transparent;
}

.ds-step.active .ds-step-num {
  background: var(--td-brand-color);
  color: #fff;
  border-color: var(--td-brand-color);
}

.ds-step.done .ds-step-num {
  background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  color: var(--td-brand-color);
  border-color: transparent;
}

.ds-step-check {
  font-size: var(--app-text-base);
}

.ds-loading-center {
  text-align: center;
  padding: 24px;
}

/* --- Step 0: type cards --- */
.ds-type-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
}

.ds-type-card {
  .ds-surface-card--interactive();
  padding: 14px;
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: inherit;
}

.ds-type-card.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ds-type-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.ds-type-name {
  font-size: var(--app-text-md);
  font-weight: 600;
}

.ds-type-soon {
  font-size: var(--app-text-2xs);
  color: var(--td-text-color-placeholder);
  background: var(--td-bg-color-component);
  padding: 1px 6px;
  border-radius: 3px;
}

.ds-type-desc {
  font-size: var(--app-text-xs);
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}

/* --- Step 1: setup guide + credentials (align with ModelEditor / CredentialResource) --- */
.inline-alert {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-secondary);
  flex-wrap: wrap;
}

.inline-alert__icon {
  font-size: var(--app-text-lg);
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
}

.inline-alert__text {
  flex: 1 1 auto;
  min-width: 0;
}

.inline-alert__action {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-brand-color);
  white-space: nowrap;
  transition: color var(--app-motion-fast) ease;
}

.inline-alert__action:hover {
  color: var(--td-brand-color-active);
}

.ds-setup-guide {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.ds-setup-guide--standalone {
  margin-bottom: 4px;
}

.ds-setup-guide__toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 0;
  border: none;
  background: transparent;
  font: inherit;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-secondary);
  text-align: left;
  cursor: pointer;
  transition: color var(--app-motion-instant) ease;
}

.ds-setup-guide__toggle:hover,
.ds-setup-guide__toggle:focus-visible {
  color: var(--td-text-color-primary);
  outline: none;
}

.ds-setup-guide__icon {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
}

.ds-setup-guide__summary {
  flex: 1;
  min-width: 0;
}

.ds-setup-guide__chevron {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
}

.ds-setup-guide__body {
  padding: 0 0 0 23px;
}

.ds-setup-steps {
  margin: 10px 0 0;
  padding: 0 0 0 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ds-setup-step {
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-primary);
}

.ds-setup-step__title {
  display: block;
  font-weight: 500;
  margin-bottom: 2px;
}

.ds-setup-step__desc {
  display: block;
  color: var(--td-text-color-secondary);
}

.ds-perm-tag {
  display: inline-block;
  font-size: var(--app-text-xs);
  padding: 1px 5px;
  margin: 2px 4px 2px 0;
  border-radius: 3px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family-mono, ui-monospace, monospace);
}

.ds-setup-guide__link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
  font-size: var(--app-text-md);
}

.credential-faux-input {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 4px 0 12px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--app-radius-sm);
  font-size: var(--app-text-md);
  transition: border-color var(--app-motion-fast) ease, background-color var(--app-motion-fast) ease;
}

.credential-faux-input:hover {
  border-color: var(--td-brand-color-hover);
}

.credential-faux-input.is-empty {
  cursor: pointer;
}

.credential-faux-input.is-empty:hover {
  background: var(--td-bg-color-container-hover);
}

.credential-faux-input.is-confirm-remove {
  background: var(--td-error-color-light);
  border-color: var(--td-error-color-focus);
}

.credential-faux-text {
  flex: 1;
  min-width: 0;
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.credential-faux-text.muted {
  color: var(--td-text-color-placeholder);
}

.credential-faux-text.danger {
  color: var(--td-error-color);
  font-weight: 500;
}

.credential-status-icon {
  flex-shrink: 0;
  font-size: var(--app-text-xl);
}

.credential-status-icon.success {
  color: var(--td-success-color);
}

.credential-status-icon.muted {
  color: var(--td-text-color-placeholder);
}

.credential-status-icon.warn {
  color: var(--td-error-color);
}

.credential-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.credential-actions :deep(.t-button--variant-text) {
  height: 24px;
  padding: 0 8px;
  font-size: var(--app-text-sm);
  border-radius: var(--app-radius-xs);
}

.action-divider {
  width: 1px;
  height: 14px;
  background: var(--td-component-stroke);
  margin: 0 2px;
}

.credential-edit-actions {
  display: flex;
  justify-content: flex-end;
}

.credential-edit-actions :deep(.t-button) {
  height: 28px;
  padding: 0 12px;
  font-size: var(--app-text-sm);
}

.form-item {
  margin-bottom: 0;
}

.form-item--flat {
  margin-bottom: 0;
}

.form-item--flat :deep(.t-checkbox__label) {
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-secondary);
}

.form-label {
  display: block;
  font-size: var(--app-text-md);
  font-weight: 500;
  margin-bottom: 6px;
  color: var(--td-text-color-primary);
  line-height: 1.4;

  &.required::before {
    content: '*';
    color: var(--td-error-color);
    margin-right: 4px;
    font-weight: 500;
    line-height: 1;
  }
}

.form-desc {
  margin: 4px 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.status-icon {
  font-size: var(--app-text-xl);
  flex-shrink: 0;
}

.status-icon.available {
  color: var(--td-brand-color);
}

.status-icon.unavailable {
  color: var(--td-error-color);
}

.footer-test-message {
  font-size: var(--app-text-sm);
  line-height: 1.4;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.footer-test-message.success {
  color: var(--td-brand-color-active);
}

.footer-test-message.error {
  color: var(--td-error-color);
}

/* --- Step 2: resource picker (compact flat tree, matches KB selector) --- */
.ds-resource-section {
  gap: 10px !important;
}

.ds-resource-hint {
  margin: -8px 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

/* Drive root folder_token input - shown before the lazy-load tree. */
.drive-folder-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-container);
}

.drive-folder-input__label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-primary);

  &.required::before {
    content: '*';
    color: var(--td-error-color);
    font-weight: 500;
    line-height: 1;
  }
}

.drive-folder-input__help {
  font-size: var(--app-text-lg);
  color: var(--td-text-color-placeholder);
  cursor: help;

  &:hover {
    color: var(--td-text-color-secondary);
  }
}

.drive-folder-input__row {
  display: flex;
  gap: 8px;
  align-items: center;
  padding-bottom: 20px
}

/* Drive tree placeholder: shown before the first successful load. */
.ds-drive-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 120px;
  padding: 24px 12px;
  border: 1px dashed var(--td-border-level-2-color);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-page);
  text-align: center;
}

.ds-drive-placeholder .ds-empty-title {
  margin: 0;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.ds-drive-placeholder .ds-empty-desc {
  margin: 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.resource-picker {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.resource-picker__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0;
  background: transparent;
}

.resource-picker__count {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
}

.resource-picker__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.resource-picker__action {
  padding: 0;
  border: none;
  background: transparent;
  font: inherit;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: color var(--app-motion-instant) ease;
}

.resource-picker__action:hover,
.resource-picker__action:focus-visible {
  color: var(--td-text-color-secondary);
  outline: none;
}

.resource-picker__action-sep {
  color: var(--td-text-color-disabled);
  font-size: var(--app-text-sm);
  user-select: none;
}

.resource-picker__list {
  .ds-inset-panel();
  min-height: 360px;
  max-height: min(calc(100vh - 260px), 600px);
  overflow-y: auto;
  padding: 4px 6px;
  overscroll-behavior: contain;
}

.resource-picker__row {
  --depth: 0;
  position: relative;
  display: grid;
  grid-template-columns: 16px 16px 16px 1fr;
  align-items: center;
  column-gap: 8px;
  min-height: 34px;
  margin-bottom: 2px;
  padding: 5px 8px 5px calc(8px + var(--depth) * 14px);
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  transition: background var(--app-motion-instant) ease;
}

.resource-picker__row:last-child {
  margin-bottom: 0;
}

.resource-picker__row:hover,
.resource-picker__row.is-checked,
.resource-picker__row.is-indeterminate {
  background: var(--td-bg-color-secondarycontainer);
}

.resource-picker__expand,
.resource-picker__expand-spacer {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.resource-picker__expand {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: none;
  border-radius: var(--app-radius-xs);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: background var(--app-motion-instant) ease, color var(--app-motion-instant) ease;
}

.resource-picker__expand:hover,
.resource-picker__expand:focus-visible {
  background: color-mix(in srgb, var(--td-text-color-placeholder) 12%, transparent);
  color: var(--td-text-color-secondary);
  outline: none;
}

.resource-picker__check {
  width: 16px;
  height: 16px;
  border-radius: 3px;
  border: 1.5px solid var(--td-component-border);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-sizing: border-box;
  transition: background var(--app-motion-instant) ease, border-color var(--app-motion-instant) ease;
}

.resource-picker__check.is-checked,
.resource-picker__check.is-indeterminate {
  background: var(--td-brand-color);
  border-color: var(--td-brand-color);
}

.resource-picker__check.is-indeterminate::after {
  content: '';
  width: 8px;
  height: 2px;
  border-radius: 1px;
  background: #fff;
}

.resource-picker__icon {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-secondary);
  flex-shrink: 0;
}

.resource-picker__label {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.resource-picker__name {
  min-width: 0;
  font-size: var(--app-text-md);
  line-height: 1.4;
  color: var(--td-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.resource-picker__type {
  flex-shrink: 0;
  font-size: var(--app-text-2xs);
  line-height: 1;
  padding: 2px 5px;
  border-radius: var(--app-radius-xs);
  color: var(--td-text-color-placeholder);
  background: color-mix(in srgb, var(--td-text-color-placeholder) 8%, transparent);
}

/* --- Step 2: empty state --- */
.ds-resource-empty {
  text-align: center;
  padding: 24px 0;
}

.ds-empty-title {
  font-size: var(--app-text-base);
  font-weight: 600;
  color: var(--td-text-color-primary);
  margin: 0 0 4px;
}

.ds-empty-desc {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  margin: 0 0 16px;
}

.ds-guide-steps {
  display: flex;
  flex-direction: column;
  gap: 8px;
  text-align: left;
  max-width: 440px;
  margin: 0 auto 16px;
}

.ds-guide-step {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: var(--app-text-md);
  color: var(--td-text-color-primary);
  line-height: 1.5;
}

.ds-guide-num {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-top: 1px;
}

.ds-empty-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
}

.custom-headers-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.custom-headers-desc {
  margin: 0 0 10px 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.custom-headers-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.custom-header-row {
  display: flex;
  align-items: center;
  gap: 8px;

  .custom-header-key {
    flex: 0 0 38%;
  }

  .custom-header-value {
    flex: 1;
  }

  .custom-header-remove {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    padding: 0;
    color: var(--td-text-color-placeholder);
    border-radius: var(--app-radius-sm);
    transition: all 0.18s ease;

    &:hover {
      background: var(--td-error-color-light);
      color: var(--td-error-color);
    }
  }
}

.ds-empty-retry {
  padding: 0;
  border: none;
  background: transparent;
  font: inherit;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-brand-color);
  cursor: pointer;
  transition: color var(--app-motion-instant) ease;
}

.ds-empty-retry:hover,
.ds-empty-retry:focus-visible {
  color: var(--td-brand-color-active);
  outline: none;
}

/* --- Step 3: sync strategy option pills --- */
.option-group {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  width: fit-content;
  max-width: 100%;
}

.option-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 4px 10px;
  min-height: 28px;
  border: 1px solid transparent;
  border-radius: var(--app-radius-sm);
  background: transparent;
  font: inherit;
  font-size: var(--app-text-sm);
  line-height: 1.3;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  white-space: nowrap;
  transition: background var(--app-motion-fast) ease, color var(--app-motion-fast) ease, border-color var(--app-motion-fast) ease;
}

.option-pill:hover {
  color: var(--td-text-color-primary);
}

.option-pill:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 1px;
}

.option-pill.is-active {
  background: var(--td-bg-color-container);
  border-color: var(--td-component-stroke);
  color: var(--td-text-color-primary);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.05);
}

.gitlab-project-list {
  display: grid;
  gap: 12px;
  margin-bottom: 20px;
}

.gitlab-project-row {
  display: grid;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-container);
}

.gitlab-project-row__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>

<!--
  Drawer header logo — same white badge as the data-source list cards.
-->
<style lang="less">
.datasource-editor-drawer .setting-drawer__header-icon:has(.datasource-header-icon__img) {
  background: var(--td-bg-color-container);
  box-shadow: inset 0 0 0 1px var(--td-component-stroke);
}

.datasource-header-icon__img {
  display: block;
  width: 24px;
  height: 24px;
  object-fit: contain;
}

/* Step 2 "Resources": the whole step never scrolls - the token input stays
   fixed while the resource area (placeholder / loading / empty / tree) fills
   the remaining drawer height and the tree list scrolls internally. */
.ds-fixed-step {
  .t-drawer__body {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .setting-drawer__body {
    flex: 1;
    min-height: 0;
  }

  .ds-resource-section {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .resource-picker,
  .ds-drive-placeholder,
  .ds-loading-center,
  .ds-resource-empty {
    flex: 1;
    min-height: 0;
  }

  .ds-loading-center {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .resource-picker__list {
    flex: 1;
    min-height: 0;
    max-height: none;
  }
}
</style>
