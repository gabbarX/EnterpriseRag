<template>
  <div class="kb-upload-source-dropdown">
    <input
      ref="fileInputRef"
      type="file"
      class="hidden-file-input"
      multiple
      :accept="acceptFileTypes || undefined"
      @change="(e) => handleFilesChange(e, false)"
    />
    <input
      ref="folderInputRef"
      type="file"
      class="hidden-file-input"
      webkitdirectory
      multiple
      @change="(e) => handleFilesChange(e, true)"
    />

    <t-tooltip :content="tooltipText" placement="top">
      <t-dropdown
        :options="dropdownOptions"
        trigger="click"
        :placement="placement"
        @click="handleActionSelect"
      >
        <t-button
          variant="text"
          theme="default"
          :class="['kb-upload-source-trigger', triggerClass]"
          :data-guide="dataGuide || undefined"
          :aria-label="tooltipText"
          size="small"
        >
          <template #icon>
            <component v-if="triggerIconComponent" :is="triggerIconComponent" size="18px" :stroke-width="1.7" class="kb-source-icon" />
            <t-icon v-else :name="triggerIcon" size="18px" />
          </template>
          <span v-if="triggerLabel">{{ triggerLabel }}</span>
        </t-button>
      </t-dropdown>
    </t-tooltip>

    <t-dialog
      v-model:visible="urlDialogVisible"
      :header="'Import from URL'"
      :confirm-btn="{ content: 'Confirm', theme: 'primary' }"
      :cancel-btn="{ content: 'Cancel' }"
      width="500px"
      @confirm="handleUrlDialogConfirm"
      @cancel="handleUrlDialogCancel"
    >
      <div class="url-import-form">
        <div class="url-input-label">{{ 'URL Address' }}</div>
        <t-input
          v-model="urlInputValue"
          :placeholder="'Enter webpage URL, e.g., https://example.com'"
          clearable
          autofocus
          @enter="handleUrlDialogConfirm"
        />
        <div class="url-input-tip">{{ 'Supports importing various webpage contents. The system will automatically extract and parse text content from the webpage' }}</div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, h } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { AddIcon, FileAddIcon, UploadIcon, FolderAddIcon, LinkIcon, EditIcon } from 'tdesign-icons-vue-next'
import { filterUploadFiles } from '../utils/uploadSources'

const props = withDefaults(defineProps<{
  acceptFileTypes?: string
  supportedFileTypes?: string[]
  includeManual?: boolean
  triggerIcon?: string
  triggerLabel?: string
  triggerClass?: string
  dataGuide?: string
  tooltip?: string
  placement?: 'top' | 'bottom' | 'bottom-right' | 'bottom-left'
}>(), {
  acceptFileTypes: '',
  supportedFileTypes: () => [],
  includeManual: false,
  triggerIcon: 'file-add',
  triggerClass: '',
  dataGuide: '',
  tooltip: '',
  placement: 'bottom-right',
})

const emit = defineEmits<{
  files: [files: File[]]
  url: [url: string]
  manual: []
}>()


const fileInputRef = ref<HTMLInputElement | null>(null)
const folderInputRef = ref<HTMLInputElement | null>(null)
const urlDialogVisible = ref(false)
const urlInputValue = ref('')

const tooltipText = computed(() => props.tooltip || 'Add Document')

const triggerIconComponent = computed(() => ({
  add: AddIcon,
  'file-add': FileAddIcon,
  upload: UploadIcon,
}[props.triggerIcon]));

const sourceIconProps = { size: '18px', strokeWidth: 1.7, class: 'kb-source-icon' };

const dropdownOptions = computed(() => {
  const options = [
    {
      content: 'Upload Document',
      value: 'upload',
      prefixIcon: () => h(FileAddIcon, sourceIconProps),
    },
    {
      content: 'Upload Folder',
      value: 'uploadFolder',
      prefixIcon: () => h(FolderAddIcon, sourceIconProps),
    },
    {
      content: 'Import from URL',
      value: 'importURL',
      prefixIcon: () => h(LinkIcon, sourceIconProps),
    },
  ]
  if (props.includeManual) {
    options.push({
      content: 'Online Edit',
      value: 'manualCreate',
      prefixIcon: () => h(EditIcon, sourceIconProps),
    })
  }
  return options
})

const handleActionSelect = (data: { value: string }) => {
  switch (data.value) {
    case 'upload':
      fileInputRef.value?.click()
      break
    case 'uploadFolder':
      folderInputRef.value?.click()
      break
    case 'importURL':
      urlInputValue.value = ''
      urlDialogVisible.value = true
      break
    case 'manualCreate':
      emit('manual')
      break
    default:
      break
  }
}

const notifyFilterResult = (result: ReturnType<typeof filterUploadFiles>, emptyAllSkippedKey: string) => {
  const { validFiles, skippedCount, videoFilteredCount } = result
  if (validFiles.length === 0) {
    if (skippedCount > 0) {
      MessagePlugin.warning(emptyAllSkippedKey)
    }
    return false
  }
  if (videoFilteredCount > 0) {
    MessagePlugin.warning(`Skipped ${videoFilteredCount} video file(s) (video upload is not supported)`)
  }
  if (skippedCount > 0) {
    MessagePlugin.warning(`${skippedCount} file(s) skipped due to no available parser engine`)
  }
  return true
}

const handleFilesChange = (event: Event, fromFolder: boolean) => {
  const input = event.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return

  const result = filterUploadFiles(files, {
    supportedFileTypes: props.supportedFileTypes,
    fromFolder,
    multiFile: files.length > 1,
  })

  if (!notifyFilterResult(result, 'All selected files were skipped due to no available parser engine')) {
    input.value = ''
    return
  }

  emit('files', result.validFiles)
  input.value = ''
}

const handleUrlDialogConfirm = () => {
  const url = urlInputValue.value.trim()
  if (!url) {
    MessagePlugin.warning('Please enter a URL')
    return
  }
  try {
    new URL(url)
  } catch {
    MessagePlugin.warning('Please enter a valid URL')
    return
  }
  urlDialogVisible.value = false
  urlInputValue.value = ''
  emit('url', url)
}

const handleUrlDialogCancel = () => {
  urlDialogVisible.value = false
  urlInputValue.value = ''
}

const openUrlDialog = () => {
  urlInputValue.value = ''
  urlDialogVisible.value = true
}

defineExpose({ openUrlDialog })
</script>

<style lang="less" scoped>
.hidden-file-input {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  pointer-events: none;
}

.kb-upload-source-trigger {
  color: var(--td-text-color-secondary);

  &:hover {
    color: var(--td-brand-color);
  }
}

.url-import-form {
  .url-input-label {
    margin-bottom: 8px;
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .url-input-tip {
    margin-top: 8px;
    font-size: var(--app-text-sm);
    line-height: 1.5;
    color: var(--td-text-color-placeholder);
  }
}
</style>

<style lang="less">
// Use consistent rounded strokes for this small family of document-source actions.
.kb-source-icon path {
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>
