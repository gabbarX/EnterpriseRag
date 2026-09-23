<template>
    <div class="main" ref="dropzone" :style="{ '--sidebar-width': `${uiStore.sidebarDisplayWidth}px` }">
        <Menu></Menu>
        <div v-if="isRouterAlive" class="platform-route-outlet">
            <RouterView />
        </div>
        <div class="upload-mask" v-show="ismask">
            <UploadMask></UploadMask>
        </div>
        <Settings />
        <GlobalCommandPalette />
        <!-- Pending-invitation bell, fixed to the top right. Its z-index sits
             below the drawers, so a page's right-hand drawer covers it; rendered
             only while invitations are pending. -->
        <GlobalInvitationBell />
        <!-- Knowledge base upload progress overlay: the upload queue lives in the store, so navigating between pages does not interrupt it. -->
        <UploadTasksPanel />
        <NewUserGuide />
    </div>
</template>
<script setup lang="ts">
import Menu from '@/components/menu.vue'
import { ref, onMounted, onUnmounted, nextTick, provide, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router'
import UploadMask from '@/components/upload-mask.vue'
import Settings from '@/views/settings/Settings.vue'
import GlobalCommandPalette from '@/components/GlobalCommandPalette.vue'
import GlobalInvitationBell from '@/components/GlobalInvitationBell.vue'
import UploadTasksPanel from '@/components/upload-tasks/UploadTasksPanel.vue'
import NewUserGuide from '@/components/NewUserGuide.vue'
import { useCommandPaletteStore } from '@/stores/commandPalette'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useUIStore } from '@/stores/ui'
import { getKnowledgeBaseById } from '@/api/knowledge-base/index'
import { MessagePlugin } from 'tdesign-vue-next'
import { collectDroppedFiles } from './collectDroppedFiles'

const route = useRoute();
const router = useRouter();
const commandPaletteStore = useCommandPaletteStore();
const uiStore = useUIStore();
let ismask = ref(false)

const isRouterAlive = ref(true)
const reloadApp = () => {
    isRouterAlive.value = false
    nextTick(() => {
        isRouterAlive.value = true
    })
}
provide('app:reload', reloadApp)

// Intercept Cmd/Ctrl+R only in the Wails desktop runtime: the desktop build
// has no address bar and a full page reload leaves a blank screen, so a
// front-end soft refresh replaces it. In a browser the shortcut is left alone
// so a real reload happens; otherwise the side menu, global settings and the
// Pinia stores would not reset along with the "refresh".
// @ts-ignore
const isWailsDesktop = typeof window !== 'undefined' && !!(window as any).runtime?.EventsOn

const handleGlobalKeyDown = (e: KeyboardEvent) => {
    if (!isWailsDesktop) return
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'r') {
        e.preventDefault()
        reloadApp()
    }
}

// Counts dragenter/dragleave so that dragleave fired by child elements does not dismiss the mask
let dragCounter = 0;

const getCurrentKbId = (): string | null => {
    return (route.params as any)?.kbId as string || null
}

const CHAT_DROP_ROUTE_NAMES = new Set(['chat', 'globalCreatChat', 'kbCreatChat']);

const isChatDropRoute = () => {
    return CHAT_DROP_ROUTE_NAMES.has(String(route.name || ''));
}

const checkKnowledgeBaseInitialization = async (): Promise<boolean> => {
    const currentKbId = getCurrentKbId();
    
    if (!currentKbId) {
        MessagePlugin.error('Knowledge base ID is missing');
        return false;
    }
    
    try {
        const kbResponse = await getKnowledgeBaseById(currentKbId);
        const kb = kbResponse.data;
        
        if (!kb.summary_model_id) {
            MessagePlugin.warning('Knowledge base is not initialized. Please configure models in settings before uploading files');
            return false;
        }
        const strategy = kb.indexing_strategy;
        const needsEmbedding = !strategy || strategy.vector_enabled || strategy.keyword_enabled;
        if (needsEmbedding && !kb.embedding_model_id) {
            MessagePlugin.warning('Knowledge base is not initialized. Please configure models in settings before uploading files');
            return false;
        }
        return true;
    } catch (error) {
        MessagePlugin.error('Failed to get knowledge base information, file upload is not possible');
        return false;
    }
}


// isFileDrag distinguishes an OS file drag (the only thing the global upload
// drop zone cares about) from an in-app element drag such as the wiki
// folder/page drag-and-drop. Element drags carry only "text/*" types, never
// "Files", so we bail out and let the originating component handle the drop.
const isFileDrag = (event: DragEvent): boolean => {
    const types = event.dataTransfer?.types
    if (!types) return false
    return Array.from(types).includes('Files')
}

const shouldHandleGlobalFileDrag = (event: DragEvent): boolean => {
    if (!isFileDrag(event)) return false;
    // Keep the browser from opening dropped files, even outside upload pages.
    event.preventDefault();
    // Settings and its teleported skill drawers own their uploads. This runs
    // in document capture, before a local drop handler can stop propagation.
    const enabled = !uiStore.showSettingsModal && (
        isChatDropRoute() || (route.name === 'knowledgeBaseDetail' && !!getCurrentKbId())
    );
    if (!enabled) {
        dragCounter = 0;
        ismask.value = false;
    }
    return enabled;
}

const handleGlobalDragEnter = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter++;
    if (event.dataTransfer) {
        event.dataTransfer.effectAllowed = 'all';
    }
    ismask.value = true;
}

const handleGlobalDragOver = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    if (event.dataTransfer) {
        event.dataTransfer.dropEffect = 'copy';
    }
}

const handleGlobalDragLeave = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter--;
    if (dragCounter === 0) {
        ismask.value = false;
    }
}

const handleGlobalDrop = async (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter = 0;
    ismask.value = false;

    const droppedFiles = await collectDroppedFiles(event);
    if (droppedFiles.length === 0) {
        MessagePlugin.warning('Please drag files instead of text or links');
        return;
    }

    if (isChatDropRoute()) {
        event.stopPropagation();
        window.dispatchEvent(new CustomEvent('enterpriserag:chat-file-drop', {
            detail: { files: droppedFiles }
        }));
        return;
    }
    
    const isInitialized = await checkKnowledgeBaseInitialization();
    if (!isInitialized) {
        return;
    }

    window.dispatchEvent(new CustomEvent('enterpriserag:knowledge-file-drop', {
        detail: { kbId: getCurrentKbId(), files: droppedFiles }
    }));
}

onMounted(() => {
    document.addEventListener('dragenter', handleGlobalDragEnter, true);
    document.addEventListener('dragover', handleGlobalDragOver, true);
    document.addEventListener('dragleave', handleGlobalDragLeave, true);
    document.addEventListener('drop', handleGlobalDrop, true);
    if (isWailsDesktop) {
        window.addEventListener('keydown', handleGlobalKeyDown);
        // @ts-ignore
        window.runtime.EventsOn('app:reload', () => {
            reloadApp()
        })
    }
    // Supports opening the global command palette from a URL query parameter,
    // e.g. the legacy /platform/knowledge-search?q=foo redirects with ?cmdk=foo
    maybeOpenCmdkFromRoute()
    void useChatResourcesStore().prefetchChatInput()
});

watch(() => route.query.cmdk, () => {
    maybeOpenCmdkFromRoute()
})

function maybeOpenCmdkFromRoute() {
    if (!('cmdk' in route.query)) return
    const q = String(route.query.cmdk ?? '')
    commandPaletteStore.openPalette(q)
    // Clear the query so going back or refreshing does not reopen it repeatedly
    const newQuery = { ...route.query }
    delete (newQuery as any).cmdk
    router.replace({ path: route.path, query: newQuery, hash: route.hash })
}

onUnmounted(() => {
    document.removeEventListener('dragenter', handleGlobalDragEnter, true);
    document.removeEventListener('dragover', handleGlobalDragOver, true);
    document.removeEventListener('dragleave', handleGlobalDragLeave, true);
    document.removeEventListener('drop', handleGlobalDrop, true);
    if (isWailsDesktop) {
        window.removeEventListener('keydown', handleGlobalKeyDown);
        // @ts-ignore
        if (window.runtime?.EventsOff) {
            // @ts-ignore
            window.runtime.EventsOff('app:reload')
        }
    }
    dragCounter = 0;
});
</script>
<style lang="less">
.main {
    display: flex;
    align-items: stretch;
    width: 100%;
    height: 100%;
    min-width: 600px;
    min-height: 0;
    background: var(--td-bg-color-container);
}

/* Route area: fills the remaining width and the full column height, and passes min-height:0 down so child pages can scroll inside their own flex layout */
.platform-route-outlet {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.upload-mask {
    background-color: rgba(255, 255, 255, 0.8);
    position: fixed;
    width: 100%;
    height: 100%;
    z-index: 999;
    display: flex;
    justify-content: center;
    align-items: center;
}

img {
    -webkit-user-drag: none;
    -khtml-user-drag: none;
    -moz-user-drag: none;
    -o-user-drag: none;
    user-drag: none;
}
</style>
