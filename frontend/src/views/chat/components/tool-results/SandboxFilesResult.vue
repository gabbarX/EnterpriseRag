<template>
  <div class="sandbox-files-result">
    <div v-if="summary" class="results-summary-text">{{ summary }}</div>
    <div v-if="items.length" class="results-list">
      <ResultRow
        v-for="(item, index) in items"
        :key="item.path || `${item.name}-${index}`"
        :index="index + 1"
        :title="item.name || item.path"
        :meta="metaFor(item)"
        :popup-key="item.path || index"
        :show-popup="false"
      />
    </div>
    <div v-else class="empty-state">{{ 'No files' }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { formatFileSize } from '@/utils/files'
import {
  formatSandboxModifiedAt,
  sandboxFileListItems,
  type SkillFileListItem,
} from '@/utils/skillToolDisplay'
import type { ListSandboxFilesData } from '@/types/tool-results'
import ResultRow from './ResultRow.vue'

const props = defineProps<{
  data: ListSandboxFilesData | Record<string, unknown>
}>()


const items = computed(() => sandboxFileListItems(props.data))

const summary = computed(() => {
  if (!items.value.length) return ''
  const record = (props.data || {}) as Record<string, unknown>
  const count = typeof record.count === 'number' ? record.count : items.value.length
  const parts = [`Found ${count} file(s)`]
  if (record.truncated) {
    parts.push('List truncated')
  }
  return parts.join(' · ')
})

function sizeLabel(size?: number): string {
  if (size == null || !Number.isFinite(size) || size < 0) return ''
  if (size === 0) return '0 B'
  return formatFileSize(size)
}

function metaFor(item: SkillFileListItem): string {
  const parts = [sizeLabel(item.size), item.modifiedAt ? formatSandboxModifiedAt(item.modifiedAt) : '']
  return parts.filter(Boolean).join(' · ')
}
</script>

<style lang="less" scoped>
@import './tool-results.less';

.sandbox-files-result {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.results-summary-text {
  font-size: var(--agent-step-summary-size, 12px);
  font-weight: 400;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}

.results-list {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.empty-state {
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}
</style>
