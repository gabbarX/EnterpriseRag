<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="dialogVisible" class="upload-confirm-overlay">
        <div class="upload-confirm-modal" role="dialog" :aria-label="dialogTitle">
          <button class="close-btn" type="button" :aria-label="'Close Settings'" @click="handleCancel">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
              <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
            </svg>
          </button>

          <div class="upload-confirm-container">
            <aside class="files-panel">
              <div class="sidebar-header">
                <div class="sidebar-header-row">
                  <h2 class="sidebar-title">{{ dialogTitle }}</h2>
                  <div v-if="mode === 'file'" class="sidebar-header-actions">
                    <span class="files-count">{{ batchItemCount }}</span>
                    <KbUploadSourceDropdown
                      :accept-file-types="acceptFileTypes"
                      :supported-file-types="supportedFileTypes"
                      :tooltip="'Add more'"
                      placement="bottom-left"
                      @files="appendFiles"
                      @url="appendUrl"
                    />
                  </div>
                </div>

                <div v-if="mode === 'file'" class="destination-row">
                  <t-popup
                    v-model:visible="destinationPickerVisible"
                    trigger="click"
                    placement="bottom-left"
                    attach="body"
                    :z-index="3100"
                    overlay-class-name="upload-destination-popup"
                    destroy-on-close
                  >
                    <button
                      type="button"
                      class="destination-crumb"
                      :title="destinationFullLabel"
                      :aria-label="'Change upload location'"
                      :aria-expanded="destinationPickerVisible"
                    >
                      <span class="destination-crumb__label">{{ 'Upload location' }}</span>
                      <span class="destination-crumb__path">{{ destinationBreadcrumb }}</span>
                      <t-icon name="chevron-down" class="destination-crumb__caret" />
                    </button>
                    <template #content>
                      <div class="card-menu" @click.stop>
                        <FolderPickerMenu
                          :options="pickerFolderOptions"
                          :current-path="localTargetFolder"
                          allow-reselect
                          @create="onDestinationCreated"
                          @confirm="onDestinationPicked"
                        />
                      </div>
                    </template>
                  </t-popup>
                </div>
              </div>

              <div class="files-list-wrap">
                <div v-if="mode === 'manual' && manualPreview" class="manual-source-panel">
                  <p class="manual-source-title" :title="manualPreview.title">{{ manualPreview.title }}</p>
                  <p class="manual-source-meta">
                    {{ `${manualCharCount} characters` }}
                  </p>
                </div>
                <div v-else-if="mode === 'reparse' && reparsePreview" class="manual-source-panel">
                  <p class="manual-source-title" :title="reparsePreview.fileName">
                    {{ reparsePreview.fileName || 'Document to reparse' }}
                  </p>
                  <p class="manual-source-meta">{{ 'Reuses the last parse settings; adjust them here' }}</p>
                </div>
                <ul v-else-if="mode === 'file' && batchItemCount > 0" class="files-list">
                  <li v-for="(url, index) in localUrls" :key="`url-${url}-${index}`" class="file-item">
                    <span class="file-icon-wrap">
                      <t-icon name="link" class="file-icon" />
                    </span>
                    <div class="file-meta">
                      <span class="file-name" :title="url">{{ url }}</span>
                      <span class="file-size">{{ 'URL' }}</span>
                    </div>
                    <button
                      type="button"
                      class="file-remove"
                      :aria-label="'Remove'"
                      @click="removeUrl(index)"
                    >
                      <t-icon name="close" />
                    </button>
                  </li>
                  <li v-for="(file, index) in localFiles" :key="`${file.name}-${index}`" class="file-item">
                    <span class="file-icon-wrap">
                      <t-icon :name="getFileIcon(file.name)" class="file-icon" />
                    </span>
                    <div class="file-meta">
                      <span class="file-name" :title="fileDisplayTitle(file)">{{ file.name }}</span>
                      <span class="file-size">
                        <template v-if="fileRelativeDir(file)">
                          <span class="file-relative-dir" :title="fileRelativeDir(file)">{{ fileRelativeDir(file) }}</span>
                          <span class="file-meta-sep">·</span>
                        </template>
                        {{ formatFileSize(file.size) }}
                      </span>
                    </div>
                    <button
                      type="button"
                      class="file-remove"
                      :aria-label="'Remove'"
                      @click="removeFile(index)"
                    >
                      <t-icon name="close" />
                    </button>
                  </li>
                </ul>
                <div v-else-if="mode === 'file'" class="files-empty">{{ 'Add at least one file or URL' }}</div>
              </div>
            </aside>

            <aside class="settings-sidebar">
              <div class="settings-sidebar-header">
                <h2 class="settings-sidebar-title">{{ 'Parse settings' }}</h2>
              </div>
              <nav class="settings-nav" :aria-label="'Parse settings navigation'">
                <button
                  v-for="item in navItems"
                  :key="item.key"
                  type="button"
                  class="nav-item"
                  :class="{
                    active: activeSection === item.key,
                    'nav-item--issue': item.issue,
                  }"
                  @click="activeSection = item.key"
                >
                  <t-icon :name="item.icon" class="nav-icon" />
                  <span class="nav-label-wrap">
                    <span class="nav-label">{{ item.label }}</span>
                    <span
                      class="nav-status"
                      :class="{
                        'nav-status--warning': item.statusTone === 'warning',
                        'nav-status--error': item.statusTone === 'error',
                        'nav-status--muted': item.statusTone === 'muted',
                      }"
                      :title="item.statusFull"
                    >{{ item.status }}</span>
                  </span>
                  <span v-if="item.issue" class="nav-dot" aria-hidden="true" />
                </button>
              </nav>
            </aside>

            <div class="config-panel">
              <div class="content-wrapper upload-confirm-content">
                  <div v-show="activeSection === 'tags'" class="section">
                    <div class="section-content">
                      <div class="section-header">
                        <h2 class="section-title">{{ 'Document tags' }}</h2>
                        <p class="section-desc">{{ 'Select one or more tags for every document in this import batch' }}</p>
                      </div>
                      <div class="settings-group">
                        <div class="setting-row setting-row-vertical">
                          <div class="setting-info">
                            <label>{{ 'Select tags (multiple allowed)' }}</label>
                          </div>
                          <div class="setting-control setting-control-full">
                            <t-select
                              v-model="selectedTagIds"
                              :options="tagOptions"
                              :loading="tagsLoading"
                              multiple
                              filterable
                              clearable
                              :placeholder="'Select tags (multiple allowed)'"
                            />
                            <p v-if="tagsLoadFailed" class="field-hint field-hint--error">
                              {{ 'Failed to load tags. You can set them later from the document list.' }}
                            </p>
                            <p v-else-if="!tagsLoading && tagOptions.length === 0" class="field-hint">
                              {{ 'This knowledge base has no tags. You can create them in tag management after upload.' }}
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-show="activeSection === 'parser'" class="section">
                    <KBParserSettings
                      :relevant-extensions="batchFileExts"
                      :parser-engine-rules="uiState.chunkingConfig.parserEngineRules"
                      @update:parser-engine-rules="handleParserEngineRulesUpdate"
                    />
                    <div v-if="hasPdf" class="kb-settings-block">
                      <div class="settings-group">
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Force scanned PDF parsing' }}</label>
                            <p class="desc">{{ 'Useful for web-print, scanned, or image-heavy PDFs. Every page will be rendered as an image and processed via OCR/VLM. May increase processing time and model costs.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-switch v-model="uiState.pdfForceScanned" size="medium" />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-show="activeSection === 'chunking'" class="section">
                    <div class="section-content">
                      <div class="section-header">
                        <h2 class="section-title">{{ 'Chunking Settings' }}</h2>
                        <p class="section-desc">{{ 'Controls how uploaded documents are split before embedding. Defaults work for most cases — tune only when retrieval quality is off.' }}</p>
                      </div>
                      <div class="settings-group">
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Chunking Strategy' }}</label>
                            <p class="desc">{{ 'Choose how documents are split into chunks. The Automatic mode profiles each document and picks the best strategy.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-select
                              v-model="uiState.chunkingConfig.strategy"
                              :options="chunkingStrategyOptions"
                              :clearable="false"
                              :style="{ width: '280px' }"
                            />
                          </div>
                        </div>
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Chunk Size' }}</label>
                            <p class="desc">{{ 'Maximum characters per chunk (100–4000). Default 512 ≈ 100–130 English tokens. Smaller for FAQs (200–400), larger for narrative documents (1000–2000).' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-input-number
                              v-model="uiState.chunkingConfig.chunkSize"
                              :min="100"
                              :max="4000"
                              :step="50"
                              theme="normal"
                              :style="{ width: '200px' }"
                            />
                          </div>
                        </div>
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Chunk Overlap' }}</label>
                            <p class="desc">{{ 'Characters shared between adjacent chunks (0–500). Default 80 ≈ 15% of size — sweet spot per current research. Use 0 for FAQs/structured data, 150–200 for long-form narratives.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-input-number
                              v-model="uiState.chunkingConfig.chunkOverlap"
                              :min="0"
                              :max="500"
                              :step="20"
                              theme="normal"
                              :style="{ width: '200px' }"
                            />
                          </div>
                        </div>
                      </div>

                      <button
                        type="button"
                        class="more-options-toggle"
                        :aria-expanded="chunkingMoreOpen"
                        @click="chunkingMoreOpen = !chunkingMoreOpen"
                      >
                        <t-icon name="chevron-down" :class="{ 'is-open': chunkingMoreOpen }" />
                        <span>{{ 'More processing options' }}</span>
                      </button>

                      <div v-if="chunkingMoreOpen" class="settings-group settings-group--more">
                        <div class="setting-row setting-row--separators">
                          <div class="setting-info">
                            <label>{{ 'Separators' }}</label>
                            <p class="desc">{{ 'Characters or strings the splitter prefers when cutting. Higher-priority separators are tried first; the default order favors paragraph → sentence → punctuation breaks.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-select
                              v-model="uiState.chunkingConfig.separators"
                              :options="separatorOptions"
                              multiple
                              creatable
                              filterable
                              :style="{ width: '280px' }"
                            />
                          </div>
                        </div>
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Token limit per chunk' }}</label>
                          </div>
                          <div class="setting-control">
                            <t-input-number
                              v-model="uiState.chunkingConfig.tokenLimit"
                              :min="0"
                              :max="8192"
                              :step="64"
                              theme="normal"
                              :style="{ width: '200px' }"
                            />
                          </div>
                        </div>
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Language hints' }}</label>
                          </div>
                          <div class="setting-control">
                            <t-select
                              v-model="uiState.chunkingConfig.languages"
                              :options="languageOptions"
                              multiple
                              :style="{ width: '280px' }"
                            />
                          </div>
                        </div>
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Parent-Child Chunking' }}</label>
                            <p class="desc">{{ 'Two-level chunking: small child chunks are vector-matched (precise hits) but the larger parent chunk is returned to the LLM (richer context). Recommended for long documents (>10 pages); skip for short FAQs to save storage.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-switch v-model="uiState.chunkingConfig.enableParentChild" />
                          </div>
                        </div>
                        <div v-if="uiState.chunkingConfig.enableParentChild" class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Parent Chunk Size' }}</label>
                          </div>
                          <div class="setting-control">
                            <t-input-number v-model="uiState.chunkingConfig.parentChunkSize" :min="512" :max="8192" :step="64" theme="normal" :style="{ width: '200px' }" />
                          </div>
                        </div>
                        <div v-if="uiState.chunkingConfig.enableParentChild" class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Child Chunk Size' }}</label>
                          </div>
                          <div class="setting-control">
                            <t-input-number v-model="uiState.chunkingConfig.childChunkSize" :min="64" :max="2048" :step="32" theme="normal" :style="{ width: '200px' }" />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-show="activeSection === 'multimodal'" class="section" data-section="multimodal">
                    <div class="kb-settings-block">
                      <div class="section-header">
                        <h2 class="section-title">{{ 'Image Processing Configuration' }}</h2>
                        <p class="section-desc">{{ 'Configure image content understanding for parsing and retrieving non-text content like images' }}</p>
                      </div>
                      <div v-if="issueSectionKeys.has('multimodal')" class="section-notice">
                        <t-icon name="info-circle-filled" />
                        <span>{{ 'Images detected. Enable multimodal and select a model.' }}</span>
                      </div>
                      <div class="settings-group">
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Multimodal Feature' }}</label>
                            <p class="desc">{{ 'Enable understanding of multimodal content such as images' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-switch v-model="uiState.multimodalConfig.enabled" size="medium" />
                          </div>
                        </div>
                        <div v-if="uiState.multimodalConfig.enabled" class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'VLLM Vision Model' }} <span class="required">*</span></label>
                            <p class="desc">{{ 'Vision-language model required for multimodal understanding' }}</p>
                          </div>
                          <div class="setting-control">
                            <ModelSelector
                              model-type="VLLM"
                              :selected-model-id="uiState.multimodalConfig.vllmModelId"
                              :all-models="allModels"
                              :status="showMultimodalModelError ? 'error' : 'default'"
                              :placeholder="'Select a VLLM model (required)'"
                              @update:selected-model-id="handleMultimodalVLLMChange"
                              @add-model="handleAddVLLMModel"
                            />
                          </div>
                        </div>
                        <div v-if="uiState.multimodalConfig.enabled" class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Image Description Language' }}</label>
                            <p class="desc">{{ 'Leave empty to follow the document language' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-select
                              v-model="uiState.multimodalConfig.descriptionLanguage"
                              clearable
                              :placeholder="'Follow document language'"
                              :style="{ width: '280px' }"
                            >
                              <t-option value="English" :label="'English'" />
                            </t-select>
                          </div>
                        </div>
                        <div v-if="uiState.multimodalConfig.enabled" class="setting-row setting-row-vertical">
                          <div class="setting-info">
                            <label>{{ 'Image Processing Instructions' }}</label>
                            <p class="desc">{{ 'Add visual priorities while OCR and Markdown output contracts remain fixed' }}</p>
                          </div>
                          <div class="setting-control setting-control-full">
                            <t-textarea
                              v-model="uiState.multimodalConfig.customInstructions"
                              :placeholder="'For example: prioritize nameplates, model numbers, alarm codes, and table units…'"
                              :maxlength="4000"
                              :autosize="{ minRows: 3, maxRows: 8 }"
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-show="activeSection === 'asr'" class="section" data-section="asr">
                    <div class="kb-settings-block">
                      <div class="section-header">
                        <h2 class="section-title">{{ 'Audio Speech Recognition' }}</h2>
                        <p class="section-desc">{{ 'Configure ASR (speech-to-text). When enabled, you can upload audio files and transcribe them to text (e.g. mp3, wav, m4a, flac, ogg). Video upload is not supported.' }}</p>
                      </div>
                      <div v-if="issueSectionKeys.has('asr')" class="section-notice">
                        <t-icon name="info-circle-filled" />
                        <span>{{ 'Audio detected. Enable speech recognition and select a model.' }}</span>
                      </div>
                      <div class="settings-group">
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Enable audio speech recognition' }}</label>
                            <p class="desc">{{ 'When enabled, audio can be uploaded to the knowledge base; speech is transcribed to text for parsing and retrieval.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-switch v-model="uiState.asrConfig.enabled" size="medium" />
                          </div>
                        </div>
                        <div v-if="uiState.asrConfig.enabled" class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'ASR Model' }} <span class="required">*</span></label>
                            <p class="desc">{{ 'Speech-to-text model for audio (e.g. OpenAI Whisper)' }}</p>
                          </div>
                          <div class="setting-control">
                            <ModelSelector
                              model-type="ASR"
                              :selected-model-id="uiState.asrConfig.modelId"
                              :all-models="allModels"
                              :status="showAsrModelError ? 'error' : 'default'"
                              :placeholder="'Select an ASR model'"
                              @update:selected-model-id="(val: string) => { uiState.asrConfig.modelId = val }"
                              @add-model="handleAddASRModel"
                            />
                          </div>
                        </div>
                        <div v-if="uiState.asrConfig.enabled" class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Audio language hint' }}</label>
                            <p class="desc">{{ 'Optional. A primary language hint can improve some ASR models; leave empty for auto-detection.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-input
                              v-model="uiState.asrConfig.language"
                              clearable
                              :placeholder="'For example zh or en; empty means auto-detect'"
                              :style="{ width: '280px' }"
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-show="activeSection === 'summary'" class="section" data-section="summary">
                    <div class="kb-settings-block">
                      <div class="section-header">
                        <h2 class="section-title">{{ 'Document summary' }}</h2>
                        <p class="section-desc">{{ 'Choose whether to automatically summarize documents in this import.' }}</p>
                      </div>
                      <div class="settings-group">
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'Generate document summaries' }}</label>
                            <p class="desc">{{ 'Enabled by default. Disable to skip summaries while parsing, indexing, and other configured steps continue.' }}</p>
                          </div>
                          <div class="setting-control">
                            <t-switch v-model="uiState.summaryEnabled" size="medium" />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-show="activeSection === 'question'" class="section">
                    <div class="kb-settings-block">
                      <div class="section-header">
                        <h2 class="section-title">{{ 'AI Question Generation' }}</h2>
                        <p class="section-desc">{{ 'Generate related questions for each chunk using LLM during document parsing to improve retrieval recall. Enabling this will increase document parsing time.' }}</p>
                      </div>
                      <div class="settings-group">
                        <div class="setting-row">
                          <div class="setting-info">
                            <label>{{ 'AI Question Generation' }}</label>
                            <p class="desc">{{ 'Number of questions to generate per document chunk (1-10)' }}</p>
                          </div>
                          <div class="setting-control setting-control-inline">
                            <t-input-number
                              v-if="uiState.questionGenerationConfig.enabled"
                              v-model="uiState.questionGenerationConfig.questionCount"
                              :min="1"
                              :max="10"
                              :step="1"
                              theme="normal"
                              :style="{ width: '88px' }"
                            />
                            <t-switch v-model="uiState.questionGenerationConfig.enabled" size="medium" />
                          </div>
                        </div>
                        <div v-if="uiState.questionGenerationConfig.enabled" class="setting-row setting-row-vertical">
                          <div class="setting-info">
                            <label>{{ 'Question Generation Instructions' }}</label>
                            <p class="desc">{{ 'Specify audience, scenario, and wording while the system retains the stable output format' }}</p>
                          </div>
                          <div class="setting-control setting-control-full">
                            <t-textarea
                              v-model="uiState.questionGenerationConfig.customInstructions"
                              :placeholder="'For example: generate natural customer-support questions and avoid exam-style wording…'"
                              :maxlength="4000"
                              :autosize="{ minRows: 3, maxRows: 8 }"
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div v-if="isGraphSectionAvailable" v-show="activeSection === 'graph'" class="section">
                    <GraphSettings
                      :graph-extract="uiState.nodeExtractConfig"
                      :model-id="llmModelId"
                      :all-models="allModels"
                      @update:graphExtract="handleNodeExtractUpdate"
                    />
                  </div>
                </div>

              <footer class="modal-footer">
                <t-button theme="default" variant="outline" @click="handleCancel">
                  {{ 'Cancel' }}
                </t-button>
                <t-button theme="primary" :disabled="!canConfirm" @click="handleConfirm">
                  {{ confirmButtonText }}
                </t-button>
              </footer>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import ModelSelector from '@/components/ModelSelector.vue'
import KBParserSettings from '../settings/KBParserSettings.vue'
import GraphSettings from '../settings/GraphSettings.vue'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useEditorResourcesStore } from '@/stores/editorResources'
import { useUIStore } from '@/stores/ui'
import { formatFileSize, getFileIcon } from '@/utils/files'
import { getUploadFileKey } from '../utils/uploadSources'
import { listKnowledgeTags } from '@/api/knowledge-base'
import KbUploadSourceDropdown from './KbUploadSourceDropdown.vue'
import FolderPickerMenu, { type FolderOption } from './FolderPickerMenu.vue'
import { folderOptionFromPath, sortFolderOptions } from '../folderTree'
import type { KnowledgeProcessOverrides } from '@/types/knowledgeProcess'
import type {
  UploadConfirmManualSource,
  UploadConfirmMode,
  UploadConfirmReparseSource,
  UploadConfirmResult,
} from '@/stores/uploadConfirm'

const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp']
const AUDIO_EXTENSIONS = ['mp3', 'wav', 'm4a', 'flac', 'ogg']

type ConfigSectionKey = 'tags' | 'parser' | 'chunking' | 'multimodal' | 'asr' | 'summary' | 'question' | 'graph'
type IssueSectionKey = 'multimodal' | 'asr'

interface ChunkingUIConfig {
  chunkSize: number
  chunkOverlap: number
  separators: string[]
  parserEngineRules?: Array<{
    file_types: string[]
    engine: string
    xlsx_first_row_as_header?: boolean
  }>
  enableParentChild: boolean
  parentChunkSize: number
  childChunkSize: number
  strategy?: string
  tokenLimit?: number
  languages?: string[]
  tableMetadataInstructions?: string
}

interface UploadUIState {
  summaryEnabled: boolean
  chunkingConfig: ChunkingUIConfig
  multimodalConfig: { enabled: boolean; vllmModelId: string; descriptionLanguage?: string; customInstructions?: string }
  asrConfig: { enabled: boolean; modelId: string; language: string }
  questionGenerationConfig: { enabled: boolean; questionCount: number; customInstructions?: string }
  nodeExtractConfig: {
    enabled: boolean
    text: string
    tags: string[]
    nodes: Array<{ name: string; attributes: string[] }>
    relations: Array<{ node1: string; node2: string; type: string }>
    customInstructions?: string
  }
  graphEnabled: boolean
  pdfForceScanned: boolean
}

const props = withDefaults(defineProps<{
  visible: boolean
  kbInfo: any
  mode?: UploadConfirmMode
  files?: File[]
  urls?: string[]
  tagIds?: string[]
  manualPreview?: UploadConfirmManualSource | null
  reparsePreview?: UploadConfirmReparseSource | null
  tagId?: string
  acceptFileTypes?: string
  supportedFileTypes?: string[]
  /** Folder the batch will be uploaded into; '' means the knowledge base root. */
  targetFolder?: string
  /** Existing folders offered as upload destinations. */
  folderOptions?: FolderOption[]
}>(), {
  mode: 'file',
  files: () => [],
  urls: () => [],
  tagIds: () => [],
  manualPreview: null,
  reparsePreview: null,
  acceptFileTypes: '',
  supportedFileTypes: () => [],
  targetFolder: '',
  folderOptions: () => [],
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  confirm: [payload: UploadConfirmResult]
  cancel: []
}>()

const chatResources = useChatResourcesStore()
const editorResources = useEditorResourcesStore()
const uiStore = useUIStore()

const allModels = ref<any[]>([])
const localFiles = ref<File[]>([])
const localUrls = ref<string[]>([])
const availableTags = ref<Array<{ id: string; name: string }>>([])
const selectedTagIds = ref<string[]>([])
const tagsLoading = ref(false)
const tagsLoadFailed = ref(false)
const chunkingMoreOpen = ref(false)
const activeSection = ref<ConfigSectionKey>('tags')
const uiState = ref<UploadUIState>(createDefaultUIState())
// Destination folder for this batch. Pre-filled from the sidebar tree, but
// editable here so browsing a folder never silently decides where files land.
const localTargetFolder = ref('')
const destinationPickerVisible = ref(false)
// Folders created in this dialog before upload; the server tree only gains them
// once files land, so keep them here across picker open/close cycles.
const pendingFolderPaths = ref<string[]>([])

const pickerFolderOptions = computed(() => {
  const byPath = new Map<string, FolderOption>()
  ;(props.folderOptions || []).forEach((option) => byPath.set(option.path, option))
  pendingFolderPaths.value.forEach((path) => {
    if (!byPath.has(path)) byPath.set(path, folderOptionFromPath(path))
  })
  return sortFolderOptions([...byPath.values()])
})

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

// Deep paths are shown as root / segment / segment in the picker row.
const destinationBreadcrumb = computed(() => {
  if (!localTargetFolder.value) return 'Root'
  const parts = localTargetFolder.value.split('/').filter(Boolean)
  return ['Root', ...parts].join(' / ')
})

const destinationFullLabel = computed(() =>
  localTargetFolder.value || 'Root',
)

/**
 * Directory a folder-upload file came from, shown under its name so a batch of
 * same-named files (README.md in five folders) stays distinguishable and the
 * resulting structure is visible before confirming.
 */
function fileRelativeDir(file: File): string {
  const relativePath = (file as File & { webkitRelativePath?: string }).webkitRelativePath
  if (!relativePath) return ''
  return relativePath.split('/').filter(Boolean).slice(0, -1).join('/')
}

function fileDisplayTitle(file: File): string {
  const relativePath = (file as File & { webkitRelativePath?: string }).webkitRelativePath
  return relativePath || file.name
}

function onDestinationCreated(path: string) {
  if (!path || pendingFolderPaths.value.includes(path)) return
  pendingFolderPaths.value = [...pendingFolderPaths.value, path]
}

function onDestinationPicked(path: string) {
  localTargetFolder.value = path
  destinationPickerVisible.value = false
}

function getModelName(modelId: string): string {
  if (!modelId) return 'Not set'
  const model = allModels.value.find((m: any) => m.id === modelId)
  return model?.name || modelId
}

function truncateNavText(text: string, max = 18): string {
  if (!text) return text
  if (text.length <= max) return text
  return `${text.slice(0, max - 1)}…`
}

function hasParserCustomization(): boolean {
  const rules = uiState.value.chunkingConfig.parserEngineRules
  if (!rules?.length) return false
  return rules.some(rule => rule.engine && rule.engine !== 'builtin')
    || rules.some(rule => rule.xlsx_first_row_as_header)
}

function getFileExt(file: File): string {
  const dot = file.name.lastIndexOf('.')
  if (dot < 0) return ''
  return file.name.substring(dot + 1).toLowerCase()
}

function getExtFromUrl(url: string): string {
  try {
    const pathname = new URL(url).pathname
    const dot = pathname.lastIndexOf('.')
    if (dot < 0) return ''
    return pathname.substring(dot + 1).toLowerCase()
  } catch {
    return ''
  }
}

function inferMediaExtsFromMarkdown(content: string): string[] {
  const exts = new Set<string>()
  const patterns = [
    /\.(jpg|jpeg|png|gif|bmp|webp)(\?|#|\)|\s|$)/gi,
    /\.(mp3|wav|m4a|flac|ogg)(\?|#|\)|\s|$)/gi,
  ]
  for (const pattern of patterns) {
    let match: RegExpExecArray | null
    while ((match = pattern.exec(content)) !== null) {
      let ext = match[1].toLowerCase()
      if (ext === 'jpeg') ext = 'jpg'
      exts.add(ext)
    }
  }
  return [...exts]
}

const manualCharCount = computed(() => props.manualPreview?.content?.length ?? 0)
const batchItemCount = computed(() => localFiles.value.length + localUrls.value.length)

const dialogTitle = computed(() => {
  if (props.mode === 'manual') return 'Confirm online publish'
  if (props.mode === 'reparse') return 'Confirm reparse'
  return 'Confirm Upload'
})

const confirmButtonText = computed(() => {
  if (props.mode === 'manual') return 'Publish and parse'
  if (props.mode === 'reparse') return 'Confirm and reparse'
  return 'Upload and parse'
})

const batchFileExts = computed(() => {
  const set = new Set<string>()
  if (props.mode === 'manual' && props.manualPreview?.content) {
    for (const ext of inferMediaExtsFromMarkdown(props.manualPreview.content)) {
      set.add(ext)
    }
  }
  if (props.mode === 'reparse') {
    const ext = (props.reparsePreview?.fileType || '').toLowerCase()
    if (ext) set.add(ext)
  }
  for (const url of localUrls.value) {
    const ext = getExtFromUrl(url)
    if (ext) set.add(ext)
  }
  for (const file of localFiles.value) {
    const ext = getFileExt(file)
    if (ext) set.add(ext)
  }
  return [...set]
})

const hasPdf = computed(() => batchFileExts.value.includes('pdf'))

const chunkingStrategyOptions = computed(() => [
  { label: 'Automatic', value: 'auto' },
  { label: 'Heading-aware', value: 'heading' },
  { label: 'Structure-aware', value: 'heuristic' },
  { label: 'Length-based', value: 'legacy' },
])

const separatorOptions = computed(() => [
  { label: 'Double newline ()', value: '\n\n' },
  { label: 'Single newline ()', value: '\n' },
  { label: 'Danda (।)', value: '।' },
  { label: 'Full stop + space (. )', value: '. ' },
  { label: 'Exclamation mark + space (! )', value: '! ' },
  { label: 'Question mark + space (? )', value: '? ' },
  { label: 'Semicolon (;)', value: ';' },
  { label: 'Space ( )', value: ' ' },
])

const languageOptions = computed(() => [
  { label: 'German', value: 'de' },
  { label: 'English', value: 'en' },
  { label: 'Chinese', value: 'zh' },
])

const tagOptions = computed(() => availableTags.value.map(tag => ({
  label: tag.name,
  value: tag.id,
})))

const llmModelId = computed(() => props.kbInfo?.summary_model_id || '')

const hasImages = computed(() => {
  if (props.mode === 'manual' && props.manualPreview?.content) {
    const content = props.manualPreview.content
    if (/data:image\/|!\[[^\]]*\]\([^)]+\)/i.test(content)) return true
  }
  return batchFileExts.value.some(ext => IMAGE_EXTENSIONS.includes(ext))
})

const hasAudio = computed(() => {
  return batchFileExts.value.some(ext => AUDIO_EXTENSIONS.includes(ext))
})

const isGraphDatabaseEnabled = computed(() => {
  const engine = editorResources.systemInfo?.graph_database_engine
  return !!engine && engine !== 'Not Enabled'
})

const isGraphSectionAvailable = computed(() => {
  return isGraphDatabaseEnabled.value && uiState.value.graphEnabled
})

const showMultimodalModelError = computed(() => {
  return uiState.value.multimodalConfig.enabled && !uiState.value.multimodalConfig.vllmModelId
})

const showAsrModelError = computed(() => {
  return uiState.value.asrConfig.enabled && !uiState.value.asrConfig.modelId
})

const issueSectionKeys = computed(() => {
  const keys = new Set<IssueSectionKey>()
  if (hasImages.value) {
    if (!uiState.value.multimodalConfig.enabled || !uiState.value.multimodalConfig.vllmModelId) {
      keys.add('multimodal')
    }
  } else if (showMultimodalModelError.value) {
    keys.add('multimodal')
  }
  if (hasAudio.value) {
    if (!uiState.value.asrConfig.enabled || !uiState.value.asrConfig.modelId) {
      keys.add('asr')
    }
  } else if (showAsrModelError.value) {
    keys.add('asr')
  }
  return keys
})

const navItems = computed(() => {
  const items: Array<{
    key: ConfigSectionKey
    icon: string
    label: string
    status: string
    statusFull: string
    statusTone?: 'warning' | 'error' | 'muted'
    issue?: boolean
  }> = []

  const push = (key: ConfigSectionKey, icon: string, label: string, issue?: boolean) => {
    const statusMeta = getSectionNavStatus(key, issue)
    const full = statusMeta.status
    items.push({
      key,
      icon,
      label,
      status: truncateNavText(full),
      statusFull: full,
      statusTone: statusMeta.statusTone,
      issue,
    })
  }

  if (props.mode !== 'reparse') {
    push('tags', 'tag', 'Document tags')
  }
  push('parser', 'file-search', 'Parser Engine')
  push('chunking', 'file-copy', 'Chunking Settings')
  push(
    'multimodal',
    'image',
    'Image Processing',
    issueSectionKeys.value.has('multimodal'),
  )
  push(
    'asr',
    'sound',
    'Audio',
    issueSectionKeys.value.has('asr'),
  )
  push('summary', 'file', 'Document summary')
  push('question', 'chat', 'AI Question Generation')
  if (isGraphSectionAvailable.value) {
    push('graph', 'chart-bubble', 'Knowledge Graph')
  }
  return items
})

function getSectionNavStatus(
  key: ConfigSectionKey,
  issue?: boolean,
): { status: string; statusTone?: 'warning' | 'error' | 'muted' } {
  switch (key) {
    case 'tags':
      if (selectedTagIds.value.length === 0) {
        return { status: 'Not set', statusTone: 'muted' }
      }
      return {
        status: `${selectedTagIds.value.length} tags`,
      }
    case 'parser':
      if (uiState.value.pdfForceScanned && hasPdf.value) {
        return { status: 'Scanned mode' }
      }
      if (hasParserCustomization()) {
        return { status: 'Customized' }
      }
      return { status: 'Default', statusTone: 'muted' }
    case 'chunking': {
      const chunking = uiState.value.chunkingConfig
      const parts = [`Chunk ${chunking.chunkSize}`]
      if (chunking.enableParentChild) {
        parts.push('Parent-child')
      }
      return { status: parts.join(' · ') }
    }
    case 'multimodal': {
      if (issue) {
        return { status: 'Needs setup', statusTone: 'error' }
      }
      const mm = uiState.value.multimodalConfig
      if (!mm.enabled) {
        return { status: 'Off', statusTone: 'muted' }
      }
      return {
        status: mm.vllmModelId ? getModelName(mm.vllmModelId) : 'Not set',
        statusTone: mm.vllmModelId ? undefined : 'warning',
      }
    }
    case 'asr': {
      if (issue) {
        return { status: 'Needs setup', statusTone: 'error' }
      }
      const asr = uiState.value.asrConfig
      if (!asr.enabled) {
        return { status: 'Off', statusTone: 'muted' }
      }
      return {
        status: asr.modelId ? getModelName(asr.modelId) : 'Not set',
        statusTone: asr.modelId ? undefined : 'warning',
      }
    }
    case 'summary':
      return uiState.value.summaryEnabled
        ? { status: 'On' }
        : { status: 'Off', statusTone: 'muted' }
    case 'question': {
      const question = uiState.value.questionGenerationConfig
      if (!question.enabled) {
        return { status: 'Off', statusTone: 'muted' }
      }
      return {
        status: `${question.questionCount}`,
      }
    }
    case 'graph': {
      if (!uiState.value.graphEnabled || !uiState.value.nodeExtractConfig.enabled) {
        return { status: 'Off', statusTone: 'muted' }
      }
      const tagCount = uiState.value.nodeExtractConfig.tags?.length ?? 0
      if (tagCount > 0) {
        return { status: `${tagCount}` }
      }
      return { status: 'On' }
    }
    default:
      return { status: '' }
  }
}

const canConfirm = computed(() => {
  if (props.mode === 'file' && batchItemCount.value === 0) return false
  if (props.mode === 'manual' && !props.manualPreview?.content?.trim()) return false
  if (hasImages.value) {
    if (!uiState.value.multimodalConfig.enabled || !uiState.value.multimodalConfig.vllmModelId) {
      return false
    }
  }
  if (hasAudio.value) {
    if (!uiState.value.asrConfig.enabled || !uiState.value.asrConfig.modelId) {
      return false
    }
  }
  if (showMultimodalModelError.value || showAsrModelError.value) {
    return false
  }
  return true
})

function getDefaultSection(): ConfigSectionKey {
  if (props.mode === 'reparse') return 'parser'
  if (issueSectionKeys.value.has('multimodal')) return 'multimodal'
  if (issueSectionKeys.value.has('asr')) return 'asr'
  return 'tags'
}

function goToSection(key: ConfigSectionKey) {
  activeSection.value = key
  nextTick(() => {
    document.querySelector(`[data-section="${key}"]`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

function createDefaultUIState(): UploadUIState {
  return {
    summaryEnabled: true,
    chunkingConfig: {
      chunkSize: 512,
      chunkOverlap: 80,
      separators: ['\n\n', '\n', '।', '. '],
      parserEngineRules: undefined,
      enableParentChild: true,
      parentChunkSize: 4096,
      childChunkSize: 384,
      strategy: 'auto',
      tokenLimit: 0,
      languages: [],
      tableMetadataInstructions: '',
    },
    multimodalConfig: { enabled: false, vllmModelId: '', descriptionLanguage: '', customInstructions: '' },
    asrConfig: { enabled: false, modelId: '', language: '' },
    questionGenerationConfig: { enabled: true, questionCount: 3, customInstructions: '' },
    nodeExtractConfig: {
      enabled: false,
      text: '',
      tags: [],
      nodes: [],
      relations: [],
      customInstructions: '',
    },
    graphEnabled: false,
    pdfForceScanned: false,
  }
}

function initFromKbInfo(kb: any) {
  if (!kb) {
    uiState.value = createDefaultUIState()
    return
  }

  uiState.value = {
    summaryEnabled: true,
    chunkingConfig: {
      chunkSize: kb.chunking_config?.chunk_size || 512,
      chunkOverlap: kb.chunking_config?.chunk_overlap || 80,
      separators: kb.chunking_config?.separators || ['\n\n', '\n', '।', '. '],
      parserEngineRules: kb.chunking_config?.parser_engine_rules || undefined,
      enableParentChild: kb.chunking_config?.enable_parent_child ?? false,
      parentChunkSize: kb.chunking_config?.parent_chunk_size || 4096,
      childChunkSize: kb.chunking_config?.child_chunk_size || 384,
      strategy: kb.chunking_config?.strategy || 'auto',
      tokenLimit: kb.chunking_config?.token_limit || 0,
      languages: kb.chunking_config?.languages || [],
      tableMetadataInstructions: kb.chunking_config?.table_metadata_instructions || '',
    },
    multimodalConfig: {
      enabled: !!kb.vlm_config?.enabled,
      vllmModelId: kb.vlm_config?.model_id || '',
      descriptionLanguage: kb.vlm_config?.description_language || '',
      customInstructions: kb.vlm_config?.custom_instructions || '',
    },
    asrConfig: {
      enabled: !!kb.asr_config?.enabled,
      modelId: kb.asr_config?.model_id || '',
      language: kb.asr_config?.language || '',
    },
    questionGenerationConfig: {
      enabled: kb.question_generation_config?.enabled ?? true,
      questionCount: kb.question_generation_config?.question_count || 3,
      customInstructions: kb.question_generation_config?.custom_instructions || '',
    },
    nodeExtractConfig: {
      enabled: !!kb.extract_config?.enabled && !!kb.indexing_strategy?.graph_enabled,
      text: kb.extract_config?.text || '',
      tags: kb.extract_config?.tags || [],
      nodes: (kb.extract_config?.nodes || []).map((node: any) => ({
        name: node.name,
        attributes: node.attributes || [],
      })),
      relations: kb.extract_config?.relations || [],
      customInstructions: kb.extract_config?.custom_instructions || '',
    },
    graphEnabled: kb.indexing_strategy?.graph_enabled ?? false,
    pdfForceScanned: false,
  }
}

function buildProcessOverrides(): KnowledgeProcessOverrides {
  const state = uiState.value
  const chunking = state.chunkingConfig

  const overrides: KnowledgeProcessOverrides = {
    summary_enabled: state.summaryEnabled,
    parser_engine_rules: chunking.parserEngineRules,
    chunking_config: {
      chunk_size: chunking.chunkSize,
      chunk_overlap: chunking.chunkOverlap,
      separators: chunking.separators,
      enable_parent_child: chunking.enableParentChild,
      parent_chunk_size: chunking.parentChunkSize,
      child_chunk_size: chunking.childChunkSize,
      strategy: chunking.strategy,
      token_limit: chunking.tokenLimit,
      languages: chunking.languages,
      table_metadata_instructions: chunking.tableMetadataInstructions,
    },
    enable_multimodel: state.multimodalConfig.enabled,
    vlm_config: {
      enabled: state.multimodalConfig.enabled,
      model_id: state.multimodalConfig.vllmModelId,
      description_language: state.multimodalConfig.descriptionLanguage,
      custom_instructions: state.multimodalConfig.customInstructions,
    },
    asr_config: {
      enabled: state.asrConfig.enabled,
      model_id: state.asrConfig.modelId,
      language: state.asrConfig.language,
    },
    question_generation_config: {
      enabled: state.questionGenerationConfig.enabled,
      question_count: state.questionGenerationConfig.questionCount,
      custom_instructions: state.questionGenerationConfig.customInstructions,
    },
    graph_enabled: state.nodeExtractConfig.enabled && state.graphEnabled,
    extract_config: {
      enabled: state.nodeExtractConfig.enabled,
      text: state.nodeExtractConfig.text,
      tags: state.nodeExtractConfig.tags,
      nodes: state.nodeExtractConfig.nodes,
      relations: state.nodeExtractConfig.relations,
      custom_instructions: state.nodeExtractConfig.customInstructions,
    },
  }

  if (state.pdfForceScanned) {
    overrides.parser_engine_overrides = {
      pdf_force_scanned: 'true',
    }
  }

  return overrides
}

function applyOverridesToState(o?: KnowledgeProcessOverrides | null) {
  if (!o) return
  const s = uiState.value
  if (o.summary_enabled != null) s.summaryEnabled = o.summary_enabled
  const cc = o.chunking_config
  if (cc) {
    if (cc.chunk_size != null) s.chunkingConfig.chunkSize = cc.chunk_size
    if (cc.chunk_overlap != null) s.chunkingConfig.chunkOverlap = cc.chunk_overlap
    if (cc.separators) s.chunkingConfig.separators = cc.separators
    if (cc.enable_parent_child != null) s.chunkingConfig.enableParentChild = cc.enable_parent_child
    if (cc.parent_chunk_size != null) s.chunkingConfig.parentChunkSize = cc.parent_chunk_size
    if (cc.child_chunk_size != null) s.chunkingConfig.childChunkSize = cc.child_chunk_size
    if (cc.strategy != null) s.chunkingConfig.strategy = cc.strategy
    if (cc.token_limit != null) s.chunkingConfig.tokenLimit = cc.token_limit
    if (cc.languages) s.chunkingConfig.languages = cc.languages
    if (cc.table_metadata_instructions != null) s.chunkingConfig.tableMetadataInstructions = cc.table_metadata_instructions
    if (cc.parser_engine_rules) s.chunkingConfig.parserEngineRules = cc.parser_engine_rules
  }
  if (o.parser_engine_rules) s.chunkingConfig.parserEngineRules = o.parser_engine_rules
  if (o.enable_multimodel != null) s.multimodalConfig.enabled = o.enable_multimodel
  if (o.vlm_config) {
    if (o.vlm_config.enabled != null) s.multimodalConfig.enabled = o.vlm_config.enabled
    if (o.vlm_config.model_id != null) s.multimodalConfig.vllmModelId = o.vlm_config.model_id
    if (o.vlm_config.description_language != null) s.multimodalConfig.descriptionLanguage = o.vlm_config.description_language
    if (o.vlm_config.custom_instructions != null) s.multimodalConfig.customInstructions = o.vlm_config.custom_instructions
  }
  if (o.asr_config) {
    if (o.asr_config.enabled != null) s.asrConfig.enabled = o.asr_config.enabled
    if (o.asr_config.model_id != null) s.asrConfig.modelId = o.asr_config.model_id
    if (o.asr_config.language != null) s.asrConfig.language = o.asr_config.language
  }
  const qg = o.question_generation_config
  if (qg) {
    if (qg.enabled != null) s.questionGenerationConfig.enabled = qg.enabled
    if (qg.question_count != null) s.questionGenerationConfig.questionCount = qg.question_count
    if (qg.custom_instructions != null) s.questionGenerationConfig.customInstructions = qg.custom_instructions
  }
  const ec = o.extract_config
  if (ec) {
    if (ec.enabled != null) s.nodeExtractConfig.enabled = ec.enabled
    if (ec.text != null) s.nodeExtractConfig.text = ec.text
    if (ec.tags) s.nodeExtractConfig.tags = ec.tags
    if (ec.nodes) s.nodeExtractConfig.nodes = ec.nodes.map(n => ({ name: n.name, attributes: n.attributes || [] }))
    if (ec.relations) s.nodeExtractConfig.relations = ec.relations
    if (ec.custom_instructions != null) s.nodeExtractConfig.customInstructions = ec.custom_instructions
  }
  if (o.graph_enabled != null) s.graphEnabled = o.graph_enabled
  s.nodeExtractConfig.enabled = s.nodeExtractConfig.enabled && s.graphEnabled
  if (o.parser_engine_overrides && o.parser_engine_overrides.pdf_force_scanned === 'true') {
    s.pdfForceScanned = true
  } else {
    s.pdfForceScanned = false
  }
}

async function loadModels() {
  try {
    await chatResources.ensureModels()
    allModels.value = chatResources.allModels || []
  } catch {
    allModels.value = []
  }
}

async function loadSystemInfo() {
  try {
    await editorResources.ensureSystemInfo()
  } catch {
    // Graph section falls back to hidden when system info is unavailable.
  }
}

async function loadTags() {
  const kbId = props.kbInfo?.id
  availableTags.value = []
  tagsLoadFailed.value = false
  if (!kbId || props.mode === 'reparse') return

  tagsLoading.value = true
  try {
    const response: any = await listKnowledgeTags(kbId, { page: 1, page_size: 1000 })
    const tags = response?.data?.data || []
    availableTags.value = tags.map((tag: any) => ({
      id: String(tag.id),
      name: String(tag.name || ''),
    }))
  } catch {
    tagsLoadFailed.value = true
  } finally {
    tagsLoading.value = false
  }
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    localFiles.value = props.mode === 'file' ? [...(props.files || [])] : []
    localUrls.value = props.mode === 'file' ? [...(props.urls || [])] : []
    selectedTagIds.value = props.mode === 'reparse' ? [] : [...(props.tagIds || [])]
    localTargetFolder.value = props.mode === 'file' ? (props.targetFolder || '') : ''
    pendingFolderPaths.value = []
    destinationPickerVisible.value = false
    initFromKbInfo(props.kbInfo)
    if (props.mode === 'reparse') {
      applyOverridesToState(props.reparsePreview?.processOverrides)
    }
    activeSection.value = getDefaultSection()
    chunkingMoreOpen.value = false
    loadModels()
    loadSystemInfo()
    loadTags()
  },
)

watch(isGraphSectionAvailable, (available) => {
  if (!available && activeSection.value === 'graph') {
    activeSection.value = getDefaultSection()
  }
})

const appendFiles = (incoming: File[]) => {
  const existingKeys = new Set(localFiles.value.map(getUploadFileKey))
  const toAdd: File[] = []
  let duplicateCount = 0

  for (const file of incoming) {
    const key = getUploadFileKey(file)
    if (existingKeys.has(key)) {
      duplicateCount++
      continue
    }
    existingKeys.add(key)
    toAdd.push(file)
  }

  if (toAdd.length > 0) {
    localFiles.value = [...localFiles.value, ...toAdd]
    MessagePlugin.success(`Added ${toAdd.length} file(s)`)
  } else if (duplicateCount > 0) {
    MessagePlugin.warning('Selected files are already in the list')
  }
}

const appendUrl = (url: string) => {
  if (localUrls.value.includes(url)) {
    MessagePlugin.warning('This URL is already in the list')
    return
  }
  localUrls.value = [...localUrls.value, url]
  MessagePlugin.success('URL added')
}

const removeUrl = (index: number) => {
  localUrls.value = localUrls.value.filter((_, i) => i !== index)
}

const removeFile = (index: number) => {
  localFiles.value = localFiles.value.filter((_, i) => i !== index)
}

const handleParserEngineRulesUpdate = (rules: Array<{
  file_types: string[]
  engine: string
  xlsx_first_row_as_header?: boolean
}>) => {
  uiState.value.chunkingConfig.parserEngineRules = rules
}

const handleMultimodalVLLMChange = (modelId: string) => {
  uiState.value.multimodalConfig.vllmModelId = modelId
}

const handleAddVLLMModel = () => {
  uiStore.openSettings('models', 'vllm')
}

const handleAddASRModel = () => {
  uiStore.openSettings('models', 'asr')
}

const handleNodeExtractUpdate = (config: UploadUIState['nodeExtractConfig']) => {
  uiState.value.nodeExtractConfig = { ...config }
  uiState.value.graphEnabled = config.enabled
}

const validateBeforeConfirm = (): boolean => {
  if (hasImages.value) {
    if (!uiState.value.multimodalConfig.enabled || !uiState.value.multimodalConfig.vllmModelId) {
      MessagePlugin.warning('Configure a multimodal model')
      uiState.value.multimodalConfig.enabled = true
      goToSection('multimodal')
      return false
    }
  } else if (showMultimodalModelError.value) {
    MessagePlugin.warning('Multimodal is enabled. Please select a VLM model.')
    goToSection('multimodal')
    return false
  }

  if (hasAudio.value) {
    if (!uiState.value.asrConfig.enabled || !uiState.value.asrConfig.modelId) {
      MessagePlugin.warning('Configure a speech recognition model')
      uiState.value.asrConfig.enabled = true
      goToSection('asr')
      return false
    }
  } else if (showAsrModelError.value) {
    MessagePlugin.warning('Speech recognition is enabled. Please select an ASR model.')
    goToSection('asr')
    return false
  }
  return true
}

const handleCancel = () => {
  emit('cancel')
  emit('update:visible', false)
}

const handleConfirm = () => {
  if (props.mode === 'file' && batchItemCount.value === 0) {
    MessagePlugin.warning('Add at least one file or URL')
    return
  }
  if (!validateBeforeConfirm()) return

  const processConfig = buildProcessOverrides()
  if (props.mode === 'manual' && props.manualPreview) {
    emit('confirm', {
      processConfig,
      mode: 'manual',
      tagIds: [...selectedTagIds.value],
      manual: { ...props.manualPreview, tagIds: [...selectedTagIds.value] },
    })
  } else if (props.mode === 'reparse' && props.reparsePreview) {
    emit('confirm', { processConfig, mode: 'reparse', reparse: { ...props.reparsePreview } })
  } else {
    emit('confirm', {
      processConfig,
      mode: 'file',
      tagIds: [...selectedTagIds.value],
      files: [...localFiles.value],
      urls: [...localUrls.value],
      targetFolder: localTargetFolder.value,
    })
  }
  emit('update:visible', false)
}
</script>

<style lang="less" scoped>
.upload-confirm-overlay {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
}

.upload-confirm-modal {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 92vw;
  max-width: 1160px;
  height: 85vh;
  max-height: 750px;
  overflow: hidden;
  border-radius: var(--app-radius-xl);
  background: var(--td-bg-color-container);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
}

.close-btn {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  cursor: pointer;

  &:hover {
    color: var(--td-text-color-primary);
  }
}

.upload-confirm-container {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.files-panel {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 220px;
  background: var(--td-bg-color-settings-modal);
  border-right: 1px solid var(--td-component-stroke);
}

.sidebar-header,
.settings-sidebar-header {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  box-sizing: border-box;
  min-height: 56px;
  padding: 12px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.sidebar-header {
  flex-direction: column;
  align-items: stretch;
  gap: 10px;
}

.sidebar-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  min-width: 0;
}

.sidebar-title {
  margin: 0;
  flex: 1;
  min-width: 0;
  padding-right: 0;
  font-size: var(--app-text-xl);
  font-weight: 600;
  line-height: 1.35;
  color: var(--td-text-color-primary);
}

.sidebar-header-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
}

.files-count {
  flex-shrink: 0;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: var(--app-radius-lg);
  font-size: var(--app-text-xs);
  font-weight: 600;
  line-height: 20px;
  text-align: center;
  background: var(--td-bg-color-component);
  color: var(--td-text-color-secondary);
}

// Destination sits under the title inside the header block.
.destination-row {
  flex-shrink: 0;
  padding: 0;
}

.destination-row :deep(.t-popup__reference) {
  display: block;
  max-width: 100%;
}

.destination-crumb {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  min-width: 0;
  padding: 0;
  border: 0;
  background: transparent;
  font-family: var(--app-font-family);
  font-size: var(--app-text-sm);
  line-height: 18px;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: color var(--app-motion-fast) ease;

  &:hover {
    color: var(--td-brand-color);

    .destination-crumb__path,
    .destination-crumb__caret {
      color: var(--td-brand-color);
    }
  }
}

.destination-crumb__label {
  flex-shrink: 0;
}

.destination-crumb__path {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-primary);
  font-weight: 500;
}

.destination-crumb__caret {
  flex-shrink: 0;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
}

.files-list-wrap {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  padding: 6px 8px 12px;
  overflow: hidden;
}

.files-list {
  flex: 1;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
  padding: 6px 6px 6px 8px;
  border-radius: var(--app-radius-sm);
  transition: background-color var(--app-motion-fast) ease;

  &:last-child {
    margin-bottom: 0;
  }

  &:hover {
    background: var(--td-bg-color-container-hover);
  }
}

.file-icon-wrap {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
}

.file-icon {
  font-size: var(--app-text-xl);
  color: var(--td-text-color-secondary);
}

.file-item:hover .file-icon {
  color: var(--td-brand-color);
}

.file-meta {
  flex: 1;
  min-width: 0;
}

.file-name {
  display: block;
  overflow: hidden;
  font-size: var(--app-text-sm);
  font-weight: 500;
  line-height: 1.35;
  color: var(--td-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  display: block;
  margin-top: 1px;
  font-size: var(--app-text-xs);
  line-height: 1.3;
  color: var(--td-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-relative-dir {
  color: var(--td-text-color-secondary);
}

.file-meta-sep {
  margin: 0 4px;
  color: var(--td-text-color-placeholder);
}

.file-remove {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  border-radius: var(--app-radius-xs);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: opacity var(--app-motion-fast) ease, color var(--app-motion-fast) ease, background-color var(--app-motion-fast) ease;
  opacity: 0.45;

  .file-item:hover &,
  &:focus-visible {
    opacity: 1;
  }

  &:hover {
    color: var(--td-text-color-primary);
    background: var(--td-bg-color-component);
  }
}

.files-empty {
  flex: 1;
  padding: 16px 8px;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  text-align: center;
}

.manual-source-panel {
  flex: 1;
  min-height: 0;
  padding: 0;
  overflow-y: auto;
}

.manual-source-title {
  margin: 0;
  padding: 8px 10px;
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-container-hover);
  font-size: var(--app-text-md);
  font-weight: 500;
  line-height: 1.4;
  color: var(--td-text-color-primary);
  word-break: break-word;
}

.manual-source-meta {
  margin: 6px 2px 0;
  font-size: var(--app-text-xs);
  color: var(--td-text-color-placeholder);
}

.config-panel {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.settings-sidebar {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 216px;
  min-height: 0;
  background-color: var(--td-bg-color-settings-modal);
  border-right: 1px solid var(--td-component-stroke);
}

.settings-sidebar-header {
  width: 100%;
}

.settings-sidebar-title {
  margin: 0;
  font-size: var(--app-text-xl);
  font-weight: 600;
  line-height: 1.35;
  color: var(--td-text-color-primary);
}

.settings-nav {
  flex: 1;
  padding: 10px 6px 12px;
  overflow-y: auto;
  min-height: 0;
}

.nav-group-title {
  padding: 6px 14px 2px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  font-weight: 600;
  letter-spacing: 0.02em;

  .settings-nav > &:first-child {
    padding-top: 2px;
  }

  .settings-nav > &:not(:first-child) {
    padding-top: 8px;
  }
}

.nav-item {
  display: flex;
  align-items: flex-start;
  width: 100%;
  margin-bottom: 2px;
  padding: 9px 10px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  font-size: var(--app-text-base);
  color: var(--td-text-color-primary);
  text-align: left;
  cursor: pointer;
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

    .nav-label {
      color: var(--td-brand-color);
    }

    .nav-status {
      color: var(--td-text-color-secondary);
    }
  }

  &--issue .nav-label {
    color: var(--td-error-color);
  }
}

.nav-icon {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
  margin-top: 2px;
  font-size: var(--app-text-xl);
  color: inherit;
}

.nav-label-wrap {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.nav-label {
  overflow: hidden;
  font-size: var(--app-text-md);
  font-weight: 500;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nav-status {
  overflow: hidden;
  font-size: var(--app-text-sm);
  line-height: 1.35;
  color: var(--td-text-color-placeholder);
  text-overflow: ellipsis;
  white-space: nowrap;

  &--muted {
    color: var(--td-text-color-placeholder);
  }

  &--warning {
    color: var(--td-warning-color);
  }

  &--error {
    color: var(--td-error-color);
  }
}

.nav-dot {
  flex-shrink: 0;
  width: 6px;
  height: 6px;
  margin-left: 6px;
  border-radius: 50%;
  background: var(--td-error-color);
}

.content-wrapper {
  flex: 1;
  min-width: 0;
  min-height: 0;
  padding: 22px 32px 28px;
  overflow-y: auto;
  background: var(--td-bg-color-container);
}

.upload-confirm-content {
  .setting-info {
    flex: 0 0 40%;
    max-width: 40%;
  }

  .setting-control {
    flex: 1 1 56%;
    max-width: 56%;
    min-width: 280px;
  }
}

.section-notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 16px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-secondary);

  .t-icon {
    flex-shrink: 0;
    margin-top: 1px;
    font-size: var(--app-text-xl);
    color: var(--td-brand-color);
  }
}

.section {
  margin-bottom: 32px;

  &:last-child {
    margin-bottom: 0;
  }
}

.section-content {
  .section-header {
    margin-bottom: 16px;
  }

  .section-title {
    margin: 0 0 6px;
    font-size: var(--app-text-3xl);
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  .section-desc {
    margin: 0;
    font-size: var(--app-text-base);
    line-height: 22px;
    color: var(--td-text-color-placeholder);
  }
}

.kb-settings-block {
  width: 100%;

  .section-header {
    margin-bottom: 20px;
  }

  .section-title {
    margin: 0 0 6px;
    font-size: var(--app-text-3xl);
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  .section-desc {
    margin: 0;
    font-size: var(--app-text-base);
    line-height: 1.5;
    color: var(--td-text-color-secondary);
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;

  &--more {
    margin-top: 4px;
  }
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
    display: block;
    margin-bottom: 4px;
    font-size: var(--app-text-lg);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .desc {
    margin: 0;
    font-size: var(--app-text-md);
    line-height: 1.5;
    color: var(--td-text-color-secondary);
  }

  .warn {
    margin: 4px 0 0;
    font-size: var(--app-text-sm);
    line-height: 1.4;
    color: var(--td-warning-color);
  }
}

.setting-control {
  display: flex;
  flex: 0 0 55%;
  max-width: 55%;
  align-items: center;
  justify-content: flex-end;

  &-full {
    flex: none;
    width: 100%;
    max-width: none;
    justify-content: flex-start;
  }

  &-inline {
    gap: 12px;
    justify-content: flex-end;
  }
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

.required {
  margin-left: 2px;
  font-weight: 500;
  color: var(--td-error-color);
}

.field-hint {
  margin: 6px 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);

  &--error {
    color: var(--td-error-color);
  }
}

.more-options-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 4px;
  padding: 6px 0;
  border: none;
  background: transparent;
  color: var(--td-brand-color);
  font-size: var(--app-text-md);
  cursor: pointer;

  .t-icon {
    transition: transform 0.18s ease;

    &.is-open {
      transform: rotate(180deg);
    }
  }
}

:deep(.t-input-number) {
  width: 100%;
}

.modal-footer {
  display: flex;
  flex-shrink: 0;
  justify-content: flex-end;
  gap: 12px;
  padding: 14px 20px;
  border-top: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

@media (max-width: 800px) {
  .upload-confirm-container {
    flex-direction: column;
  }

  .files-panel {
    width: auto;
    max-height: 140px;
    border-right: none;
    border-bottom: 1px solid var(--td-component-stroke);
  }

  .settings-sidebar {
    width: auto;
    border-right: none;
    border-bottom: 1px solid var(--td-component-stroke);
  }

  .settings-nav {
    display: flex;
    flex-wrap: nowrap;
    gap: 4px;
    flex: none;
    padding: 8px;
    overflow-x: auto;
  }

  .nav-group-title {
    display: none;
  }

  .nav-item {
    flex: 0 0 auto;
    width: auto;
    margin-bottom: 0;
    white-space: nowrap;
  }

  .content-wrapper {
    padding: 16px;
  }

  .setting-info,
  .setting-control {
    flex: 1 1 100%;
    max-width: none;
  }

  .setting-row {
    flex-direction: column;
    gap: 12px;
  }
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity var(--app-motion-base) ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>

<style lang="less">
// Must sit above the upload modal (z-index 3000). Do not reuse card-more-popup here —
// its global z-index: 99 !important would hide the menu behind the modal overlay.
.upload-destination-popup {
  z-index: 3100 !important;

  .t-popup__content {
    padding: 4px !important;
    margin-top: 6px !important;
    min-width: 208px;
    border-radius: var(--app-radius-lg) !important;
    background: var(--td-bg-color-container) !important;
    border: 0.5px solid var(--td-component-stroke) !important;
    box-shadow:
      0 0 0 0.5px rgba(0, 0, 0, 0.03),
      0 2px 4px rgba(0, 0, 0, 0.04),
      0 8px 24px rgba(0, 0, 0, 0.1) !important;
  }
}
</style>
