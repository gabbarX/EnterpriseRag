<template>
  <div class="model-selector">
    <t-select
      :value="selectedModelId"
      @change="handleModelChange"
      :placeholder="placeholderText"
      :disabled="disabled"
      :loading="loading"
      :status="status"
      :clearable="clearable"
      filterable
      style="width: 100%;"
    >
      <template v-if="selectedModel && showContextWindow" #valueDisplay>
        <span class="selected-model">
          <span class="selected-model__name">{{ modelDisplayName(selectedModel) }}</span>
          <span
            class="model-ctx"
            :class="{ 'model-ctx--default': isDefaultContextWindow(selectedModel.parameters?.context_window) }"
            :title="contextWindowTitle(selectedModel.parameters?.context_window)"
          >{{ formatContextWindow(selectedModel.parameters?.context_window) }}</span>
        </span>
      </template>
      <t-option
        v-for="model in models"
        :key="model.id"
        :value="model.id"
        :label="modelDisplayName(model)"
      >
        <div class="model-option">
          <t-icon name="check-circle-filled" class="model-icon" />
          <span class="model-name">{{ modelDisplayName(model) }}</span>
          <span v-if="model.display_name" class="model-raw-name">{{ model.name }}</span>
          <t-tag v-if="model.is_builtin" size="small" theme="primary">{{ 'Built-in' }}</t-tag>
          <t-tag v-if="model.is_default" size="small" theme="success">{{ 'Default' }}</t-tag>
          <span
            v-if="showContextWindow"
            class="model-ctx"
            :class="{ 'model-ctx--default': isDefaultContextWindow(model.parameters?.context_window) }"
            :title="contextWindowTitle(model.parameters?.context_window)"
          >{{ formatContextWindow(model.parameters?.context_window) }}</span>
        </div>
      </t-option>
      
      <t-option
        v-if="!disabled"
        value="__add_model__"
        class="add-model-option"
      >
        <div class="model-option add">
          <t-icon name="add" class="add-icon" />
          <span class="model-name">{{ 'Go to global settings to add models' }}</span>
        </div>
      </t-option>
    </t-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { type ModelConfig } from '@/api/model'
import { useChatResourcesStore } from '@/stores/chatResources'
import { MessagePlugin } from 'tdesign-vue-next'
import { filterModelsByType } from './modelSelectorFilter'
import {
  formatContextWindow,
  isDefaultContextWindow,
  effectiveContextWindow,
  modelHasContextWindow,
} from '@/utils/contextWindow'

interface Props {
  modelType: 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM' | 'ASR'
  selectedModelId?: string
  disabled?: boolean
  placeholder?: string
  status?: 'default' | 'success' | 'warning' | 'error'
  clearable?: boolean
  allModels?: ModelConfig[]
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  placeholder: '',
  status: 'default',
  clearable: false,
})

const emit = defineEmits<{
  'update:selectedModelId': [value: string]
  'add-model': []
}>()

const models = ref<ModelConfig[]>([])
const loading = ref(false)
const chatResources = useChatResourcesStore()

const placeholderText = computed(() => {
  return props.placeholder || 'Select a model'
})

const modelDisplayName = (model: ModelConfig) => {
  const displayName = model.display_name?.trim()
  return displayName || model.name
}

const showContextWindow = computed(() => modelHasContextWindow(props.modelType))

const contextWindowTitle = (tokens?: number) => {
  if (isDefaultContextWindow(tokens)) {
    return `Unset, using default ${formatContextWindow(tokens)}`
  }
  return `${effectiveContextWindow(tokens)} tokens`
}

watch(
  () => [props.allModels, props.modelType, chatResources.allModels] as const,
  ([newModels]) => {
    const source = Array.isArray(newModels) ? newModels : chatResources.allModels
    models.value = filterModelsByType(source, props.modelType)
  },
  { immediate: true },
)

const selectedModel = computed(() => {
  if (!props.selectedModelId) return null
  return models.value.find(m => m.id === props.selectedModelId)
})

const loadModels = async () => {
  if (props.allModels) {
    return
  }

  loading.value = true
  try {
    await chatResources.ensureModels()
    models.value = filterModelsByType(chatResources.allModels, props.modelType)
  } catch (error) {
    console.error('Failed to load model list', error)
    MessagePlugin.error('Failed to load model list')
    models.value = []
  } finally {
    loading.value = false
  }
}

const handleModelChange = (value?: string) => {
  if (value === '__add_model__') {
    emit('add-model')
    return
  }
  emit('update:selectedModelId', value || '')
}

defineExpose({
  refresh: loadModels
})

onMounted(() => {
  if (!props.allModels) {
    loadModels()
  }
})
</script>

<style lang="less" scoped>
.model-selector {
  width: 100%;
}

.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .model-icon {
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
  }
  
  .add-icon {
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
  }
  
  .model-name {
    flex: 0 1 auto;
    min-width: 0;
    font-size: var(--app-text-md);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .model-raw-name {
    flex: 1;
    min-width: 0;
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .model-ctx {
    margin-left: auto;
  }
  
  &.add {
    .model-name {
      color: var(--td-brand-color);
      font-weight: 500;
    }
  }
}

.model-ctx {
  flex-shrink: 0;
  font-size: var(--app-text-xs);
  font-variant-numeric: tabular-nums;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  padding: 0 6px;
  border-radius: var(--app-radius-xs);
  line-height: 18px;

  &--default {
    color: var(--td-text-color-placeholder);
  }
}

.selected-model {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;

  &__name {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
