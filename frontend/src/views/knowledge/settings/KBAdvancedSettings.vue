<template>
  <div class="kb-advanced-settings" :class="{ 'kb-advanced-settings--embedded': embedded }">
    <div v-if="!embedded" class="section-header">
      <h2>{{ 'Advanced Settings' }}</h2>
      <p class="section-description">{{ 'Configure question generation and other advanced features' }}</p>
    </div>

    <div class="settings-group">
      <!-- Question Generation feature (only useful for RAG indexing) -->
      <template v-if="ragEnabled !== false">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'AI Question Generation' }}</label>
          <p class="desc">{{ 'Generate related questions for each chunk using LLM during document parsing to improve retrieval recall. Enabling this will increase document parsing time.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localQuestionGeneration.enabled"
            @change="handleQuestionGenerationToggle"
            size="medium"
          />
        </div>
      </div>

      <!-- Question Generation configuration -->
      <div v-if="localQuestionGeneration.enabled" class="subsection">
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Question Count' }}</label>
            <p class="desc">{{ 'Number of questions to generate per document chunk (1-10)' }}</p>
          </div>
          <div class="setting-control">
            <t-input-number
              v-model="localQuestionGeneration.questionCount"
              :min="1"
              :max="10"
              :step="1"
              theme="normal"
              @change="handleQuestionGenerationChange"
              style="width: 120px;"
            />
          </div>
        </div>
        <div class="setting-row setting-row-vertical">
          <div class="setting-info">
            <label>{{ 'Question Generation Instructions' }}</label>
            <p class="desc">{{ 'Specify audience, scenario, and wording while the system retains the stable output format' }}</p>
          </div>
          <div class="setting-control">
            <t-textarea
              v-model="localQuestionGeneration.customInstructions"
              :placeholder="'For example: generate natural customer-support questions and avoid exam-style wording…'"
              :maxlength="4000"
              :autosize="{ minRows: 3, maxRows: 8 }"
              @change="handleQuestionGenerationChange"
            />
          </div>
        </div>
      </div>
      </template>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Automatic Tagging' }}</label>
          <p class="desc">{{ 'After parsing, select suitable tags from the existing knowledge-base tags. Tags are never created or deleted, but one extra model call is required.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch v-model="localAutoTag.enabled" size="medium" @change="emitAutoTag" />
        </div>
      </div>

      <div v-if="localAutoTag.enabled" class="subsection">
        <div class="setting-row setting-row-vertical">
          <div class="setting-info">
            <label>{{ 'Classification Model' }}</label>
            <p class="desc">{{ 'Leave empty to reuse the knowledge-base summary model.' }}</p>
          </div>
          <div class="setting-control">
            <ModelSelector
              model-type="KnowledgeQA"
              :selected-model-id="localAutoTag.modelId"
              :all-models="allModels"
              clearable
              :placeholder="'Select a classification model'"
              @update:selected-model-id="(value: string) => { localAutoTag.modelId = value; emitAutoTag() }"
            />
          </div>
        </div>
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Maximum tags per document' }}</label>
            <p class="desc">{{ 'Automatically associate 1-10 existing tags per document.' }}</p>
          </div>
          <div class="setting-control">
            <t-input-number
              v-model="localAutoTag.maxTags"
              :min="1"
              :max="10"
              :step="1"
              theme="normal"
              style="width: 120px;"
              @change="emitAutoTag"
            />
          </div>
        </div>
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ 'Skip documents that already have tags' }}</label>
            <p class="desc">{{ 'When enabled, documents tagged manually at upload time are left untouched, so a deliberate classification is not diluted.' }}</p>
          </div>
          <div class="setting-control">
            <t-switch v-model="localAutoTag.skipIfTagged" size="medium" @change="emitAutoTag" />
          </div>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Auto-generate knowledge base description' }}</label>
          <p class="desc">{{ 'After documents are added, removed or re-summarized, derive the description from the aggregated document profiles. The aggregation itself makes no model call; one small call runs only when the aggregate changed.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch v-model="localProfile.enabled" size="medium" @change="emitProfile" />
        </div>
      </div>

      <div v-if="localProfile.enabled" class="subsection">
        <div class="setting-row setting-row-vertical">
          <div class="setting-info">
            <label>{{ 'Generation model' }}</label>
            <p class="desc">{{ 'Leave empty to reuse the knowledge-base summary model.' }}</p>
          </div>
          <div class="setting-control">
            <ModelSelector
              model-type="KnowledgeQA"
              :selected-model-id="localProfile.modelId"
              :all-models="allModels"
              clearable
              :placeholder="'Select a generation model'"
              @update:selected-model-id="(value: string) => { localProfile.modelId = value; emitProfile() }"
            />
          </div>
        </div>
        <div class="setting-row setting-row-vertical">
          <div class="setting-info">
            <label>{{ 'Description instructions' }}</label>
            <p class="desc">{{ 'Name the audience, terms to keep or the tone; the output format stays fixed.' }}</p>
          </div>
          <div class="setting-control">
            <t-textarea
              v-model="localProfile.customInstructions"
              :placeholder="'e.g. Written for support agents; describe product lines in plain words and keep model numbers…'"
              :maxlength="2000"
              :autosize="{ minRows: 2, maxRows: 6 }"
              @change="emitProfile"
            />
          </div>
        </div>
      </div>

      <div class="setting-row setting-row-vertical">
        <div class="setting-info">
          <label>{{ 'Table Metadata Instructions' }}</label>
          <p class="desc">{{ 'Add business context and field semantics to CSV/Excel summaries for better retrieval' }}</p>
        </div>
        <div class="setting-control">
          <t-textarea
            :model-value="tableMetadataInstructions"
            :placeholder="'For example: this is a sales-order table, amounts are in CNY, and status uses internal codes…'"
            :maxlength="4000"
            :autosize="{ minRows: 3, maxRows: 8 }"
            @change="(value: string) => emit('update:tableMetadataInstructions', value)"
          />
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ModelSelector from '@/components/ModelSelector.vue'

interface QuestionGenerationConfig {
  enabled: boolean
  questionCount: number
  customInstructions?: string
}

interface AutoTagConfig {
  enabled: boolean
  modelId: string
  maxTags: number
  skipIfTagged: boolean
}

interface ProfileConfig {
  enabled: boolean
  modelId: string
  customInstructions: string
}

interface Props {
  questionGeneration?: QuestionGenerationConfig
  autoTag?: AutoTagConfig
  profileConfig?: ProfileConfig
  ragEnabled?: boolean
  allModels?: any[]
  embedded?: boolean
  tableMetadataInstructions?: string
}

const props = withDefaults(defineProps<Props>(), {
  embedded: false,
})

const emit = defineEmits<{
  'update:questionGeneration': [value: QuestionGenerationConfig]
  'update:autoTag': [value: AutoTagConfig]
  'update:profileConfig': [value: ProfileConfig]
  'update:tableMetadataInstructions': [value: string]
}>()

const localProfile = ref<ProfileConfig>(
  props.profileConfig
    ? { ...props.profileConfig }
    : { enabled: false, modelId: '', customInstructions: '' }
)

watch(() => props.profileConfig, (newVal) => {
  if (newVal) localProfile.value = { ...newVal }
}, { deep: true })

const emitProfile = () => {
  emit('update:profileConfig', { ...localProfile.value })
}

const localQuestionGeneration = ref<QuestionGenerationConfig>(
  props.questionGeneration
    ? { ...props.questionGeneration, customInstructions: props.questionGeneration.customInstructions || '' }
    : { enabled: false, questionCount: 3, customInstructions: '' }
)

const localAutoTag = ref<AutoTagConfig>(
  props.autoTag ? { ...props.autoTag } : { enabled: false, modelId: '', maxTags: 3, skipIfTagged: true }
)

watch(() => props.questionGeneration, (newVal) => {
  if (newVal) {
    localQuestionGeneration.value = { customInstructions: '', ...newVal }
  }
}, { deep: true })

watch(() => props.autoTag, (newVal) => {
  if (newVal) localAutoTag.value = { ...newVal }
}, { deep: true })

const emitAutoTag = () => {
  if (!localAutoTag.value.maxTags) localAutoTag.value.maxTags = 3
  localAutoTag.value.maxTags = Math.min(10, Math.max(1, Math.trunc(localAutoTag.value.maxTags)))
  emit('update:autoTag', { ...localAutoTag.value })
}

const handleQuestionGenerationToggle = () => {
  if (!localQuestionGeneration.value.enabled) {
    localQuestionGeneration.value.questionCount = 3
  }
  emit('update:questionGeneration', localQuestionGeneration.value)
}

const handleQuestionGenerationChange = () => {
  emit('update:questionGeneration', localQuestionGeneration.value)
}
</script>

<style lang="less" scoped>
.kb-advanced-settings {
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
  }

  .desc {
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }

  .hint {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
    margin: 6px 0 0 0;
    line-height: 1.5;
  }
}

.setting-control {
  flex: 0 0 55%;
  max-width: 55%;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.setting-row-vertical {
  flex-direction: column;
  gap: 12px;

  .setting-info,
  .setting-control {
    flex: none;
    width: 100%;
    max-width: none;
    padding-right: 0;
  }

  .setting-control {
    display: block;
  }
}

.subsection {
  padding: 16px 20px;
  margin: 12px 0 0 0;
  background: var(--td-bg-color-container);
  border-radius: var(--app-radius-md);
  border-left: 3px solid var(--td-brand-color);
  position: relative;
}

.required {
  color: var(--td-error-color);
  margin-left: 2px;
  font-weight: 500;
}

.kb-advanced-settings--embedded {
  .setting-row {
    padding: 12px 0;
  }

  .setting-row:has(.t-switch) {
    flex-direction: row;
    align-items: center;
    gap: 16px;

    .setting-info {
      flex: 1;
      min-width: 0;
      max-width: none;
      padding-right: 0;
    }

    .setting-control {
      flex: none;
      align-self: center;
    }
  }

  .subsection .setting-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;

    .setting-info {
      flex: none;
      max-width: none;
      padding-right: 0;
    }

    .setting-control {
      align-self: flex-start;
    }
  }

  .subsection {
    margin-top: 0;
    padding: 0;
    border: none;
    background: none;
  }
}

</style>
