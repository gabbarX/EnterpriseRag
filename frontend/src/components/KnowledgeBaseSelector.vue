<template>
  <div v-if="visible" class="kb-overlay" @click="close">
    <div class="kb-dropdown" @click.stop @wheel.stop :style="dropdownStyle">
      <div class="kb-search">
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="text"
          :placeholder="'Search knowledge bases...'"
          class="kb-search-input"
          @keydown.down.prevent="moveSelection(1)"
          @keydown.up.prevent="moveSelection(-1)"
          @keydown.enter.prevent="toggleSelection"
          @keydown.esc="close"
        />
      </div>

      <div class="kb-list" ref="kbList" @wheel.stop>
        <div
          v-for="(kb, index) in filteredKnowledgeBases"
          :key="kb.id"
          :class="['kb-item', { selected: isSelected(kb.id), highlighted: highlightedIndex === index }]"
          @click="toggleKb(kb.id)"
          @mouseenter="highlightedIndex = index"
        >
          <div class="kb-item-left">
            <div class="checkbox" :class="{ checked: isSelected(kb.id) }">
              <svg v-if="isSelected(kb.id)" width="12" height="12" viewBox="0 0 12 12" fill="none">
                <path d="M10 3L4.5 8.5L2 6" stroke="#fff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </div>
            <div class="kb-icon" :class="{ 'faq': kb.type === 'faq' }">
              <svg v-if="kb.type === 'faq'" width="14" height="14" viewBox="0 0 24 24" fill="none">
                <path d="M12 22C17.5228 22 22 17.5228 22 12C22 6.47715 17.5228 2 12 2C6.47715 2 2 6.47715 2 12C2 17.5228 6.47715 22 12 22Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                <path d="M9 9C9 7.89543 9.89543 7 11 7H13C14.1046 7 15 7.89543 15 9C15 10.1046 14.1046 11 13 11H12V14" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                <circle cx="12" cy="17" r="1" fill="currentColor"/>
              </svg>
              <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none">
                <path d="M22 19C22 19.5304 21.7893 20.0391 21.4142 20.4142C21.0391 20.7893 20.5304 21 20 21H4C3.46957 21 2.96086 20.7893 2.58579 20.4142C2.21071 20.0391 2 19.5304 2 19V5C2 4.46957 2.21071 3.96086 2.58579 3.58579C2.96086 3.21071 3.46957 3 4 3H9L11 6H20C20.5304 6 21.0391 6.21071 21.4142 6.58579C21.7893 6.96086 22 7.46957 22 8V19Z" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
            </div>
            <div class="kb-name-wrap">
              <span class="kb-name">{{ kb.name }}</span>
              <span class="kb-docs">({{ kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count || 0) }})</span>
            </div>
          </div>
        </div>

        <div v-if="filteredKnowledgeBases.length === 0" class="kb-empty">
          {{ searchQuery ? 'No matching knowledge base found' : 'No knowledge bases available' }}
        </div>
      </div>

      <div class="kb-actions">
        <button @click="selectAll" class="kb-btn">{{ 'Select All' }}</button>
        <button @click="clearAll" class="kb-btn">{{ 'Clear' }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { getRootZoom, rectToCssPx, cssViewportSize } from '@/utils/zoom'

interface KnowledgeBase {
  id: string
  name: string
  type?: 'document' | 'faq'
  knowledge_count?: number
  chunk_count?: number
  embedding_model_id?: string
  summary_model_id?: string
}


const props = defineProps<{
  visible: boolean
  anchorEl?: any | null
  dropdownWidth?: number
  offsetY?: number
}>()

const emit = defineEmits(['close', 'update:visible'])

const settingsStore = useSettingsStore()

const searchQuery = ref('')
const highlightedIndex = ref(0)
const knowledgeBases = ref<KnowledgeBase[]>([])
const searchInput = ref<HTMLInputElement | null>(null)
const kbList = ref<HTMLElement | null>(null)
const dropdownStyle = ref<Record<string, string>>({})

const dropdownWidth = props.dropdownWidth ?? 300
const offsetY = props.offsetY ?? 8

const filteredKnowledgeBases = computed(() => {
  const valid = knowledgeBases.value.filter(
    k => k.embedding_model_id && k.summary_model_id
  )
  if (!searchQuery.value) return valid
  const q = searchQuery.value.toLowerCase()
  return valid.filter(k => k.name.toLowerCase().includes(q))
})

const selectedKbIds = computed(() => settingsStore.settings.selectedKnowledgeBases || [])

const resolveAnchorEl = () => {
  const a = props.anchorEl
  if (!a) return null
  if (typeof a === 'object' && 'value' in a) {
    return a.value ?? null
  }
  if (typeof a === 'object' && '$el' in a) {
    // @ts-ignore
    return a.$el ?? null
  }
  return a
}

const isSelected = (id: string) => selectedKbIds.value.includes(id)

const toggleKb = (id: string) => {
  isSelected(id) ? settingsStore.removeKnowledgeBase(id) : settingsStore.addKnowledgeBase(id)
}

const toggleSelection = () => {
  const kb = filteredKnowledgeBases.value[highlightedIndex.value]
  if (kb) toggleKb(kb.id)
}

const moveSelection = (dir: number) => {
  const max = filteredKnowledgeBases.value.length
  if (max === 0) return
  highlightedIndex.value = Math.max(0, Math.min(max - 1, highlightedIndex.value + dir))
  nextTick(() => {
    const items = kbList.value?.querySelectorAll('.kb-item')
    items?.[highlightedIndex.value]?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

const selectAll = () => settingsStore.selectKnowledgeBases(filteredKnowledgeBases.value.map(k => k.id))
const clearAll = () => settingsStore.clearKnowledgeBases()

const close = () => {
  emit('update:visible', false)
  emit('close')
}

const loadKnowledgeBases = async () => {
  try {
    const res: any = await listKnowledgeBases()
    if (res?.data && Array.isArray(res.data)) knowledgeBases.value = res.data
  } catch (e) {
    console.error('Failed to load knowledge bases', e)
  }
}

const updateDropdownPosition = () => {
  const anchor = resolveAnchorEl()
  
  // Cache root zoom for this update. Both fallback and rect-anchored paths
  // need to convert visual measurements to CSS pixels (see utils/zoom.ts).
  const zoom = getRootZoom()
  const { width: vwFallback, height: vhFallback } = cssViewportSize(zoom)

  const applyFallback = () => {
    const topFallback = Math.max(80, vhFallback / 2 - 160);
    dropdownStyle.value = {
      position: 'fixed',
      width: `${dropdownWidth}px`,
      left: `${Math.round((vwFallback - dropdownWidth) / 2)}px`,
      top: `${Math.round(topFallback)}px`,
      transform: 'none',
      margin: '0',
      padding: '0',
    };
  };
  
  if (!anchor) {
    applyFallback()
    return
  }

  let rawRect: { top: number; left: number; right: number; bottom: number; width: number; height: number } | null = null
  try {
    if (typeof anchor.getBoundingClientRect === 'function') {
      const r = anchor.getBoundingClientRect()
      rawRect = { top: r.top, left: r.left, right: r.right, bottom: r.bottom, width: r.width, height: r.height }
    } else if (anchor.width !== undefined && anchor.left !== undefined) {
      // Already a DOMRect-like
      rawRect = anchor as DOMRect
    }
  } catch (e) {
    console.error('[KnowledgeBaseSelector] Error getting bounding rect:', e)
  }

  if (!rawRect || rawRect.width === 0 || rawRect.height === 0) {
    applyFallback()
    return
  }

  // Convert to CSS pixels so subsequent comparisons against the dropdown's
  // own width/height stay in one coordinate system.
  const rect = rectToCssPx(rawRect, zoom)
  console.log('[KB Selector] Button rect (css px):', rect)
  const vw = vwFallback
  const vh = vhFallback
  
  let left = Math.floor(rect.left)
  
  const minLeft = 16
  const maxLeft = Math.max(16, vw - dropdownWidth - 16)
  left = Math.max(minLeft, Math.min(maxLeft, left))

  const preferredDropdownHeight = 280
  const maxDropdownHeight = 360
  const minDropdownHeight = 200
  const topMargin = 20
  const spaceBelow = vh - rect.bottom
  const spaceAbove = rect.top
  
  console.log('[KB Selector] Space check:', {
    spaceBelow,
    spaceAbove,
    windowHeight: vh
  })
  
  let actualHeight: number
  let shouldOpenBelow: boolean
  
  if (spaceBelow >= minDropdownHeight + offsetY) {
    actualHeight = Math.min(preferredDropdownHeight, spaceBelow - offsetY - 16)
    shouldOpenBelow = true
    console.log('[KB Selector] Position: below button', { actualHeight })
  } else {
    const availableHeight = spaceAbove - offsetY - topMargin
    if (availableHeight >= preferredDropdownHeight) {
      actualHeight = preferredDropdownHeight
    } else {
      actualHeight = Math.max(minDropdownHeight, availableHeight)
    }
    shouldOpenBelow = false
    console.log('[KB Selector] Position: above button', { actualHeight })
  }
  
  if (shouldOpenBelow) {
    const top = Math.floor(rect.bottom + offsetY)
    console.log('[KB Selector] Opening below, top:', top)
    dropdownStyle.value = {
      position: 'fixed',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      top: `${top}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none',
      margin: '0',
      padding: '0'
    }
  } else {
    const bottom = vh - rect.top + offsetY
    console.log('[KB Selector] Opening above, bottom:', bottom)
    dropdownStyle.value = {
      position: 'fixed',
      width: `${dropdownWidth}px`,
      left: `${left}px`,
      bottom: `${bottom}px`,
      maxHeight: `${actualHeight}px`,
      transform: 'none',
      margin: '0',
      padding: '0'
    }
  }
}

let resizeHandler: (() => void) | null = null
let scrollHandler: (() => void) | null = null

watch(() => props.visible, async (v) => {
  if (v) {
    await loadKnowledgeBases();
    await nextTick();
    requestAnimationFrame(() => {
      updateDropdownPosition();
      requestAnimationFrame(() => {
        updateDropdownPosition();
        setTimeout(() => {
          updateDropdownPosition();
        }, 50);
      });
    });
    nextTick(() => searchInput.value?.focus());
    resizeHandler = () => updateDropdownPosition();
    scrollHandler = () => updateDropdownPosition();
    window.addEventListener('resize', resizeHandler, { passive: true });
    window.addEventListener('scroll', scrollHandler, { passive: true, capture: true });
  } else {
    searchQuery.value = '';
    highlightedIndex.value = 0;
    if (resizeHandler) {
      window.removeEventListener('resize', resizeHandler);
      resizeHandler = null;
    }
    if (scrollHandler) {
      window.removeEventListener('scroll', scrollHandler, { capture: true });
      scrollHandler = null;
    }
  }
});
</script>

<style scoped lang="less">
.kb-overlay,
.kb-overlay *,
.kb-overlay *::before,
.kb-overlay *::after {
  box-sizing: border-box;
}

.kb-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: transparent;
  touch-action: none;
}

.kb-dropdown {
  position: fixed !important;
  background: var(--td-bg-color-container);
  border: .5px solid var(--td-component-border);
  border-radius: var(--app-radius-lg);
  box-shadow: var(--td-shadow-2);
  overflow: hidden;
  animation: fadeIn 0.15s ease-out;
  z-index: 10000;
  margin: 0;
  transform-origin: top left;
  display: flex;
  flex-direction: column;
}

.kb-search {
  padding: 8px 10px;
  border-bottom: .5px solid var(--td-component-stroke);
}
.kb-search-input {
  width: 100%;
  padding: 6px 10px;
  font-size: var(--app-text-sm);
  border: .5px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  outline: none;
  transition: border var(--app-motion-instant);
}
.kb-search-input:focus {
  border-color: var(--td-success-color);
  background: var(--td-bg-color-container);
}

.kb-list {
  flex: 1;
  min-height: 0;
  max-height: 260px;
  overflow-y: auto;
  padding: 6px 8px;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
}

.kb-item {
  display: flex;
  align-items: center;
  padding: 6px 8px;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  transition: background var(--app-motion-instant);
  margin-bottom: 4px;
}
.kb-item:last-child { margin-bottom: 0; }

.kb-item:hover,
.kb-item.highlighted { background: var(--td-bg-color-secondarycontainer); }

.kb-item.selected { background: var(--td-brand-color-light); }

.kb-item-left {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.checkbox {
  width: 16px; height: 16px;
  border-radius: 3px;
  border: 1.5px solid var(--td-component-border);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.checkbox.checked {
  background: var(--td-success-color);
  border-color: var(--td-success-color);
}
.checkbox.checked svg {
  width: 10px;
  height: 10px;
}
.kb-icon {
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--td-brand-color-active);
  
  &.faq {
    color: var(--td-brand-color);
  }
}
.kb-name-wrap { display:flex; flex-direction: row; align-items: center; gap: 4px; min-width: 0; }
.kb-name { font-size: var(--app-text-sm); color: var(--td-text-color-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; line-height: 1.4; }
.kb-docs { font-size: var(--app-text-xs); color: var(--td-text-color-placeholder); flex-shrink: 0; }

.kb-empty { padding: 20px 8px; text-align: center; color: var(--td-text-color-placeholder); font-size: var(--app-text-sm); }

.kb-actions {
  display: flex;
  gap: 8px;
  padding: 8px 10px;
  border-top: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
}
.kb-btn {
  flex: 1;
  padding: 6px 10px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: all var(--app-motion-instant);
}
.kb-btn:hover {
  border-color: var(--td-success-color);
  color: var(--td-success-color);
  background: var(--td-brand-color-light);
}

@keyframes fadeIn {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}
</style>
