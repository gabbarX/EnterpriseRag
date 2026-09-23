<template>
  <div class="parser-engine-settings">
    <div class="section-header">
      <h2>{{ 'Parser Engine' }}</h2>
      <p class="section-description">
        {{ 'Document parser engine status and configuration. Settings here take priority over server environment variables. Leave empty to use environment variable defaults.' }}
      </p>
    </div>

    <div v-if="loading" class="loading-state">
      <t-loading size="small" />
      <span>{{ 'Loading...' }}</span>
    </div>

    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadAll">{{ 'Retry' }}</t-button>
        </template>
      </t-alert>
    </div>

    <template v-else>
      <div v-if="engines.length === 0 && !hasBuiltinEngine" class="empty-state">
        <p class="empty-text">{{ 'No parser engine detected. Please ensure the DocReader service is running properly.' }}</p>
      </div>

      <div v-else class="engine-cards">
        <button
          v-if="!hasBuiltinEngine"
          type="button"
          class="engine-card engine-card--builtin"
          :class="{ 'engine-card--active': drawerVisible && currentEngine?.Name === 'builtin' }"
          @click="openDrawer({ Name: 'builtin' } as any)"
        >
          <div class="engine-card__badge">{{ engineInitial('builtin') }}</div>
          <div class="engine-card__body">
            <div class="engine-card__header">
              <h3 class="engine-card__title">{{ getEngineDisplayName('builtin') }}</h3>
              <span
                class="engine-card__status"
                :class="connected ? 'engine-card__status--on' : 'engine-card__status--err'"
              >
                <span class="engine-card__status-dot" />
                {{ connected ? 'Connected' : 'Disconnected' }}
              </span>
            </div>
            <p class="engine-card__desc">{{ 'DocReader built-in parser engine (docx/pdf/xlsx and other complex formats)' }}</p>
          </div>
        </button>

        <button
          v-for="engine in sortedEngines"
          :key="engine.Name"
          type="button"
          class="engine-card"
          :class="[
            `engine-card--${engine.Name}`,
            { 'engine-card--active': drawerVisible && currentEngine?.Name === engine.Name }
          ]"
          @click="openDrawer(engine)"
        >
          <div class="engine-card__badge">{{ engineInitial(engine.Name) }}</div>
          <div class="engine-card__body">
            <div class="engine-card__header">
              <h3 class="engine-card__title">{{ getEngineDisplayName(engine.Name) }}</h3>
              <span v-if="engine.Available" class="engine-card__status engine-card__status--on">
                <span class="engine-card__status-dot" />
                {{ 'Available' }}
              </span>
              <t-tooltip
                v-else-if="engine.UnavailableReason"
                :content="engine.UnavailableReason"
                placement="top"
              >
                <span class="engine-card__status engine-card__status--err engine-card__status--help">
                  <span class="engine-card__status-dot" />
                  {{ 'Unavailable' }}
                </span>
              </t-tooltip>
              <span v-else class="engine-card__status engine-card__status--err">
                <span class="engine-card__status-dot" />
                {{ 'Unavailable' }}
              </span>
            </div>
            <p class="engine-card__desc">{{ getEngineDisplayDesc(engine.Name, engine.Description) }}</p>
          </div>
        </button>
      </div>

    </template>

    <SettingDrawer
      v-model:visible="drawerVisible"
      :title="drawerTitle"
      :class="currentEngine ? `parser-engine-drawer parser-engine-drawer--${currentEngine.Name}` : 'parser-engine-drawer'"
      :hide-footer="!authStore.hasRole('admin') && !needsTestButton"
      :confirm-loading="saving"
      @confirm="onSave"
      @cancel="drawerVisible = false"
    >
      <!-- Header icon. Per-engine colouring is injected by the non-scoped
           .parser-engine-drawer--{name} rules at the bottom of this file. -->
      <template v-if="currentEngine" #headerIcon>
        <span class="header-icon__text">{{ engineInitial(currentEngine.Name) }}</span>
      </template>
      <template v-if="currentEngine" #subtitle>
        <span>{{ getEngineDisplayDesc(currentEngine.Name, currentEngine.Description) }}</span>
        <a
          v-if="engineDocLink(currentEngine.Name)"
          :href="engineDocLink(currentEngine.Name)"
          target="_blank"
          rel="noopener noreferrer"
          class="doc-link doc-link--inline"
        >
          {{ engineDocLabel(currentEngine.Name) }}
          <t-icon name="link" class="link-icon" />
        </a>
      </template>
      <template v-if="needsTestButton" #footer-left>
        <t-button variant="outline" :loading="checking" @click="onCheck">
          <template #icon>
            <t-icon v-if="!checking && saveSuccess && checkMessage" name="check-circle-filled"
              class="status-icon available" />
            <t-icon v-else-if="!checking && checkMessage && !saveSuccess" name="close-circle-filled"
              class="status-icon unavailable" />
          </template>
          {{ checking ? 'Testing…' : 'Test Connection' }}
        </t-button>
        <span v-if="checkMessage" :class="['footer-test-message', saveSuccess ? 'success' : 'error']" :title="checkMessage">
          {{ checkMessage }}
        </span>
      </template>

      <div v-if="currentEngine">
        <section
          v-if="currentEngine.FileTypes && currentEngine.FileTypes.length"
          class="setting-drawer__section"
        >
          <h4 class="setting-drawer__section-title">{{ 'Supported Formats' }}</h4>
          <div class="file-types">
            <span v-for="ft in currentEngine.FileTypes" :key="ft" class="file-type-chip">
              {{ ft }}
            </span>
          </div>
        </section>

        <section v-if="currentEngine.Name === 'builtin'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Status' }}</h4>

          <div v-if="currentEngine.Name === 'builtin'" class="docreader-block">
            <div class="status-line">
              <t-tag v-if="connected" theme="success" variant="light" size="small">
                {{ 'Connected' }}
              </t-tag>
              <t-tag v-else theme="danger" variant="light" size="small">
                {{ 'Disconnected' }}
              </t-tag>
              <t-tag theme="default" variant="light" size="small">
                {{ docreaderTransport === 'http' ? 'HTTP' : 'gRPC' }}
              </t-tag>
              <span v-if="docreaderAddrEnv" class="env-hint">
                {{ 'Current' }}: {{ docreaderAddrEnv }}
              </span>
            </div>
            <p class="form-desc">{{ 'To modify, set environment variables DOCREADER_ADDR and DOCREADER_TRANSPORT (grpc/http), then restart the service.' }}</p>
          </div>

        </section>

        <section v-if="currentEngine.Name === 'mineru'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Configuration' }}</h4>

          <div class="form-item">
            <label class="form-label">{{ 'Self-hosted Endpoint' }}</label>
            <t-input
              v-model="config.mineru_endpoint"
              :placeholder="'e.g. https://your-mineru.example.com'"
              clearable
            />
          </div>
          <div class="form-item">
            <label class="form-label">Backend</label>
            <t-select v-model="config.mineru_model" :placeholder="'Default pipeline'" clearable>
              <t-option value="pipeline" label="pipeline" />
              <t-option value="vlm-auto-engine" label="vlm-auto-engine" />
              <t-option value="vlm-http-client" label="vlm-http-client" />
              <t-option value="hybrid-auto-engine" label="hybrid-auto-engine" />
              <t-option value="hybrid-http-client" label="hybrid-http-client" />
            </t-select>
          </div>
          <div class="form-item">
            <label class="form-label">vLLM {{ 'Server URL' }}</label>
            <t-input
              v-model="config.mineru_vlm_server_url"
              :placeholder="'e.g. http://your-vllm-server:8000'"
              clearable
            />
            <p class="form-desc">{{ 'Required when Backend is vlm-http-client or hybrid-http-client' }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ 'PDF Parsing Method' }}</label>
            <t-select v-model="config.mineru_parse_method">
              <t-option value="auto" :label="'Auto-detect (Recommended)'" />
              <t-option value="ocr" :label="'Force OCR'" />
              <t-option value="txt" :label="'Text extraction only'" />
            </t-select>
            <p class="form-desc">{{ 'Auto mode uses OCR for scanned PDFs and extracts the native text layer from digital PDFs.' }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ 'Features' }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.mineru_enable_formula">{{ 'Formula Recognition' }}</t-checkbox>
              <t-checkbox v-model="config.mineru_enable_table">{{ 'Table Recognition' }}</t-checkbox>
            </div>
          </div>
          <div class="form-item">
            <label class="form-label">{{ 'Language' }}</label>
            <t-input
              v-model="config.mineru_language"
              :placeholder="'e.g. ch, en, ja (default ch)'"
              clearable
            />
          </div>
        </section>

        <section v-if="currentEngine.Name === 'mineru_cloud'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Configuration' }}</h4>

          <div class="form-item">
            <label class="form-label required">API Key</label>
            <t-input
              v-model="config.mineru_api_key"
              type="password"
              :placeholder="'MinerU Cloud API Key'"
              clearable
            >
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </div>
          <div class="form-item">
            <label class="form-label">Model Version</label>
            <t-select v-model="config.mineru_cloud_model" :placeholder="'Default pipeline'" clearable>
              <t-option value="pipeline" label="pipeline" />
              <t-option value="vlm" :label="'vlm (Visual Language Model)'" />
              <t-option value="MinerU-HTML" :label="'MinerU-HTML (HTML Parsing)'" />
            </t-select>
          </div>
          <div class="form-item">
            <label class="form-label">{{ 'Features' }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.mineru_cloud_enable_formula">{{ 'Formula Recognition' }}</t-checkbox>
              <t-checkbox v-model="config.mineru_cloud_enable_table">{{ 'Table Recognition' }}</t-checkbox>
              <t-checkbox v-model="config.mineru_cloud_enable_ocr">OCR</t-checkbox>
            </div>
          </div>
          <div class="form-item">
            <label class="form-label">{{ 'Language' }}</label>
            <t-input
              v-model="config.mineru_cloud_language"
              :placeholder="'e.g. ch, en, ja (default ch)'"
              clearable
            />
          </div>
        </section>

        <section v-if="currentEngine.Name === 'paddleocr_vl'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Configuration' }}</h4>

          <div class="form-item">
            <label class="form-label required">{{ 'Self-hosted Endpoint' }}</label>
            <t-input
              v-model="config.paddleocr_vl_endpoint"
              :placeholder="'e.g. http://your-paddleocr-vl:8080'"
              clearable
            />
            <p class="form-desc">{{ 'Base URL of the full PaddleOCR-VL pipeline service; no /layout-parsing suffix needed' }}</p>
          </div>
          <div class="form-item">
            <label class="form-label">{{ 'Features' }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.paddleocr_vl_use_seal_recognition">{{ 'Seal Recognition' }}</t-checkbox>
              <t-checkbox v-model="config.paddleocr_vl_use_chart_recognition">{{ 'Chart Recognition' }}</t-checkbox>
            </div>
          </div>
        </section>

        <section v-if="currentEngine.Name === 'paddleocr_vl_cloud'" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Configuration' }}</h4>

          <div class="form-item">
            <label class="form-label required">Token</label>
            <t-input
              v-model="config.paddleocr_vl_cloud_token"
              type="password"
              :placeholder="'PaddleOCR-VL AI Studio Token'"
              clearable
            >
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </div>
          <div class="form-item">
            <label class="form-label">Model</label>
            <t-input
              v-model="config.paddleocr_vl_cloud_model"
              placeholder="PaddleOCR-VL-1.6"
              clearable
            />
          </div>
          <div class="form-item">
            <label class="form-label">{{ 'Features' }}</label>
            <div class="form-toggles">
              <t-checkbox v-model="config.paddleocr_vl_cloud_use_seal_recognition">{{ 'Seal Recognition' }}</t-checkbox>
              <t-checkbox v-model="config.paddleocr_vl_cloud_use_chart_recognition">{{ 'Chart Recognition' }}</t-checkbox>
            </div>
          </div>
        </section>
      </div>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { MessagePlugin } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import {
  getParserEngines,
  getParserEngineConfig,
  updateParserEngineConfig,
  checkParserEngines,
  type ParserEngineInfo,
  type ParserEngineConfig,
} from '@/api/system'

const uiStore = useUIStore()
const authStore = useAuthStore()

const CONFIGURABLE_ENGINES = new Set(['mineru', 'mineru_cloud', 'paddleocr_vl', 'paddleocr_vl_cloud'])

const ENGINE_DOC_LINKS: Record<string, string> = {
  markitdown: 'https://github.com/microsoft/markitdown',
  mineru: 'https://github.com/opendatalab/MinerU',
  mineru_cloud: 'https://mineru.net/apiManage/docs',
  paddleocr_vl: 'https://github.com/PaddlePaddle/PaddleOCR',
  paddleocr_vl_cloud: 'https://aistudio.baidu.com/paddleocr',
}

/** Parser engine config defaults — keep in sync with the DocReader/Python side. */
const DEFAULT_PARSER_CONFIG: ParserEngineConfig = {
  docreader_addr: '',
  docreader_transport: 'grpc',
  mineru_endpoint: '',
  mineru_api_key: '',
  mineru_model: 'pipeline',
  mineru_vlm_server_url: '',
  mineru_enable_formula: true,
  mineru_enable_table: true,
  mineru_parse_method: 'auto',
  mineru_enable_ocr: true,
  mineru_language: 'en',
  mineru_cloud_model: 'pipeline',
  mineru_cloud_enable_formula: true,
  mineru_cloud_enable_table: true,
  mineru_cloud_enable_ocr: true,
  mineru_cloud_language: 'en',
  paddleocr_vl_endpoint: '',
  paddleocr_vl_use_seal_recognition: true,
  paddleocr_vl_use_chart_recognition: false,
  paddleocr_vl_cloud_token: '',
  paddleocr_vl_cloud_model: 'PaddleOCR-VL-1.6',
  paddleocr_vl_cloud_use_seal_recognition: true,
  paddleocr_vl_cloud_use_chart_recognition: false,
}

const engines = ref<ParserEngineInfo[]>([])
const docreaderAddrEnv = ref('')
const docreaderTransport = ref<'grpc' | 'http'>('grpc')
const connected = ref(false)
const loading = ref(true)
const error = ref('')

const config = ref<ParserEngineConfig>({ ...DEFAULT_PARSER_CONFIG })
const saving = ref(false)
const saveMessage = ref('')
const saveSuccess = ref(false)
const checking = ref(false)
const checkMessage = ref('')

const hasBuiltinEngine = computed(() => engines.value.some(e => e.Name === 'builtin'))

const drawerVisible = ref(false)
const currentEngine = ref<ParserEngineInfo | null>(null)
const drawerTitle = computed(() => {
  return currentEngine.value ? getEngineDisplayName(currentEngine.value.Name) : ''
})

// Whether the footer test-connection button should appear. Engines without
// configurable fields and that aren't the builtin DocReader (whose connection
// status is the whole point of the drawer) skip the test affordance — for
// e.g. simple/markitdown there's nothing to validate beyond presence.
const needsTestButton = computed(() => {
  if (!currentEngine.value) return false
  return hasConfigFields(currentEngine.value.Name) || currentEngine.value.Name === 'builtin'
})

const ENGINE_ORDER: Record<string, number> = {
  builtin: 0,
  simple: 2,
  anydoc: 3,
  markitdown: 4,
  mineru: 5,
  mineru_cloud: 6,
  paddleocr_vl: 7,
  paddleocr_vl_cloud: 8,
}

const sortedEngines = computed(() => {
  return [...engines.value].sort((a, b) => {
    const oa = ENGINE_ORDER[a.Name] ?? 100
    const ob = ENGINE_ORDER[b.Name] ?? 100
    if (oa !== ob) return oa - ob
    return a.Name.localeCompare(b.Name)
  })
})

function hasConfigFields(engineName: string): boolean {
  return CONFIGURABLE_ENGINES.has(engineName)
}

function engineDocLink(name: string): string | undefined {
  return ENGINE_DOC_LINKS[name]
}

function engineDocLabel(_name: string): string {
  return 'Docs'
}

function engineInitial(engineName: string): string {
  const display = getEngineDisplayName(engineName)
  return (display.trim().charAt(0) || engineName.charAt(0) || '?').toUpperCase()
}

function getEngineDisplayName(engineName: string): string {
  const key = `kbSettings.parser.engines.${engineName}.name`
  const translated = key
  return translated !== key ? translated : engineName
}

function getEngineDisplayDesc(engineName: string, fallback: string): string {
  const key = `kbSettings.parser.engines.${engineName}.desc`
  const translated = key
  return translated !== key ? translated : fallback
}

function openDrawer(engine: ParserEngineInfo) {
  currentEngine.value = engine
  drawerVisible.value = true
  saveMessage.value = ''
  checkMessage.value = ''
}

async function loadEngines() {
  try {
    const res = await getParserEngines()
    engines.value = res?.data ?? []
    docreaderAddrEnv.value = res?.docreader_addr ?? ''
    const transport = (res?.docreader_transport ?? 'grpc').toLowerCase()
    docreaderTransport.value = transport === 'http' ? 'http' : 'grpc'
    connected.value = res?.connected ?? (engines.value.length > 0)
  } catch (e: any) {
    error.value = e?.message || 'Failed to load parser engine list'
    engines.value = []
    connected.value = false
  }
}

async function loadConfig() {
  try {
    const res = await getParserEngineConfig()
    const data = res?.data
    config.value = {
      docreader_addr: data?.docreader_addr ?? DEFAULT_PARSER_CONFIG.docreader_addr ?? '',
      docreader_transport: data?.docreader_transport ?? DEFAULT_PARSER_CONFIG.docreader_transport ?? 'grpc',
      mineru_endpoint: data?.mineru_endpoint ?? DEFAULT_PARSER_CONFIG.mineru_endpoint ?? '',
      mineru_api_key: data?.mineru_api_key ?? DEFAULT_PARSER_CONFIG.mineru_api_key ?? '',
      mineru_model: data?.mineru_model ?? DEFAULT_PARSER_CONFIG.mineru_model ?? '',
      mineru_vlm_server_url: data?.mineru_vlm_server_url ?? DEFAULT_PARSER_CONFIG.mineru_vlm_server_url ?? '',
      mineru_enable_formula: data?.mineru_enable_formula ?? DEFAULT_PARSER_CONFIG.mineru_enable_formula ?? true,
      mineru_enable_table: data?.mineru_enable_table ?? DEFAULT_PARSER_CONFIG.mineru_enable_table ?? true,
      mineru_parse_method: data?.mineru_parse_method ?? (data?.mineru_enable_ocr === false ? 'txt' : 'auto'),
      mineru_enable_ocr: data?.mineru_enable_ocr ?? DEFAULT_PARSER_CONFIG.mineru_enable_ocr ?? true,
      mineru_language: data?.mineru_language ?? DEFAULT_PARSER_CONFIG.mineru_language ?? 'en',
      mineru_cloud_model: data?.mineru_cloud_model ?? DEFAULT_PARSER_CONFIG.mineru_cloud_model ?? '',
      mineru_cloud_enable_formula: data?.mineru_cloud_enable_formula ?? DEFAULT_PARSER_CONFIG.mineru_cloud_enable_formula ?? true,
      mineru_cloud_enable_table: data?.mineru_cloud_enable_table ?? DEFAULT_PARSER_CONFIG.mineru_cloud_enable_table ?? true,
      mineru_cloud_enable_ocr: data?.mineru_cloud_enable_ocr ?? DEFAULT_PARSER_CONFIG.mineru_cloud_enable_ocr ?? true,
      mineru_cloud_language: data?.mineru_cloud_language ?? DEFAULT_PARSER_CONFIG.mineru_cloud_language ?? 'en',
      paddleocr_vl_endpoint: data?.paddleocr_vl_endpoint ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_endpoint ?? '',
      paddleocr_vl_use_seal_recognition: data?.paddleocr_vl_use_seal_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_use_seal_recognition ?? true,
      paddleocr_vl_use_chart_recognition: data?.paddleocr_vl_use_chart_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_use_chart_recognition ?? false,
      paddleocr_vl_cloud_token: data?.paddleocr_vl_cloud_token ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_token ?? '',
      paddleocr_vl_cloud_model: data?.paddleocr_vl_cloud_model ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_model ?? 'PaddleOCR-VL-1.6',
      paddleocr_vl_cloud_use_seal_recognition: data?.paddleocr_vl_cloud_use_seal_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_use_seal_recognition ?? true,
      paddleocr_vl_cloud_use_chart_recognition: data?.paddleocr_vl_cloud_use_chart_recognition ?? DEFAULT_PARSER_CONFIG.paddleocr_vl_cloud_use_chart_recognition ?? false,
    }
  } catch {
    config.value = { ...DEFAULT_PARSER_CONFIG }
  }
}

async function loadAll() {
  loading.value = true
  error.value = ''
  await Promise.all([loadEngines(), loadConfig()])
  loading.value = false
}

function buildConfigPayload(): ParserEngineConfig {
  return {
    docreader_addr: config.value.docreader_addr?.trim() ?? '',
    docreader_transport: (config.value.docreader_transport ?? 'grpc').trim() || 'grpc',
    mineru_endpoint: config.value.mineru_endpoint?.trim() ?? '',
    mineru_api_key: config.value.mineru_api_key?.trim() ?? '',
    mineru_model: config.value.mineru_model?.trim() ?? '',
    mineru_vlm_server_url: config.value.mineru_vlm_server_url?.trim() ?? '',
    mineru_enable_formula: config.value.mineru_enable_formula,
    mineru_enable_table: config.value.mineru_enable_table,
    mineru_parse_method: config.value.mineru_parse_method ?? 'auto',
    // Keep the legacy toggle during rolling upgrades. New servers prefer parse_method.
    mineru_enable_ocr: config.value.mineru_parse_method !== 'txt',
    mineru_language: config.value.mineru_language?.trim() ?? '',
    mineru_cloud_model: config.value.mineru_cloud_model?.trim() ?? '',
    mineru_cloud_enable_formula: config.value.mineru_cloud_enable_formula,
    mineru_cloud_enable_table: config.value.mineru_cloud_enable_table,
    mineru_cloud_enable_ocr: config.value.mineru_cloud_enable_ocr,
    mineru_cloud_language: config.value.mineru_cloud_language?.trim() ?? '',
    paddleocr_vl_endpoint: config.value.paddleocr_vl_endpoint?.trim() ?? '',
    paddleocr_vl_use_seal_recognition: config.value.paddleocr_vl_use_seal_recognition,
    paddleocr_vl_use_chart_recognition: config.value.paddleocr_vl_use_chart_recognition,
    paddleocr_vl_cloud_token: config.value.paddleocr_vl_cloud_token?.trim() ?? '',
    paddleocr_vl_cloud_model: config.value.paddleocr_vl_cloud_model?.trim() ?? '',
    paddleocr_vl_cloud_use_seal_recognition: config.value.paddleocr_vl_cloud_use_seal_recognition,
    paddleocr_vl_cloud_use_chart_recognition: config.value.paddleocr_vl_cloud_use_chart_recognition,
  }
}

async function onCheck() {
  if (!connected) {
    checkMessage.value = 'Please ensure the DocReader service is configured via environment variables and connected'
    return
  }
  checking.value = true
  checkMessage.value = ''
  saveMessage.value = ''
  try {
    const res = await checkParserEngines(buildConfigPayload())
    engines.value = res?.data ?? []
    if (res?.connected !== undefined) {
      connected.value = res.connected
    }

    if (currentEngine.value) {
      if (currentEngine.value.Name === 'builtin') {
        if (connected.value) {
          checkMessage.value = 'Test Connection Successful'
          saveSuccess.value = true
        } else {
          checkMessage.value = 'Check failed'
          saveSuccess.value = false
        }
      } else {
        const updatedEngine = engines.value.find(e => e.Name === currentEngine.value!.Name)
        if (updatedEngine) {
          if (updatedEngine.Available) {
            checkMessage.value = 'Test Connection Successful'
            saveSuccess.value = true
          } else {
            checkMessage.value = updatedEngine.UnavailableReason || 'Check failed'
            saveSuccess.value = false
          }
        } else {
          checkMessage.value = 'Check failed'
          saveSuccess.value = false
        }
      }
    } else {
      checkMessage.value = 'Checked with current parameters. Status above has been updated.'
      saveSuccess.value = true
    }

    setTimeout(() => { checkMessage.value = '' }, 3000)
  } catch (e: any) {
    checkMessage.value = e?.message || 'Check failed'
    saveSuccess.value = false
  } finally {
    checking.value = false
  }
}

async function onSave() {
  saving.value = true
  saveMessage.value = ''
  try {
    await updateParserEngineConfig(buildConfigPayload())
    saveSuccess.value = true
    saveMessage.value = 'Saved successfully'
    drawerVisible.value = false
    loadEngines()
  } catch (e: any) {
    saveSuccess.value = false
    saveMessage.value = e?.message || 'Save failed'
  } finally {
    saving.value = false
  }
}

onMounted(loadAll)
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/provider-card.less';

@import (reference) '@/components/css/settings-section.less';

.parser-engine-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-base);
}

.error-inline {
  padding: 16px 0;
}

.empty-state {
  padding: 48px 0;
  text-align: center;

  .empty-text {
    font-size: var(--app-text-base);
    color: var(--td-text-color-placeholder);
    margin: 0;
  }
}

.engine-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
  margin-top: 24px;
}

.engine-card {
  .provider-card();
  .provider-card-interactive();
  text-align: left;
  font: inherit;
  color: inherit;
  cursor: pointer;

  &--active {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-1);
  }
}

.engine-card__badge {
  .provider-card-badge();
  .provider-card-badge-color(#0052d9);
}

.engine-card--builtin .engine-card__badge {
  .provider-card-badge-color(#9444e6);
}
.engine-card--simple .engine-card__badge {
  .provider-card-badge-color(#464646);
}
.engine-card--markitdown .engine-card__badge {
  .provider-card-badge-color(#0089ff);
}
.engine-card--mineru .engine-card__badge,
.engine-card--mineru_cloud .engine-card__badge,
.engine-card--paddleocr_vl .engine-card__badge,
.engine-card--paddleocr_vl_cloud .engine-card__badge {
  .provider-card-badge-color(#6235bb);
}

.engine-card__body {
  .provider-card-body();
}

.engine-card__header {
  .provider-card-header();
}

.engine-card__title {
  .provider-card-title();
}

.engine-card__status {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 1px 8px 1px 6px;
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 16px;
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-secondarycontainer);

  &--on {
    color: var(--td-success-color-7);

    .engine-card__status-dot { background: var(--td-success-color); }
  }

  &--err {
    color: var(--td-error-color-7);

    .engine-card__status-dot { background: var(--td-error-color); }
  }

  &--help {
    cursor: help;
  }
}

.engine-card__status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.engine-card__desc {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  margin: 0;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.form-item {
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: var(--app-text-md);
  font-weight: 500;
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
  margin: 4px 0 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: var(--app-text-md);
}

:deep(.t-checkbox) {
  font-size: var(--app-text-md);

  .t-checkbox__label {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
  }
}

.docreader-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  .form-desc {
    margin-top: 0;
  }
}

.status-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.env-hint {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
}

.file-types {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.file-type-chip {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  font-size: var(--app-text-xs);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-component);
  border-radius: var(--app-radius-xs);
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  letter-spacing: 0.02em;
}

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

.inline-alert--ok .inline-alert__icon {
  color: var(--td-success-color);
}

.inline-alert--warn {
  color: var(--td-text-color-primary);

  .inline-alert__icon {
    color: var(--td-warning-color);
  }
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
  cursor: pointer;
  white-space: nowrap;
  transition: color var(--app-motion-fast) ease;

  &:hover {
    color: var(--td-brand-color-active);
  }

  .t-icon {
    font-size: var(--app-text-base);
  }
}

.spinning {
  animation: wk-spin 1s linear infinite;
}

.form-toggles {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 8px 0 0;
}

.footer-test-message {
  font-size: var(--app-text-sm);
  line-height: 1.4;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &.success {
    color: var(--td-brand-color-active);
  }

  &.error {
    color: var(--td-error-color);
  }
}

.status-icon {
  font-size: var(--app-text-xl);
  flex-shrink: 0;

  &.available {
    color: var(--td-brand-color);
  }

  &.unavailable {
    color: var(--td-error-color);
  }
}

.doc-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-brand-color);
  text-decoration: none;
  transition: color var(--app-motion-fast) ease;

  &:hover {
    color: var(--td-brand-color-active);
  }

  .link-icon {
    font-size: var(--app-text-base);
  }

  &--inline {
    margin-left: 6px;
    font-size: var(--app-text-sm);
    font-weight: 500;
    vertical-align: baseline;

    .link-icon {
      font-size: var(--app-text-sm);
    }
  }
}

.header-icon__text {
  font-size: var(--app-text-lg);
  font-weight: 600;
  letter-spacing: 0.02em;
}
</style>

<!--
  Non-scoped block: per-engine header-icon coloring. Keep these rules global
  so they always apply
  regardless of whether the drawer panel inherits the parent's scoped
  data attributes. Each rule mirrors the matching .engine-card--{name}
  .engine-card__badge from the scoped block above.
-->
<style lang="less">
.parser-engine-drawer--builtin .setting-drawer__header-icon {
  background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
  color: var(--td-brand-color);
}
.parser-engine-drawer--simple .setting-drawer__header-icon {
  background: rgba(70, 70, 70, 0.1);
  color: #464646;
}
.parser-engine-drawer--markitdown .setting-drawer__header-icon {
  background: rgba(0, 137, 255, 0.12);
  color: #0089FF;
}
.parser-engine-drawer--mineru .setting-drawer__header-icon,
.parser-engine-drawer--mineru_cloud .setting-drawer__header-icon,
.parser-engine-drawer--paddleocr_vl .setting-drawer__header-icon,
.parser-engine-drawer--paddleocr_vl_cloud .setting-drawer__header-icon {
  background: rgba(98, 53, 187, 0.12);
  color: #6235BB;
}
</style>
