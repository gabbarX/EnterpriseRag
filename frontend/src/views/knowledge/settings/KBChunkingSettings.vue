<template>
  <div class="kb-chunking-settings" :class="{ 'kb-chunking-settings--embedded': embedded }">
    <div v-if="!embedded" class="section-header">
      <div class="section-header-text">
        <h2>{{ 'Chunking Settings' }}</h2>
        <p class="section-description">{{ 'Controls how uploaded documents are split before embedding. Defaults work for most cases — tune only when retrieval quality is off.' }}</p>
      </div>
    </div>

    <div class="settings-group">
      <!-- Strategy -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Chunking Strategy' }}</label>
          <p class="desc">{{ 'Choose how documents are split into chunks. The Automatic mode profiles each document and picks the best strategy.' }}</p>
        </div>
        <div class="setting-control strategy-control">
          <t-select
            v-model="localStrategy"
            :options="strategyOptions"
            :placeholder="'Select a chunking strategy (splits by length if left empty)'"
            :clearable="true"
            @change="handleStrategyChange"
            :style="selectStyle"
          />
          <!-- Test trigger sits right next to the strategy picker so users
               discover it exactly when they're deciding which strategy to
               use on their content. -->
          <KBChunkingDebug v-if="!embedded" :config="debugConfig" />
        </div>
      </div>

      <!-- Strategy explanation panel -->
      <div v-if="currentStrategyInfo" class="strategy-info-panel">
        <p>
          <strong>{{ currentStrategyInfo.label }}:</strong>
          {{ currentStrategyInfo.tooltip }}
        </p>
      </div>

      <!-- Chunk Size -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Chunk Size' }}</label>
          <p class="desc">{{ 'Maximum characters per chunk (100–4000). Default 512 ≈ 100–130 English tokens. Smaller for FAQs (200–400), larger for narrative documents (1000–2000).' }}</p>
        </div>
        <div class="setting-control">
          <div class="slider-container">
            <t-slider
              v-model="localChunkSize"
              :min="100"
              :max="4000"
              :step="50"
              :marks="embedded ? undefined : chunkSizeMarks"
              @change="handleChunkSizeChange"
              :style="sliderStyle"
            />
            <span class="value-display">{{ localChunkSize }} {{ 'characters' }}</span>
          </div>
        </div>
      </div>

      <!-- Chunk Overlap -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Chunk Overlap' }}</label>
          <p class="desc">{{ 'Characters shared between adjacent chunks (0–500). Default 80 ≈ 15% of size — sweet spot per current research. Use 0 for FAQs/structured data, 150–200 for long-form narratives.' }}</p>
          <p v-if="overlapTooHigh" class="warn">{{ 'Overlap is large compared to chunk size — chunks will share most of their content.' }}</p>
        </div>
        <div class="setting-control">
          <div class="slider-container">
            <t-slider
              v-model="localChunkOverlap"
              :min="0"
              :max="500"
              :step="20"
              :marks="embedded ? undefined : chunkOverlapMarks"
              @change="handleChunkOverlapChange"
              :style="sliderStyle"
            />
            <span class="value-display">{{ localChunkOverlap }} {{ 'characters' }}</span>
          </div>
        </div>
      </div>

      <!-- Separators -->
      <div class="setting-row setting-row--separators">
        <div class="setting-info">
          <label>{{ 'Separators' }}</label>
          <p class="desc">{{ 'Characters or strings the splitter prefers when cutting. Higher-priority separators are tried first; the default order favors paragraph → sentence → punctuation breaks.' }}</p>
        </div>
        <div class="setting-control">
          <t-select
            v-model="localSeparators"
            :options="separatorOptions"
            multiple
            creatable
            filterable
            :placeholder="'Select or customize separators'"
            @change="handleSeparatorsChange"
            :style="selectStyle"
          />
        </div>
      </div>

      <!-- Parent-Child Chunking -->
      <div class="setting-row setting-row--toggle">
        <div class="setting-info">
          <label>{{ 'Parent-Child Chunking' }}</label>
          <p class="desc">{{ 'Two-level chunking: small child chunks are vector-matched (precise hits) but the larger parent chunk is returned to the LLM (richer context). Recommended for long documents (>10 pages); skip for short FAQs to save storage.' }}</p>
        </div>
        <div class="setting-control">
          <t-switch
            v-model="localEnableParentChild"
            @change="handleParentChildChange"
          />
        </div>
      </div>

      <!-- Parent Chunk Size -->
      <div v-if="localEnableParentChild" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Parent Chunk Size' }}</label>
          <p class="desc">{{ 'Size of the context chunk returned to the LLM (512–8192). Default 4096 ≈ 1000 English tokens, fits comfortably in any modern LLM context window.' }}</p>
        </div>
        <div class="setting-control">
          <div class="slider-container">
            <t-slider
              v-model="localParentChunkSize"
              :min="512"
              :max="8192"
              :step="64"
              :marks="embedded ? undefined : parentChunkSizeMarks"
              @change="handleParentChunkSizeChange"
              :style="sliderStyle"
            />
            <span class="value-display">{{ localParentChunkSize }} {{ 'characters' }}</span>
          </div>
        </div>
      </div>

      <!-- Child Chunk Size -->
      <div v-if="localEnableParentChild" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Child Chunk Size' }}</label>
          <p class="desc">{{ 'Size of the embedded chunk used for vector match (64–2048). Default 384 ≈ 80 tokens — sweet spot for sentence-transformer / BGE-style embedders.' }}</p>
        </div>
        <div class="setting-control">
          <div class="slider-container">
            <t-slider
              v-model="localChildChunkSize"
              :min="64"
              :max="2048"
              :step="32"
              :marks="embedded ? undefined : childChunkSizeMarks"
              @change="handleChildChunkSizeChange"
              :style="sliderStyle"
            />
            <span class="value-display">{{ localChildChunkSize }} {{ 'characters' }}</span>
          </div>
        </div>
      </div>

      <!-- Advanced section toggle -->
      <button type="button" class="advanced-toggle" @click="advancedOpen = !advancedOpen">
        <chevron-right-icon class="toggle-arrow" :class="{ open: advancedOpen }" />
        <span>{{ 'Advanced options' }}</span>
      </button>

      <div v-if="advancedOpen" class="advanced-section">
        <!-- Token Limit -->
        <div class="setting-row" :class="{ disabled: advancedDisabled }">
          <div class="setting-info">
            <label>{{ 'Token limit per chunk' }}</label>
            <p class="desc">{{ 'Hard token cap per chunk (0–8192). 0 = off (chunk size in characters only). Activate when your embedding model has a small token limit: 200 for MiniLM (256 tok), 400 for BGE/Cohere (512 tok). Modern embedders (OpenAI, Voyage, Jina-v3) accept >2000 tokens — leave at 0.' }}</p>
          </div>
          <div class="setting-control">
            <t-input-number
              v-model="localTokenLimit"
              :min="0"
              :max="8192"
              :step="64"
              :disabled="advancedDisabled"
              @change="handleTokenLimitChange"
              style="width: 200px;"
            />
          </div>
        </div>

        <!-- Languages -->
        <div class="setting-row" :class="{ disabled: advancedDisabled }">
          <div class="setting-info">
            <label>{{ 'Language hints' }}</label>
            <p class="desc">{{ 'Restricts heuristic patterns to the chosen languages (DE/EN/ZH). Empty = auto-detect from sample. Set explicitly for homogeneous corpora to avoid false-positive matches across languages.' }}</p>
          </div>
          <div class="setting-control">
            <t-select
              v-model="localLanguages"
              :options="languageOptions"
              multiple
              :disabled="advancedDisabled"
              :placeholder="'Auto-detect'"
              @change="handleLanguagesChange"
              :style="selectStyle"
            />
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { ChevronRightIcon } from 'tdesign-icons-vue-next'
import KBChunkingDebug from './KBChunkingDebug.vue'

interface ParserEngineRule {
  file_types: string[]
  engine: string
  xlsx_first_row_as_header?: boolean
}

// Slider ranges defined in this file (min/max props on t-slider) mirror
// the validated bounds in the backend splitter:
//   ChunkSize:      100–4000  (default 512). 100 = too fragmented to be
//                   useful; 4000 = approaches the 7500-char absoluteMaxSize
//                   that the splitter hard-caps to anyway.
//   ChunkOverlap:   0–500     (default 80). Backend caps to ChunkSize/2
//                   when set higher than that.
//   ParentChunkSize: 512–8192 (default 4096 ≈ 1000 EN tokens).
//   ChildChunkSize:  64–2048  (default 384 ≈ 80 EN tokens, sweet spot for
//                   sentence-transformer / BGE embedders).
//   TokenLimit:      0–8192   (default 0 = off, char-based budget only).
//                   Set to 200 for MiniLM (256-tok limit), 400 for BGE/
//                   Cohere (512-tok), leave at 0 for OpenAI/Voyage/Jina-v3.
interface ChunkingConfig {
  chunkSize: number
  chunkOverlap: number
  separators: string[]
  parserEngineRules?: ParserEngineRule[]
  enableParentChild: boolean
  parentChunkSize: number
  childChunkSize: number
  // Adaptive chunking strategy. Empty string = legacy / not set.
  strategy?: string
  // Cap chunk size in approx tokens. 0 = char-based budget only.
  tokenLimit?: number
  // Language hints for heuristic patterns (de/en/zh).
  languages?: string[]
}

interface Props {
  config: ChunkingConfig
  embedded?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  embedded: false,
})

const selectStyle = computed(() => (props.embedded ? { width: '100%' } : { width: '280px' }))
const sliderStyle = computed(() => (props.embedded ? { width: '100%' } : { width: '200px' }))

const chunkSizeMarks = { 100: '100', 1000: '1000', 2000: '2000', 4000: '4000' }
const chunkOverlapMarks = { 0: '0', 250: '250', 500: '500' }
const parentChunkSizeMarks = { 512: '512', 2048: '2048', 4096: '4096', 8192: '8192' }
const childChunkSizeMarks = { 64: '64', 384: '384', 1024: '1024', 2048: '2048' }

const emit = defineEmits<{
  'update:config': [value: ChunkingConfig]
}>()


const localChunkSize = ref(props.config.chunkSize)
const localChunkOverlap = ref(props.config.chunkOverlap)
const localSeparators = ref([...props.config.separators])
const localEnableParentChild = ref(props.config.enableParentChild ?? false)
const localParentChunkSize = ref(props.config.parentChunkSize || 4096)
const localChildChunkSize = ref(props.config.childChunkSize || 384)
const localStrategy = ref(props.config.strategy ?? '')
const localTokenLimit = ref(props.config.tokenLimit ?? 0)
const localLanguages = ref<string[]>([...(props.config.languages ?? [])])
const advancedOpen = ref(false)

const strategyOptions = computed(() => [
  {
    label: 'Automatic',
    value: 'auto',
    tooltip: 'A document profiler picks between heading-aware, structure-aware and length-based splitting per upload.'
  },
  {
    label: 'Heading-aware',
    value: 'heading',
    tooltip: 'Splits at Markdown heading boundaries (#, ##, ###); each chunk carries its heading path. Best for well-structured Markdown.'
  },
  {
    label: 'Structure-aware',
    value: 'heuristic',
    tooltip: 'Splits on detected structural cues: page-breaks, numbered sections, multilingual chapter markers (DE/EN/ZH), all-caps titles. Ideal for PDFs without Markdown headings.'
  },
  {
    label: 'Length-based',
    value: 'legacy',
    tooltip: 'Ignores structure; splits recursively by character count and separators — the original behavior. Use when the structure-aware strategies misbehave on your content.'
  }
])

const currentStrategyInfo = computed(() => {
  if (!localStrategy.value) {
    return null
  }
  return strategyOptions.value.find(o => o.value === localStrategy.value) ?? null
})

const advancedDisabled = computed(() => localStrategy.value === 'legacy')

const overlapTooHigh = computed(
  () => localChunkOverlap.value > 0 && localChunkOverlap.value >= localChunkSize.value / 2
)

// Live config snapshot for the debug panel — uses current local form values
// so the panel reflects edits immediately without waiting for save.
const debugConfig = computed(() => ({
  chunkSize: localChunkSize.value,
  chunkOverlap: localChunkOverlap.value,
  separators: localSeparators.value,
  enableParentChild: localEnableParentChild.value,
  parentChunkSize: localParentChunkSize.value,
  childChunkSize: localChildChunkSize.value,
  strategy: localStrategy.value,
  tokenLimit: localTokenLimit.value,
  languages: localLanguages.value
}))

const languageOptions = computed(() => [
  { label: 'German', value: 'de' },
  { label: 'English', value: 'en' },
  { label: 'Chinese', value: 'zh' }
])

const separatorOptions = computed(() => [
  { label: 'Double newline ()', value: '\n\n' },
  { label: 'Single newline ()', value: '\n' },
  { label: 'Danda (।)', value: '।' },
  { label: 'Full stop + space (. )', value: '. ' },
  { label: 'Exclamation mark + space (! )', value: '! ' },
  { label: 'Question mark + space (? )', value: '? ' },
  { label: 'Semicolon (;)', value: ';' },
  { label: 'Space ( )', value: ' ' }
])

watch(() => props.config, (newConfig) => {
  localChunkSize.value = newConfig.chunkSize
  localChunkOverlap.value = newConfig.chunkOverlap
  localSeparators.value = [...newConfig.separators]
  localEnableParentChild.value = newConfig.enableParentChild ?? false
  localParentChunkSize.value = newConfig.parentChunkSize || 4096
  localChildChunkSize.value = newConfig.childChunkSize || 384
  localStrategy.value = newConfig.strategy ?? ''
  localTokenLimit.value = newConfig.tokenLimit ?? 0
  localLanguages.value = [...(newConfig.languages ?? [])]
}, { deep: true })

const handleChunkSizeChange = () => { emitUpdate() }
const handleChunkOverlapChange = () => { emitUpdate() }
const handleSeparatorsChange = () => { emitUpdate() }
const handleParentChildChange = () => { emitUpdate() }
const handleParentChunkSizeChange = () => { emitUpdate() }
const handleChildChunkSizeChange = () => { emitUpdate() }
const handleStrategyChange = () => { emitUpdate() }
const handleTokenLimitChange = () => { emitUpdate() }
const handleLanguagesChange = () => { emitUpdate() }

const emitUpdate = () => {
  // Spread arrays so the parent gets its own copy. Mutating the emitted
  // arrays from outside must not leak back into our reactive state and
  // cause two-way ref drift between the form and the editor model.
  emit('update:config', {
    chunkSize: localChunkSize.value,
    chunkOverlap: localChunkOverlap.value,
    separators: [...localSeparators.value],
    parserEngineRules: props.config.parserEngineRules,
    enableParentChild: localEnableParentChild.value,
    parentChunkSize: localParentChunkSize.value,
    childChunkSize: localChildChunkSize.value,
    strategy: localStrategy.value,
    tokenLimit: localTokenLimit.value,
    languages: [...localLanguages.value]
  })
}
</script>

<style lang="less" scoped>
.kb-chunking-settings {
  width: 100%;
}

.section-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  // Stick to the top of the scrollable section so the section title remains
  // visible while the user scrolls through a long form. Negative top and
  // matching negative margins compensate for content-wrapper's padding
  // (24px 32px) so the sticky band visually spans the full width when stuck.
  position: sticky;
  top: -24px;
  z-index: 5;
  background: var(--td-bg-color-container);
  padding: 24px 32px 12px 32px;
  margin: -24px -32px 16px -32px;
  border-bottom: 1px solid var(--td-component-stroke);

  .section-header-text {
    flex: 1;
    min-width: 0;
  }

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

  &.disabled {
    opacity: 0.5;
  }
}

.strategy-info-panel {
  margin: 0 0 16px 0;
  padding: 10px 14px;
  background: var(--td-bg-color-container-hover);
  border-left: 3px solid var(--td-brand-color);
  border-radius: 0 4px 4px 0;
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  line-height: 1.5;

  p {
    margin: 0;
    word-break: break-word;
  }

  strong {
    color: var(--td-text-color-primary);
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

  .warn {
    font-size: var(--app-text-sm);
    color: var(--td-warning-color);
    margin: 4px 0 0 0;
    line-height: 1.4;
  }
}

.setting-control {
  flex: 0 0 55%;
  max-width: 55%;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

// Strategy row stacks the picker above the test trigger so the action has
// room to breathe without competing with the select for horizontal space.
// Both children stay right-aligned under the section's right column.
.strategy-control {
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.slider-container {
  display: flex;
  align-items: center;
  gap: 16px;
  width: 100%;
  justify-content: flex-end;
}

.value-display {
  font-size: var(--app-text-base);
  color: var(--td-text-color-primary);
  font-weight: 500;
  min-width: 80px;
  text-align: right;
}

.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 16px 0 8px 0;
  margin: 0;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  user-select: none;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
    border-radius: var(--app-radius-xs);
  }
}

.toggle-arrow {
  font-size: var(--app-text-xl);
  transition: transform var(--app-motion-fast) ease;

  &.open {
    transform: rotate(90deg);
  }
}

.advanced-section {
  // Visually grouped via the toggle above; avoid a left rule that hugs the
  // panel edge and looks detached from the rest of the form.
  margin-top: 4px;
}

.kb-chunking-settings--embedded {
  .setting-row {
    flex-direction: column;
    align-items: stretch;
    gap: 10px;
    padding: 14px 0;
  }

  .setting-row--toggle {
    flex-direction: row;
    align-items: center;
    gap: 16px;

    .setting-info {
      flex: 1;
      min-width: 0;
    }

    .setting-control {
      flex: none;
      width: auto;
      justify-content: flex-end;
    }
  }

  .setting-row--separators {
    padding-bottom: 18px;
  }

  .setting-info {
    flex: none;
    max-width: none;
    padding-right: 0;

    label {
      font-size: var(--app-text-base);
    }

    .desc {
      font-size: var(--app-text-sm);
    }
  }

  .setting-control {
    flex: none;
    max-width: none;
    justify-content: flex-start;
    align-items: stretch;
    width: 100%;
  }

  .strategy-control {
    flex-direction: column;
    align-items: stretch;
    width: 100%;
  }

  .strategy-info-panel {
    margin: -4px 0 10px;
    padding: 8px 12px;
    font-size: var(--app-text-sm);
  }

  .slider-container {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    width: 100%;
    justify-content: flex-start;
  }

  .value-display {
    order: -1;
    text-align: left;
    min-width: 0;
  }

  :deep(.t-slider) {
    padding-bottom: 0;
  }

  :deep(.t-select__wrap) {
    max-width: 100%;
  }

  :deep(.t-tag) {
    max-width: 100%;
  }

  .advanced-toggle {
    padding-top: 10px;
  }

  .advanced-section .setting-row {
    padding: 14px 0;
  }
}
</style>
