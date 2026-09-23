<template>
  <!-- Anchored in-place popup for folder actions. The trigger is icon-only to
       keep the directory row compact; everything (menu, name input, delete
       confirm) happens inside this one popup so no full-page dialog is spawned.
       click.stop on the trigger and content prevents the surrounding directory
       row from treating the interaction as an expand/collapse toggle. -->
  <t-popup v-model:visible="open" trigger="click" placement="bottom-start" destroy-on-close
    :overlay-class-name="popupOverlayClass" @visible-change="onVisibleChange">
    <span :class="['wiki-directory-action', 'wiki-directory-action--reveal', { 'is-open': open }]"
      :title="'Folder actions'" @click.stop @dragstart.prevent.stop>
      <t-icon name="more" />
    </span>
    <template #content>
      <div class="wiki-folder-menu" @click.stop>
        <div v-if="mode === 'menu'" class="popup-menu">
          <div class="popup-menu-item" @click="enterMode('create')">
            <t-icon name="folder-add" class="menu-icon" />
            <span>{{ 'New Subfolder' }}</span>
          </div>
          <div class="popup-menu-item" @click="emitRename">
            <t-icon name="edit" class="menu-icon" />
            <span>{{ 'Rename Folder' }}</span>
          </div>
          <div class="popup-menu-item delete" @click="enterMode('delete')">
            <t-icon name="delete" class="menu-icon" />
            <span>{{ 'Delete Folder' }}</span>
          </div>
        </div>

        <div v-else-if="mode === 'create'" class="anchored-form-popup-inner">
          <div class="anchored-form-popup-title">{{ 'New Subfolder' }}</div>
          <t-input
            ref="inputRef"
            v-model="nameInput"
            :placeholder="'Enter folder name'"
            @enter="submitName"
          />
          <div class="anchored-form-popup-footer">
            <t-button variant="outline" @click="open = false">
              {{ 'Cancel' }}
            </t-button>
            <t-button theme="primary" :disabled="!nameInput.trim()" @click="submitName">
              {{ 'Confirm' }}
            </t-button>
          </div>
        </div>

        <div v-else class="anchored-form-popup-inner">
          <div class="anchored-form-popup-title">{{ 'Delete Folder' }}</div>
          <div class="anchored-form-popup-body">
            {{
              deletable
                ? `Delete folder \u0022${name}\u0022?`
                : 'Folder is not empty. Remove its pages and sub-folders first.'
            }}
          </div>
          <div class="anchored-form-popup-footer">
            <t-button variant="outline" @click="open = false">
              {{ deletable ? 'Cancel' : 'Confirm' }}
            </t-button>
            <t-button v-if="deletable" theme="danger" @click="submitDelete">
              {{ 'Confirm' }}
            </t-button>
          </div>
        </div>
      </div>
    </template>
  </t-popup>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'

const props = withDefaults(defineProps<{
  name?: string
  pageCount?: number
  hasChildren?: boolean
}>(), {
  name: '',
  pageCount: 0,
  hasChildren: false,
})

const emit = defineEmits<{
  (e: 'create', name: string): void
  (e: 'rename'): void
  (e: 'delete'): void
}>()


const open = ref(false)
const mode = ref<'menu' | 'create' | 'delete'>('menu')
const nameInput = ref('')
const inputRef = ref<{ focus: () => void } | null>(null)

const deletable = computed(() => props.pageCount === 0 && !props.hasChildren)

const popupOverlayClass = computed(() => (
  mode.value === 'menu'
    ? 'card-more-popup wiki-folder-action-overlay'
    : 'anchored-form-popup-overlay'
))

function onVisibleChange(visible: boolean) {
  if (!visible) {
    mode.value = 'menu'
    nameInput.value = ''
    return
  }
  mode.value = 'menu'
}

function enterMode(next: 'create' | 'delete') {
  mode.value = next
  if (next === 'create') {
    nameInput.value = ''
    nextTick(() => inputRef.value?.focus())
  }
}

function emitRename() {
  emit('rename')
  open.value = false
}

function submitName() {
  const value = nameInput.value.trim()
  if (!value) return
  emit('create', value)
  open.value = false
}

function submitDelete() {
  emit('delete')
  open.value = false
}
</script>

<style lang="less" scoped>
.wiki-directory-action {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  font-size: var(--app-text-lg);
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: color var(--app-motion-fast), opacity var(--app-motion-fast);

  &:hover {
    color: var(--td-brand-color);
  }
}

.wiki-directory-action--reveal {
  opacity: 0;

  &.is-open {
    opacity: 1;
  }
}
</style>

<style lang="less">
// Menu chrome comes from card-more-popup + .popup-menu in dropdown-menu.less.
.wiki-folder-action-overlay {
  .wiki-folder-menu {
    min-width: 188px;
  }
}
</style>
