<script setup lang="ts">
import { ref } from 'vue';
import FolderPickerMenu, { type FolderOption } from './FolderPickerMenu.vue';

defineProps<{
  count: number;
  deleteLoading?: boolean;
  reparseLoading?: boolean;
  tagLoading?: boolean;
  downloadLoading?: boolean;
  canDownload?: boolean;
  canMutate?: boolean;
  // When true the bar stays visible even with 0 selections, so users can exit
  // batch mode from here without selecting anything first.
  visible?: boolean;
  /** Hidden when the knowledge base has no folder structure to file into. */
  showMoveToFolder?: boolean;
  folderOptions?: FolderOption[];
}>();

const emit = defineEmits<{
  (e: 'cancel'): void;
  (e: 'delete'): void;
  (e: 'reparse'): void;
  (e: 'batchTag'): void;
  (e: 'moveToFolder', folderPath: string): void;
  (e: 'download'): void;
  (e: 'selectLoaded'): void;
}>();


const folderPickerVisible = ref(false);
</script>

<template>
  <transition name="batch-bar-fade">
    <div v-if="visible || count > 0" class="doc-batch-bar" role="region"
      :aria-label="`${count} selected`">
      <div class="batch-bar-inner">
        <div class="batch-bar-left">
          <span class="batch-bar-count">{{ `${count} selected` }}</span>
          <t-button variant="text" theme="default" size="small" class="batch-bar-clear"
            :disabled="downloadLoading" @click="emit('selectLoaded')">
            {{ 'Select loaded' }}
          </t-button>
          <t-button variant="text" theme="default" size="small" class="batch-bar-clear"
            :disabled="downloadLoading" @click="emit('cancel')">
            {{ 'Deselect all' }}
          </t-button>
        </div>
        <div class="batch-bar-actions">
          <t-tooltip v-if="canDownload" :content="'Download a ZIP of up to 200 documents and 512 MiB of original content per batch. Select all includes loaded documents only; web pages without original files are skipped. The ZIP keeps knowledge-base folders.'">
            <span class="batch-download-trigger">
              <t-button theme="primary" size="small" :loading="downloadLoading"
                :disabled="count === 0 || count > 200 || deleteLoading || reparseLoading || tagLoading || downloadLoading"
                @click="emit('download')">
                <template #icon><t-icon name="download" size="14px" /></template>
                {{ (downloadLoading ? 'Preparing download…' : 'Download selected') }}
              </t-button>
            </span>
          </t-tooltip>

          <t-popconfirm v-if="canMutate" theme="warning" :content="`Rebuild ${count} selected documents? Existing content will be cleared and each document will be re-parsed.`"
            :confirm-btn="{ content: 'Confirm and reparse', theme: 'warning' }"
            :cancel-btn="{ content: 'Cancel' }" placement="top" @confirm="emit('reparse')">
            <t-button theme="default" variant="outline" size="small"
              :disabled="count === 0 || deleteLoading || reparseLoading || tagLoading || downloadLoading" :loading="reparseLoading" @click.stop>
              <template #icon><t-icon name="refresh" size="14px" /></template>
              {{ 'Rebuild Document' }}
            </t-button>
          </t-popconfirm>

          <t-button v-if="canMutate" theme="default" variant="outline" size="small"
            :disabled="count === 0 || deleteLoading || reparseLoading || tagLoading || downloadLoading" :loading="tagLoading"
            @click="emit('batchTag')">
            <template #icon><t-icon name="tag" size="14px" /></template>
            {{ 'Batch Tag' }}
          </t-button>

          <t-popup v-if="canMutate && showMoveToFolder" v-model:visible="folderPickerVisible" trigger="click"
            placement="top" overlay-class-name="card-more" destroy-on-close>
            <t-button theme="default" variant="outline" size="small"
              :disabled="count === 0 || deleteLoading || reparseLoading || tagLoading || downloadLoading">
              <template #icon><t-icon name="folder" size="14px" /></template>
              {{ 'Move to folder' }}
            </t-button>
            <template #content>
              <div class="card-menu">
                <FolderPickerMenu :options="folderOptions || []"
                  @confirm="(path: string) => { folderPickerVisible = false; emit('moveToFolder', path) }" />
              </div>
            </template>
          </t-popup>

          <t-popconfirm v-if="canMutate" theme="warning" :content="`Delete ${count} selected documents? This action cannot be undone.`"
            :confirm-btn="{ content: 'Confirm Delete', theme: 'danger' }"
            :cancel-btn="{ content: 'Cancel' }" placement="top" @confirm="emit('delete')">
            <t-button theme="danger" variant="outline" size="small"
              :disabled="count === 0 || deleteLoading || reparseLoading || tagLoading || downloadLoading" :loading="deleteLoading" @click.stop>
              <template #icon><t-icon name="delete" size="14px" /></template>
              {{ 'Delete selected' }}
            </t-button>
          </t-popconfirm>
        </div>
      </div>
    </div>
  </transition>
</template>

<style scoped lang="less">
.doc-batch-bar {
  position: relative;
  z-index: 5;
  width: 100%;
  max-width: 920px;
  margin: 0 auto;
  padding: 0 4px;
  box-sizing: border-box;
}

.batch-bar-inner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.08);
}

.batch-bar-left {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
  min-width: 0;
  flex: 0 0 auto;
}

.batch-bar-count {
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  white-space: nowrap;
}

.batch-bar-clear {
  flex-shrink: 0;
  padding: 0 6px !important;
  height: 28px !important;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary) !important;

  &:hover {
    color: var(--td-text-color-primary) !important;
  }
}

.batch-bar-actions {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.batch-bar-actions > * { flex-shrink: 0; }

.batch-download-trigger {
  display: inline-flex;
}

.batch-bar-fade-enter-active,
.batch-bar-fade-leave-active {
  transition: transform var(--app-motion-base) ease, opacity var(--app-motion-base) ease;
}

.batch-bar-fade-enter-from,
.batch-bar-fade-leave-to {
  opacity: 0;
  transform: translateY(6px);
}
</style>
