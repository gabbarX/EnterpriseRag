<template>
  <div class="kb-model-config">
    <div class="section-header">
      <h2>{{ 'Model Configuration' }}</h2>
      <p class="section-description">{{ 'Select appropriate AI models for the knowledge base' }}</p>
    </div>

    <div class="settings-group">
      <div class="setting-row" data-guide="kb-create-llm">
        <div class="setting-info">
          <label>{{ 'LLM Model' }} <span class="required">*</span></label>
          <p class="desc">{{ 'Large language model used for summarization and abstract generation (optional)' }}</p>
        </div>
        <div class="setting-control">
          <ModelSelector
            ref="llmSelectorRef"
            model-type="KnowledgeQA"
            :selected-model-id="config.llmModelId"
            :all-models="allModels"
            @update:selected-model-id="handleLLMChange"
            @add-model="handleAddModel('chat')"
            :placeholder="'Select an LLM model (optional)'"
          />
        </div>
      </div>

      <div v-if="ragEnabled !== false || wikiEnabled" class="setting-row" data-guide="kb-create-embedding">
        <div class="setting-info">
          <label>
            {{ 'Embedding Model' }}
            <span v-if="ragEnabled" class="required">*</span>
            <span v-else-if="wikiEnabled" class="optional">{{ '(Optional)' }}</span>
          </label>
          <p class="desc">
            {{ (wikiEnabled && ragEnabled === false)
              ? 'Optional. When set, it powers similarity matching for Wiki directory classification so new pages reuse existing folders better; otherwise the full directory tree is used.'
              : 'Embedding model used for text vectorization' }}
          </p>
          <t-alert
            v-if="ragEnabled && hasFiles"
            theme="warning"
            :message="'Knowledge base already has files. Embedding model cannot be modified'"
            style="margin-top: 8px;"
          />
        </div>
        <div class="setting-control">
          <ModelSelector
            ref="embeddingSelectorRef"
            model-type="Embedding"
            :selected-model-id="config.embeddingModelId"
            :all-models="allModels"
            :disabled="ragEnabled && hasFiles"
            :clearable="ragEnabled === false && wikiEnabled"
            @update:selected-model-id="handleEmbeddingChange"
            @add-model="handleAddModel('embedding')"
            :placeholder="'Select an embedding model'"
          />
        </div>
      </div>

      <div v-if="wikiEnabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Wiki Synthesis Model' }}</label>
          <p class="desc">{{ 'Falls back to the summary model if not set' }}</p>
        </div>
        <div class="setting-control">
          <ModelSelector
            model-type="KnowledgeQA"
            :selected-model-id="config.wikiSynthesisModelId"
            :all-models="allModels"
            clearable
            @update:selected-model-id="handleWikiModelChange"
            @add-model="handleAddModel('knowledgeqa')"
            :placeholder="'Select the LLM model for Wiki generation'"
          />
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useUIStore } from '@/stores/ui'
import ModelSelector from '@/components/ModelSelector.vue'

interface ModelConfig {
  llmModelId?: string
  embeddingModelId?: string
  vllmModelId?: string
  wikiSynthesisModelId?: string
}

interface Props {
  config: ModelConfig
  hasFiles: boolean
  wikiEnabled?: boolean
  ragEnabled?: boolean
  allModels?: any[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:config': [value: ModelConfig]
}>()

const uiStore = useUIStore()

const llmSelectorRef = ref<InstanceType<typeof ModelSelector>>()
const embeddingSelectorRef = ref<InstanceType<typeof ModelSelector>>()

const handleLLMChange = (modelId: string) => {
  emit('update:config', {
    ...props.config,
    llmModelId: modelId
  })
}

const handleEmbeddingChange = (modelId: string) => {
  emit('update:config', {
    ...props.config,
    embeddingModelId: modelId
  })
}

const handleWikiModelChange = (modelId: string) => {
  emit('update:config', {
    ...props.config,
    wikiSynthesisModelId: modelId
  })
}

const handleAddModel = (subSection: string) => {
  uiStore.openSettings('models', subSection)
}
</script>

<style lang="less" scoped>
.kb-model-config {
  width: 100%;
}

.section-header {
  margin-bottom: 20px;

  h2 {
    font-size: var(--app-text-3xl);
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 6px 0;
  }

  .section-description {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child {
    border-bottom: none;
  }
}

.setting-info {
  flex: 0 0 40%;
  max-width: 40%;
  padding-right: 24px;

  label {
    font-size: var(--app-text-lg);
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;

    .required {
      color: var(--td-error-color);
      margin-left: 2px;
    }

    .optional {
      color: var(--td-text-color-placeholder);
      font-size: var(--app-text-sm);
      font-weight: 400;
      margin-left: 4px;
    }
  }

  .desc {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex: 0 0 55%;
  max-width: 55%;
  display: flex;
  justify-content: flex-end;
  align-items: flex-start;
}
</style>

