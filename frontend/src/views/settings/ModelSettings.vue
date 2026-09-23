<template>
  <div class="model-settings">
    <div class="section-header">
      <div class="section-header__top">
        <div>
          <h2>{{ 'Model Settings' }}</h2>
          <p class="section-description">{{ 'Manage different types of AI models, including local Ollama and remote APIs' }}</p>
        </div>
        <t-button
          v-if="authStore.hasRole('admin')"
          type="button"
          theme="primary"
          variant="text"
          size="medium"
          class="model-test-trigger"
          @click="showDebugDrawer = true"
        >
          <template #icon><play-circle-icon /></template>
          {{ 'Model Test' }}
        </t-button>
      </div>

      <div class="builtin-models-hint" role="note">
        <p class="builtin-hint-label">{{ 'Built-in Models' }}</p>
        <p class="builtin-hint-text">
          {{ (authStore.isSystemAdmin ? 'Built-in models are visible to all workspaces. System administrators can edit configuration and credentials; deletion remains deployment-managed.' : 'Built-in models are visible to all workspaces. Sensitive information is hidden, and they cannot be edited or deleted.') }}
        </p>
        <a class="doc-link" href="https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/README.md" target="_blank"
          rel="noopener noreferrer">
          {{ 'View Built-in Models Guide' }}
          <t-icon name="link" class="link-icon" />
        </a>
      </div>
    </div>

    <t-tabs v-model="activeTypeFilter" class="model-type-tabs" data-guide="settings-models">
      <t-tab-panel value="all" :label="`${'All'}(${allLegacyModels.length})`" />
      <t-tab-panel value="chat" :label="`${'Chat'}(${countByType('chat')})`" />
      <t-tab-panel value="embedding"
        :label="`${'Embedding'}(${countByType('embedding')})`" />
      <t-tab-panel value="rerank" :label="`${'ReRank'}(${countByType('rerank')})`" />
      <t-tab-panel value="vllm" :label="`${'Vision'}(${countByType('vllm')})`" />
      <t-tab-panel value="asr" :label="`${'Speech'}(${countByType('asr')})`" />
    </t-tabs>

    <t-loading :loading="loading" size="small" class="model-list-loading">
      <div v-if="!loading && filteredModels.length === 0 && !authStore.hasRole('admin')" class="empty-state">
        <t-empty :description="emptyHint" />
      </div>
      <div v-else-if="!loading" class="model-grid">
        <div v-for="model in filteredModels" :key="`${model._modelType}-${model.id}`" class="model-card" :class="[
          `model-card--${model._modelType}`,
          {
            'model-card--builtin': model.isBuiltin,
            'model-card--clickable': isModelCardClickable(model),
          },
        ]" :role="isModelCardClickable(model) ? 'button' : undefined"
          :tabindex="isModelCardClickable(model) ? 0 : undefined"
          @click="onModelCardClick($event, model._modelType, model)"
          @keydown.enter="onModelCardClick($event, model._modelType, model)">
          <div class="model-card__badge" :aria-label="typeLabel(model._modelType)">
            <t-icon :name="typeIcon(model._modelType)" size="18px" />
          </div>
          <div class="model-card__body">
            <div class="model-card__header">
              <h3 class="model-card__title">{{ modelDisplayName(model) }}</h3>
              <span v-if="model.isBuiltin" class="model-card__lock" :title="'Built-in'"
                :aria-label="'Built-in'">
                <t-icon :name="authStore.isSystemAdmin ? 'edit-1' : 'lock-on'" />
              </span>
              <div v-if="canManageModel(model)" class="model-card__actions" @click.stop>
                <t-dropdown :options="getModelOptions(model._modelType, model)" placement="bottom-right" attach="body"
                  trigger="click"
                  @click="(data: any) => handleMenuAction({ value: data.value }, model._modelType, model)">
                  <t-button variant="text" shape="square" size="small" class="model-card__action-btn model-card__more">
                    <t-icon name="ellipsis" />
                  </t-button>
                </t-dropdown>
                <t-popconfirm
                  v-if="canDeleteModel(model)"
                  :content="`Delete model \u0022${modelDisplayName(model)}\u0022?`"
                  :confirm-btn="{ content: 'Delete', theme: 'danger' }"
                  :cancel-btn="{ content: 'Cancel' }"
                  placement="bottom-right"
                  @confirm="deleteModel(model._modelType, model.id)"
                >
                  <t-tooltip :content="'Delete'" placement="top">
                    <t-button
                      theme="danger"
                      shape="square"
                      variant="text"
                      size="small"
                      class="model-card__action-btn model-card__delete"
                      @click.stop
                    >
                      <template #icon><t-icon name="delete" /></template>
                    </t-button>
                  </t-tooltip>
                </t-popconfirm>
              </div>
            </div>
            <p class="model-card__subtitle">
              <span class="model-card__vendor">
                <img v-if="vendorIcon(model)" :src="vendorIcon(model)" class="model-card__vendor-icon" alt="" />
                <span>{{ vendorLabel(model) }}</span>
              </span>
              <template v-if="model._modelType === 'embedding' && model.dimension">
                <span class="model-card__sep">·</span>
                <span>{{ 'Vector Dimension' }} {{ model.dimension }}</span>
              </template>
              <template v-if="model._modelType === 'chat' || model._modelType === 'vllm'">
                <span class="model-card__sep">·</span>
                <span
                  class="model-card__ctx"
                  :class="{ 'model-card__ctx--default': isDefaultContextWindow(model.contextWindow) }"
                  :title="contextWindowTitle(model.contextWindow)"
                >{{ formatContextWindow(model.contextWindow) }}</span>
              </template>
              <template v-if="model._modelType === 'chat' && model.supportsVision">
                <span class="model-card__sep">·</span>
                <span class="model-card__vision" :title="'Supports Vision / Multimodal'"
                  :aria-label="'Supports Vision / Multimodal'">
                  <t-icon name="image" size="12px" />
                </span>
              </template>
            </p>
          </div>
        </div>
        <button
          v-if="authStore.hasRole('admin')"
          type="button"
          class="model-card model-card--add"
          data-guide="settings-add-model"
          @click="openAddDialog"
        >
          <span class="model-card--add__icon" aria-hidden="true">
            <add-icon />
          </span>
          <span class="model-card--add__label">{{ 'Add Model' }}</span>
        </button>
      </div>
    </t-loading>

    <t-dialog
      v-model:visible="showUsageDialog"
      :header="'Model cannot be deleted'"
      :footer="false"
      width="680px"
      destroy-on-close
    >
      <div v-if="usageConflict" class="model-usage-dialog">
        <p class="model-usage-dialog__description">
          {{ `Model \u0022${usageConflictModelName}\u0022 is still referenced by the following settings. Open each configuration and choose another model before deleting it.` }}
        </p>

        <div class="model-usage-dialog__content">
          <section v-if="usageConflict.knowledge_bases.length" class="model-usage-group">
            <h3>
              {{ `Knowledge bases (${modelUsageResourceCount(usageConflict.knowledge_bases, usageConflict.knowledge_base_total)})` }}
            </h3>
            <ul>
              <li v-for="resource in usageConflict.knowledge_bases" :key="resource.id">
                <div class="model-usage-resource">
                  <strong>{{ resource.name || resource.id }}</strong>
                  <div class="model-usage-bindings">
                    <t-tag
                      v-for="(binding, index) in resource.bindings"
                      :key="`${binding}-${index}`"
                      size="small"
                      variant="light"
                    >
                      {{ modelUsageBindingLabel(binding) }}
                    </t-tag>
                  </div>
                </div>
                <t-button
                  theme="primary"
                  variant="text"
                  size="small"
                  @click="openUsageResource('knowledge_base', resource.id, resource.bindings)"
                >
                  {{ 'Open settings' }}
                </t-button>
              </li>
            </ul>
            <p
              v-if="modelUsageListTruncated(usageConflict.knowledge_bases, usageConflict.knowledge_base_total)"
              class="model-usage-truncated"
            >
              {{ `Showing the first ${usageConflict.knowledge_bases.length} of ${modelUsageResourceCount(usageConflict.knowledge_bases, usageConflict.knowledge_base_total)}` }}
            </p>
          </section>

          <section v-if="usageConflict.agents.length" class="model-usage-group">
            <h3>{{ `Agents (${modelUsageResourceCount(usageConflict.agents, usageConflict.agent_total)})` }}</h3>
            <ul>
              <li v-for="resource in usageConflict.agents" :key="resource.id">
                <div class="model-usage-resource">
                  <strong>{{ resource.name || resource.id }}</strong>
                  <div class="model-usage-bindings">
                    <t-tag
                      v-for="(binding, index) in resource.bindings"
                      :key="`${binding}-${index}`"
                      size="small"
                      variant="light"
                    >
                      {{ modelUsageBindingLabel(binding) }}
                    </t-tag>
                  </div>
                </div>
                <t-button
                  theme="primary"
                  variant="text"
                  size="small"
                  @click="openUsageResource('agent', resource.id, resource.bindings)"
                >
                  {{ 'Open settings' }}
                </t-button>
              </li>
            </ul>
            <p
              v-if="modelUsageListTruncated(usageConflict.agents, usageConflict.agent_total)"
              class="model-usage-truncated"
            >
              {{ `Showing the first ${usageConflict.agents.length} of ${modelUsageResourceCount(usageConflict.agents, usageConflict.agent_total)}` }}
            </p>
          </section>

          <section v-if="usageConflict.long_term_memory.bindings.length" class="model-usage-group">
            <h3>{{ 'Long-term memory' }}</h3>
            <div class="model-usage-memory">
              <div class="model-usage-bindings">
                <t-tag
                  v-for="(binding, index) in usageConflict.long_term_memory.bindings"
                  :key="`${binding}-${index}`"
                  size="small"
                  variant="light"
                >
                  {{ modelUsageBindingLabel(binding) }}
                </t-tag>
              </div>
            </div>
          </section>
        </div>

        <div class="model-usage-dialog__actions">
          <t-button @click="showUsageDialog = false">{{ 'Close' }}</t-button>
        </div>
      </div>
    </t-dialog>

    <ModelEditorDialog v-model:visible="showDialog" :model-type="currentModelType" :model-data="editingModel"
      :save-model="handleModelSave" />
    <ModelDebugDrawer v-model:visible="showDebugDrawer" :models="allModels" />

  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { AddIcon, PlayCircleIcon } from 'tdesign-icons-vue-next'
import { useRouter } from 'vue-router'
import ModelEditorDialog from '@/components/ModelEditorDialog.vue'
import ModelDebugDrawer from '@/components/ModelDebugDrawer.vue'
import {
  listModels,
  createModel,
  updateModel as updateModelAPI,
  deleteModel as deleteModelAPI,
  ModelInUseError,
  modelUsageBindingLabel,
  modelUsageKnowledgeBaseSection,
  modelUsageListTruncated,
  modelUsageResourceCount,
  modelUsageResourceRoute,
  type ModelConfig,
  type ModelUsageDetails,
  type ModelUsageResourceKind,
} from '@/api/model'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { focusKbEditorSection } from '@/config/contextualGuides'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useModelProvidersStore } from '@/stores/modelProviders'
import {
  formatContextWindow,
  isDefaultContextWindow,
  effectiveContextWindow,
} from '@/utils/contextWindow'

const authStore = useAuthStore()
const uiStore = useUIStore()
const chatResources = useChatResourcesStore()
const providersStore = useModelProvidersStore()
const router = useRouter()
type ModelType = 'chat' | 'embedding' | 'rerank' | 'vllm' | 'asr'
type FilterType = 'all' | ModelType

const showDialog = ref(false)
const showDebugDrawer = ref(false)
const showUsageDialog = ref(false)
const usageConflict = ref<ModelUsageDetails | null>(null)
const usageConflictModelName = ref('')
const currentModelType = ref<ModelType>('chat')
const editingModel = ref<any>(null)
const loading = ref(true)
const activeTypeFilter = ref<FilterType>('all')

const MODEL_TAB_TYPES: FilterType[] = ['chat', 'embedding', 'rerank', 'vllm', 'asr']
const KNOWLEDGE_BASE_EDITOR_HOST_ROUTES = new Set([
  'home',
  'knowledgeBaseList',
  'knowledgeBaseDetail',
  'globalCreatChat',
  'kbCreatChat',
  'chat',
])
watch(
  () => uiStore.settingsInitialSubSection,
  (sub) => {
    if (sub && MODEL_TAB_TYPES.includes(sub as FilterType)) {
      activeTypeFilter.value = sub as FilterType
    }
  },
  { immediate: true },
)

const allModels = ref<ModelConfig[]>([])

const backendTypeToModelType: Record<string, ModelType> = {
  KnowledgeQA: 'chat',
  Embedding: 'embedding',
  Rerank: 'rerank',
  VLLM: 'vllm',
  ASR: 'asr'
}

// apiKey is always blank here: the server's main GET response does not
// include it (see internal/handler/dto/model.go — ModelParametersDTO omits
// secret fields). Credential read/write happens inside the editor dialog
// via the dedicated /credentials subresource.
function convertToLegacyFormat(model: ModelConfig) {
  return {
    id: model.id!,
    name: model.name,
    displayName: model.display_name || '',
    source: model.source,
    modelName: model.name,
    baseUrl: model.parameters.base_url || '',
    apiKey: '',
    provider: model.parameters.provider || '',
    dimension: model.parameters.embedding_parameters?.dimension,
    supportsDimensionOverride: model.parameters.embedding_parameters?.supports_dimension_override || false,
    isBuiltin: model.is_builtin || false,
    supportsVision: model.parameters.supports_vision || false,
    contextWindow: model.parameters.context_window || undefined,
    maxConcurrency: model.parameters.max_concurrency,
    maxOutputTokens: model.parameters.max_output_tokens || undefined,
    customHeaders: model.parameters.custom_headers
      ? Object.entries(model.parameters.custom_headers).map(([key, value]) => ({ key, value: String(value) }))
      : [],
    extraConfig: Object.fromEntries(
      Object.entries(model.parameters.extra_config || {}).filter(([key]) => key !== 'thinking_control'),
    ) as Record<string, string>,
    thinkingControl: model.parameters.extra_config?.thinking_control || '',
    spec: model.parameters.spec || null,
    capabilities: model.capabilities,
    _modelType: backendTypeToModelType[model.type] || 'chat' as ModelType,
    // Preserve the credential metadata map so the editor dialog can render
    // the "Configured" state without an extra round-trip.
    credentials: model.credentials,
  }
}

const allLegacyModels = computed(() => allModels.value.map(convertToLegacyFormat))
const filteredModels = computed(() => {
  if (activeTypeFilter.value === 'all') return allLegacyModels.value
  return allLegacyModels.value.filter(m => m._modelType === activeTypeFilter.value)
})

const countByType = (type: ModelType) => allLegacyModels.value.filter(m => m._modelType === type).length

const typeIcon = (type: ModelType): string => {
  const map: Record<ModelType, string> = {
    chat: 'chat',
    embedding: 'chart-bubble',
    rerank: 'filter-sort',
    vllm: 'image',
    asr: 'sound',
  }
  return map[type]
}

const typeLabel = (type: ModelType) => {
  const map: Record<ModelType, string> = {
    chat: 'Chat',
    embedding: 'Embedding',
    rerank: 'ReRank',
    vllm: 'Vision',
    asr: 'Speech'
  }
  return map[type]
}

const sourceLabel = (type: ModelType) => {
  if (type === 'vllm' || type === 'asr') {
    return 'OpenAI-compatible'
  }
  return 'Remote'
}

// Maps a backend `provider` id (e.g. "openai", "anthropic", "gemini")
// to the vendor's localized name from the backend catalog (modelProviders
// store), so the model card and the editor dropdown always agree. Falls
// back to the raw id when the catalog has not loaded yet, and to '' when
// the backend didn't store a provider — caller falls back to sourceLabel().
const providerLabel = (model: any): string => {
  const id = model.provider
  if (!id) return ''
  return providersStore.labelFor(id, 'en-US') || id
}

const vendorIcon = (model: any): string => {
  if (model.source === 'local' || !model.provider) return ''
  return providersStore.iconFor(model.provider)
}

// What the vendor chip on a card shows. Keeps the chip text uniformly
// short so cards line up:
//   local  → "Ollama"
//   remote → the provider's short name. For the catch-all "generic"
//            provider we render a single short word ("Custom") — the
//            editor dropdown's longer "OpenAI-compatible" label blows
//            out the card chip row, and that framing isn't meaningful to
//            most end users (they didn't pick "I want OpenAI
//            compatibility", they just pasted a base URL).
const vendorLabel = (model: any): string => {
  if (model.source === 'local') return 'Ollama'
  if (model.provider === 'generic') {
    return 'Custom'
  }
  return providerLabel(model) || sourceLabel(model._modelType)
}

const modelDisplayName = (model: any) => {
  const displayName = typeof model.displayName === 'string' ? model.displayName.trim() : ''
  return displayName || model.name
}

const contextWindowTitle = (tokens?: number) => {
  if (isDefaultContextWindow(tokens)) {
    return `Unset, using default ${formatContextWindow(tokens)}`
  }
  return `${effectiveContextWindow(tokens)} tokens`
}

const emptyHint = computed(() => {
  if (activeTypeFilter.value === 'all') return 'No chat models'
  const map: Record<ModelType, string> = {
    chat: 'No chat models',
    embedding: 'No embedding models',
    rerank: 'No re-rank models',
    vllm: 'No VLLM models',
    asr: 'No ASR models'
  }
  return map[activeTypeFilter.value as ModelType]
})

const loadModels = async () => {
  loading.value = true
  // Vendor icons / display names come from the catalog store. Loaded in
  // parallel with the model list; a failure must not block card rendering.
  void providersStore.ensureLoaded('').catch(() => {})
  try {
    const models = await listModels()
    allModels.value = models
    // Write straight back into the workspace-level cache after this page's
    // own listModels. Otherwise the chat composer / agent editor keep
    // serving the stale context_window from the 60s TTL until a page reload.
    chatResources.replaceModels(models)
  } catch (error: any) {
    console.error('Failed to load model list:', error)
    MessagePlugin.error(error.message)
  } finally {
    loading.value = false
  }
}

const openAddDialog = () => {
  currentModelType.value = activeTypeFilter.value === 'all' ? 'chat' : activeTypeFilter.value
  editingModel.value = null
  showDialog.value = true
}

// Tenant Admin+ manages tenant models; only SystemAdmin manages shared
// built-in models. The backend repeats this distinction authoritatively.
const canEditModel = (model: any) =>
  model.isBuiltin ? authStore.isSystemAdmin : authStore.hasRole('admin')

const isModelCardClickable = (model: any) => canEditModel(model)

const canManageModel = (model: any) => canEditModel(model)

// Built-in lifecycle remains deployment-managed (YAML / SQL). The UI only
// exposes configuration and credential editing to SystemAdmin.
const canDeleteModel = (model: any) =>
  authStore.hasRole('admin') && !model.isBuiltin

const onModelCardClick = (event: Event, type: ModelType, model: any) => {
  if (!isModelCardClickable(model)) return
  if (event.type === 'keydown') {
    const ke = event as KeyboardEvent
    if (ke.key !== 'Enter' && ke.key !== ' ') return
    ke.preventDefault()
  }
  const target = event.target as HTMLElement | null
  if (target?.closest('.model-card__actions')) return
  editModel(type, model)
}

const editModel = (type: ModelType, model: any) => {
  if (model.isBuiltin && !authStore.isSystemAdmin) {
    MessagePlugin.warning('Built-in models cannot be edited')
    return
  }
  if (!model.isBuiltin && !authStore.hasRole('admin')) {
    return
  }
  currentModelType.value = type
  editingModel.value = { ...model }
  showDialog.value = true
}

const handleModelSave = async (modelData: any) => {
  const saveType: ModelType = modelData.modelType ?? currentModelType.value
  currentModelType.value = saveType

  try {
    if (!modelData.modelName || !modelData.modelName.trim()) {
      throw new Error('Model name cannot be empty')
    }

    if (modelData.modelName.trim().length > 100) {
      throw new Error('Model name cannot exceed 100 characters')
    }

    if (modelData.displayName && modelData.displayName.trim().length > 100) {
      throw new Error('Display name cannot exceed 100 characters')
    }

    if (modelData.source === 'remote') {
      if (!modelData.baseUrl || !modelData.baseUrl.trim()) {
        throw new Error('Base URL is required for remote APIs')
      }

      try {
        new URL(modelData.baseUrl.trim())
      } catch {
        throw new Error('Invalid Base URL, please enter a valid URL')
      }
    }

    if (saveType === 'embedding') {
      if (!modelData.dimension || modelData.dimension < 128 || modelData.dimension > 4096) {
        throw new Error('Embedding dimension must be between 128 and 4096')
      }
    }

    const customHeadersMap: Record<string, string> = {}
    if (Array.isArray(modelData.customHeaders)) {
      for (const item of modelData.customHeaders) {
        const key = (item?.key ?? '').trim()
        const value = (item?.value ?? '').trim()
        if (key && value) {
          customHeadersMap[key] = value
        }
      }
    }

    // api_key flows in only on initial create (modelData.apiKey is wiped on
    // every edit-mode open). Edits to existing models commit credentials via
    // the /credentials subresource (handled inside ModelEditorDialog).
    const trimmedApiKey = (modelData.apiKey ?? '').trim()
    const apiKeyFields: { api_key?: string } =
      !editingModel.value && trimmedApiKey ? { api_key: trimmedApiKey } : {}
    const trimmedAppSecret = (modelData.appSecret ?? '').trim()
    const appSecretFields: { app_secret?: string } =
      !editingModel.value && trimmedAppSecret ? { app_secret: trimmedAppSecret } : {}
    // extra_config: vendor extra fields + advanced overrides (already trimmed
    // by the editor) plus the legacy thinking_control for rows that still
    // carry it. Empty values are dropped so cleared keys disappear.
    const extraConfig: Record<string, string> = {}
    if (modelData.source === 'remote') {
      for (const [key, value] of Object.entries(modelData.extraConfig || {})) {
        const trimmed = String(value ?? '').trim()
        if (key && trimmed) extraConfig[key] = trimmed
      }
      const legacyThinking = String(modelData.thinkingControl || '').trim()
      if (saveType === 'chat' && legacyThinking) {
        extraConfig.thinking_control = legacyThinking
      } else {
        delete extraConfig.thinking_control
      }
    }
    // Always send extra_config for remote models, even when it ends up empty:
    // PUT /models/:id restores the stored map when the field is absent
    // (internal/handler/model.go), so omitting it would make "clear the last
    // vendor field / protocol override" silently no-op. Local rows keep the
    // omit-when-empty behaviour — this form never edits their extra_config.
    const extraConfigFields = modelData.source === 'remote'
      ? { extra_config: extraConfig }
      : {}

    // parameters.spec: keep whatever the row already had, replace compat with
    // the (validated) JSON from the advanced textarea; drop spec when empty.
    const specText = String(modelData.specCompat ?? '').trim()
    const baseSpec = (modelData.spec && typeof modelData.spec === 'object') ? { ...modelData.spec } : {}
    if (specText) {
      const compat = JSON.parse(specText)
      if (!compat || typeof compat !== 'object' || Array.isArray(compat)) {
        throw new Error('Invalid JSON')
      }
      baseSpec.compat = compat
    } else {
      delete baseSpec.compat
    }
    const specFields = Object.keys(baseSpec).length > 0 ? { spec: baseSpec } : {}

    const apiModelData: ModelConfig = {
      name: modelData.modelName.trim(),
      display_name: modelData.displayName?.trim() || '',
      type: getModelType(saveType),
      source: modelData.source,
      description: '',
      parameters: {
        base_url: modelData.baseUrl?.trim() || '',
        ...apiKeyFields,
        ...appSecretFields,
        provider: modelData.provider || '',
        ...extraConfigFields,
        ...(Object.keys(customHeadersMap).length > 0 ? { custom_headers: customHeadersMap } : {}),
        ...(saveType === 'embedding' && modelData.dimension ? {
          embedding_parameters: {
            dimension: modelData.dimension,
            truncate_prompt_tokens: 0,
            supports_dimension_override: modelData.supportsDimensionOverride ?? false
          }
        } : {}),
        ...(saveType === 'vllm' ? {
          supports_vision: true
        } : saveType === 'chat' ? {
          supports_vision: modelData.supportsVision ?? false
        } : {}),
        ...((saveType === 'chat' || saveType === 'vllm')
          && Number(modelData.contextWindow) >= 1024
          ? { context_window: Math.round(Number(modelData.contextWindow)) }
          : {}),
        ...((saveType === 'chat' || saveType === 'vllm')
          && Number(modelData.maxOutputTokens) > 0
          ? { max_output_tokens: Math.round(Number(modelData.maxOutputTokens)) }
          : {}),
        ...specFields,
        // Backend concurrency cap: only chat/embedding/vllm are governed, and only >0 is sent (0/empty keeps the global default).
        ...(['chat', 'embedding', 'vllm'].includes(saveType)
          && Number(modelData.maxConcurrency) > 0
          ? { max_concurrency: Number(modelData.maxConcurrency) }
          : {})
      }
    }

    if (editingModel.value && editingModel.value.id) {
      await updateModelAPI(editingModel.value.id, apiModelData)
      MessagePlugin.success('Model updated')
    } else {
      await createModel(apiModelData)
      MessagePlugin.success('Model added')
    }

    await loadModels()
  } catch (error: any) {
    console.error('Failed to save model:', error)
    throw error
  }
}

const deleteModel = async (_type: ModelType, modelId: string) => {
  const model = allModels.value.find(m => m.id === modelId)
  if (model?.is_builtin) {
    MessagePlugin.warning('Built-in models cannot be deleted')
    return
  }

  try {
    await deleteModelAPI(modelId)
    MessagePlugin.success('Model deleted')
    await loadModels()
  } catch (error: any) {
    console.error('Failed to delete model:', error)
    if (error instanceof ModelInUseError) {
      usageConflict.value = error.details
      usageConflictModelName.value = model?.display_name || model?.name || modelId
      showUsageDialog.value = true
      return
    }
    MessagePlugin.error(error.message || 'Failed to delete model')
  }
}

const openUsageResource = async (
  kind: ModelUsageResourceKind,
  id: string,
  bindings: readonly string[] = [],
) => {
  showUsageDialog.value = false
  const knowledgeBaseSection = modelUsageKnowledgeBaseSection(bindings)
  // The UI store can outlive the route component that owns the KB editor.
  // Only focus an existing editor when the current route still mounts one;
  // otherwise a stale visible/id pair would make a direct Settings page no-op.
  const routeName = router.currentRoute.value.name
  const canFocusExistingKnowledgeBase = kind === 'knowledge_base'
    && uiStore.showKBEditorModal
    && uiStore.currentKBId === id
    && typeof routeName === 'string'
    && KNOWLEDGE_BASE_EDITOR_HOST_ROUTES.has(routeName)

  uiStore.closeSettings()
  await nextTick()

  if (canFocusExistingKnowledgeBase) {
    // Keep unsaved edits when the referenced KB is already open underneath
    // global Settings; only move the existing editor to the relevant section.
    focusKbEditorSection(knowledgeBaseSection)
    return
  }

  if (kind === 'knowledge_base') {
    uiStore.closeKBEditor()
    await nextTick()
  }
  // Both the KB editor and global Settings can already be open underneath the
  // usage dialog. Flush their false state before reusing either component so
  // its visible watcher reloads the target object instead of keeping stale data.
  await router.push(modelUsageResourceRoute(kind, id, bindings))
  if (kind === 'knowledge_base') {
    uiStore.openKBSettings(id, knowledgeBaseSection)
  }
}

const getModelOptions = (type: ModelType, model: any) => {
  const options: any[] = []

  if (model.isBuiltin) {
    if (authStore.isSystemAdmin) {
      options.push({
        content: 'Edit',
        value: `edit-${type}-${model.id}`
      })
    }
    return options
  }

  // Models are tenant-wide infrastructure (LLM credentials); the
  // backend gates every mutation behind Admin+ (see RegisterModelRoutes).
  // Non-Admins get an empty action menu — viewing is fine, but editing,
  // copying (also goes through createModel), and deleting are not.
  if (!authStore.hasRole('admin')) {
    return options
  }

  options.push({
    content: 'Edit',
    value: `edit-${type}-${model.id}`
  })

  options.push({
    content: 'Copy',
    value: `copy-${type}-${model.id}`
  })

  return options
}

const handleMenuAction = (data: { value: string }, type: ModelType, model: any) => {
  const value = data.value

  if (value.indexOf('edit-') === 0) {
    editModel(type, model)
  } else if (value.indexOf('copy-') === 0) {
    copyModel(type, model.id)
  }
}

const generateCopyName = (originalName: string): string => {
  const suffix = ' Copy'
  const existingNames = new Set(allModels.value.map(m => m.name))
  let candidate = `${originalName}${suffix}`
  let counter = 2
  while (existingNames.has(candidate)) {
    candidate = `${originalName}${suffix} ${counter}`
    counter += 1
  }
  return candidate
}

const copyModel = async (_type: ModelType, modelId: string) => {
  const source = allModels.value.find(m => m.id === modelId)
  if (!source) {
    return
  }
  if (source.is_builtin) {
    MessagePlugin.warning('Built-in models cannot be copied')
    return
  }

  try {
    const newModel: ModelConfig = {
      name: generateCopyName(source.name),
      display_name: source.display_name || '',
      type: source.type,
      source: source.source,
      description: source.description || '',
      parameters: JSON.parse(JSON.stringify(source.parameters || {}))
    }

    await createModel(newModel)
    MessagePlugin.success('Model copied')
    await loadModels()
  } catch (error: any) {
    console.error('Failed to copy model:', error)
    MessagePlugin.error(error.message || 'Failed to copy model')
  }
}

function getModelType(type: ModelType): 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM' | 'ASR' {
  const typeMap = {
    chat: 'KnowledgeQA' as const,
    embedding: 'Embedding' as const,
    rerank: 'Rerank' as const,
    vllm: 'VLLM' as const,
    asr: 'ASR' as const
  }
  return typeMap[type]
}

onMounted(() => {
  loadModels()
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/provider-card.less';

@import (reference) '@/components/css/settings-section.less';

.model-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.model-test-trigger {
  --td-bg-color-container-hover: transparent;
  flex-shrink: 0;
  padding-left: 0;
  padding-right: 0;
  font-weight: 600;

  &:hover,
  &:focus,
  &.t-is-active,
  &:active {
    background-color: transparent !important;
    color: var(--td-brand-color-hover);
  }

  &:active {
    color: var(--td-brand-color-active);
  }
}

.builtin-models-hint {
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
}

.builtin-hint-label {
  margin: 0 0 4px 0;
  font-size: var(--app-text-sm);
  font-weight: 500;
  color: var(--td-text-color-placeholder);
  letter-spacing: 0.02em;
}

.builtin-hint-text {
  margin: 0 0 6px 0;
  font-size: var(--app-text-md);
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.builtin-models-hint .doc-link {
  font-size: var(--app-text-md);
}

.model-list-loading {
  min-height: 120px;
}

.model-type-tabs {
  margin-bottom: 16px;

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

.model-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;

  .model-card--add {
    width: 100%;
    height: 100%;
  }
}

.model-card {
  .provider-card();
  .provider-card-interactive();

  &--add {
    .provider-card-add();

    &:hover,
    &:focus-visible {
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

  &--builtin {
    background: var(--td-bg-color-secondarycontainer);

    &:hover {
      box-shadow: none;
      border-color: var(--td-component-stroke);
    }
  }

  &--clickable {
    .provider-card-interactive();


  }
}

.model-card__badge {
  .provider-card-badge();
  .provider-card-badge-color(#0052d9);
}

.model-card--chat .model-card__badge {
  .provider-card-badge-color(#0052d9);
}

.model-card--embedding .model-card__badge {
  .provider-card-badge-color(#6235bb);
}

.model-card--rerank .model-card__badge {
  .provider-card-badge-color(#b85c00);
}

.model-card--vllm .model-card__badge {
  .provider-card-badge-color(#c93e3e);
}

.model-card--asr .model-card__badge {
  .provider-card-badge-color(#118053);
}

.model-card__body {
  .provider-card-body();
}

.model-card__header {
  .provider-card-header();
}

.model-card__title {
  .provider-card-title();
}

/*
  Built-in lock indicator. Most cards in a typical install ARE built-in,
  so loud styling everywhere becomes noise — instead the lock is muted
  and small by default, and lights up on hover. The signal that matters
  to users is "which models did I add" → user-added cards stand out by
  the absence of the lock.
*/
.model-card__lock {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  color: var(--td-text-color-placeholder);
  opacity: 0.6;
  transition: color var(--app-motion-fast) ease, opacity var(--app-motion-fast) ease;

  .t-icon {
    font-size: var(--app-text-md);
  }
}

.model-card:hover .model-card__lock {
  opacity: 1;
  color: var(--td-text-color-secondary);
}

.model-card__subtitle {
  // A flex row, not inline text: .model-card__vendor is an inline-flex box
  // whose baseline comes from its first item — the 14px icon, whose baseline
  // is its bottom edge — so as inline content it sat a couple of pixels off
  // the "· 200K" beside it. Aligning the row by centre instead of by
  // baseline puts every part of the line on one optical line.
  display: flex;
  align-items: center;
  min-width: 0;
  margin: 2px 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-secondary);
}

.model-card__vendor {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 0;

  // The vendor name is the only part long enough to need truncating; the
  // context window and the badges after it must stay readable.
  > span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.model-card__sep,
.model-card__ctx,
.model-card__vision {
  flex: none;
  white-space: nowrap;
}

.model-card__vendor-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  border-radius: 3px;
  object-fit: contain;
}

.model-card__sep {
  margin: 0 4px;
  color: var(--td-text-color-placeholder);
}

.model-card__vision {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.model-card__ctx {
  font-variant-numeric: tabular-nums;
}

.model-card__ctx--default {
  color: var(--td-text-color-placeholder);
}

.model-card__actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 2px;
}

.model-card__action-btn {
  flex-shrink: 0;
  padding: 2px;
  opacity: 0;
  transition: opacity var(--app-motion-fast) ease;
}

.model-card__more {
  color: var(--td-text-color-placeholder);

  &:hover,
  &:focus-visible {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }
}

.model-card:hover .model-card__action-btn,
.model-card:focus-within .model-card__action-btn,
.model-card__actions:focus-within .model-card__action-btn {
  opacity: 1;
}

.empty-state {
  padding: 64px 0;
  text-align: center;

  :deep(.t-empty__description) {
    font-size: var(--app-text-base);
    color: var(--td-text-color-placeholder);
    margin-bottom: 16px;
  }
}

.model-usage-dialog__description {
  margin: 0 0 16px;
  color: var(--td-text-color-secondary);
  line-height: 1.6;
}

.model-usage-dialog__content {
  max-height: min(58vh, 560px);
  overflow-y: auto;
  padding-right: 4px;
}

.model-usage-group {
  & + & {
    margin-top: 20px;
  }

  h3 {
    margin: 0 0 8px;
    font-size: var(--app-text-base);
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
    border: 1px solid var(--td-component-stroke);
    border-radius: var(--app-radius-md);
    overflow: hidden;
  }

  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 12px;

    & + li {
      border-top: 1px solid var(--td-component-stroke);
    }
  }
}

.model-usage-truncated {
  margin: 8px 0 0;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}

.model-usage-resource {
  min-width: 0;

  strong {
    display: block;
    margin-bottom: 6px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.model-usage-bindings {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.model-usage-memory {
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
}

.model-usage-dialog__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}
</style>
