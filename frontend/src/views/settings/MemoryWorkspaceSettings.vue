<template>
  <div class="memory-workspace-settings">
    <div class="section-header">
      <h2>{{ 'Long-term memory' }}</h2>
      <p class="section-description">{{ 'Let the assistant remember what members tell it — who they are, how they like to work, stable facts and what they are working on — across conversations.' }}</p>
    </div>

    <!-- The switch defaults to off because memory retains what users say
         across sessions. That makes the feature easy to miss, so the intro
         states plainly what turning it on does. -->
    <div class="intro">
      <t-icon name="info-circle" class="intro-icon" />
      <div>
        <p class="intro-title">{{ 'Off by default, you have to turn it on' }}</p>
        <p class="intro-desc">{{ 'Long-term memory retains what members say in conversations, so it does not arrive enabled. Once on, each member has their own isolated memory space and can review, edit, delete or switch it off entirely under \u0022My memory\u0022. Active profile and preference memories are included in every later turn; facts and ongoing tasks are recalled only when the question is related.' }}</p>
      </div>
    </div>

    <div class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Enable long-term memory in this workspace' }}</label>
          <p class="desc">{{ 'When off, no conversation in this workspace reads or writes memory.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch v-model="config.enabled" :disabled="!canEdit" @change="debouncedSave" />
        </div>
      </div>

      <div v-if="config.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'How memories are written' }}</label>
          <p class="desc">{{ 'Controls what gets remembered.' }}</p>
          <p class="desc hint">
            {{
              config.write_mode === 'auto'
                ? 'Additionally makes one background model call after a conversation to distill what is worth keeping from what the member said.'
                : 'Only records what a member explicitly asks to remember, plus entries added by hand. No extra model call.'
            }}
          </p>
        </div>
        <div class="setting-control">
          <t-radio-group v-model="config.write_mode" :disabled="!canEdit" @change="debouncedSave">
            <t-radio-button value="explicit_only">
              {{ 'Explicit only' }}
            </t-radio-button>
            <t-radio-button value="auto">
              {{ 'Distill automatically' }}
            </t-radio-button>
          </t-radio-group>
        </div>
      </div>

      <div v-if="config.enabled && config.write_mode === 'auto'" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Distillation model' }}</label>
          <p class="desc">{{ 'Leave blank to use the model the conversation itself used.' }}</p>
        </div>
        <div class="setting-control" style="min-width: 280px">
          <ModelSelector
            model-type="KnowledgeQA"
            :selected-model-id="config.extract_model_id"
            :disabled="!canEdit"
            @update:selected-model-id="handleModelChange"
            @add-model="handleAddModel('chat')"
          />
        </div>
      </div>

      <div v-if="config.enabled && config.write_mode === 'auto'" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Distillation delay' }}</label>
          <p class="desc">{{ 'How long a finished turn waits before distillation runs. Waiting lets one model call cover the several messages a user usually sends in a row.' }}</p>
        </div>
        <div class="setting-control">
          <t-input-number
            v-model="config.extract_delay_seconds"
            :min="5"
            :max="3600"
            :step="15"
            suffix="s"
            :disabled="!canEdit"
            @change="debouncedSave"
          />
        </div>
      </div>

      <div v-if="config.enabled && config.write_mode === 'auto'" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Minimum interval between runs' }}</label>
          <p class="desc">{{ 'The floor between two distillation runs for one person, used to bound cost. Messages produced inside the interval are not dropped — they are carried over to the next run.' }}</p>
        </div>
        <div class="setting-control">
          <t-input-number
            v-model="config.extract_min_interval_seconds"
            :min="0"
            :max="86400"
            :step="60"
            suffix="s"
            :disabled="!canEdit"
            @change="debouncedSave"
          />
        </div>
      </div>

      <div v-if="config.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Match memory by meaning' }}</label>
          <p class="desc">{{ 'Adds semantic matching on top of wording, so a memory still surfaces after the user re-phrases the subject — and most memories get re-phrased eventually. Costs one embedding call per turn, and falls back to wording-only matching on timeout.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch v-model="config.vector_recall" :disabled="!canEdit" @change="debouncedSave" />
        </div>
      </div>

      <div v-if="config.enabled && config.vector_recall" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Memory embedding model' }}</label>
          <p class="desc">{{ 'Semantic recall uses this one model, independent of whichever embedding models knowledge bases bind. Leave blank for wording-only matching. After a change, new memories use the new model immediately; existing ones stay wording-only until they are re-embedded.' }}</p>
        </div>
        <div class="setting-control" style="min-width: 280px">
          <ModelSelector
            model-type="Embedding"
            :selected-model-id="config.embedding_model_id"
            :disabled="!canEdit"
            :clearable="true"
            @update:selected-model-id="handleEmbeddingModelChange"
            @add-model="handleAddModel('embedding')"
          />
        </div>
      </div>

      <div v-if="config.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Let memory shape retrieval' }}</label>
          <p class="desc">{{ 'Memory takes part in query rewriting and document ranking rather than only being appended to the answer prompt. This is where memory earns its keep in a knowledge-base product.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="config.retrieval_conditioning"
            :disabled="!canEdit"
            @change="debouncedSave"
          />
        </div>
      </div>

      <div v-if="config.enabled && config.write_mode === 'auto'" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Questions before a topic becomes an interest' }}</label>
          <p class="desc">{{ 'A subject is recorded only after it has come up this many times. Setting it to 1 records every passing question, which is usually too noisy.' }}</p>
        </div>
        <div class="setting-control">
          <t-input-number
            v-model="config.interest_threshold"
            :min="1"
            :max="20"
            :step="1"
            :disabled="!canEdit"
            @change="debouncedSave"
          />
        </div>
      </div>

      <div v-if="config.enabled && config.write_mode === 'auto'" class="setting-row instructions-row">
        <div class="setting-info">
          <label>{{ 'Custom distillation rules' }}</label>
          <p class="desc">{{ 'Workspace rules appended to the distillation prompt, for policies the product cannot guess — for example \u0022never record customer names\u0022.' }}</p>
        </div>
        <div class="setting-control instructions-control">
          <t-textarea
            v-model="config.extract_instructions"
            :autosize="{ minRows: 3, maxRows: 8 }"
            :maxlength="1000"
            :disabled="!canEdit"
            :placeholder="'One rule per line, for example: never record customer names'"
            @blur="debouncedSave"
          />
        </div>
      </div>

      <div v-if="config.enabled" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Memories per member' }}</label>
          <p class="desc">{{ 'Beyond this, the lowest ranked memories are archived by importance and recency. Archived memories stay visible under \u0022My memory\u0022.' }}</p>
        </div>
        <div class="setting-control">
          <t-input-number
            v-model="config.max_items"
            :min="10"
            :max="2000"
            :step="10"
            :disabled="!canEdit"
            @change="debouncedSave"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import ModelSelector from '@/components/ModelSelector.vue'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { getTenantMemoryConfig, updateTenantMemoryConfig, type MemoryConfig } from '@/api/memory'

const authStore = useAuthStore()
const uiStore = useUIStore()

const config = reactive<MemoryConfig>({
  enabled: false,
  write_mode: 'explicit_only',
  extract_model_id: '',
  max_items: 200,
  extract_delay_seconds: 90,
  extract_min_interval_seconds: 300,
  extract_instructions: '',
  interest_threshold: 3,
  retrieval_conditioning: true,
  embedding_model_id: '',
  vector_recall: true,
})
const isInitializing = ref(true)

const canEdit = computed(() => authStore.hasRole('admin'))

const loadConfig = async () => {
  try {
    const response = await getTenantMemoryConfig()
    if (response.data) {
      config.enabled = response.data.enabled ?? false
      config.write_mode = response.data.write_mode === 'auto' ? 'auto' : 'explicit_only'
      config.extract_model_id = response.data.extract_model_id || ''
      config.max_items = response.data.max_items || 200
      config.extract_delay_seconds = response.data.extract_delay_seconds || 90
      config.extract_min_interval_seconds = response.data.extract_min_interval_seconds || 300
      config.extract_instructions = response.data.extract_instructions || ''
      config.interest_threshold = response.data.interest_threshold || 3
      config.retrieval_conditioning = response.data.retrieval_conditioning !== false
      config.embedding_model_id = response.data.embedding_model_id || ''
      config.vector_recall = response.data.vector_recall !== false
    }
  } catch (error: any) {
    console.error('Failed to load memory config:', error)
  } finally {
    // Give the switches a tick to settle so binding the loaded values does not
    // immediately fire a save.
    setTimeout(() => {
      isInitializing.value = false
    }, 100)
  }
}

const saveConfig = async () => {
  try {
    await updateTenantMemoryConfig({ ...config })
    MessagePlugin.success('Long-term memory settings saved')
  } catch (error: any) {
    MessagePlugin.error(
      `Failed to save: ${error?.message || ''}`,
    )
  }
}

let saveTimer: number | null = null
const debouncedSave = () => {
  if (isInitializing.value || !canEdit.value) return
  if (saveTimer) clearTimeout(saveTimer)
  saveTimer = window.setTimeout(() => {
    saveConfig().catch(() => {})
  }, 500)
}

const handleModelChange = (modelId: string) => {
  config.extract_model_id = modelId
  debouncedSave()
}

const handleEmbeddingModelChange = (modelId: string) => {
  config.embedding_model_id = modelId || ''
  debouncedSave()
}

const handleAddModel = (subSection: 'chat' | 'embedding') => {
  uiStore.openSettings('models', subSection)
  window.dispatchEvent(
    new CustomEvent('settings-nav', { detail: { section: 'models', subsection: subSection } }),
  )
}

onMounted(loadConfig)
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.memory-workspace-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.intro {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 14px 16px;
  margin-bottom: 8px;
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);
}

.intro-icon {
  color: var(--td-brand-color);
  margin-top: 2px;
  flex-shrink: 0;
}

.intro-title {
  margin: 0 0 4px 0;
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.intro-desc {
  margin: 0;
  font-size: var(--app-text-md);
  line-height: 1.6;
  color: var(--td-text-color-secondary);
}

.settings-group {
  display: flex;
  flex-direction: column;
}

.setting-row {
  .setting-row();
}

.setting-info {
  .setting-info();

  .hint {
    margin-top: 4px !important;
    color: var(--td-text-color-placeholder);
  }
}

.setting-control {
  .setting-control();
}

// The custom prompt needs room to read, so this row stacks instead of putting a
// paragraph of rules into a narrow right-hand column.
.instructions-row {
  flex-direction: column;
  align-items: stretch;

  .setting-info {
    max-width: 100%;
    padding-right: 0;
    margin-bottom: 10px;
  }
}

.instructions-control {
  width: 100%;
  justify-content: stretch;
}
</style>
