<template>
    <div class="aside_box" :class="{ 'aside_box--collapsed': uiStore.sidebarCollapsed, 'aside_box--resizing': uiStore.sidebarResizing }">
        <div class="logo_row" v-if="!uiStore.sidebarCollapsed">
            <div class="logo_box" @click="router.push('/platform/knowledge-bases')" style="cursor: pointer;">
                <img class="logo logo--light" src="@/assets/img/sustainability-logo.png" alt="sustainability.ai">
                <img class="logo logo--dark" src="@/assets/img/sustainability-logo-dark.png" alt="sustainability.ai">
                <sup v-if="isLiteEdition" class="lite-badge">Lite</sup>
            </div>
            <div class="logo_actions">
                <t-tooltip placement="bottom">
                    <template #content>
                        <span class="cmdk-tip">
                            <span class="cmdk-tip-label">{{ 'Search' }}</span>
                            <span class="cmdk-tip-keys">{{ cmdModKeyLabel }}K</span>
                        </span>
                    </template>
                    <div class="header-icon-btn" @click="commandPaletteStore.openPalette('')"
                        :aria-label="'Search'">
                        <img class="header-icon-img" :src="getImgSrc('search.svg')" alt="">
                    </div>
                </t-tooltip>
                <div class="sidebar-toggle" @click="uiStore.toggleSidebar" :title="'Collapse Sidebar'">
                    <svg viewBox="0 0 20 20" width="18" height="18" fill="none" xmlns="http://www.w3.org/2000/svg">
                        <rect x="1.5" y="1.5" width="17" height="17" rx="3" stroke="currentColor" stroke-width="1.2" />
                        <line x1="7.5" y1="1.5" x2="7.5" y2="18.5" stroke="currentColor" stroke-width="1.2" />
                        <line x1="4" y1="7.5" x2="4" y2="12.5" stroke="currentColor" stroke-width="1.2"
                            stroke-linecap="round" />
                    </svg>
                </div>
            </div>
        </div>
        <t-tooltip v-else :content="'Expand Sidebar'" placement="right">
            <div class="menu_item sidebar-toggle-item" @click="uiStore.toggleSidebar">
                <div class="menu_item-box">
                    <div class="menu_icon">
                        <svg class="icon" viewBox="0 0 20 20" width="20" height="20" fill="none"
                            xmlns="http://www.w3.org/2000/svg">
                            <rect x="1.5" y="1.5" width="17" height="17" rx="3" stroke="currentColor"
                                stroke-width="1.2" />
                            <line x1="7.5" y1="1.5" x2="7.5" y2="18.5" stroke="currentColor" stroke-width="1.2" />
                            <line x1="5" y1="10" x2="3" y2="8" stroke="currentColor" stroke-width="1.2"
                                stroke-linecap="round" />
                            <line x1="5" y1="10" x2="3" y2="12" stroke="currentColor" stroke-width="1.2"
                                stroke-linecap="round" />
                        </svg>
                    </div>
                </div>
            </div>
        </t-tooltip>

        <TenantSelector v-if="canAccessAllTenants && !uiStore.sidebarCollapsed" />

        <PanelResizeHandle edge="right" :label="'Drag to resize panel width'"
            :value="uiStore.sidebarDisplayWidth" :min="SIDEBAR_COLLAPSED_WIDTH" :max="SIDEBAR_MAX_WIDTH"
            @start="startSidebarResize" @resize="resizeSidebar" @end="uiStore.sidebarResizing = false" />

        <div class="menu_top" ref="scrollContainer" @scroll="handleScroll">
            <div class="menu_box menu_box--cmdk" v-if="uiStore.sidebarCollapsed">
                <t-tooltip placement="right">
                    <template #content>
                        <span class="cmdk-tip">
                            <span class="cmdk-tip-label">{{ 'Search' }}</span>
                            <span class="cmdk-tip-keys">{{ cmdModKeyLabel }}K</span>
                        </span>
                    </template>
                    <div class="menu_item menu_item--cmdk" @click="commandPaletteStore.openPalette('')">
                        <div class="menu_item-box">
                            <div class="menu_icon">
                                <img class="icon" :src="getImgSrc('search.svg')" alt="">
                            </div>
                        </div>
                    </div>
                </t-tooltip>
            </div>
            <div class="menu_box" :class="{ 'menu_box--sticky': item.children && !uiStore.sidebarCollapsed }"
                v-for="(item, index) in topMenuItems" :key="index">
                <t-tooltip :content="item.title" placement="right" :disabled="!uiStore.sidebarCollapsed">
                    <div @click="handleMenuClick(item.path)" @mouseenter="mouseenteMenu(item.path)"
                        @mouseleave="mouseleaveMenu(item.path)" :data-guide="`nav-${item.path}`"
                        :class="['menu_item', item.childrenPath && item.childrenPath == currentpath ? 'menu_item_c_active' : isMenuItemActive(item.path) ? 'menu_item_active' : '']">
                        <div class="menu_item-box">
                            <div class="menu_icon">
                                <img class="icon"
                                    :src="getImgSrc(item.icon == 'zhishiku' ? knowledgeIcon : item.icon == 'agent' ? agentIcon : item.icon == 'artifact' ? artifactIcon : item.icon == 'organization' ? organizationIcon : item.icon == 'logout' ? logoutIcon : item.icon == 'setting' ? settingIcon : prefixIcon)"
                                    alt="">
                            </div>
                            <template v-if="!uiStore.sidebarCollapsed">
                                <span class="menu_title" :title="item.title">{{ item.title }}</span>
                                <span v-if="item.path === 'organizations' && orgStore.totalPendingJoinRequestCount > 0"
                                    class="menu-pending-badge"
                                    :title="'Pending join requests to review'">{{
                                        orgStore.totalPendingJoinRequestCount }}</span>
                            </template>
                        </div>
                    </div>
                </t-tooltip>
            </div>

            <div class="submenu" v-if="!uiStore.sidebarCollapsed">
                <!-- Stable, always-mounted source filter: reserving its row here
                     (instead of embedding it in the first date group, which
                     appears/disappears while a bucket loads) prevents the
                     top-right control from jumping when switching session type. -->
                <div v-if="showSessionSourceFilter && !batchMode" class="session-list-scope-header">
                    <SessionSourceFilter inline :emphasized="sessionScopeFilterPinned" :sources="sessionSourceOptions"
                        :current="activeSessionBucketKey" @select="switchSessionBucket" />
                </div>
                <template v-if="sessionListBooting && !hasAnySession">
                    <div v-for="n in 4" :key="'skel-' + n" class="submenu_item_p session-chat-row">
                        <div class="session-list-row session-list-row--flat">
                            <t-skeleton animation="gradient" class="session-list-row__body"
                                :row-col="[{ width: '100%', height: '14px' }]" />
                        </div>
                    </div>
                </template>

                <div v-else class="session-filtered-list">
                    <template
                        v-if="activeBucket?.loading && !activeBucket.loaded && filteredGroupedSessions.length === 0">
                        <div v-for="n in 4" :key="'bucket-skel-' + n" class="submenu_item_p session-chat-row">
                            <div class="session-list-row session-list-row--flat">
                                <t-skeleton animation="gradient" class="session-list-row__body"
                                    :row-col="[{ width: '100%', height: '14px' }]" />
                            </div>
                        </div>
                    </template>
                    <template v-else-if="activeBucket?.loaded && filteredGroupedSessions.length === 0">
                        <div class="submenu_empty">{{ 'No conversations yet' }}</div>
                    </template>
                    <template v-else>
                        <template v-for="group in filteredGroupedSessions" :key="group.key">
                            <div v-if="group.label" class="timeline_header session-list-row session-list-row--flat">
                                <span class="session-list-row__body">
                                    <span class="timeline_header-label">{{ group.label }}</span>
                                </span>
                            </div>
                            <div v-for="subitem in group.items" :key="subitem.id"
                                class="submenu_item_p session-chat-row" :data-session-id="subitem.id" :class="{
                                    'session-chat-row--active': !batchMode && subitem.path === currentSecondpath,
                                    'session-chat-row--selected': batchMode && batchSelectedIds.includes(subitem.id),
                                    'session-chat-row--revealed': revealedSessionId === subitem.id,
                                }">
                                <div class="session-list-row session-list-row--flat">
                                    <div class="session-list-row__body">
                                        <SessionSidebarRow :item="subitem" :batch-mode="batchMode"
                                            :running="Boolean(sessionActivityEntries[subitem.id])"
                                            :active-path="currentSecondpath" :selected-ids="batchSelectedIds"
                                            :menu-options="buildSessionMenuOptions(subitem)"
                                            @navigate="gotopage(subitem.path)"
                                            @toggle-select="toggleBatchSelect(subitem.id)"
                                            @menu-click="handleSessionMenuClick($event, subitem)"
                                            @rename-submit="renameSessionTitle(subitem, $event.title)"
                                            @hover-in="mouseenteBotDownr(subitem.id)" @hover-out="mouseleaveBotDown" />
                                    </div>
                                </div>
                            </div>
                        </template>
                        <div v-if="activeBucket?.loading && filteredGroupedSessions.length > 0"
                            class="session-list-loading session-list-row session-list-row--flat">
                            <span class="session-list-row__body">
                                <t-loading size="small" />
                            </span>
                        </div>
                    </template>
                </div>
            </div>
        </div>

        <div v-if="batchMode && !uiStore.sidebarCollapsed" class="batch-inline-footer">
            <div class="batch-footer-left">
                <t-checkbox :checked="isAllBatchSelected" :indeterminate="isBatchIndeterminate"
                    @change="toggleBatchSelectAll">
                    {{ 'Select All' }}
                </t-checkbox>
            </div>
            <div class="batch-footer-right">
                <t-button size="small" variant="text" @click="exitBatchMode">
                    {{ 'Cancel' }}
                </t-button>
                <t-button size="small" theme="danger" variant="base" :disabled="batchSelectedIds.length === 0"
                    :loading="batchDeleting" @click="handleInlineBatchDelete">
                    {{ 'Delete Conversations' }}{{ batchSelectedIds.length > 0 ? `(${batchDisplayCount})` : '' }}
                </t-button>
            </div>
        </div>

        <div class="menu_bottom">
            <UserMenu />
        </div>

    </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia';
import { onMounted, onUnmounted, watch, computed, ref, h, nextTick } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { getSessionsList, batchDelSessions, deleteAllSessions, getSession } from "@/api/chat/index";
import { useChatResourcesStore } from '@/stores/chatResources';
import { listAllIMChannels } from '@/api/agent/index';
import SessionSidebarRow from './SessionSidebarRow.vue';
import PanelResizeHandle from './PanelResizeHandle.vue';
import { SIDEBAR_COLLAPSED_WIDTH, SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH } from '@/utils/sidebarWidth';
import {
    clearSession,
    removeSession,
    renameSession,
    SESSION_MUTATION_EVENT,
    setSessionPinned,
    type SessionMutationDetail,
} from './sessionMutations';
import SessionSourceFilter from './SessionSourceFilter.vue';
import {
    SIDEBAR_BUCKET_PAGE_SIZE,
    applyBucketCountProbe,
    buildBucketDefinitions,
    bucketHasMore,
    bucketVisible,
    createEmptyBucket,
    flattenBucketItems,
    isChannelBucket,
    isChannelBucketKey,
    mergeBucketPage,
    prependSessionToWebBucket,
    removeSessionFromBuckets,
    type SidebarSessionBucket,
} from './sessionSidebarBuckets';
import type { SessionForGrouping } from './sessionGrouping';
import { listAllEmbedChannels } from '@/api/embed/index';
import {
    classifyDateBucket,
    configuredPlatforms,
    groupSessionsByDate,
    originGroupKey,
    resolveSessionOrigin,
    type DateBucketKey,
} from './sessionGrouping';
import {
    DEFAULT_SESSION_BUCKET_KEY,
    buildSessionSourceOptions,
    findSessionBucketKey,
    shouldShowSessionSourceFilter,
} from './sessionSidebarSourceFilter';
import { logout as logoutApi } from '@/api/auth';
import { useMenuStore } from '@/stores/menu';
import { useSessionActivityStore } from '@/stores/sessionActivity';
import { useAuthStore } from '@/stores/auth';
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities';
import { useOrganizationStore } from '@/stores/organization';
import { useUIStore } from '@/stores/ui';
import { useCommandPaletteStore } from '@/stores/commandPalette';
import { MessagePlugin, DialogPlugin, Icon as TIcon } from "tdesign-vue-next";
import UserMenu from '@/components/UserMenu.vue';
import TenantSelector from '@/components/TenantSelector.vue';
import { getSystemInfo } from '@/api/system';

const AGENT_EDITOR_IM_LABELS: Record<string, string> = {
  title: 'IM Integration',
  description: 'Connect agent to instant messaging platforms like Slack, Telegram and Mattermost',
  slack: 'Slack',
  telegram: 'Telegram',
  mattermost: 'Mattermost',
  addChannel: 'Add Channel',
  channelsTitle: 'IM Channels',
  disabled: 'Disabled',
  editChannel: 'Edit Channel',
  deleteConfirm: 'Are you sure you want to delete this channel? This action cannot be undone.',
  channelName: 'Channel Name',
  channelNamePlaceholder: 'Enter a name for easy identification',
  channelNameDefaultHint: 'Defaults to the platform name; you can customize it, or leave blank to use the platform name on save',
  platform: 'Platform',
  mode: 'Connection Mode',
  outputMode: 'Output Mode',
  outputStream: 'Streaming',
  outputFull: 'Full Response',
  callbackUrl: 'Callback URL',
  empty: 'No IM channels yet',
  unnamed: 'Unnamed Channel',
  docLink: 'Integration Guide',
  slackConsole: 'Slack API Console',
  telegramConsole: 'Telegram BotFather',
  mattermostConsole: 'Mattermost integrations',
  mattermostModeHint: 'Mattermost only supports Webhook mode (outgoing webhook + bot token).',
  mattermostPostToMain: 'Post replies in channel timeline',
  mattermostPostToMainHint: 'When on, bot replies are new top-level posts in the channel. When off (default), they stay in the thread and the main view only shows “N replies”.',
  modeHint: 'WebSocket is recommended for easier setup',
  consoleTip: 'to get credentials',
  fileKnowledgeBase: 'File Storage Knowledge Base',
  fileKnowledgeBasePlaceholder: 'Select a knowledge base (optional)',
  fileKnowledgeBaseHint: 'When configured, files sent by users will be automatically saved to this knowledge base',
  sessionMode: 'Session Mode',
  sessionModeUser: 'Per User (default)',
  sessionModeThread: 'Per Thread',
  sessionModeHint: 'User mode: each person has their own conversation. Use /clear to start fresh. Thread mode: each message thread is a separate conversation. Multiple people can collaborate in the same thread.',
  sectionChannel: 'Channel info',
  sectionConnection: 'Connection',
  sectionCredentials: 'Platform credentials',
  selectPlatform: 'Select platform',
  selectPlatformDesc: 'Choose the messaging platform to connect',
  enabled: 'Enable channel',
  stepBasic: 'Basic info',
  stepConnection: 'Connection',
  stepKnowledge: 'File storage',
  stepCredentials: 'Credentials',
  sectionAccess: 'Access & output',
  sectionSession: 'Session',
  sectionCallback: 'Callback URL',
  sectionKnowledge: 'File storage',
  sectionStatus: 'Status',
}

const chatResources = useChatResourcesStore();
// Platform logos reused from IMChannelsOverviewPanel — keeps the session list
// visually consistent with the channels admin view.
import slackLogo from '@/assets/img/im/slack.svg';
import telegramLogo from '@/assets/img/im/telegram.svg';
import mattermostLogo from '@/assets/img/im/mattermost.svg';

const PLATFORM_LOGO: Record<string, string> = {
    slack: slackLogo,
    telegram: telegramLogo,
    mattermost: mattermostLogo,
};

const platformLogo = (p: string): string => (p ? PLATFORM_LOGO[p] || '' : '');

const usemenuStore = useMenuStore();
const sessionActivity = useSessionActivityStore();
const { entries: sessionActivityEntries } = storeToRefs(sessionActivity);
let sessionActivityTimer: ReturnType<typeof setInterval> | undefined;
const authStore = useAuthStore();
const deploymentCapabilities = useDeploymentCapabilitiesStore();
const orgStore = useOrganizationStore();
const uiStore = useUIStore();
const commandPaletteStore = useCommandPaletteStore();

// Platform-aware label for the ⌘K hint. navigator.platform is deprecated but
// the alternatives (userAgentData.platform) aren't universally available yet;
// this check is good enough for Mac vs. non-Mac.
const isMacLike = typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform || '');
const cmdModKeyLabel = isMacLike ? '⌘' : 'Ctrl';
const route = useRoute();
const router = useRouter();
const currentpath = ref('');
const total = ref(0);
const sessionBuckets = ref<Record<string, SidebarSessionBucket>>({});
const bucketOrder = ref<string[]>([]);
let bucketRequestToken = 0;
const sessionListBooting = ref(false);
const currentSecondpath = ref('');
const scrollContainer = ref<HTMLElement | null>(null);
const imPlatforms = ref<string[]>([]);
const embedChannelNames = ref<Record<string, string>>({});
const activeSessionBucketKey = ref(DEFAULT_SESSION_BUCKET_KEY);
const sessionListCanScroll = ref(false);
const visibleChannelBuckets = computed(() =>
    bucketOrder.value
        .map((key) => sessionBuckets.value[key])
        .filter((bucket): bucket is SidebarSessionBucket => !!bucket && isChannelBucket(bucket) && bucketVisible(bucket)),
);
const showSessionSourceFilter = computed(() =>
    shouldShowSessionSourceFilter(visibleChannelBuckets.value.length),
);
const sessionScopeFilterPinned = computed(() =>
    activeSessionBucketKey.value !== DEFAULT_SESSION_BUCKET_KEY,
);
const sessionSourceOptions = computed(() =>
    buildSessionSourceOptions(
        'My chats',
        visibleChannelBuckets.value.map((bucket) => ({
            key: bucket.key,
            label: bucket.label,
            platform: bucket.platform,
        })),
        (platform) => platformLogo(platform),
    ),
);
const activeBucket = computed(() => sessionBuckets.value[activeSessionBucketKey.value]);
const hasAnySession = computed(() =>
    Object.values(sessionBuckets.value).some((bucket) => bucket.items.length > 0),
);
type MenuItem = { title: string; icon: string; path: string; childrenPath?: string; children?: any[] };
const { menuArr, visibleMenuArr } = storeToRefs(usemenuStore);
let activeSubmenu = ref<string>('');
const isLiteEdition = ref(false);

const batchMode = ref(false)
const batchSelectedIds = ref<string[]>([])
const batchDeleting = ref(false)

const allSessionIds = computed(() => {
    const chatMenu = (menuArr.value as unknown as MenuItem[]).find((item: MenuItem) => item.path === 'creatChat');
    if (!chatMenu?.children) return [];
    return (chatMenu.children as any[]).map((s: any) => s.id);
})

const isAllBatchSelected = computed(() =>
    allSessionIds.value.length > 0 && batchSelectedIds.value.length === allSessionIds.value.length
)

const isBatchIndeterminate = computed(() =>
    batchSelectedIds.value.length > 0 && batchSelectedIds.value.length < allSessionIds.value.length
)

const batchDisplayCount = computed(() =>
    isAllBatchSelected.value ? total.value : batchSelectedIds.value.length
)

const canAccessAllTenants = computed(() => authStore.canAccessAllTenants);

const isInKnowledgeBase = computed<boolean>(() => {
    return route.name === 'knowledgeBaseDetail' ||
        route.name === 'kbCreatChat' ||
        route.name === 'knowledgeBaseSettings';
});

const isInKnowledgeBaseList = computed<boolean>(() => {
    return route.name === 'knowledgeBaseList';
});

const isInCreatChat = computed<boolean>(() => {
    return route.name === 'globalCreatChat' || route.name === 'kbCreatChat';
});

const isInChatDetail = computed<boolean>(() => route.name === 'chat');

const isInAgentList = computed<boolean>(() => route.name === 'agentList');

const isInOrganizationList = computed<boolean>(() => route.name === 'organizationList');

const isMenuItemActive = (itemPath: string): boolean => {
    const currentRoute = route.name;

    switch (itemPath) {
        case 'knowledge-bases':
            return currentRoute === 'knowledgeBaseList' ||
                currentRoute === 'knowledgeBaseDetail' ||
                currentRoute === 'knowledgeBaseSettings';
        case 'agents':
            return currentRoute === 'agentList';
        case 'artifacts':
            return currentRoute === 'artifactLibrary';
        case 'organizations':
            return currentRoute === 'organizationList';
        case 'creatChat':
            return currentRoute === 'kbCreatChat' || currentRoute === 'globalCreatChat';
        case 'settings':
            return currentRoute === 'settings';
        default:
            return itemPath === currentpath.value;
    }
};

const getIconActiveState = (itemPath: string) => {
    const currentRoute = route.name;

    return {
        isKbActive: itemPath === 'knowledge-bases' && (
            currentRoute === 'knowledgeBaseList' ||
            currentRoute === 'knowledgeBaseDetail' ||
            currentRoute === 'knowledgeBaseSettings'
        ),
        isCreatChatActive: itemPath === 'creatChat' && (currentRoute === 'kbCreatChat' || currentRoute === 'globalCreatChat'),
        isSettingsActive: itemPath === 'settings' && currentRoute === 'settings',
        isChatActive: itemPath === 'chat' && currentRoute === 'chat'
    };
};

const TOP_MENU_PATHS = new Set(['creatChat', 'knowledge-bases', 'artifacts', 'agents', 'organizations']);

const topMenuItems = computed<MenuItem[]>(() => {
    return (visibleMenuArr.value as unknown as MenuItem[]).filter((item: MenuItem) => TOP_MENU_PATHS.has(item.path));
});

const bottomMenuItems = computed<MenuItem[]>(() => {
    return (visibleMenuArr.value as unknown as MenuItem[]).filter((item: MenuItem) => !TOP_MENU_PATHS.has(item.path));
});

const currentKbName = ref<string>('')
const currentKbInfo = ref<any>(null)

const pinningIds = ref<Set<string>>(new Set())

const dateBucketLabels = computed<Record<DateBucketKey, string>>(() => ({
    pinned: 'Pinned',
    today: 'Today',
    yesterday: 'Yesterday',
    last7Days: 'Last 7 Days',
    last30Days: 'Last 30 Days',
    lastYear: 'Last Year',
    earlier: 'Earlier',
}));

const filteredGroupedSessions = computed(() => {
    const bucket = activeBucket.value;
    if (!bucket?.items.length) return [];
    return groupSessionsByDate(
        bucket.items.map((item) => ({
            ...item,
            path: `chat/${item.id}`,
            title: item.title || '',
        })),
        dateBucketLabels.value,
        (session) => classifyDateBucket(session.updated_at || session.created_at),
    );
});

// Only a locally created fork requests attention; loading history and switching
// between existing sessions must not replay the entrance animation.
const pendingForkRevealId = ref('');
const revealedSessionId = ref('');
let forkRevealTimer: ReturnType<typeof setTimeout> | undefined;
usemenuStore.$onAction(({ name, args, after }) => {
    if (name !== 'updataMenuChildren' || !args[0]?.parent_session_id) return;
    const sessionId = String(args[0].id);
    after(() => { pendingForkRevealId.value = sessionId; });
});

watch(
    () => {
        const id = pendingForkRevealId.value;
        return id && !uiStore.sidebarCollapsed && currentSecondpath.value === `chat/${id}`
            && filteredGroupedSessions.value.some((group) => group.items.some((item) => item.id === id))
            ? id : '';
    },
    (id) => {
        if (!id) return;
        const container = scrollContainer.value;
        const row = Array.from(container?.querySelectorAll<HTMLElement>('[data-session-id]') ?? [])
            .find((element) => element.dataset.sessionId === id);
        if (!container || !row) return;

        // Reveal within the sidebar only, without moving the conversation pane.
        const bounds = container.getBoundingClientRect();
        const rowBounds = row.getBoundingClientRect();
        if (rowBounds.top < bounds.top) container.scrollTop += rowBounds.top - bounds.top;
        else if (rowBounds.bottom > bounds.bottom) container.scrollTop += rowBounds.bottom - bounds.bottom;

        clearTimeout(forkRevealTimer);
        revealedSessionId.value = id;
        pendingForkRevealId.value = '';
        forkRevealTimer = setTimeout(() => { revealedSessionId.value = ''; }, 350);
    },
    { flush: 'post' },
);

const refreshSessionListScrollability = async () => {
    await nextTick();
    const container = scrollContainer.value;
    sessionListCanScroll.value = !!container && container.scrollHeight > container.clientHeight + 1;
};

const ensureBucketFillsViewport = async (key: string) => {
    const MAX_ITERATIONS = 20;
    for (let i = 0; i < MAX_ITERATIONS; i++) {
        await nextTick();
        await new Promise<void>((resolve) => requestAnimationFrame(() => resolve()));
        const container = scrollContainer.value;
        const bucket = sessionBuckets.value[key];
        if (!container || !bucket || !bucketHasMore(bucket) || bucket.loading) break;

        const hasOverflow = container.scrollHeight > container.clientHeight + 1;
        if (hasOverflow) break;

        const prevCount = bucket.items.length;
        await loadBucketPage(key);
        if ((sessionBuckets.value[key]?.items.length ?? 0) <= prevCount) break;
    }
};

const mouseenteBotDownr = (val: string) => {
    activeSubmenu.value = val;
}
const mouseleaveBotDown = () => {
    activeSubmenu.value = '';
}

const enterBatchMode = () => {
    batchMode.value = true
    batchSelectedIds.value = []
}

const exitBatchMode = () => {
    batchMode.value = false
    batchSelectedIds.value = []
}

const toggleBatchSelect = (id: string) => {
    const idx = batchSelectedIds.value.indexOf(id)
    if (idx > -1) {
        batchSelectedIds.value.splice(idx, 1)
    } else {
        batchSelectedIds.value.push(id)
    }
}

const toggleBatchSelectAll = (checked: boolean) => {
    batchSelectedIds.value = checked ? [...allSessionIds.value] : []
}

const handleInlineBatchDelete = () => {
    if (batchSelectedIds.value.length === 0) return
    const isDeleteAll = isAllBatchSelected.value
    const displayCount = batchDisplayCount.value
    const confirmDialog = DialogPlugin.confirm({
        header: 'Delete Conversations',
        body: isDeleteAll
            ? 'Are you sure you want to delete all conversations? This action cannot be undone.'
            : `Are you sure you want to delete the selected ${displayCount} conversation(s)? This action cannot be undone.`,
        confirmBtn: { content: 'Delete Conversations', theme: 'danger' as const },
        cancelBtn: 'Cancel',
        theme: 'warning',
        onConfirm: async () => {
            batchDeleting.value = true
            try {
                let res: any
                if (isDeleteAll) {
                    res = await deleteAllSessions()
                } else {
                    res = await batchDelSessions([...batchSelectedIds.value])
                }
                if (res && res.success === true) {
                    if (isDeleteAll) {
                        usemenuStore.clearMenuArr();
                        total.value = 0;
                        await getMessageList();
                    } else {
                        let next = sessionBuckets.value;
                        for (const id of batchSelectedIds.value) {
                            next = removeSessionFromBuckets(next, id);
                        }
                        sessionBuckets.value = next;
                        syncMenuStoreFromBuckets();
                    }
                    const currentChatId = route.params.chatid as string;
                    if (currentChatId && (isDeleteAll || batchSelectedIds.value.includes(currentChatId))) {
                        router.push('/platform/creatChat');
                    }
                    batchSelectedIds.value = []
                    MessagePlugin.success('Deleted successfully')
                    exitBatchMode()
                } else {
                    MessagePlugin.error('Delete failed, please try again later')
                }
            } catch {
                MessagePlugin.error('Delete failed, please try again later')
            }
            batchDeleting.value = false
            confirmDialog.destroy()
        },
    })
}

const handleSessionMenuClick = (data: { value: string }, item: any) => {
    if (data?.value === 'delete') {
        delCard(item);
    } else if (data?.value === 'clearMessages') {
        clearMessages(item);
    } else if (data?.value === 'batchManage') {
        enterBatchMode()
    } else if (data?.value === 'pin' || data?.value === 'unpin') {
        togglePin(item, data.value === 'pin');
    }
};


const buildSessionMenuOptions = (item: any) => {
    const options: any[] = [];
    if (item.is_pinned) {
        options.push({
            content: 'Unpin',
            value: 'unpin',
            prefixIcon: () => h(TIcon, { name: 'pin-filled' }),
        });
    } else {
        options.push({
            content: 'Pin',
            value: 'pin',
            prefixIcon: () => h(TIcon, { name: 'pin' }),
        });
    }
    options.push(
        { content: 'Rename', value: 'rename', prefixIcon: () => h(TIcon, { name: 'edit-1' }) },
        { content: 'Clear Messages', value: 'clearMessages', prefixIcon: () => h(TIcon, { name: 'clear' }) },
        { content: 'Batch Manage', value: 'batchManage', prefixIcon: () => h(TIcon, { name: 'queue' }) },
        { content: 'Delete Record', value: 'delete', theme: 'error', prefixIcon: () => h(TIcon, { name: 'delete' }) },
    );
    return options;
};

const updateSessionInBuckets = (
    sessionId: string,
    patch: Partial<{ is_pinned: boolean; pinned_at: string | null; title: string; isNoTitle?: boolean }>,
) => {
    const next: Record<string, SidebarSessionBucket> = {};
    for (const [key, bucket] of Object.entries(sessionBuckets.value)) {
        next[key] = {
            ...bucket,
            items: bucket.items.map((row) => (row.id === sessionId ? { ...row, ...patch } : row)),
        };
    }
    sessionBuckets.value = next;
    syncMenuStoreFromBuckets();
};

const renameSessionTitle = async (item: any, title: string) => {
    try {
        await renameSession(item.id, title, item.description || '');
        MessagePlugin.success('Title updated');
    } catch {
        MessagePlugin.error('Failed to update title, please try again later');
    }
};

const togglePin = (item: any, pin: boolean) => {
    if (pinningIds.value.has(item.id)) return;
    pinningIds.value.add(item.id);

    setSessionPinned(item.id, pin).catch(() => {
        MessagePlugin.error(pin ? 'Failed to pin, please try again later' : 'Failed to unpin, please try again later');
    }).finally(() => {
        pinningIds.value.delete(item.id);
    });
};

const clearMessages = (item: any) => {
    clearSession(item.id).then(() => {
        MessagePlugin.success('Messages cleared');
    }).catch(() => {
        MessagePlugin.error('Failed to clear messages, please try again later');
    });
};

const delCard = (item: any) => {
    removeSession(item.id).catch(() => MessagePlugin.error('Delete failed, please try again later!'))
}


const debounce = (fn: (...args: any[]) => void, delay: number) => {
    let timer: ReturnType<typeof setTimeout>
    return (...args: any[]) => {
        clearTimeout(timer)
        timer = setTimeout(() => fn(...args), delay)
    }
}
const mapSessionRow = (item: any) => ({
    title: item.title ? item.title : 'New Chat',
    path: `chat/${item.id}`,
    id: item.id,
    isMore: false,
    isNoTitle: item.title ? false : true,
    created_at: item.created_at,
    updated_at: item.updated_at,
    is_pinned: !!item.is_pinned,
    pinned_at: item.pinned_at || null,
    im_platform: item.im_platform || '',
    description: item.description || '',
    user_id: item.user_id || '',
    parent_session_id: item.parent_session_id || '',
});

const syncMenuStoreFromBuckets = () => {
    usemenuStore.clearMenuArr();
    const flat = flattenBucketItems(sessionBuckets.value, bucketOrder.value);
    flat.forEach((item) => usemenuStore.updatemenuArr(item));
    total.value = flat.length;
};

const menuChildToSessionRow = (item: Record<string, unknown>): SessionForGrouping & { path: string } => {
    const id = String(item.id);
    return {
        id,
        path: typeof item.path === 'string' ? item.path : `chat/${id}`,
        title: typeof item.title === 'string' ? item.title : undefined,
        is_pinned: !!item.is_pinned,
        created_at: typeof item.created_at === 'string' ? item.created_at : undefined,
        updated_at: typeof item.updated_at === 'string' ? item.updated_at : undefined,
        im_platform: typeof item.im_platform === 'string' ? item.im_platform : '',
        description: typeof item.description === 'string' ? item.description : '',
        user_id: typeof item.user_id === 'string' ? item.user_id : '',
        parent_session_id: typeof item.parent_session_id === 'string' ? item.parent_session_id : '',
    };
};

const sessionExistsInBuckets = (sessionId: string) =>
    Object.values(sessionBuckets.value).some((bucket) => bucket.items.some((row) => row.id === sessionId));

const ensureSessionInSidebar = (sessionId: string) => {
    if (!sessionId || sessionExistsInBuckets(sessionId)) return;

    const web = sessionBuckets.value.web;
    if (!web) return;

    const chatMenu = (menuArr.value as unknown as MenuItem[]).find((item) => item.path === 'creatChat');
    const fromStore = (chatMenu?.children as Record<string, unknown>[] | undefined)
        ?.find((item) => item.id === sessionId);
    if (!fromStore) return;

    sessionBuckets.value = {
        ...sessionBuckets.value,
        web: prependSessionToWebBucket(web, menuChildToSessionRow(fromStore)),
    };
    total.value = flattenBucketItems(sessionBuckets.value, bucketOrder.value).length;
};

const rebuildBucketDefinitions = () => buildBucketDefinitions(
    imPlatforms.value,
    embedChannelNames.value,
    {
        web: 'My chats',
        imPlatform: (platform) => (AGENT_EDITOR_IM_LABELS[platform] ?? ''),
        embedChannel: (name) => name,
        api: 'API sessions',
    },
    { includeAdminChannelBuckets: authStore.hasRole('admin') },
);

const probeChannelBucketCounts = async (keys: string[], token: number) => {
    const targets = keys.filter((key) => isChannelBucketKey(key));
    await Promise.all(
        targets.map(async (key) => {
            const bucket = sessionBuckets.value[key];
            if (!bucket) return;
            try {
                const res: any = await getSessionsList(1, 1, bucket.apiSource);
                if (token !== bucketRequestToken) return;
                sessionBuckets.value = {
                    ...sessionBuckets.value,
                    [key]: applyBucketCountProbe(bucket, res?.total ?? 0),
                };
            } catch {
                if (token !== bucketRequestToken) return;
                sessionBuckets.value = {
                    ...sessionBuckets.value,
                    [key]: applyBucketCountProbe(bucket, 0),
                };
            }
        }),
    );
};

const loadBucketPage = async (key: string, page?: number, token?: number) => {
    const activeToken = token ?? bucketRequestToken;
    const bucket = sessionBuckets.value[key];
    if (!bucket || bucket.loading) return;

    const nextPage = page ?? bucket.page + 1;
    sessionBuckets.value = {
        ...sessionBuckets.value,
        [key]: { ...bucket, loading: true },
    };

    try {
        const res: any = await getSessionsList(nextPage, SIDEBAR_BUCKET_PAGE_SIZE, bucket.apiSource);
        if (activeToken !== bucketRequestToken) return;
        const rows = (res?.data || []).map((item: any) => mapSessionRow(item));
        const current = sessionBuckets.value[key];
        sessionBuckets.value = {
            ...sessionBuckets.value,
            [key]: mergeBucketPage(current, rows, res?.total ?? rows.length, nextPage),
        };
        syncMenuStoreFromBuckets();
        await refreshSessionListScrollability();
    } catch {
        if (activeToken !== bucketRequestToken) return;
        const current = sessionBuckets.value[key];
        sessionBuckets.value = {
            ...sessionBuckets.value,
            [key]: { ...current, loading: false, loaded: true },
        };
    }
};

const switchSessionBucket = async (key: string) => {
    if (key === activeSessionBucketKey.value) return;
    activeSessionBucketKey.value = key;
    const bucket = sessionBuckets.value[key];
    if (bucket && !bucket.loaded && !bucket.loading) {
        await loadBucketPage(key, 1);
    }
    await ensureBucketFillsViewport(key);
    await refreshSessionListScrollability();
};

const syncActiveBucketFromChat = async (sessionId: string | undefined) => {
    if (!sessionId) return;

    let bucketKey = findSessionBucketKey(sessionBuckets.value, sessionId);
    if (!bucketKey) {
        const chatMenu = (menuArr.value as unknown as MenuItem[]).find((item) => item.path === 'creatChat');
        const fromStore = (chatMenu?.children as Record<string, unknown>[] | undefined)
            ?.find((item) => item.id === sessionId);
        if (fromStore) {
            bucketKey = originGroupKey(resolveSessionOrigin(menuChildToSessionRow(fromStore)));
        }
    }
    // On a hard refresh only the web bucket is loaded, so a session opened from
    // any other folder (IM, embed, or the admin-only API folder) isn't in any
    // bucket or the menu store. Fetch its detail and classify its origin folder
    // so the sidebar stays in sync with the chat pane instead of snapping back
    // to "my chats". Only switch when that folder is actually present.
    if (!bucketKey) {
        try {
            const res: any = await getSession(sessionId);
            const candidate = originGroupKey(resolveSessionOrigin({
                id: sessionId,
                im_platform: res?.data?.im_platform || '',
                description: res?.data?.description || '',
                user_id: res?.data?.user_id || '',
            }));
            if (sessionBuckets.value[candidate]) {
                bucketKey = candidate;
            }
        } catch {
            // Fall through: leave the default bucket active on lookup failure.
        }
    }
    if (!bucketKey || bucketKey === activeSessionBucketKey.value) return;

    activeSessionBucketKey.value = bucketKey;
    const bucket = sessionBuckets.value[bucketKey];
    if (bucket && !bucket.loaded && !bucket.loading) {
        await loadBucketPage(bucketKey, 1);
    }
};

const initSessionBuckets = async () => {
    const token = ++bucketRequestToken;
    sessionListBooting.value = true;

    const defs = rebuildBucketDefinitions();
    bucketOrder.value = defs.map((def) => def.key);
    const buckets: Record<string, SidebarSessionBucket> = {};
    for (const def of defs) {
        buckets[def.key] = createEmptyBucket(def);
    }
    sessionBuckets.value = buckets;

    const channelKeys = defs.map((def) => def.key).filter((key) => isChannelBucketKey(key));
    await Promise.all([
        loadBucketPage('web', 1, token),
        probeChannelBucketCounts(channelKeys, token),
    ]);

    if (token === bucketRequestToken) {
        sessionListBooting.value = false;
        syncMenuStoreFromBuckets();
        await ensureBucketFillsViewport('web');
        await refreshSessionListScrollability();
    }
};

const getMessageList = async () => {
    await initSessionBuckets();
};

const checkScrollBottom = async () => {
    const container = scrollContainer.value;
    const key = activeSessionBucketKey.value;
    const bucket = sessionBuckets.value[key];
    if (!container || !bucket || !bucketHasMore(bucket) || bucket.loading) return;

    const { scrollTop, scrollHeight, clientHeight } = container;
    const hasOverflow = scrollHeight > clientHeight + 1;
    if (!hasOverflow) {
        await ensureBucketFillsViewport(key);
        return;
    }

    const isNearBottom = scrollHeight - (scrollTop + clientHeight) < 100;
    if (!isNearBottom) return;

    await loadBucketPage(key);
};

const handleScroll = debounce(checkScrollBottom, 200);

async function loadCurrentKbInfo(kbId: string) {
    if (!kbId || !isInKnowledgeBase.value) {
        currentKbName.value = ''
        currentKbInfo.value = null
        return
    }
    const data = await chatResources.fetchKnowledgeBaseById(kbId)
    if (data) {
        currentKbName.value = data.name || ''
        currentKbInfo.value = data
    } else {
        currentKbInfo.value = null
    }
}

const loadSessionOriginMeta = async () => {
    try {
        const res: any = await listAllIMChannels();
        imPlatforms.value = configuredPlatforms(res?.data || []);
    } catch {
        imPlatforms.value = [];
    }
    try {
        const res: any = await listAllEmbedChannels();
        const names: Record<string, string> = {};
        for (const ch of res?.data || []) {
            if (ch?.id && ch?.name) names[ch.id] = ch.name;
        }
        embedChannelNames.value = names;
    } catch {
        embedChannelNames.value = {};
    }
};

const handleSessionMutation = (event: Event) => {
    const detail = (event as CustomEvent<SessionMutationDetail>).detail;
    if (!detail?.sessionId) return;
    if (detail.removed || detail.messagesCleared) sessionActivity.update(detail.sessionId, false);
    if (detail.patch) {
        updateSessionInBuckets(detail.sessionId, {
            ...detail.patch,
            ...(detail.patch.title ? { isNoTitle: false } : {}),
        });
    }
    if (detail.removed) {
        sessionBuckets.value = removeSessionFromBuckets(sessionBuckets.value, detail.sessionId);
        syncMenuStoreFromBuckets();
        if (detail.sessionId === route.params.chatid) {
            router.push('/platform/creatChat');
        }
    }
};

onMounted(async () => {
    sessionActivityTimer = setInterval(() => { void sessionActivity.refresh(); }, 5000);
    const routeName = typeof route.name === 'string' ? route.name : (route.name ? String(route.name) : '')
    currentpath.value = routeName;
    if (route.params.chatid) {
        currentSecondpath.value = `chat/${route.params.chatid}`;
    }

    window.addEventListener(SESSION_MUTATION_EVENT, handleSessionMutation);

    isLiteEdition.value = authStore.isLiteMode
    getSystemInfo().then(res => {
        if (res.data?.edition === 'lite') {
            isLiteEdition.value = true
            authStore.setLiteMode(true)
        }
    }).catch(() => { })

    await loadCurrentKbInfo((route.params as any)?.kbId as string)

    await loadSessionOriginMeta();
    await getMessageList();
    const initialChatId = route.params.chatid as string | undefined;
    if (initialChatId) {
        ensureSessionInSidebar(initialChatId);
        await syncActiveBucketFromChat(initialChatId);
    }
    if (deploymentCapabilities.isSupported('organizations') && orgStore.organizations.length === 0) {
        orgStore.fetchOrganizations();
    }
});

onUnmounted(() => {
    clearInterval(sessionActivityTimer);
    clearTimeout(forkRevealTimer);
    sessionActivity.clear();
    window.removeEventListener(SESSION_MUTATION_EVENT, handleSessionMutation);
});

watch([() => route.name, () => route.params], (newvalue, oldvalue) => {
    const nameStr = typeof newvalue[0] === 'string' ? (newvalue[0] as string) : (newvalue[0] ? String(newvalue[0]) : '')
    currentpath.value = nameStr;
    if (newvalue[1].chatid) {
        currentSecondpath.value = `chat/${newvalue[1].chatid}`;
    } else {
        currentSecondpath.value = "";
    }

    const newChatId = (newvalue[1] as any)?.chatid as string | undefined;
    if (nameStr === 'chat' && newChatId) {
        ensureSessionInSidebar(newChatId);
        void syncActiveBucketFromChat(newChatId);
    }

    getIcon(nameStr);

    if (newvalue[1].kbId !== oldvalue?.[1]?.kbId) {
        loadCurrentKbInfo((newvalue[1] as any)?.kbId as string);
    }
});
let knowledgeIcon = ref('zhishiku-green.svg');
let prefixIcon = ref('prefixIcon.svg');
let logoutIcon = ref('logout.svg');
let settingIcon = ref('setting.svg');
let agentIcon = ref('agent.svg');
let artifactIcon = ref('artifact.svg');
let organizationIcon = ref('organization.svg');
let pathPrefix = ref(route.name)
const getIcon = (path: string) => {
    const kbActiveState = getIconActiveState('knowledge-bases');
    const creatChatActiveState = getIconActiveState('creatChat');
    const settingsActiveState = getIconActiveState('settings');
    const agentsActiveState = route.name === 'agentList';
    const artifactsActiveState = route.name === 'artifactLibrary';
    const organizationsActiveState = route.name === 'organizationList';

    knowledgeIcon.value = kbActiveState.isKbActive ? 'zhishiku-green.svg' : 'zhishiku.svg';

    agentIcon.value = agentsActiveState ? 'agent-green.svg' : 'agent.svg';

    artifactIcon.value = artifactsActiveState ? 'artifact-green.svg' : 'artifact.svg';

    organizationIcon.value = organizationsActiveState ? 'organization-green.svg' : 'organization.svg';

    prefixIcon.value = creatChatActiveState.isCreatChatActive ? 'prefixIcon-green.svg' : 'prefixIcon.svg';

    settingIcon.value = settingsActiveState.isSettingsActive ? 'setting-green.svg' : 'setting.svg';

    logoutIcon.value = 'logout.svg';
}
getIcon(typeof route.name === 'string' ? route.name as string : (route.name ? String(route.name) : ''))
const handleMenuClick = async (path: string) => {
    if (path === 'knowledge-bases') {
        const kbId = await getCurrentKbId()
        if (kbId) {
            router.push(`/platform/knowledge-bases/${kbId}`)
        } else {
            router.push('/platform/knowledge-bases')
        }
    } else if (path === 'agents') {
        router.push('/platform/agents')
    } else if (path === 'organizations') {
        router.push('/platform/organizations')
    } else if (path === 'settings') {
        uiStore.openSettings()
        router.push('/platform/settings')
    } else {
        gotopage(path)
    }
}

const handleLogout = () => {
    gotopage('logout')
}

const getCurrentKbId = async (): Promise<string | null> => {
    const kbId = (route.params as any)?.kbId as string
    if (isInKnowledgeBase.value && kbId) {
        return kbId
    }
    return null
}

const gotopage = async (path: string) => {
    pathPrefix.value = path;
    if (path === 'logout') {
        try {
            await logoutApi();
        } catch (error) {
            console.error('Logout API call failed:', error);
        }
        authStore.logout();
        MessagePlugin.success('Logged out successfully');
        router.push('/login');
        return;
    } else {
        if (path === 'creatChat') {
            if (isInKnowledgeBase.value) {
                router.push('/platform/creatChat')
            } else {
                router.push(`/platform/creatChat`)
            }
        } else {
            router.push(`/platform/${path}`);
        }
    }
    getIcon(path)
}

const getImgSrc = (url: string) => {
    return new URL(`/src/assets/img/${url}`, import.meta.url).href;
}

const mouseenteMenu = (path: string) => {
}
const mouseleaveMenu = (path: string) => {
}

let sidebarResizeStartWidth = 0
const startSidebarResize = () => {
    sidebarResizeStartWidth = uiStore.sidebarDisplayWidth
    uiStore.sidebarResizing = true
}
const resizeSidebar = (delta: number, keyboard: boolean) => {
    if (keyboard && uiStore.sidebarCollapsed && delta > 0) {
        uiStore.expandSidebar()
    } else if (keyboard && uiStore.sidebarWidth === SIDEBAR_MIN_WIDTH && delta < 0) {
        uiStore.collapseSidebar()
    } else {
        uiStore.resizeSidebar(sidebarResizeStartWidth + delta)
    }
}


</script>
<style lang="less" scoped>
.aside_box {
    --sidebar-inset-x: 14px;
    --sidebar-icon-size: 18px;
    --sidebar-channel-icon: 14px;
    --sidebar-icon-gap: 8px;
    --sidebar-text-inset: calc(var(--sidebar-inset-x) + var(--sidebar-icon-size) + var(--sidebar-icon-gap)); // 40px

    min-width: 0;
    width: var(--sidebar-width, 260px);
    flex-shrink: 0;
    padding: 8px 6px 6px;
    background: var(--td-bg-color-sidebar);
    box-sizing: border-box;
    /* Avoid 100vh because <html> carries a `zoom` multiplier for font-size
       control; 100vh is evaluated against the unscaled viewport and then
       scaled, so at "large" the sidebar would extend past the window. The
       ancestor chain (html/body/#app/.main) is already height: 100%. */
    height: 100%;
    overflow: visible;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--td-component-stroke);
    box-shadow: 1px 0 0 rgba(0, 0, 0, 0.02);
    transition: width 0.25s ease, min-width 0.25s ease;
    position: relative;

    html.wails-desktop & {
        padding-top: 30px;
    }

    &--resizing {
        transition: none;
    }

    &--collapsed {
        min-width: 60px;
        width: 60px;
        padding: 8px 3px 6px;
        overflow: visible;

        .menu_item {
            justify-content: center;
            padding: 9px 0;

            .menu_item-box {
                justify-content: center;
                width: auto;
            }

            .menu_icon {
                margin-right: 0;
            }
        }

        .menu_bottom {
            align-items: center;
        }

        .menu_top {
            margin-right: 0;
            padding-right: 0;
        }
    }

    .logo_row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        height: 50px;
        flex-shrink: 0;
        padding: 0 10px 0 var(--sidebar-inset-x);
    }

    .sidebar-toggle {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 18px;
        height: 18px;
        flex-shrink: 0;
        cursor: pointer;
        color: var(--td-text-color-secondary);
        border-radius: var(--app-radius-xs);
        transition: background-color var(--app-motion-base) ease;
        box-sizing: border-box;

        &:hover {
            background: var(--td-bg-color-container-hover);
            color: var(--td-text-color-primary);
        }
    }

    .logo_box {
        display: flex;
        align-items: center;
        flex: 1;
        min-width: 0;
        overflow: hidden;

        .logo {
            width: 150px;
            height: auto;
        }

        .logo--dark {
            display: none;
        }

        .lite-badge {
            margin-left: 2px;
            align-self: flex-start;
            margin-top: 2px;
            font-size: 9px;
            font-weight: 600;
            color: var(--td-text-color-placeholder);
            user-select: none;
            white-space: nowrap;
        }
    }

    .menu_top {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow-y: auto;
        overflow-x: hidden;
        min-height: 0;
        margin-right: -4px;
        padding-right: 4px;

        scrollbar-width: thin;
        scrollbar-color: transparent transparent;
        transition: scrollbar-color var(--app-motion-base) ease;

        &::-webkit-scrollbar {
            width: 6px;
        }

        &::-webkit-scrollbar-track {
            background: transparent;
        }

        &::-webkit-scrollbar-thumb {
            background-color: transparent;
            border-radius: var(--app-radius-sm);
            transition: background-color var(--app-motion-base) ease;
        }

        &:hover {
            scrollbar-color: var(--td-scrollbar-color) transparent;

            &::-webkit-scrollbar-thumb {
                background-color: var(--td-scrollbar-color);
            }
        }

        &::-webkit-scrollbar-thumb:hover {
            background-color: var(--td-scrollbar-hover-color);
        }
    }

    .menu_bottom {
        flex-shrink: 0;
        display: flex;
        flex-direction: column;
    }

    .menu_box {
        display: flex;
        flex-direction: column;

        &--sticky {
            position: sticky;
            top: 0;
            z-index: 2;
            background: var(--td-bg-color-sidebar);
        }
    }


    .active-upload {
        color: var(--td-brand-color);
    }

    .menu_item_active {
        border-radius: var(--app-radius-xs);
        background: var(--td-bg-color-secondarycontainer) !important;

        .menu_icon,
        .menu_title {
            color: var(--td-brand-color) !important;
        }
    }

    .menu_item_c_active {

        .menu_icon,
        .menu_title {
            color: var(--td-text-color-primary);
        }
    }

    .menu_item {
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: space-between;
        height: 38px;
        padding: 8px 10px 8px var(--sidebar-inset-x);
        box-sizing: border-box;
        margin-bottom: 2px;
        border-radius: var(--app-radius-xs);
        transition: background-color var(--app-motion-base) ease;

        .menu_item-box {
            display: flex;
            align-items: center;
        }

        &:hover {
            border-radius: var(--app-radius-xs);
            background: var(--td-bg-color-container-hover);

            .menu_icon,
            .menu_title {
                color: var(--td-text-color-primary);
            }
        }
    }

    .menu_icon {
        display: flex;
        flex: 0 0 var(--sidebar-icon-size);
        width: var(--sidebar-icon-size);
        margin-right: var(--sidebar-icon-gap);
        color: var(--td-text-color-secondary);

        .icon {
            width: 18px;
            height: 18px;
            overflow: hidden;
        }
    }

    .menu_title {
        color: var(--td-text-color-primary);
        text-overflow: ellipsis;
        font-family: var(--app-font-family);
        font-size: var(--app-text-base);
        font-style: normal;
        font-weight: 600;
        line-height: 20px;
        overflow: hidden;
        white-space: nowrap;
        max-width: 120px;
        flex: 1;
    }

    .submenu {
        position: relative;
        font-family: var(--app-font-family);
        font-size: var(--app-text-base);
        font-style: normal;
        min-width: 0;
        padding-top: 3px;
    }

    :deep(.submenu_pin_icon) {
        color: inherit;
        font-size: var(--app-text-sm);
        margin-right: 4px;
        vertical-align: middle;
        flex-shrink: 0;
    }

    .submenu_source_icon {
        width: 14px;
        height: 14px;
        margin-right: 0px;
        vertical-align: middle;
        object-fit: contain;
        flex-shrink: 0;
        filter: grayscale(1);
        opacity: 0.55;
        transition: filter var(--app-motion-fast) ease, opacity var(--app-motion-fast) ease;
    }

    :deep(.submenu_item:hover .submenu_source_icon),
    :deep(.submenu_item_active .submenu_source_icon) {
        filter: none;
        opacity: 1;
    }

    .session-list-row {
        display: flex;
        align-items: center;
        gap: var(--sidebar-icon-gap);
        padding: 0 10px 0 var(--sidebar-inset-x);
        min-width: 0;
        box-sizing: border-box;
    }

    .session-list-row__icon {
        flex: 0 0 var(--sidebar-icon-size);
        width: var(--sidebar-icon-size);
        height: var(--sidebar-icon-size);
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }

    .session-list-row__body {
        flex: 1 1 auto;
        min-width: 0;
        overflow: hidden;
    }

    .session-list-row--flat {
        padding-left: var(--sidebar-inset-x);
        gap: 0;
    }

    .session-list-loading {
        display: flex;
        align-items: center;
        min-height: 26px;
        color: var(--td-text-color-placeholder);
    }

    .timeline_header {
        font-family: var(--app-font-family);
        font-size: var(--app-text-xs);
        font-weight: 600;
        color: var(--td-text-color-disabled);
        padding-top: 4px;
        padding-bottom: 1px;
        margin-top: 0;
        line-height: 16px;
        user-select: none;
    }

    .timeline_header-label {
        white-space: nowrap;
    }

    // Stable filter control: always mounted and absolutely pinned to the list's
    // never jumps when switching session type reloads a bucket. It overlays the
    // empty right side of the first header row, so it needs no reserved height.
    .session-list-scope-header {
        position: absolute;
        top: 4px;
        right: 10px;
        z-index: 2;
        display: flex;
        justify-content: flex-end;
        max-width: calc(100% - var(--sidebar-inset-x) - 10px);

        :deep(.session-source-filter--inline) {
            flex: 0 1 auto;
            min-width: 0;
            max-width: 100%;
            opacity: 0;
            transition: opacity var(--app-motion-fast) ease;
        }
    }

    .submenu:hover .session-list-scope-header :deep(.session-source-filter--inline),
    .session-list-scope-header:hover :deep(.session-source-filter--inline),
    .session-list-scope-header:focus-within :deep(.session-source-filter--inline),
    .session-list-scope-header :deep(.session-source-filter--inline.session-source-filter--emphasized) {
        opacity: 1;
    }

    .submenu_item_p {
        padding: 0;
        box-sizing: border-box;
        min-width: 0;
        overflow: hidden;

        &.session-chat-row .session-list-row {
            min-height: 30px;
            padding-right: 6px;
            border-radius: var(--app-radius-sm);
            transition: background var(--app-motion-fast) ease, color var(--app-motion-fast) ease;
        }

        &.session-chat-row--revealed {
            animation: session-fork-enter 280ms ease-out both;
        }

        &.session-chat-row:hover .session-list-row {
            background: var(--td-bg-color-container-hover);

            :deep(.menu-more) {
                color: var(--td-text-color-primary);
            }

        }

        &.session-chat-row--active .session-list-row {
            background: var(--td-bg-color-container-hover);

            :deep(.submenu_item) {
                color: var(--td-brand-color);
            }

            :deep(.menu-more) {
                color: var(--td-text-color-primary);
            }
        }

        &.session-chat-row--selected .session-list-row {
            background: color-mix(in srgb, var(--td-brand-color) 5%, transparent);
        }
    }

    :deep(.submenu_item) {
        cursor: pointer;
        display: flex;
        align-items: center;
        color: var(--td-text-color-primary);
        font-weight: 400;
        font-size: var(--app-text-base);
        line-height: 20px;
        height: 100%;
        width: 100%;
        padding: 6px 0;
        position: relative;
        min-width: 0;
        background: transparent;

        .submenu_title {
            display: flex;
            align-items: center;
            flex: 1 1 auto;
            min-width: 0;
            overflow: hidden;
        }

        .session-running-indicator {
            flex: 0 0 16px;
            flex-shrink: 0;
        }

        .submenu_title-text {
            flex: 1 1 auto;
            min-width: 0;
            overflow: hidden;
            white-space: nowrap;
            text-overflow: ellipsis;
        }

        .menu-more-wrap {
            transition: opacity var(--app-motion-base) ease;
            flex-shrink: 0;
        }

        .menu-more {
            display: inline-block;
            font-weight: bold;
            color: var(--td-brand-color);
        }

        .submenu_title--batch {
            margin-left: 4px;
        }

        &.submenu_item_batch {
            padding-left: 0;
        }
    }

    :deep(.submenu_item_batch) {
        cursor: pointer;
        user-select: none;
    }

    .batch-checkbox {
        flex-shrink: 0;
    }

}

.batch-inline-footer {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 12px;
    border-top: 1px solid var(--td-component-stroke);
    background: var(--td-bg-color-container);

    .batch-footer-left {
        display: flex;
        align-items: center;
        font-size: var(--app-text-md);
        color: var(--td-text-color-placeholder);
    }

    .batch-footer-right {
        display: flex;
        align-items: center;
        gap: 6px;
    }
}

.menu_item-box {
    display: flex;
    align-items: center;
    width: 100%;
    position: relative;
}

/* Empty state when there are no sessions. */
.submenu_empty {
    padding: 24px 14px;
    text-align: center;
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
    user-select: none;
}

.logo_actions {
    display: flex;
    align-items: center;
    gap: 4px;
    flex-shrink: 0;
}

.header-icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 26px;
    height: 26px;
    flex-shrink: 0;
    cursor: pointer;
    border-radius: var(--app-radius-sm);
    color: var(--td-text-color-secondary);
    transition: background-color var(--app-motion-base) ease;
    box-sizing: border-box;

    &:hover {
        background: var(--td-bg-color-container-hover);
    }

    .header-icon-img {
        width: 18px;
        height: 18px;
        display: block;
    }
}

.cmdk-tip {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    white-space: nowrap;

    .cmdk-tip-label {
        font-size: var(--app-text-md);
    }

    .cmdk-tip-keys {
        font-size: var(--app-text-md);
        opacity: 0.6;
        letter-spacing: 0.5px;
    }
}

.menu-pending-badge {
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    margin-left: 6px;
    border-radius: 9px;
    background: rgba(250, 173, 20, 0.2);
    color: var(--td-warning-color);
    font-size: var(--app-text-sm);
    font-weight: 600;
    line-height: 18px;
    text-align: center;
    flex-shrink: 0;
}

.menu_box {
    position: relative;
}

@keyframes session-fork-enter {
    from { opacity: 0; transform: translateX(-10px); }
    to { opacity: 1; transform: translateX(0); }
}

@media (prefers-reduced-motion: reduce) {
    .aside_box .submenu_item_p.session-chat-row--revealed {
        animation: none;
    }
}
</style>
<style lang="less">
// Dark mode: swap to the light-ink logo variant
html[theme-mode="dark"] .aside_box .logo_box .logo--light {
    display: none;
}

html[theme-mode="dark"] .aside_box .logo_box .logo--dark {
    display: block;
}

html[theme-mode="dark"] .aside_box .menu_top:hover {
    scrollbar-color: rgba(255, 255, 255, 0.22) transparent;
}

html[theme-mode="dark"] .aside_box .menu_top:hover::-webkit-scrollbar-thumb {
    background-color: rgba(255, 255, 255, 0.22);
}

html[theme-mode="dark"] .aside_box .menu_top::-webkit-scrollbar-thumb:hover {
    background-color: rgba(255, 255, 255, 0.38);
}

// Dark mode: invert the top search icon button image to match text color
html[theme-mode="dark"] .aside_box .header-icon-img {
    filter: invert(1);
    opacity: 0.55;
}

html[theme-mode="dark"] .aside_box .header-icon-btn:hover .header-icon-img {
    opacity: 0.9;
}

// Dark mode: make SVG icons match text color (loaded via <img>, currentColor won't work)
html[theme-mode="dark"] .aside_box .menu_icon img.icon {
    filter: invert(1);
    opacity: 0.55;
}

// Hover state: brighter icon like text
html[theme-mode="dark"] .aside_box .menu_item:hover .menu_icon img.icon {
    opacity: 0.9;
}

// menu_item_c_active: text is primary, so icon should match
html[theme-mode="dark"] .aside_box .menu_item_c_active .menu_icon img.icon {
    opacity: 0.9;
}

// Active (green) icons should not be inverted
html[theme-mode="dark"] .aside_box .menu_item_active .menu_icon img.icon {
    filter: none;
    opacity: 1;
}


:deep(.t-popconfirm) {
    .t-popconfirm__content {
        background: var(--td-bg-color-container);
        border: 1px solid var(--td-component-stroke);
        border-radius: var(--app-radius-sm);
        box-shadow: var(--td-shadow-3);
        padding: 12px 16px;
        font-size: var(--app-text-base);
        color: var(--td-text-color-primary);
        max-width: 200px;
    }

    .t-popconfirm__arrow {
        border-bottom-color: var(--td-component-stroke);
    }

    .t-popconfirm__arrow::after {
        border-bottom-color: var(--td-bg-color-container);
    }

    .t-popconfirm__buttons {
        margin-top: 8px;
        display: flex;
        justify-content: flex-end;
        gap: 8px;
    }

    .t-button--variant-outline {
        border-color: var(--td-component-border);
        color: var(--td-text-color-secondary);
    }

    .t-button--theme-danger {
        background-color: var(--td-error-color);
        border-color: var(--td-error-color);
    }

    .t-button--theme-danger:hover {
        background-color: var(--td-error-color);
        border-color: var(--td-error-color);
    }
}
</style>
