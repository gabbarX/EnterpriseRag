<template>
  <!-- Fallback for ToolResultRenderer when used outside AgentStreamDisplay. -->
  <div class="knowledge-chunks-list">
    <div v-if="summaryHtml" class="results-summary-text" v-html="summaryHtml" />
    <div v-else class="empty-state">{{ 'No matching content found' }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { getKnowledgeChunksSummaryHtml } from '@/utils/knowledgeChunksDisplay';
import type { KnowledgeChunksListData } from '@/types/tool-results';

const props = defineProps<{
  data: KnowledgeChunksListData;
}>();


const summaryHtml = computed(() => getKnowledgeChunksSummaryHtml(props.data));
</script>

<style lang="less" scoped>
.knowledge-chunks-list {
  .results-summary-text {
    font-size: var(--agent-step-summary-size, 12px);
    font-weight: 400;
    color: var(--td-text-color-secondary);
    line-height: 1.5;

    :deep(strong) {
      color: var(--td-text-color-secondary);
      font-weight: 500;
    }
  }

  .empty-state {
    font-size: var(--app-text-md);
    color: var(--td-text-color-placeholder);
  }
}
</style>
