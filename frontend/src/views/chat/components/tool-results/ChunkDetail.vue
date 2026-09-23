<template>
  <div class="chunk-detail">
    <div class="info-section">
      <div class="info-field">
        <span class="field-label">{{ 'Chunk ID:' }}</span>
        <span class="field-value"><code>{{ data.chunk_id }}</code></span>
      </div>
      <div class="info-field">
        <span class="field-label">{{ 'Document ID:' }}</span>
        <span class="field-value"><code>{{ data.knowledge_id }}</code></span>
      </div>
      <div class="info-field">
        <span class="field-label">{{ 'Position:' }}</span>
        <span class="field-value">{{ `Chunk #${data.chunk_index}` }}</span>
      </div>
      <div v-if="data.content_length" class="info-field">
        <span class="field-label">{{ 'Content length:' }}</span>
        <span class="field-value">{{ `${data.content_length} characters` }}</span>
      </div>
    </div>

    <div class="info-section">
      <div class="info-section-title">{{ 'Full content' }}</div>
      <div class="full-content">{{ data.content }}</div>
    </div>

    <div class="info-section">
      <div class="action-buttons">
        <button class="action-button" @click="copyToClipboard">
          📋 {{ 'Copy content' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ChunkDetailData } from '@/types/tool-results';
import { copyToClipboard as copyTextToClipboard } from '@/utils/clipboard';

const props = defineProps<{
  data: ChunkDetailData;
}>();


const copyToClipboard = () => {
  void copyTextToClipboard(props.data.content);
};
</script>

<style lang="less" scoped>
@import './tool-results.less';

.chunk-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 8px 0;
}

code {
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-xs);
  background: var(--td-bg-color-secondarycontainer);
  padding: 2px 4px;
  border-radius: 3px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}
</style>
