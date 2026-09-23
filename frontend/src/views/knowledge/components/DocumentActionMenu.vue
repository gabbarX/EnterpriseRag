<script setup lang="ts">
import { computed } from 'vue';

interface KnowledgeItem {
  id: string;
  file_name?: string;
  title?: string;
  type?: string;
  parse_status?: string;
}

const props = defineProps<{
  item: KnowledgeItem;
  canDownload: boolean;
  canMutateKnowledge: boolean;
  traceVisible: boolean;
  /** Whether the knowledge base has a folder structure to file documents into. */
  foldersAvailable?: boolean;
}>();

const emit = defineEmits<{
  (e: 'download'): void;
  (e: 'edit'): void;
  (e: 'view-trace'): void;
  (e: 'reparse'): void;
  (e: 'cancel-parse'): void;
  (e: 'move'): void;
  (e: 'move-folder'): void;
  (e: 'batch-manage'): void;
  (e: 'delete'): void;
}>();


const CANCELABLE_PARSE_STATUSES = new Set(['pending', 'processing', 'finalizing']);

const isParseInFlight = computed(() =>
  CANCELABLE_PARSE_STATUSES.has(String(props.item.parse_status ?? ''))
);

const fileName = computed(() => props.item.file_name || props.item.title || props.item.id);
</script>

<template>
  <div
    v-if="canDownload && (item.type === 'file' || item.type === 'manual')"
    class="doc-action-menu-item"
    @click.stop="emit('download')"
  >
    <t-icon class="icon" name="download" />
    <span>{{ 'Download' }}</span>
  </div>

  <div v-if="item.type === 'manual'" class="doc-action-menu-item" @click.stop="emit('edit')">
    <t-icon class="icon" name="edit" />
    <span>{{ 'Edit Document' }}</span>
  </div>

  <div v-if="traceVisible" class="doc-action-menu-item" @click.stop="emit('view-trace')">
    <t-icon class="icon" name="chart-bar" />
    <span>{{ 'View trace' }}</span>
  </div>

  <!-- Rebuild (in-flight: no popconfirm, just emits) -->
  <div v-if="isParseInFlight" class="doc-action-menu-item" @click.stop="emit('reparse')">
    <t-icon class="icon" name="refresh" />
    <span>{{ 'Rebuild Document' }}</span>
  </div>

  <!-- Rebuild (normal: with popconfirm) -->
  <t-popconfirm v-else theme="warning"
    :content="`Rebuild document \u0022${fileName}\u0022? This will clear existing chunks and parse it again.`"
    :confirm-btn="{ content: 'Confirm', theme: 'primary' }"
    :cancel-btn="{ content: 'Cancel' }" placement="left"
    @confirm="emit('reparse')">
    <div class="doc-action-menu-item" @click.stop>
      <t-icon class="icon" name="refresh" />
      <span>{{ 'Rebuild Document' }}</span>
    </div>
  </t-popconfirm>

  <t-popconfirm v-if="isParseInFlight" theme="warning"
    :content="`Stop parsing \u0022${fileName}\u0022? Already-written chunks are kept and can be re-parsed later via \u0022Rebuild\u0022; pending optimization tasks (summary / Q&A / knowledge graph) will be dropped immediately.`"
    :confirm-btn="{ content: 'Stop parsing', theme: 'danger' }"
    :cancel-btn="{ content: 'Cancel' }" placement="left"
    @confirm="emit('cancel-parse')">
    <div class="doc-action-menu-item danger" @click.stop>
      <t-icon class="icon" name="close-circle" />
      <span>{{ 'Stop parsing' }}</span>
    </div>
  </t-popconfirm>

  <div v-if="canMutateKnowledge" class="doc-action-menu-item" @click.stop="emit('move-folder')">
    <t-icon class="icon" name="folder" />
    <span>{{ 'Move to folder' }}</span>
  </div>

  <div v-if="canMutateKnowledge" class="doc-action-menu-item" @click.stop="emit('move')">
    <t-icon class="icon" name="swap" />
    <span>{{ 'Move to...' }}</span>
  </div>

  <div v-if="canMutateKnowledge || canDownload" class="doc-action-menu-item" @click.stop="emit('batch-manage')">
    <t-icon class="icon" name="queue" />
    <span>{{ 'Batch Manage' }}</span>
  </div>

  <t-popconfirm theme="warning"
    :content="`Confirm deletion of document \u0022${fileName}\u0022, recovery will be impossible after deletion`"
    :confirm-btn="{ content: 'Confirm Delete', theme: 'danger' }"
    :cancel-btn="{ content: 'Cancel' }" placement="left"
    @confirm="emit('delete')">
    <div class="doc-action-menu-item danger" @click.stop>
      <t-icon class="icon" name="delete" />
      <span>{{ 'Delete Document' }}</span>
    </div>
  </t-popconfirm>
</template>

<style scoped lang="less">
.doc-action-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  font-family: var(--app-font-family);
  font-size: var(--app-text-base);
  font-weight: 400;
  line-height: 20px;
  color: var(--td-text-color-primary);
  cursor: pointer;
  border-radius: var(--app-radius-xs);
  transition: background-color var(--app-motion-fast) cubic-bezier(0.2, 0, 0, 1);

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &:active {
    background: var(--td-bg-color-container-active);
  }

  .icon {
    font-size: var(--app-text-2xl);
    color: var(--td-text-color-secondary);
    transition: color var(--app-motion-fast) ease;
  }

  &:hover .icon {
    color: var(--td-text-color-primary);
  }

  &.danger {
    color: var(--td-error-color-6);
    margin-top: 4px;
    position: relative;

    &::before {
      content: '';
      position: absolute;
      top: -3px;
      left: 8px;
      right: 8px;
      height: 1px;
      background: var(--td-component-stroke);
    }

    .icon {
      color: var(--td-error-color-6);
    }

    &:hover {
      background: var(--td-error-color-1);
      color: var(--td-error-color-6);

      .icon {
        color: var(--td-error-color-6);
      }
    }

    &:active {
      background: var(--td-error-color-2);
    }
  }
}
</style>
