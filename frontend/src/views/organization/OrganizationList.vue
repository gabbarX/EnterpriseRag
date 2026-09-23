<template>
  <div class="org-list-container">
    <div class="org-list-content">
      <div class="header" style="--wails-draggable: drag">
        <div class="header-title" style="--wails-draggable: drag">
          <div class="title-row" style="--wails-draggable: drag">
            <h2 style="--wails-draggable: drag">
              <ResourceIcon type="organization" :size="24" />
              {{ 'Shared Spaces' }}
            </h2>
            <div class="header-actions" style="--wails-draggable: no-drag">
              <t-tooltip :content="canManageOrg ? 'Join Shared Space' : noPermissionTip" placement="bottom">
                <t-button variant="text" theme="default" size="small" class="header-action-btn"
                  style="--wails-draggable: no-drag" :disabled="!canManageOrg" @click="handleJoinOrganization">
                  <template #icon><t-icon name="enter" size="16px" /></template>
                {{ 'Join Shared Space' }}
                </t-button>
              </t-tooltip>
              <t-tooltip :content="canManageOrg ? 'Create Shared Space' : noPermissionTip" placement="bottom">
                <t-button variant="text" theme="default" size="small" class="header-action-btn"
                  style="--wails-draggable: no-drag" :disabled="!canManageOrg" @click="handleCreateOrganization">
                  <template #icon><img src="@/assets/img/organization-green.svg" class="org-create-icon" alt=""
                      aria-hidden="true" /></template>
                {{ 'Create Shared Space' }}
                </t-button>
              </t-tooltip>
            </div>
          </div>
          <p class="header-subtitle" style="--wails-draggable: drag">{{ 'Create or join shared spaces so multiple workspaces can collaborate and share knowledge bases and agents' }}</p>
        </div>
      </div>
      <ResourceListToolbar mode="organization" :model-value="spaceSelection" @update:model-value="value => spaceSelection = value === 'created' || value === 'joined' ? value : 'all'" v-model:query="keyword" :count-all="organizations.length"
      :count-created="createdCount" :count-joined="joinedCount" />
      <div class="org-list-main">
        <EmptyState v-if="keyword.trim() && !loading && filteredOrganizations.length === 0" icon="search"
          :title="'No results'">
          <t-button variant="outline" @click="keyword = ''">{{ 'Clear' }}</t-button>
        </EmptyState>
        <div v-if="loading && filteredOrganizations.length === 0" class="org-card-wrap">
          <div v-for="n in 4" :key="'skel-' + n" class="org-card org-card-skeleton is-skeleton">
            <div class="card-header">
              <t-skeleton animation="gradient"
                :row-col="[[{ width: '36px', height: '36px', type: 'circle' }, { width: '50%', height: '20px' }]]" />
            </div>
            <div style="flex:1;margin-top:12px">
              <t-skeleton animation="gradient"
                :row-col="[{ width: '100%', height: '14px' }, { width: '70%', height: '14px' }]" />
            </div>
            <div style="margin-top:auto">
              <t-skeleton animation="gradient"
                :row-col="[[{ width: '60px', height: '22px', type: 'rect' }, { width: '60px', height: '22px', type: 'rect' }]]" />
            </div>
          </div>
        </div>

        <div v-if="filteredOrganizations.length > 0" class="org-card-wrap">
          <template v-for="(org, index) in filteredOrganizations" :key="org.id">
            <div v-if="spaceSelection === 'all' && org.is_owner && index === 0" class="org-section-header"
              role="button" tabindex="0" :aria-expanded="!isOrgSectionCollapsed('created')" @click="toggleOrgSection('created')"
              @keydown.enter.prevent="toggleOrgSection('created')"
              @keydown.space.prevent="toggleOrgSection('created')">
              <t-icon name="user" size="14px" />
              <span>{{ 'Created by me' }}</span>
              <span class="org-section-count">{{ orgSectionCounts.created }}</span>
              <t-icon class="org-section-toggle"
                :name="isOrgSectionCollapsed('created') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <div v-if="spaceSelection === 'all' && !org.is_owner
              && (index === 0 || filteredOrganizations[index - 1].is_owner)" class="org-section-header" role="button"
              tabindex="0" :aria-expanded="!isOrgSectionCollapsed('joined')" @click="toggleOrgSection('joined')"
              @keydown.enter.prevent="toggleOrgSection('joined')"
              @keydown.space.prevent="toggleOrgSection('joined')">
              <t-icon name="usergroup" size="14px" />
              <span>{{ 'Joined' }}</span>
              <span class="org-section-count">{{ orgSectionCounts.joined }}</span>
              <t-icon class="org-section-toggle"
                :name="isOrgSectionCollapsed('joined') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <div v-show="!isOrgRowHidden(org)" class="org-card"
            :class="{ 'joined-org': !org.is_owner }" role="link" tabindex="0" @keydown.enter.self.prevent="handleCardClick(org)" @keydown.space.self.prevent="handleCardClick(org)" @click="handleCardClick(org)">

            <div class="card-header">
              <div class="card-header-left">
                <div class="org-avatar">
                  <SpaceAvatar :name="org.name" :avatar="org.avatar" size="small" />
                </div>
                <div class="card-title-block">
                  <span class="card-title" :title="org.name">{{ org.name }}</span>
                </div>
              </div>
              <t-popup v-model="organizationMenuVisibility[org.id]" overlayClassName="card-more-popup"
                :on-visible-change="(visible: boolean) => onVisibleChange(visible, org)" trigger="click"
                destroy-on-close placement="bottom-right">
                <button type="button" :aria-label="'Expand'" class="more-wrap" @click.stop :class="{ 'active-more': organizationMenuVisibility[org.id] }">
                  <img class="more-icon" src="@/assets/img/more.png" alt="" />
                </button>
                <template #content>
                  <div class="popup-menu" @click.stop>
                    <div class="popup-menu-item" @click.stop="handleSettings(org)">
                      <t-icon class="menu-icon" name="setting" />
                      <span>{{ 'Shared Space Settings' }}</span>
                    </div>
                    <div v-if="!org.is_owner" class="popup-menu-item delete" @click.stop="handleLeave(org)">
                      <t-icon class="menu-icon" name="logout" />
                      <span>{{ 'Leave Shared Space' }}</span>
                    </div>
                    <div v-if="org.is_owner && canManageOrg" class="popup-menu-item delete"
                      @click.stop="handleDelete(org)">
                      <t-icon class="menu-icon" name="delete" />
                      <span>{{ 'Delete' }}</span>
                    </div>
                  </div>
                </template>
              </t-popup>
            </div>

            <div class="card-content">
              <div class="card-description" :title="org.description || 'No description'">
                  {{ org.description || 'No description' }}
              </div>
            </div>

            <div class="card-bottom">
              <div class="bottom-left">
                <div class="feature-badges">
                  <t-tooltip :content="'Member count'" placement="top">
                    <div class="feature-badge stat-member">
                      <t-icon name="user" size="14px" />
                      <span class="badge-count">{{ org.member_count || 0 }}</span>
                    </div>
                  </t-tooltip>
                  <t-tooltip :content="'Knowledge Bases'" placement="top">
                    <div class="feature-badge stat-kb">
                      <t-icon name="folder" size="14px" />
                      <span class="badge-count">{{ org.share_count ?? 0 }}</span>
                    </div>
                  </t-tooltip>
                  <t-tooltip :content="'Agents'" placement="top">
                    <div class="feature-badge stat-agent">
                      <img src="@/assets/img/agent-green.svg" class="stat-agent-icon" alt="" aria-hidden="true" />
                      <span class="badge-count">{{ org.agent_share_count ?? 0 }}</span>
                    </div>
                  </t-tooltip>
                </div>
                <t-tooltip v-if="(org.pending_join_request_count ?? 0) > 0"
                  :content="'Pending join requests to review'" placement="top">
                  <span class="pending-requests-badge">{{ org.pending_join_request_count }} {{
                    'Pending' }}</span>
                </t-tooltip>
              </div>
              <div v-if="showOrgRelationTag(org)" class="bottom-right">
                <div class="relation-role-tag" :class="org.is_owner ? 'owner' : (org.my_role || '')">
                  <t-icon :name="org.is_owner ? 'usergroup-add' : 'usergroup'" size="14px" />
                  <span>{{ org.is_owner ? 'Creator' : (org.my_role ?
                    (ORGANIZATION_ROLE_LABELS[org.my_role] ?? '') :
                    'Joined') }}</span>
                </div>
              </div>
            </div>
          </div>
          </template>
        </div>

        <EmptyState v-if="!keyword.trim() && !loading && filteredOrganizations.length === 0" :title="emptyStateTitle"
          :description="emptyStateDesc">
          <template #icon><ResourceIcon type="organization" :size="32" /></template>
          <t-tooltip :content="noPermissionTip" placement="top" :disabled="canManageOrg">
            <t-button theme="default" variant="outline" class="org-join-btn" :disabled="!canManageOrg"
              @click="handleJoinOrganization">
              <template #icon><t-icon name="enter" /></template>
              {{ 'Join Shared Space' }}
            </t-button>
          </t-tooltip>
          <t-tooltip :content="noPermissionTip" placement="top" :disabled="canManageOrg">
            <t-button theme="primary" class="org-create-btn" :disabled="!canManageOrg" @click="handleCreateOrganization">
              <template #icon><img src="@/assets/img/organization-green.svg" class="org-create-icon" alt=""
                  aria-hidden="true" /></template>
              {{ 'Create Shared Space' }}
            </t-button>
          </t-tooltip>
        </EmptyState>
      </div>
    </div>

    <OrganizationSettingsModal :visible="showSettingsModal" :org-id="settingsOrgId" :mode="settingsMode"
      @update:visible="showSettingsModal = $event" />

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="showInvitePreview" class="invite-preview-overlay" @click.self="closeInvitePreview">
          <div class="invite-preview-modal" :class="{
            'is-wide': !invitePreviewData && !invitePreviewLoading && joinStep === 'search'
          }">
            <div class="invite-preview-header">
              <button v-if="invitePreviewData && !inviteCode" class="invite-preview-back" @click="backFromPreview"
                :aria-label="'Back to search'">
                <t-icon name="chevron-left" />
              </button>
              <h2 class="invite-preview-title">{{ invitePreviewData ? 'Join Shared Space' :
                'Join Shared Space' }}</h2>
              <button class="invite-preview-close" @click="closeInvitePreview" :aria-label="'Close'">
                <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                  <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
                </svg>
              </button>
            </div>

            <div class="invite-preview-body-wrap" :style="inviteBodyWrapStyle">
              <div ref="inviteBodyInnerRef" class="invite-body-inner">
                <div v-if="!invitePreviewLoading && !invitePreviewData"
                  class="invite-preview-body invite-preview-input">
                  <div class="join-mode-pills">
                    <button type="button" :class="['join-mode-pill', { active: joinStep === 'invite' }]"
                      @click="joinStep = 'invite'">
                      {{ 'Enter invite code' }}
                    </button>
                    <button type="button" :class="['join-mode-pill', { active: joinStep === 'search' }]"
                      @click="handleSearchTabClick">
                      {{ 'Search shared spaces' }}
                    </button>
                  </div>

                  <div ref="tabContentWrapperRef" class="join-tab-content-wrapper">
                    <div v-if="joinStep === 'invite'" class="join-tab-content">
                      <template v-if="!invitePreviewError">
                        <div class="join-form-item">
                          <label class="join-form-label">{{ 'Invite Code' }}</label>
                          <p class="join-form-desc">{{ 'Enter the invite code (or paste from an invite link) to view the shared space and join' }}</p>
                          <t-input v-model="joinInputCode" :placeholder="'Enter invite code'"
                            size="medium" :maxlength="32" clearable @keyup.enter="doPreviewFromInput" />
                          <p class="join-form-tip">{{ 'The invite code is generated by shared space admins, please ask them for it' }}</p>
                        </div>
                      </template>
                      <template v-else>
                        <div class="invite-preview-error-inline">
                          <t-icon name="error-circle" size="20px" />
                          <span>{{ invitePreviewError }}</span>
                        </div>
                        <div class="join-form-item">
                          <label class="join-form-label">{{ 'Invite Code' }}</label>
                          <t-input v-model="joinInputCode" :placeholder="'Enter invite code'"
                            size="medium" :maxlength="32" clearable @keyup.enter="doPreviewFromInput" />
                        </div>
                      </template>
                      <div class="invite-preview-footer invite-preview-footer-single">
                        <t-button theme="default" variant="outline" size="medium" @click="closeInvitePreview">
                          {{ 'Cancel' }}
                        </t-button>
                        <t-button theme="primary" size="medium" :loading="invitePreviewLoading"
                          @click="doPreviewFromInput">
                          {{ 'View' }}
                        </t-button>
                      </div>
                    </div>

                    <div v-else-if="joinStep === 'search'" class="join-tab-content join-tab-search">
                      <div class="join-form-item join-form-item--compact">
                        <label class="join-form-label">{{ 'Search shared spaces' }}</label>
                        <p class="join-form-desc">{{ 'Browse or search shared spaces that are open for discovery; join without an invite code' }}</p>
                        <t-input v-model="searchQuery" :placeholder="'Search by shared space name, description or ID'"
                          size="medium" clearable @input="doSearchSearchableDebounced" @keyup.enter="doSearchSearchable">
                          <template #prefix-icon>
                            <t-icon name="search" />
                          </template>
                        </t-input>
                      </div>
                      <div class="searchable-list-wrap">
                        <t-loading :loading="searchLoading">
                          <div v-if="searchableList.length === 0 && !searchLoading" class="searchable-empty">
                            <t-empty :description="searchQuery ? 'No matching shared spaces' :
                              'No discoverable shared spaces yet, or try a search'" />
                          </div>
                          <div v-else class="searchable-list">
                            <div v-for="org in searchableList" :key="org.id" class="searchable-row"
                              :class="{ 'is-full': isOrgFull(org) }"
                              @click="!isOrgFull(org) && previewSearchableOrg(org)">
                              <div class="searchable-row-main">
                                <SpaceAvatar :name="org.name" :avatar="org.avatar" size="small" />
                                <div class="searchable-row-info">
                                  <span class="searchable-row-title" :title="org.name">{{ org.name }}</span>
                                  <span class="searchable-row-desc">{{ org.description || 'No description' }}</span>
                                </div>
                              </div>
                              <div class="searchable-row-meta">
                                <span class="searchable-meta-item">
                                  <t-icon name="user" size="12px" />
                                  <template v-if="org.member_limit > 0">{{ org.member_count }}/{{ org.member_limit }}</template>
                                  <template v-else>{{ org.member_count }}</template>
                                </span>
                                <t-tag v-if="org.require_approval" size="small" variant="light" theme="warning">
                                  {{ 'Requires approval' }}
                                </t-tag>
                                <t-tag v-if="isOrgFull(org)" size="small" variant="light">
                                  {{ 'Full' }}
                                </t-tag>
                                <t-button v-if="!isOrgFull(org)" theme="primary" variant="outline" size="small"
                                  @click.stop="previewSearchableOrg(org)">
                                  {{ 'View' }}
                                </t-button>
                              </div>
                            </div>
                          </div>
                        </t-loading>
                      </div>
                      <div class="invite-preview-footer invite-preview-footer-single">
                        <t-button theme="default" variant="outline" size="medium" @click="closeInvitePreview">
                          {{ 'Cancel' }}
                        </t-button>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Loading -->
                <div v-else-if="invitePreviewLoading" class="invite-preview-body invite-preview-loading">
                  <t-loading size="medium" />
                  <span class="invite-preview-loading-text">{{ 'Loading...' }}</span>
                </div>

                <div v-else-if="invitePreviewData" class="invite-preview-body invite-preview-body-preview">
                  <div class="preview-space-hero">
                    <div class="preview-space-avatar-wrap">
                      <SpaceAvatar :name="invitePreviewData.name" :avatar="invitePreviewData.avatar" size="large" />
                    </div>
                    <h3 class="preview-space-name">{{ invitePreviewData.name }}</h3>
                    <p class="preview-space-desc">{{ invitePreviewData.description || 'No description' }}</p>
                    <div class="feature-badges preview-space-badges">
                      <t-tooltip :content="'Member count'" placement="top">
                        <div class="feature-badge stat-member">
                          <t-icon name="user" size="14px" />
                          <span class="badge-count">{{ invitePreviewData.member_count }}</span>
                        </div>
                      </t-tooltip>
                      <t-tooltip :content="'Knowledge Bases'" placement="top">
                        <div class="feature-badge stat-kb">
                          <t-icon name="folder" size="14px" />
                          <span class="badge-count">{{ invitePreviewData.share_count }}</span>
                        </div>
                      </t-tooltip>
                      <t-tooltip :content="'Agents'" placement="top">
                        <div class="feature-badge stat-agent">
                          <img src="@/assets/img/agent-green.svg" class="stat-agent-icon" alt="" aria-hidden="true" />
                          <span class="badge-count">{{ invitePreviewData.agent_share_count ?? 0 }}</span>
                        </div>
                      </t-tooltip>
                    </div>
                    <button type="button" class="preview-space-id-chip" @click="copyPreviewSpaceId">
                      <span class="preview-space-id-label">{{ 'Shared Space ID' }}</span>
                      <code>{{ shortPreviewSpaceId }}</code>
                      <t-icon name="file-copy" size="14px" />
                    </button>
                  </div>

                  <div v-if="invitePreviewData.is_already_member" class="preview-member-status">
                    <t-icon name="check-circle" size="18px" />
                    <span>{{ 'You are already a member of this shared space' }}</span>
                  </div>

                  <div v-else class="preview-join-summary">
                    <div class="preview-info-row">
                      <span class="preview-info-label">{{ 'Join method' }}</span>
                      <t-tag size="small"
                        :theme="invitePreviewData.require_approval ? 'warning' : 'success'" variant="light">
                        {{ invitePreviewData.require_approval ? 'Requires approval' :
                          'No approval required' }}
                      </t-tag>
                    </div>
                    <p v-if="!invitePreviewData.require_approval" class="preview-info-desc">
                      {{ `Default role after joining: ${'Viewer'}` }}
                    </p>
                    <template v-else>
                      <p class="preview-info-desc preview-info-desc--warning">
                        {{ 'This shared space requires admin approval to join' }}
                      </p>
                      <div class="preview-join-fields">
                        <div class="join-form-item join-form-item--compact">
                          <label class="join-form-label">{{ 'Requested role' }}</label>
                          <t-select v-model="inviteRequestRole" size="medium"
                            :placeholder="'Select role'" :options="orgRoleOptions" />
                        </div>
                        <div class="join-form-item join-form-item--compact">
                          <label class="join-form-label">{{ 'Application note (optional)' }}</label>
                          <t-textarea v-model="inviteRequestMessage" size="medium"
                            :placeholder="'Optional: message (e.g. intro or reason to join)'" :maxlength="500"
                            :autosize="{ minRows: 2, maxRows: 4 }" />
                        </div>
                      </div>
                    </template>
                  </div>

                  <div class="invite-preview-footer">
                    <t-button theme="default" variant="outline" size="medium" @click="backFromPreview">
                      {{ !inviteCode ? 'Back to search' : 'Cancel' }}
                    </t-button>
                    <t-button v-if="!invitePreviewData.is_already_member" theme="primary" size="medium"
                      :loading="inviteJoining" @click="confirmJoinOrganization">
                      {{ invitePreviewData.require_approval ? 'Request to Join' :
                        'Join' }}
                    </t-button>
                    <t-button v-else theme="primary" size="medium" @click="viewOrganizationFromPreview">
                      {{ 'View Shared Space' }}
                    </t-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import EmptyState from '@/components/EmptyState.vue'
import ResourceIcon from '@/components/icons/ResourceIcon.vue'
import { useConfirmDelete } from '@/components/settings/useConfirmDelete'
import { useOrganizationStore } from '@/stores/organization'
import { useAuthStore } from '@/stores/auth'
import type { Organization, OrganizationPreview, SearchableOrganizationItem } from '@/api/organization'
import { previewOrganization, submitJoinRequest } from '@/api/organization'
import { copyWithToast } from '@/utils/clipboard'
import OrganizationSettingsModal from './OrganizationSettingsModal.vue'
import SpaceAvatar from '@/components/SpaceAvatar.vue'
import ResourceListToolbar from '@/components/ResourceListToolbar.vue'
import { matchesResourceQuery } from '@/utils/resourceListSearch'
import { shouldShowOrgRelationTag } from '@/utils/card-list-badge'

const ORGANIZATION_ROLE_LABELS: Record<string, string> = {
  admin: 'Admin',
  editor: 'Editor',
  viewer: 'Viewer',
}

type OrgWithUI = Organization

const route = useRoute()
const router = useRouter()
const orgStore = useOrganizationStore()
const authStore = useAuthStore()

// Every write under /api/v1/organizations (create, join, request to join, invite,
// approve, change settings) requires admin in the current workspace at the route
// layer; this flag only drives UI rendering, the real boundary is server-side.
const canManageOrg = computed(
  () => authStore.hasRole('admin') || authStore.canAccessAllTenants
)
const noPermissionTip = computed(() => 'This action requires the admin role (or higher) in the current workspace. Please contact the workspace owner.')

const orgRoleOptions = [
  { label: 'Viewer', value: 'viewer' },
  { label: 'Editor', value: 'editor' },
  { label: 'Admin', value: 'admin' },
]
const inviteRequestRole = ref<'viewer' | 'editor' | 'admin'>('viewer')
const inviteRequestMessage = ref('')

// State
const showSettingsModal = ref(false)
const settingsOrgId = ref('')
const settingsMode = ref<'create' | 'edit'>('edit')
const confirmDelete = useConfirmDelete()

const showInvitePreview = ref(false)
const invitePreviewLoading = ref(false)
const inviteJoining = ref(false)
const inviteCode = ref('')
const joinInputCode = ref('')
const invitePreviewData = ref<OrganizationPreview | null>(null)
const invitePreviewError = ref('')

const joinStep = ref<'invite' | 'search'>('invite')
const searchQuery = ref('')
const searchableList = computed(() => orgStore.searchableOrganizations)
const searchLoading = ref(false)
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null
const organizationMenuVisibility = reactive<Record<string, boolean>>({})

const tabContentWrapperRef = ref<HTMLElement | null>(null)

const inviteBodyInnerRef = ref<HTMLElement | null>(null)
const inviteBodyHeightPx = ref<number>(0)
let inviteBodyResizeObserver: ResizeObserver | null = null

const inviteBodyWrapStyle = computed(() => {
  const px = inviteBodyHeightPx.value
  if (px <= 0) return {}
  return { maxHeight: `${px}px`, minHeight: `${px}px` }
})

const shortPreviewSpaceId = computed(() => {
  const id = invitePreviewData.value?.id
  if (!id) return ''
  return id.length > 8 ? `${id.slice(0, 8)}…` : id
})

function updateInviteBodyHeight() {
  const el = inviteBodyInnerRef.value
  if (!el || !showInvitePreview.value) return
  const h = el.scrollHeight
  // Never write 0 as the height, it collapses the panel; only update on a valid height
  if (h > 0) inviteBodyHeightPx.value = h
}

function setupInviteBodyResizeObserver() {
  if (inviteBodyResizeObserver) return
  const el = inviteBodyInnerRef.value
  if (!el || !showInvitePreview.value) return
  inviteBodyResizeObserver = new ResizeObserver((entries) => {
    const entry = entries[0]
    if (!entry) return
    const h = entry.contentRect.height
    // A height of 0 can be read mid-switch and would collapse the panel
    if (h > 0 || inviteBodyHeightPx.value <= 0) inviteBodyHeightPx.value = h
  })
  inviteBodyResizeObserver.observe(el)
  inviteBodyHeightPx.value = el.scrollHeight
}

function teardownInviteBodyResizeObserver() {
  if (inviteBodyResizeObserver) {
    inviteBodyResizeObserver.disconnect()
    inviteBodyResizeObserver = null
  }
  inviteBodyHeightPx.value = 0
}

watch(
  [showInvitePreview, inviteBodyInnerRef],
  ([show, inner]) => {
    if (!show) {
      teardownInviteBodyResizeObserver()
      return
    }
    if (inner) {
      nextTick(() => {
        setupInviteBodyResizeObserver()
      })
    }
  },
  { flush: 'post' }
)

// Read the new content height after layout so the height transition stays visible
watch(
  [() => invitePreviewLoading.value, () => invitePreviewData.value],
  () => {
    if (!showInvitePreview.value || !inviteBodyInnerRef.value) return
    nextTick(() => {
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          updateInviteBodyHeight()
        })
      })
    })
  },
  { flush: 'post' }
)

const updateTabContentHeight = () => {
  if (!tabContentWrapperRef.value) return

  tabContentWrapperRef.value.style.height = 'auto'
  const naturalHeight = tabContentWrapperRef.value.scrollHeight

  tabContentWrapperRef.value.style.height = `${naturalHeight}px`
}

watch(joinStep, () => {
  if (!tabContentWrapperRef.value) return

  const currentHeight = tabContentWrapperRef.value.scrollHeight
  tabContentWrapperRef.value.style.height = `${currentHeight}px`

  requestAnimationFrame(() => {
    updateTabContentHeight()

    setTimeout(() => {
      if (tabContentWrapperRef.value) {
        tabContentWrapperRef.value.style.height = 'auto'
      }
    }, 300) // keep in sync with the CSS transition duration
  })
}, { flush: 'post' })

watch([searchableList, searchLoading], () => {
  if (joinStep.value === 'search') {
    nextTick(() => {
      updateTabContentHeight()
    })
  }
})

const handleOrganizationDialogEvent = ((event: CustomEvent<{ type: 'create' | 'join' }>) => {
  if (!canManageOrg.value) {
    MessagePlugin.warning(
      event.detail?.type === 'create'
        ? 'Your workspace role is insufficient to create a shared space'
        : 'Your workspace role is insufficient to join or request to join a shared space'
    )
    return
  }
  if (event.detail?.type === 'create') {
    settingsOrgId.value = ''
    settingsMode.value = 'create'
    showSettingsModal.value = true
  } else if (event.detail?.type === 'join') {
    joinInputCode.value = ''
    inviteCode.value = ''
    invitePreviewData.value = null
    invitePreviewError.value = ''
    invitePreviewLoading.value = false
    joinStep.value = 'invite'
    searchQuery.value = ''
    orgStore.clearSearchableOrganizations()
    showInvitePreview.value = true
  }
}) as EventListener

const spaceSelection = ref<'all' | 'created' | 'joined'>('all')
const keyword = ref('')

// Computed
const loading = computed(() => orgStore.loading)
const organizations = computed<OrgWithUI[]>(() => orgStore.organizations)

const createdCount = computed(() => organizations.value.filter(o => o.is_owner).length)
const joinedCount = computed(() => organizations.value.filter(o => !o.is_owner).length)

const unsearchedFilteredOrganizations = computed(() => {
  if (spaceSelection.value === 'created') return organizations.value.filter(o => o.is_owner)
  if (spaceSelection.value === 'joined') return organizations.value.filter(o => !o.is_owner)
  // In the "all" view put the spaces I created first and the joined ones after, so the
  // section headers land exactly on the transition - same as the KB / agent lists.
  return [...organizations.value].sort((a, b) => {
    if (a.is_owner === b.is_owner) return 0
    return a.is_owner ? -1 : 1
  })
})
const filteredOrganizations = computed(() => unsearchedFilteredOrganizations.value.filter(item => matchesResourceQuery(item, keyword.value)))

type OrgSectionKey = 'created' | 'joined'
const collapsedOrgSections = ref<Set<OrgSectionKey>>(new Set())
const isOrgSectionCollapsed = (key: OrgSectionKey) => collapsedOrgSections.value.has(key)
const toggleOrgSection = (key: OrgSectionKey) => {
  const next = new Set(collapsedOrgSections.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedOrgSections.value = next
}
const orgSectionOf = (org: { is_owner?: boolean }): OrgSectionKey => (org.is_owner ? 'created' : 'joined')
const isOrgRowHidden = (org: { is_owner?: boolean }) =>
  spaceSelection.value === 'all' && isOrgSectionCollapsed(orgSectionOf(org))
const orgSectionCounts = computed<Record<OrgSectionKey, number>>(() => {
  const c: Record<OrgSectionKey, number> = { created: 0, joined: 0 }
  filteredOrganizations.value.forEach(o => { c[orgSectionOf(o)]++ })
  return c
})

function showOrgRelationTag(org: { is_owner?: boolean; my_role?: string }): boolean {
  return shouldShowOrgRelationTag({
    spaceSelection: spaceSelection.value,
    isOwner: !!org.is_owner,
    myRole: org.my_role,
  })
}

const emptyStateTitle = computed(() => {
  if (spaceSelection.value === 'created') return 'You have not created any shared space yet'
  if (spaceSelection.value === 'joined') return 'You have not joined any shared space yet'
  return 'You have not joined any shared space yet'
})

const emptyStateDesc = computed(() => {
  if (spaceSelection.value === 'created') return 'Click "Create Shared Space" to create a new one'
  if (spaceSelection.value === 'joined') return 'Join an existing shared space with an invite code'
  return 'Create a shared space or join an existing one with an invite code'
})

// Methods
const onVisibleChange = (visible: boolean, org: OrgWithUI) => {
  if (!visible) {
    organizationMenuVisibility[org.id] = false
  }
}

function handleCreateOrganization() {
  if (!canManageOrg.value) {
    MessagePlugin.warning('Your workspace role is insufficient to create a shared space')
    return
  }
  settingsOrgId.value = ''
  settingsMode.value = 'create'
  showSettingsModal.value = true
}

function handleJoinOrganization() {
  if (!canManageOrg.value) {
    MessagePlugin.warning('Your workspace role is insufficient to join or request to join a shared space')
    return
  }
  joinInputCode.value = ''
  inviteCode.value = ''
  invitePreviewData.value = null
  invitePreviewError.value = ''
  invitePreviewLoading.value = false
  joinStep.value = 'invite'
  searchQuery.value = ''
  orgStore.clearSearchableOrganizations()
  showInvitePreview.value = true
}

function handleCardClick(org: OrgWithUI) {
  if (organizationMenuVisibility[org.id]) {
    return
  }
  settingsOrgId.value = org.id
  settingsMode.value = 'edit'
  showSettingsModal.value = true
}

function handleSettings(org: OrgWithUI) {
  organizationMenuVisibility[org.id] = false
  settingsOrgId.value = org.id
  settingsMode.value = 'edit'
  showSettingsModal.value = true
}

function handleLeave(org: OrgWithUI) {
  organizationMenuVisibility[org.id] = false
  confirmDelete({
    title: 'Leave Shared Space',
    body: `Are you sure you want to leave shared space "${org.name}"? You will no longer have access to knowledge bases shared in this shared space.`,
    confirmText: 'Leave Shared Space',
    onConfirm: async () => {
      const success = await orgStore.leave(org.id)
      if (success) {
        MessagePlugin.success('Left shared space successfully')
      } else {
        MessagePlugin.error(orgStore.error || 'Failed to leave shared space')
      }
    },
  })
}

function handleDelete(org: OrgWithUI) {
  organizationMenuVisibility[org.id] = false
  if (!canManageOrg.value) {
    MessagePlugin.warning('Your workspace role is insufficient to manage this shared space')
    return
  }
  confirmDelete({
    title: 'Delete Shared Space',
    body: `Are you sure you want to delete shared space "${org.name}"? All members will be removed. This action cannot be undone.`,
    onConfirm: async () => {
      const success = await orgStore.remove(org.id)
      if (success) {
        MessagePlugin.success('Shared space deleted')
      } else {
        MessagePlugin.error(orgStore.error || 'Failed to delete shared space')
      }
    },
  })
}

async function handleInvitePreview(code: string) {
  inviteCode.value = code
  invitePreviewLoading.value = true
  invitePreviewError.value = ''
  invitePreviewData.value = null
  showInvitePreview.value = true

  try {
    const result = await previewOrganization(code)
    if (result.success && result.data) {
      invitePreviewData.value = result.data
      if (result.data.is_already_member) {
        invitePreviewError.value = 'You are already a member of this shared space'
      }
    } else {
      invitePreviewError.value = result.message || 'Invite code is invalid or expired'
    }
  } catch (e: any) {
    invitePreviewError.value = e?.message || 'Preview failed, please try again'
  } finally {
    invitePreviewLoading.value = false
  }
}

async function confirmJoinOrganization() {
  if (!invitePreviewData.value || invitePreviewData.value.is_already_member) return
  if (!canManageOrg.value) {
    MessagePlugin.warning('Your workspace role is insufficient to join or request to join a shared space')
    return
  }

  // No invite code means this came from search, so use the search join flow
  if (!inviteCode.value && invitePreviewData.value.id) {
    await joinBySearchOrg()
    return
  }

  if (!inviteCode.value) return

  inviteJoining.value = true
  try {
    if (invitePreviewData.value.require_approval) {
      const result = await submitJoinRequest({
        invite_code: inviteCode.value,
        message: inviteRequestMessage.value?.trim() || undefined,
        role: inviteRequestRole.value,
      })
      if (result.success) {
        MessagePlugin.success('Request submitted, please wait for admin approval')
        showInvitePreview.value = false
        inviteCode.value = ''
        invitePreviewData.value = null
        // Clear the invite_code query parameter
        router.replace({ path: route.path, query: {} })
      } else {
        MessagePlugin.error(result.message || 'Failed to submit request, please try again')
      }
    } else {
      const result = await orgStore.join(inviteCode.value)
      if (result) {
        MessagePlugin.success('Successfully joined shared space!')
        showInvitePreview.value = false
        inviteCode.value = ''
        invitePreviewData.value = null
        // Clear the invite_code query parameter
        router.replace({ path: route.path, query: {} })
      } else {
        MessagePlugin.error(orgStore.error || 'Failed to join, please try again')
      }
    }
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to join, please try again')
  } finally {
    inviteJoining.value = false
  }
}

async function doPreviewFromInput() {
  const code = joinInputCode.value?.trim()
  if (!code) {
    MessagePlugin.warning('Please enter invite code')
    return
  }
  invitePreviewError.value = ''
  await handleInvitePreview(code)
}

function closeInvitePreview() {
  showInvitePreview.value = false
  inviteCode.value = ''
  joinInputCode.value = ''
  invitePreviewData.value = null
  invitePreviewError.value = ''
  joinStep.value = 'invite'
  searchQuery.value = ''
  orgStore.clearSearchableOrganizations()
  inviteRequestRole.value = 'viewer'
  inviteRequestMessage.value = ''
  router.replace({ path: route.path, query: {} })
}

function backFromPreview() {
  const fromSearch = !inviteCode.value
  invitePreviewData.value = null
  inviteRequestRole.value = 'viewer'
  inviteRequestMessage.value = ''
  if (fromSearch) {
    joinStep.value = 'search'
  }
}

function handleSearchTabClick() {
  joinStep.value = 'search'
  void doSearchSearchable()
}

async function doSearchSearchable() {
  const currentQuery = searchQuery.value.trim()

  searchLoading.value = true
  try {
    await orgStore.fetchSearchableOrganizations(currentQuery, { limit: 20 })
  } catch (e) {
    orgStore.clearSearchableOrganizations()
  } finally {
    searchLoading.value = false
  }
}

function doSearchSearchableDebounced() {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => doSearchSearchable(), 300)
}

function isOrgFull(org: SearchableOrganizationItem): boolean {
  return org.member_limit > 0 && org.member_count >= org.member_limit
}

function previewSearchableOrg(org: SearchableOrganizationItem) {
  invitePreviewData.value = {
    id: org.id,
    name: org.name,
    description: org.description,
    avatar: org.avatar,
    member_count: org.member_count,
    share_count: org.share_count,
    agent_share_count: org.agent_share_count ?? 0,
    is_already_member: org.is_already_member,
    require_approval: org.require_approval,
    created_at: '', // the search list does not return a created time
  }
  // No invite code: this space was reached through search
  inviteCode.value = ''
}

function viewOrganizationFromPreview() {
  if (!invitePreviewData.value) return
  settingsOrgId.value = invitePreviewData.value.id
  settingsMode.value = 'edit'
  showSettingsModal.value = true
}

async function copyPreviewSpaceId() {
  await copyWithToast(invitePreviewData.value?.id, 'Copied')
}

async function joinBySearchOrg() {
  if (!invitePreviewData.value || invitePreviewData.value.is_already_member) return
  if (!canManageOrg.value) {
    MessagePlugin.warning('Your workspace role is insufficient to join or request to join a shared space')
    return
  }

  inviteJoining.value = true
  try {
    const message = invitePreviewData.value.require_approval ? inviteRequestMessage.value?.trim() || undefined : undefined
    const role = invitePreviewData.value.require_approval ? inviteRequestRole.value : undefined
    const result = await orgStore.joinById(
      invitePreviewData.value.id,
      message,
      role,
      { requiresApproval: invitePreviewData.value.require_approval }
    )
    if (result.success) {
      if (invitePreviewData.value.require_approval) {
        MessagePlugin.success('Request submitted, please wait for admin approval')
      } else {
        MessagePlugin.success('Successfully joined shared space!')
      }
      showInvitePreview.value = false
      invitePreviewData.value = null
      orgStore.clearSearchableOrganizations()
      searchQuery.value = ''
      joinStep.value = 'invite'
      inviteRequestRole.value = 'viewer'
      inviteRequestMessage.value = ''
    } else {
      MessagePlugin.error(result.message || 'Failed to join, please try again')
    }
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to join, please try again')
  } finally {
    inviteJoining.value = false
  }
}

// Lifecycle
onMounted(async () => {
  void orgStore.fetchOrganizations()
  window.addEventListener('openOrganizationDialog', handleOrganizationDialogEvent)

  const code = route.query.invite_code as string
  if (code) {
    await handleInvitePreview(code)
  }

  const orgId = route.query.orgId as string
  if (orgId) {
    settingsOrgId.value = orgId
    settingsMode.value = 'edit'
    showSettingsModal.value = true
    // Clear the orgId query parameter so a refresh does not reopen the modal
    const newQuery = { ...route.query }
    delete newQuery.orgId
    router.replace({ path: route.path, query: newQuery })
  }
})

onUnmounted(() => {
  window.removeEventListener('openOrganizationDialog', handleOrganizationDialogEvent)
  teardownInviteBodyResizeObserver()
})
// A new search reveals matching rows even if their group was previously collapsed.
watch(keyword, () => { collapsedOrgSections.value = new Set() })
</script>

<style scoped lang="less">
@import (reference) '@/components/css/resource-card.less';

.org-list-container {
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
  display: flex;
}

.org-list-content { .resource-list-content(); }

.org-list-main { .resource-list-main(); }

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-shrink: 0;

  .header-title {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  h2 {
    display: flex;
    align-items: center;
    gap: 8px;
    margin: 0;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: var(--app-text-4xl);
    font-weight: 600;
    line-height: 32px;
  }
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.org-join-btn {
  border-color: color-mix(in srgb, var(--td-brand-color) 50%, transparent);
  color: var(--td-brand-color);
  font-weight: 500;
  transition: all var(--app-motion-base) ease;

  .t-icon {
    color: var(--td-brand-color);
  }

  &:hover {
    background: color-mix(in srgb, var(--td-brand-color) 8%, transparent);
    border-color: var(--td-brand-color);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }
  }
}

.org-create-btn .org-create-icon {
  width: 16px;
  height: 16px;
  filter: brightness(0) invert(1);
}

.header-subtitle {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: var(--app-text-base);
  font-weight: 400;
  line-height: 20px;
}

.header-action-btn {
  padding: 0;
  min-width: 28px;
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--td-bg-color-container) 72%, transparent);
  transition: background var(--app-motion-base), border-color var(--app-motion-base), color var(--app-motion-base);

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    border-color: var(--td-component-stroke);
    color: var(--td-text-color-primary);
  }

  :deep(.t-icon),
  :deep(.btn-icon-wrapper),
  :deep(.org-create-icon) {
    color: var(--td-brand-color);
  }

  :deep(.org-create-icon) {
    width: 16px;
    height: 16px;
  }
}

.org-card-wrap {
  .resource-card-grid();
}

.org-section-header {
  .resource-section-header();
}

.org-card {
  .resource-card();
}

.org-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.bottom-left {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.feature-badge {
  .resource-feature-badge();

  .t-icon {
    flex-shrink: 0;
  }

  .badge-count {
    line-height: 1;
  }

  &.stat-member {
    background: color-mix(in srgb, var(--td-text-color-secondary) 8%, transparent);
    color: var(--td-text-color-secondary);

    .t-icon {
      color: var(--td-text-color-secondary);
    }

    &:hover {
      background: color-mix(in srgb, var(--td-text-color-secondary) 12%, transparent);
    }
  }

  &.stat-kb {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);

    .t-icon {
      color: var(--td-text-color-secondary);
    }

    &:hover {
      background: var(--td-bg-color-container-hover);
    }
  }

  &.stat-agent {
    background: color-mix(in srgb, var(--app-accent-purple) 8%, transparent);
    color: var(--td-brand-color);

    .stat-agent-icon {
      width: 14px;
      height: 14px;
      flex-shrink: 0;
      filter: brightness(0) saturate(100%) invert(48%) sepia(79%) saturate(2476%) hue-rotate(236deg);
    }

    &:hover {
      background: color-mix(in srgb, var(--app-accent-purple) 12%, transparent);
    }
  }
}

.pending-requests-badge {
  display: inline-flex;
  align-items: center;
  height: 22px;
  padding: 0 6px;
  border-radius: var(--app-radius-sm);
  font-size: var(--app-text-sm);
  font-weight: 500;
  background: rgba(250, 173, 20, 0.12);
  color: var(--td-warning-color);
  white-space: nowrap;
}

.bottom-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.relation-role-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 22px;
  padding: 0 6px;
  border-radius: var(--app-radius-sm);
  font-size: var(--app-text-sm);
  font-weight: 500;
  font-family: var(--app-font-family);
  background: color-mix(in srgb, var(--td-text-color-secondary) 8%, transparent);
  color: var(--td-text-color-secondary);

  .t-icon {
    flex-shrink: 0;
    color: var(--td-text-color-secondary);
  }

  &.owner {
    background: color-mix(in srgb, var(--app-accent-purple) 10%, transparent);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }
  }

  &.admin {
    background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }
  }

  &.editor {
    background: color-mix(in srgb, var(--td-brand-color) 8%, transparent);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }
  }

  &.viewer {
    background: color-mix(in srgb, var(--td-text-color-secondary) 8%, transparent);
    color: var(--td-text-color-secondary);

    .t-icon {
      color: var(--td-text-color-secondary);
    }
  }
}

:deep(.t-dialog__position.t-dialog--top) {
  padding-top: 40vh !important;
}

.resource-list-header();

</style>

<style lang="less">

.invite-preview-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  backdrop-filter: blur(4px);
}

.invite-preview-modal {
  position: relative;
  width: 100%;
  max-width: 480px;
  max-height: 90vh;
  background: var(--td-bg-color-container);
  border-radius: var(--app-radius-xl);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  overflow: hidden;
  display: flex;
  flex-direction: column;

  &.is-wide {
    max-width: 560px;
  }
}

.invite-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 48px 16px 20px;
  background: var(--td-bg-color-container);
  border-bottom: 1px solid var(--td-component-stroke);
  flex-shrink: 0;
  gap: 12px;
}

.invite-preview-back {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-secondary);
  transition: background var(--app-motion-base) ease, color var(--app-motion-base) ease;

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }
}

.invite-preview-title {
  margin: 0;
  font-size: var(--app-text-xl);
  font-weight: 600;
  color: var(--td-text-color-primary);
  flex: 1;
  min-width: 0;
}

.invite-preview-close {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-secondary);
  transition: background var(--app-motion-base) ease, color var(--app-motion-base) ease;
  z-index: 10;

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }

  &:active {
    background: var(--td-bg-color-secondarycontainer);
  }
}

.invite-preview-body-wrap {
  flex: 0 0 auto;
  overflow: hidden;
  height: auto;
  transition:
    min-height 0.35s cubic-bezier(0.4, 0, 0.2, 1),
    max-height 0.35s cubic-bezier(0.4, 0, 0.2, 1);
}

.invite-body-inner {
  display: block;
}

.invite-preview-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 20px 24px 0;
  min-height: 0;
  max-height: calc(90vh - 120px);

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--td-bg-color-component-disabled);
    border-radius: 3px;
    transition: background var(--app-motion-base);

    &:hover {
      background: var(--td-brand-color);
    }
  }
}

.join-mode-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 20px;
}

.join-mode-pill {
  display: inline-flex;
  align-items: center;
  padding: 6px 14px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  font: inherit;
  font-size: var(--app-text-md);
  line-height: 1.4;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: color var(--app-motion-fast) ease, background var(--app-motion-fast) ease;

  &:hover,
  &:focus-visible {
    color: var(--td-brand-color);
    background: color-mix(in srgb, var(--td-brand-color) 8%, var(--td-bg-color-secondarycontainer));
    outline: none;
  }

  &.active {
    background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
    color: var(--td-brand-color);
    font-weight: 500;
  }
}

.join-form-item {
  margin-bottom: 20px;

  &--compact {
    margin-bottom: 12px;
  }

  .join-form-label {
    display: block;
    margin-bottom: 4px;
    font-size: var(--app-text-base);
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .join-form-desc {
    margin: 0 0 10px;
    font-size: var(--app-text-md);
    color: var(--td-text-color-secondary);
    line-height: 1.5;
  }

  .join-form-tip {
    margin: 8px 0 0;
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
    line-height: 1.45;
  }

  :deep(.t-input),
  :deep(.t-select),
  :deep(.t-textarea) {
    width: 100%;
  }
}

.join-tab-content-wrapper {
  transition: height var(--app-motion-slow) cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
}

.join-tab-content {
  width: 100%;
}

.searchable-list-wrap {
  max-height: 320px;
  min-height: 120px;
  overflow-y: auto;
  margin-bottom: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-container);

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--td-bg-color-component-disabled);
    border-radius: 3px;
    transition: background var(--app-motion-base);

    &:hover {
      background: var(--td-brand-color);
    }
  }
}

.searchable-empty {
  padding: 24px 16px;
}

.searchable-list {
  display: flex;
  flex-direction: column;
}

.searchable-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--td-component-stroke);
  cursor: pointer;
  transition: background var(--app-motion-fast) ease;

  &:last-child {
    border-bottom: none;
  }

  &:hover:not(.is-full) {
    background: var(--td-bg-color-container-hover);
  }

  &.is-full {
    cursor: default;
    opacity: 0.72;
  }
}

.searchable-row-main {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  flex: 1;
}

.searchable-row-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.searchable-row-title {
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.searchable-row-desc {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.searchable-row-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.searchable-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
}

.invite-preview-input {
  .invite-preview-error-inline {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
    padding: 10px 12px;
    border-radius: var(--app-radius-md);
    background: var(--td-error-color-light);
    color: var(--td-error-color);
    font-size: var(--app-text-md);
  }

  .invite-preview-footer-single {
    margin-top: 4px;
    padding: 16px 0 20px;
    border-top: 1px solid var(--td-component-stroke);
  }
}

.invite-preview-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64px 28px;
  gap: 20px;

  .invite-preview-loading-text {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-family);
  }
}

.invite-preview-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 40px 28px;

  .invite-preview-error-icon {
    color: var(--td-error-color);
    margin-bottom: 20px;
  }

  .invite-preview-error-title {
    font-size: var(--app-text-2xl);
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 8px;
    font-family: var(--app-font-family);
  }

  .invite-preview-error-desc {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
    margin: 0 0 24px;
    line-height: 1.5;
    font-family: var(--app-font-family);
  }
}

.invite-preview-body-preview {
  padding: 8px 24px 0;

  > .invite-preview-footer {
    margin: 16px -24px 0;
  }
}

.preview-space-hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 8px 0 20px;
}

.preview-space-avatar-wrap {
  margin-bottom: 12px;
}

.preview-space-name {
  margin: 0 0 6px;
  font-size: var(--app-text-2xl);
  font-weight: 600;
  line-height: 1.35;
  color: var(--td-text-color-primary);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preview-space-desc {
  margin: 0 0 14px;
  max-width: 360px;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-secondary);
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
}

.preview-space-badges {
  justify-content: center;
  margin-bottom: 12px;
}

.preview-space-id-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 4px 10px;
  border: none;
  border-radius: var(--app-radius-pill);
  background: var(--td-bg-color-secondarycontainer);
  font: inherit;
  font-size: var(--app-text-sm);
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: background var(--app-motion-fast) ease, color var(--app-motion-fast) ease;

  code {
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-xs);
    color: var(--td-text-color-secondary);
    background: transparent;
    border: none;
    padding: 0;
  }

  .t-icon {
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);
  }

  &:hover {
    background: color-mix(in srgb, var(--td-brand-color) 8%, var(--td-bg-color-secondarycontainer));
    color: var(--td-text-color-secondary);

    code,
    .t-icon {
      color: var(--td-brand-color);
    }
  }
}

.preview-member-status {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 0 4px;
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-brand-color);
}

.preview-join-summary {
  padding-top: 16px;
  border-top: 1px solid var(--td-component-stroke);
}

.preview-info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 28px;
}

.preview-info-label {
  font-size: var(--app-text-base);
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.preview-info-desc {
  margin: 8px 0 0;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-secondary);

  &--warning {
    color: var(--td-warning-color-active);
  }
}

.preview-join-fields {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px dashed var(--td-component-stroke);

  .t-select,
  .t-textarea {
    width: 100%;
  }

  .join-form-item--compact:last-child {
    margin-bottom: 0;
  }
}

.invite-preview-footer {
  padding: 12px 24px 20px;
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
  background: var(--td-bg-color-container);
}

.modal-enter-active,
.modal-leave-active {
  transition: all var(--app-motion-slow) ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;

  .invite-preview-modal {
    transform: scale(0.95);
  }
}
</style>
