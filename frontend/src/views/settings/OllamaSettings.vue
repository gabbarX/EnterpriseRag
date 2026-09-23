<template>
  <div class="ollama-settings">
    <div class="section-header">
      <h2>{{ 'Ollama Settings' }}</h2>
      <p class="section-description">{{ 'Manage local Ollama service and view/download models' }}</p>
    </div>

    <div class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Ollama Service Status' }}</label>
          <p class="desc">{{ 'Automatically detect local Ollama service availability. If the service is down or the URL is incorrect, status will be \u0022Unavailable\u0022.' }}</p>
        </div>
        <div class="setting-control">
          <div class="status-display">
            <t-tag 
              v-if="testing"
              theme="default"
              variant="light"
            >
              <t-icon name="loading" class="status-icon spinning" />
              {{ 'Testing' }}
            </t-tag>
            <t-tag 
              v-else-if="connectionStatus === true"
              theme="success"
              variant="light"
            >
              <t-icon name="check-circle-filled" />
              {{ 'Available' }}
            </t-tag>
            <t-tag 
              v-else-if="connectionStatus === false"
              theme="danger"
              variant="light"
            >
              <t-icon name="close-circle-filled" />
              {{ 'Unavailable' }}
            </t-tag>
            <t-tag 
              v-else
              theme="default"
              variant="light"
            >
              <t-icon name="help-circle" />
              {{ 'Not Tested' }}
            </t-tag>
            <t-button 
              size="small" 
              variant="text"
              :loading="testing"
              @click="testConnection"
            >
              <template #icon>
                <t-icon name="refresh" />
              </template>
              {{ 'Retest' }}
            </t-button>
          </div>
        </div>
      </div>

      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Service URL' }}</label>
          <p class="desc">{{ 'The API address of the local Ollama service, auto-detected by the system. To modify, set it in the .env file.' }}</p>
        </div>
        <div class="setting-control">
          <div class="url-control-group">
            <t-input 
              v-model="localBaseUrl" 
              :placeholder="'http://localhost:11434'"
              disabled
              style="flex: 1;"
            />
          </div>
          <t-alert 
            v-if="connectionStatus === false"
            theme="warning"
            :message="'Connection failed. Please check whether Ollama is running or the URL is correct'"
            style="margin-top: 8px;"
          />
        </div>
      </div>

    </div>

    <div v-if="connectionStatus === true" class="model-category-section">
      <div class="category-header">
        <div class="header-info">
          <h3>{{ 'Download Models' }}</h3>
          <p>
            {{ 'Enter a model name to download,' }}
            <a href="https://ollama.com/search" target="_blank" rel="noopener noreferrer" class="doc-link">
              {{ 'Browse Ollama model library' }}
              <t-icon name="link" class="link-icon" />
            </a>
          </p>
        </div>
      </div>
      
      <div class="download-content">
        <div class="input-group">
          <t-input 
            v-model="downloadModelName" 
            :placeholder="'e.g. qwen2.5:0.5b'"
            style="flex: 1;"
          />
          <t-button
            variant="base"
            theme="default"
            size="small"
            class="download-btn"
            :loading="downloading"
            :disabled="!downloadModelName.trim()"
            @click="downloadModel"
          >
            <template #icon><t-icon name="download" /></template>
            {{ 'Download' }}
          </t-button>
        </div>
        
        <div v-if="downloadProgress > 0" class="download-progress">
          <div class="progress-info">
            <span>{{ `Downloading: ${downloadModelName}` }}</span>
            <span>{{ downloadProgress.toFixed(2) }}%</span>
          </div>
          <t-progress :percentage="downloadProgress" size="small" />
        </div>
      </div>
    </div>

    <div v-if="connectionStatus === true" class="model-category-section">
      <div class="category-header">
        <div class="header-info">
          <h3>{{ 'Installed Models' }}</h3>
          <p>{{ 'Models installed in Ollama' }}</p>
        </div>
        <t-button 
          size="small" 
          variant="text"
          :loading="loadingModels"
          @click="refreshModels"
        >
          <template #icon>
            <t-icon name="refresh" />
          </template>
          {{ 'Refresh' }}
        </t-button>
      </div>
      
      <div v-if="loadingModels" class="loading-state">
        <t-loading size="small" />
        <span>{{ 'Loading...' }}</span>
      </div>
      <div v-else-if="downloadedModels.length > 0" class="model-list-container">
        <div v-for="model in downloadedModels" :key="model.name" class="model-card">
          <div class="model-info">
            <div class="model-name">{{ model.name }}</div>
            <div class="model-meta">
              <span class="model-size">{{ formatSize(model.size) }}</span>
              <span class="model-modified">{{ formatDate(model.modified_at) }}</span>
            </div>
          </div>
        </div>
      </div>
      <div v-else class="empty-state">
        <p class="empty-text">{{ 'No installed models' }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { MessagePlugin } from 'tdesign-vue-next'
import { checkOllamaStatus, listOllamaModels, downloadOllamaModel, getDownloadProgress, type OllamaModelInfo } from '@/api/initialization'

const settingsStore = useSettingsStore()

const localBaseUrl = ref(settingsStore.settings.ollamaConfig?.baseUrl ?? '')

const testing = ref(false)
const connectionStatus = ref<boolean | null>(null)
const loadingModels = ref(false)
const downloadedModels = ref<OllamaModelInfo[]>([])
const downloading = ref(false)
const downloadModelName = ref('')
const downloadProgress = ref(0)

const testConnection = async () => {
  testing.value = true
  connectionStatus.value = null
  
  try {
    settingsStore.updateOllamaConfig({ baseUrl: localBaseUrl.value })
    
    const result = await checkOllamaStatus()
    
    if (result.baseUrl && result.baseUrl !== localBaseUrl.value) {
      localBaseUrl.value = result.baseUrl
      settingsStore.updateOllamaConfig({ baseUrl: result.baseUrl })
    }
    
    connectionStatus.value = result.available
    
    if (connectionStatus.value) {
      MessagePlugin.success('Connected successfully')
      refreshModels()
    } else {
      MessagePlugin.error(result.error || 'Connection failed. Please check whether Ollama is running')
    }
  } catch (error: any) {
    connectionStatus.value = false
    MessagePlugin.error(error.message || 'Connection failed. Please check whether Ollama is running')
  } finally {
    testing.value = false
  }
}

const refreshModels = async () => {
  loadingModels.value = true
  
  try {
    const models = await listOllamaModels()
    downloadedModels.value = models
  } catch (error: any) {
    console.error('Failed to get model list:', error)
    MessagePlugin.error(error.message || 'Failed to get model list')
  } finally {
    loadingModels.value = false
  }
}

const formatSize = (bytes: number): string => {
  if (!bytes || bytes === 0 || isNaN(bytes)) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const formatDate = (dateStr: string): string => {
  if (!dateStr) return 'Unknown'

  const date = new Date(dateStr)
  if (isNaN(date.getTime())) return 'Unknown'

  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))

  if (days === 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days < 7) return `${days} days ago`
  return date.toLocaleDateString()
}

const downloadModel = async () => {
  if (!downloadModelName.value.trim()) return
  
  downloading.value = true
  downloadProgress.value = 0
  
  try {
    const result = await downloadOllamaModel(downloadModelName.value)
    
    if (result.status === 'failed') {
      MessagePlugin.error('Download failed. Please try again later')
      downloading.value = false
      downloadProgress.value = 0
      return
    }
    
    MessagePlugin.success(`Started downloading model ${downloadModelName.value}`)
    
    const taskId = result.taskId
    const progressInterval = setInterval(async () => {
      try {
        const task = await getDownloadProgress(taskId)
        downloadProgress.value = task.progress
        
        if (task.status === 'completed') {
          clearInterval(progressInterval)
          MessagePlugin.success(`Model ${downloadModelName.value} downloaded successfully`)
          downloadModelName.value = ''
          downloadProgress.value = 0
          downloading.value = false
          refreshModels()
        } else if (task.status === 'failed') {
          clearInterval(progressInterval)
          MessagePlugin.error(task.message || 'Download failed. Please try again later')
          downloading.value = false
          downloadProgress.value = 0
        }
      } catch (error) {
        clearInterval(progressInterval)
        MessagePlugin.error('Failed to query download progress')
        downloading.value = false
        downloadProgress.value = 0
      }
    }, 1000)
  } catch (error: any) {
    console.error('Download failed:', error)
    MessagePlugin.error(error.message || 'Download failed. Please try again later')
    downloading.value = false
    downloadProgress.value = 0
  }
}

const initOllamaBaseUrl = async () => {
  try {
    const result = await checkOllamaStatus()
    if (result.baseUrl) {
      localBaseUrl.value = result.baseUrl
      if (!settingsStore.settings.ollamaConfig?.baseUrl) {
        settingsStore.updateOllamaConfig({ baseUrl: result.baseUrl })
      }
    } else if (!localBaseUrl.value) {
      localBaseUrl.value = 'http://localhost:11434'
    }
    
      connectionStatus.value = result.available
      if (result.available) {
        refreshModels()
    }
    
    return result
  } catch (error) {
    console.error('Failed to initialise Ollama base URL:', error)
    if (!localBaseUrl.value) {
      localBaseUrl.value = 'http://localhost:11434'
    }
    return null
  }
}

onMounted(async () => {
  await initOllamaBaseUrl()
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.ollama-settings {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  .setting-row();
}

.setting-info {
  .setting-info();
}

.setting-control {
  .setting-control();
  min-width: 360px;
  max-width: 360px;
  flex-direction: column;
  align-items: flex-end;
}

.status-display {
  display: flex;
  align-items: center;
  gap: 12px;

  .status-icon.spinning {
    animation: wk-spin 1s linear infinite;
  }
}

.url-control-group {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
}

.model-category-section {
  margin-top: 32px;
  margin-bottom: 32px;
  padding-top: 32px;
  border-top: 1px solid var(--td-component-stroke);

  &:first-of-type {
    margin-top: 24px;
    padding-top: 24px;
  }

  &:last-child {
    margin-bottom: 0;
  }
}

.category-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;

  .header-info {
    flex: 1;

    h3 {
      font-size: 17px;
      font-weight: 600;
      color: var(--td-text-color-primary);
      margin: 0 0 6px 0;
    }

    p {
      font-size: var(--app-text-md);
      color: var(--td-text-color-placeholder);
      margin: 0;
      line-height: 1.5;
    }

  }
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

.model-list-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;

  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
}

.model-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  transition: all var(--app-motion-base);

  &:hover {
    border-color: var(--td-brand-color);
    background: var(--td-bg-color-container);
  }
}

.model-info {
  flex: 1;
  min-width: 0;

  .model-name {
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--td-text-color-primary);
    margin-bottom: 4px;
    font-family: var(--app-font-family-mono);
  }

  .model-meta {
    display: flex;
    gap: 12px;
    font-size: var(--app-text-sm);
    color: var(--td-text-color-secondary);
  }
}

.download-content {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .input-group {
    display: flex;
    gap: 8px;
    align-items: center;

    .download-btn {
      flex-shrink: 0;
      height: 32px;
    }
  }

  .download-progress {
    padding: 16px;
    background: var(--td-bg-color-secondarycontainer);
    border-radius: var(--app-radius-md);
    border: 1px solid var(--td-component-stroke);

    .progress-info {
      display: flex;
      justify-content: space-between;
      margin-bottom: 10px;
      font-size: var(--app-text-md);
      color: var(--td-text-color-primary);
      font-weight: 500;
    }
  }
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

</style>
