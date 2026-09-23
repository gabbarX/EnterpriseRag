<template>
  <div class="kb-list-container">
    <div class="kb-list-content">
      <div class="header" style="--wails-draggable: drag">
        <div class="header-title" style="--wails-draggable: drag">
          <div class="title-row" style="--wails-draggable: drag">
            <h2 style="--wails-draggable: drag">
              <ResourceIcon type="knowledge" :size="24" />
              {{ 'Knowledge Base' }}
            </h2>
            <t-tooltip v-if="authStore.hasRole('contributor')" :content="'Create Knowledge Base'" placement="bottom">
              <t-button variant="text" theme="default" size="small" class="header-action-btn"
                data-guide="kb-list-create" style="--wails-draggable: no-drag" @click="handleCreateKnowledgeBase">
                <template #icon><t-icon name="folder-add" size="16px" /></template>
              {{ 'Create Knowledge Base' }}
              </t-button>
            </t-tooltip>
          </div>
          <p class="header-subtitle" style="--wails-draggable: drag">{{ 'Manage and organize your knowledge bases, supporting document-based and FAQ-based knowledge bases' }}</p>
        </div>
      </div>
      <ResourceListToolbar :hide-scopes="authStore.isLiteMode" v-model="spaceSelection" v-model:query="keyword" :count-all="allKnowledgeBases"
      :count-mine="kbs.length" :count-by-org="effectiveSharedCountByOrg" :count-favorites="kbFavoritesCount"
      :count-recents="kbRecentsCount" />
      <div class="kb-list-main">
        <EmptyState v-if="keyword.trim() && !(loading || spaceKbsLoading) && visibleResultCount === 0" icon="search"
          :title="'No results'">
          <t-button variant="outline" @click="keyword = ''">{{ 'Clear' }}</t-button>
        </EmptyState>
        <!-- creator filter intentionally removed from chrome: every card
             already shows its creator via ResourceOriginBadge / avatar, so
             a dedicated horizontal switch added more noise than signal.
             The backend `?creator=mine|others` param and the URL-state
             field are kept so a future "filter by member" entry point
             (e.g. clicking an avatar) can deep-link without re-plumbing. -->

        <div v-if="hasUninitializedKbs" class="warning-banner">
          <t-icon name="info-circle" size="16px" />
          <span>{{ 'Some knowledge bases are not initialized. Configure model information in settings before adding documents.' }}</span>
        </div>

        <div v-if="loading && kbs.length === 0" class="kb-card-wrap">
          <div v-for="n in 6" :key="'skel-' + n" class="kb-card kb-card-skeleton is-skeleton">
            <div class="card-header">
              <t-skeleton animation="gradient" :row-col="[{ width: '60%', height: '20px' }]" />
            </div>
            <div class="card-content">
              <t-skeleton animation="gradient"
                :row-col="[{ width: '100%', height: '14px' }, { width: '80%', height: '14px' }]" />
            </div>
            <div class="card-bottom">
              <t-skeleton animation="gradient"
                :row-col="[[{ width: '28px', height: '28px', type: 'rect' }, { width: '28px', height: '28px', type: 'rect' }]]" />
            </div>
          </div>
        </div>

        <div
          v-if="(spaceSelection === 'all' || spaceSelection === 'favorites' || spaceSelection === 'recents') && filteredKnowledgeBases.length > 0"
          class="kb-card-wrap">
          <div
            v-if="filteredKnowledgeBases[0] && filteredKnowledgeBases[0].isMine && filteredKnowledgeBases[0].is_pinned"
            class="kb-section-header kb-section-header-pinned" role="button" tabindex="0"
            :aria-expanded="!isKbSectionCollapsed('pinned')" @click="toggleKbSection('pinned')"
            @keydown.enter.prevent="toggleKbSection('pinned')"
            @keydown.space.prevent="toggleKbSection('pinned')">
            <t-icon name="pin-filled" size="14px" />
            <span>{{ 'Pinned' }}</span>
            <span class="kb-section-count">{{ filteredKbSectionCounts.pinned }}</span>
            <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('pinned') ? 'chevron-right' : 'chevron-down'"
              size="14px" />
          </div>
        <!-- All: my KBs plus the KBs shared with me. The pinned group is owned by the
             header above; the remaining segments (created by me / workspace ·
             view only / shared with me) each render their own header. The old
             "Other" transition header is meaningless under the per-user pin model, so it is gone. -->
          <template v-for="(kb, index) in filteredKnowledgeBases" :key="kb.id">
            <!-- Created by me: header before the first non-pinned card I created,
                 shown for every role whether or not a pinned group sits above. -->
            <div v-if="kb.isMine
              && isMyKb(kb as KB)
              && !kb.is_pinned
              && (index === 0
                || (filteredKnowledgeBases[index - 1] as any).is_pinned)" class="kb-section-header" role="button"
              tabindex="0" :aria-expanded="!isKbSectionCollapsed('mine')" @click="toggleKbSection('mine')"
              @keydown.enter.prevent="toggleKbSection('mine')"
              @keydown.space.prevent="toggleKbSection('mine')">
              <t-icon name="user" size="14px" />
              <span>{{ 'Created by me' }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.mine }}</span>
              <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('mine') ? 'chevron-right' : 'chevron-down'"
                size="14px" />
            </div>
            <!-- Workspace · View only: KBs a teammate created in this workspace that the
                 current contributor cannot edit. The card must be non-pinned (otherwise it
                 belongs to Pinned) and the previous card is either absent, shared with me,
                 created by me, or pinned (the pinned to non-pinned transition needs this header too). -->
            <div v-if="kb.isMine
              && !isMyKb(kb as KB)
              && !kb.is_pinned
              && (index === 0
                || !filteredKnowledgeBases[index - 1].isMine
                || isMyKb(filteredKnowledgeBases[index - 1] as KB)
                || (filteredKnowledgeBases[index - 1] as any).is_pinned)" class="kb-section-header" role="button"
              tabindex="0" :aria-expanded="!isKbSectionCollapsed('tenantOthers')" @click="toggleKbSection('tenantOthers')"
              @keydown.enter.prevent="toggleKbSection('tenantOthers')"
              @keydown.space.prevent="toggleKbSection('tenantOthers')">
              <t-icon :name="tenantSectionIconName" size="14px" />
              <span>{{ tenantSectionLabelKey }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.tenantOthers }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('tenantOthers') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · Can edit: first transition from my own cards into shared + editable -->
            <div v-if="!kb.isMine
              && isSharedKbEditable((kb as any).permission)
              && (index === 0 || filteredKnowledgeBases[index - 1].isMine)" class="kb-section-header" role="button"
              tabindex="0" :aria-expanded="!isKbSectionCollapsed('sharedEditable')" @click="toggleKbSection('sharedEditable')"
              @keydown.enter.prevent="toggleKbSection('sharedEditable')"
              @keydown.space.prevent="toggleKbSection('sharedEditable')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="edit-1" size="12px" class="kb-section-subicon" />
              <span>{{ 'Shared with me · Can edit' }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.sharedEditable }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedEditable') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · View only: transition from editable shares / my cards into viewer shares -->
            <div v-if="!kb.isMine
              && !isSharedKbEditable((kb as any).permission)
              && (index === 0
                || filteredKnowledgeBases[index - 1].isMine
                || isSharedKbEditable((filteredKnowledgeBases[index - 1] as any).permission))"
              class="kb-section-header" role="button" tabindex="0" :aria-expanded="!isKbSectionCollapsed('sharedReadonly')" @click="toggleKbSection('sharedReadonly')"
              @keydown.enter.prevent="toggleKbSection('sharedReadonly')"
              @keydown.space.prevent="toggleKbSection('sharedReadonly')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="browse" size="12px" class="kb-section-subicon" />
              <span>{{ 'Shared with me · View only' }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.sharedReadonly }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedReadonly') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <div v-if="kb.isMine" v-show="!isKbSectionCollapsed(kbSectionOf(kb))" class="kb-card" :class="{
              'uninitialized': !isInitialized(kb),
              'kb-type-document': (kb.type || 'document') === 'document',
              'kb-type-faq': kb.type === 'faq',
              'highlight-flash': highlightedKbId !== null && highlightedKbId === kb.id
            }"
              :ref="el => { if (highlightedKbId !== null && highlightedKbId === kb.id && el) highlightedCardRef = el as HTMLElement }"
              role="link" tabindex="0" @keydown.enter.self.prevent="handleCardClick(kb)" @keydown.space.self.prevent="handleCardClick(kb)" @click="handleCardClick(kb)">
              <button type="button" class="kb-favorite-star" :class="{ 'is-favorited': isKbFavorited(kb.id) }"
                :aria-label="'Starred'" :aria-pressed="isKbFavorited(kb.id)" @click.stop="toggleFavoriteKb(kb.id, $event)">
                <t-icon :name="isKbFavorited(kb.id) ? 'star-filled' : 'star'" size="14px" />
              </button>
              <div class="card-header">
                <span class="card-title" :title="kb.name">
                  <KbWikiBadge v-if="isWikiKb(kb)" />
                  <span class="card-title-text">{{ kb.name }}</span>
                </span>
                <!-- The card menu always exists when the card is visible: pin
                     is now per-user and available to anyone who can see the KB
                     (backend route only requires KB read access). Settings /
                     Delete are mutations, so they stay behind canManageKBCard. -->
                <t-popup overlayClassName="card-more-popup" trigger="click" destroy-on-close
                  placement="bottom-right">
                  <button type="button" :aria-label="'Expand'" class="more-wrap" @click.stop>
                    <img class="more-icon" src="@/assets/img/more.png" alt="" />
                  </button>
                  <template #content>
                    <div class="popup-menu" @click.stop>
                      <div class="popup-menu-item" @click.stop="handleTogglePinById(kb.id)">
                        <t-icon class="menu-icon" :name="kb.is_pinned ? 'pin-filled' : 'pin'" />
                        <span>{{ kb.is_pinned ? 'Unpin' : 'Pin to Top' }}</span>
                      </div>
                      <div v-if="canDuplicateKBCard(kb)" class="popup-menu-item"
                        @click.stop="handleDuplicateById(kb.id)">
                        <t-icon class="menu-icon" name="file-copy" />
                        <span>{{ 'Duplicate' }}</span>
                      </div>
                      <template v-if="canManageKBCard(kb)">
                        <div class="popup-menu-item" @click.stop="handleSettingsById(kb.id)">
                          <t-icon class="menu-icon" name="setting" />
                          <span>{{ 'Settings' }}</span>
                        </div>
                        <div class="popup-menu-item delete" @click.stop="handleDeleteById(kb.id)">
                          <t-icon class="menu-icon" name="delete" />
                          <span>{{ 'Delete' }}</span>
                        </div>
                      </template>
                    </div>
                  </template>
                </t-popup>
              </div>

              <div class="card-content">
                <div class="card-description" :title="kb.description || 'No description'">
                  {{ kb.description || 'No description' }}
                </div>
              </div>

              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="kb.type === 'faq' ? 'FAQ Q&A' : 'Document-based'"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (kb.type || 'document') === 'document', 'type-faq': kb.type === 'faq' }">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" size="14px" />
                        <span class="badge-count">{{ kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count ||
                          0) }}</span>
                        <t-icon v-if="kb.isProcessing" name="loading" size="12px" class="processing-icon" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.extract_config?.enabled" :content="'Knowledge Graph'"
                      placement="top">
                      <div class="feature-badge kg">
                        <t-icon name="relation" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.vlm_config?.enabled" :content="'Multimodal'"
                      placement="top">
                      <div class="feature-badge multimodal">
                        <t-icon name="image" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.question_generation_config?.enabled"
                      :content="'Question Generation'" placement="top">
                      <div class="feature-badge question">
                        <t-icon name="help-circle" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.share_count && kb.share_count > 0"
                      :content="`Shared to ${kb.share_count} space(s)`" placement="top">
                      <div class="feature-badge shared">
                        <t-icon name="share" size="14px" />
                      </div>
                    </t-tooltip>
                  </div>
                </div>
                <div v-if="!authStore.isLiteMode && showKbOriginBadge(kb)" class="bottom-right">
                  <ResourceOriginBadge :variant="kbOriginVariant(kb)" :creator-name="kb.creator_name" />
                </div>
              </div>
            </div>

            <div v-else v-show="!isKbSectionCollapsed(kbSectionOf(kb))" class="kb-card shared-kb-card" :class="{
              'kb-type-document': (kb.type || 'document') === 'document',
              'kb-type-faq': kb.type === 'faq'
            }" role="link" tabindex="0" @keydown.enter.self.prevent="handleSharedKbClickFromAll(kb)" @keydown.space.self.prevent="handleSharedKbClickFromAll(kb)" @click="handleSharedKbClickFromAll(kb)">
              <button type="button" class="kb-favorite-star" :class="{ 'is-favorited': isKbFavorited(kb.id) }"
                :aria-label="'Starred'" :aria-pressed="isKbFavorited(kb.id)" @click.stop="toggleFavoriteKb(kb.id, $event)">
                <t-icon :name="isKbFavorited(kb.id) ? 'star-filled' : 'star'" size="14px" />
              </button>
              <div class="card-header">
                <span class="card-title" :title="kb.name">
                  <KbWikiBadge v-if="isWikiKb(kb)" />
                  <span class="card-title-text">{{ kb.name }}</span>
                </span>
                <t-tooltip :content="'View Details'" placement="top">
                  <button type="button" class="shared-detail-trigger" @click.stop="openSharedDetailFromAll(kb)"
                    :aria-label="'View Details'">
                    <t-icon name="info-circle" size="16px" />
                  </button>
                </t-tooltip>
              </div>

              <div class="card-content">
                <div class="card-description" :title="kb.description || 'No description'">
                  {{ kb.description || 'No description' }}
                </div>
              </div>

              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="kb.type === 'faq' ? 'FAQ Q&A' : 'Document-based'"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (kb.type || 'document') === 'document', 'type-faq': kb.type === 'faq' }">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" size="14px" />
                        <span class="badge-count">{{ kb.type === 'faq' ? (kb.chunk_count || '-') : (kb.knowledge_count
                          || '-')
                        }}</span>
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.extract_config?.enabled" :content="'Knowledge Graph'"
                      placement="top">
                      <div class="feature-badge kg">
                        <t-icon name="relation" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip
                      v-if="kb.vlm_config?.enabled || (kb.storage_provider_config?.provider && kb.storage_provider_config.provider !== 'local')"
                      :content="'Multimodal'" placement="top">
                      <div class="feature-badge multimodal">
                        <t-icon name="image" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.question_generation_config?.enabled"
                      :content="'Question Generation'" placement="top">
                      <div class="feature-badge question">
                        <t-icon name="help-circle" size="14px" />
                      </div>
                    </t-tooltip>
                  </div>
                </div>
                <div class="bottom-right">
                  <t-tooltip :content="kb.org_name" placement="top">
                    <div class="org-source">
                      <img src="@/assets/img/organization-green.svg" class="org-source-icon" alt=""
                        aria-hidden="true" />
                      <span>{{ kb.org_name }}</span>
                    </div>
                  </t-tooltip>
                </div>
              </div>
            </div>
          </template>
        </div>

        <div v-if="spaceSelection === 'mine' && sortedMineKbs.length > 0" class="kb-card-wrap">
          <div v-if="sortedMineKbs[0] && sortedMineKbs[0].is_pinned" class="kb-section-header kb-section-header-pinned"
            role="button" tabindex="0" :aria-expanded="!isKbSectionCollapsed('pinned')" @click="toggleKbSection('pinned')"
            @keydown.enter.prevent="toggleKbSection('pinned')"
            @keydown.space.prevent="toggleKbSection('pinned')">
            <t-icon name="pin-filled" size="14px" />
            <span>{{ 'Pinned' }}</span>
            <span class="kb-section-count">{{ mineKbSectionCounts.pinned }}</span>
            <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('pinned') ? 'chevron-right' : 'chevron-down'"
              size="14px" />
          </div>
          <!-- My knowledge bases. The pinned group is owned by the header above; the other
               segments each render their own header — see the matching comment in the "all" tab. -->
          <template v-for="(kb, index) in sortedMineKbs" :key="kb.id">
            <!-- Created by me: header before the first non-pinned card I created, shown
                 whether or not a pinned group sits above it, matching "Workspace · View
                 only" — see the matching comment in the "all" tab. -->
            <div v-if="isMyKb(kb)
              && !kb.is_pinned
              && (index === 0 || sortedMineKbs[index - 1].is_pinned)" class="kb-section-header" role="button"
              tabindex="0" :aria-expanded="!isKbSectionCollapsed('mine')" @click="toggleKbSection('mine')"
              @keydown.enter.prevent="toggleKbSection('mine')"
              @keydown.space.prevent="toggleKbSection('mine')">
              <t-icon name="user" size="14px" />
              <span>{{ 'Created by me' }}</span>
              <span class="kb-section-count">{{ mineKbSectionCounts.mine }}</span>
              <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('mine') ? 'chevron-right' : 'chevron-down'"
                size="14px" />
            </div>
            <!-- Workspace · View only: a non-pinned teammate KB whose previous card is either
                 absent, created by me, or pinned (the pinned to non-pinned transition). -->
            <div v-if="!isMyKb(kb)
              && !kb.is_pinned
              && (index === 0
                || isMyKb(sortedMineKbs[index - 1])
                || sortedMineKbs[index - 1].is_pinned)" class="kb-section-header" role="button" tabindex="0"
              :aria-expanded="!isKbSectionCollapsed('tenantOthers')" @click="toggleKbSection('tenantOthers')"
              @keydown.enter.prevent="toggleKbSection('tenantOthers')"
              @keydown.space.prevent="toggleKbSection('tenantOthers')">
              <t-icon :name="tenantSectionIconName" size="14px" />
              <span>{{ tenantSectionLabelKey }}</span>
              <span class="kb-section-count">{{ mineKbSectionCounts.tenantOthers }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('tenantOthers') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <div v-show="!isKbSectionCollapsed(kbSectionOf(kb))" class="kb-card" :class="{
              'uninitialized': !isInitialized(kb),
              'kb-type-document': (kb.type || 'document') === 'document',
              'kb-type-faq': kb.type === 'faq',
              'highlight-flash': highlightedKbId !== null && highlightedKbId === kb.id
            }"
              :ref="el => { if (highlightedKbId !== null && highlightedKbId === kb.id && el) highlightedCardRef = el as HTMLElement }"
              role="link" tabindex="0" @keydown.enter.self.prevent="handleCardClick(kb)" @keydown.space.self.prevent="handleCardClick(kb)" @click="handleCardClick(kb)">
              <button type="button" class="kb-favorite-star" :class="{ 'is-favorited': isKbFavorited(kb.id) }"
                :aria-label="'Starred'" :aria-pressed="isKbFavorited(kb.id)" @click.stop="toggleFavoriteKb(kb.id, $event)">
                <t-icon :name="isKbFavorited(kb.id) ? 'star-filled' : 'star'" size="14px" />
              </button>
              <div class="card-header">
                <span class="card-title" :title="kb.name">
                  <KbWikiBadge v-if="isWikiKb(kb)" />
                  <span class="card-title-text">{{ kb.name }}</span>
                </span>
                <!-- See the matching block in the "all" tab template for why
                     this is no longer gated by canManageKBCard. -->
                <t-popup v-model="kb.showMore" overlayClassName="card-more-popup"
                  :on-visible-change="onVisibleChange" trigger="click" destroy-on-close placement="bottom-right">
                  <button type="button" :aria-label="'Expand'" class="more-wrap" @click.stop="openMore(index)"
                    :class="{ 'active-more': currentMoreIndex === index }">
                    <img class="more-icon" src="@/assets/img/more.png" alt="" />
                  </button>
                  <template #content>
                    <div class="popup-menu" @click.stop>
                      <div class="popup-menu-item" @click.stop="handleTogglePin(kb)">
                        <t-icon class="menu-icon" :name="kb.is_pinned ? 'pin-filled' : 'pin'" />
                        <span>{{ kb.is_pinned ? 'Unpin' : 'Pin to Top' }}</span>
                      </div>
                      <div v-if="canDuplicateKBCard(kb)" class="popup-menu-item" @click.stop="handleDuplicate(kb)">
                        <t-icon class="menu-icon" name="file-copy" />
                        <span>{{ 'Duplicate' }}</span>
                      </div>
                      <template v-if="canManageKBCard(kb)">
                        <div class="popup-menu-item" @click.stop="handleSettings(kb)">
                          <t-icon class="menu-icon" name="setting" />
                          <span>{{ 'Settings' }}</span>
                        </div>
                        <div class="popup-menu-item delete" @click.stop="handleDelete(kb)">
                          <t-icon class="menu-icon" name="delete" />
                          <span>{{ 'Delete' }}</span>
                        </div>
                      </template>
                    </div>
                  </template>
                </t-popup>
              </div>

              <div class="card-content">
                <div class="card-description" :title="kb.description || 'No description'">
                  {{ kb.description || 'No description' }}
                </div>
              </div>

              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="kb.type === 'faq' ? 'FAQ Q&A' : 'Document-based'"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (kb.type || 'document') === 'document', 'type-faq': kb.type === 'faq' }">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" size="14px" />
                        <span class="badge-count">{{ kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count ||
                          0) }}</span>
                        <t-icon v-if="kb.isProcessing" name="loading" size="12px" class="processing-icon" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.extract_config?.enabled" :content="'Knowledge Graph'"
                      placement="top">
                      <div class="feature-badge kg">
                        <t-icon name="relation" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip
                      v-if="kb.vlm_config?.enabled || (kb.storage_provider_config?.provider && kb.storage_provider_config.provider !== 'local')"
                      :content="'Multimodal'" placement="top">
                      <div class="feature-badge multimodal">
                        <t-icon name="image" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.question_generation_config?.enabled"
                      :content="'Question Generation'" placement="top">
                      <div class="feature-badge question">
                        <t-icon name="help-circle" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="(kb.share_count ?? 0) > 0"
                      :content="`Shared to ${kb.share_count ?? 0} space(s)`" placement="top">
                      <div class="feature-badge shared">
                        <t-icon name="share" size="14px" />
                      </div>
                    </t-tooltip>
                  </div>
                </div>
                <div v-if="!authStore.isLiteMode && showKbOriginBadge(kb)" class="bottom-right">
                  <ResourceOriginBadge :variant="kbOriginVariant(kb)" :creator-name="kb.creator_name" />
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- The Collaboration / Shared-with-me aggregate view was removed: shared KBs now appear under "All" or under a specific space -->

        <div v-if="spaceSelectionOrgId && spaceKbsLoading" class="kb-list-main-loading">
          <t-loading size="medium" text="" />
        </div>
        <div v-else-if="spaceSelectionOrgId && sortedSpaceKbsList.length > 0" class="kb-card-wrap">
          <template v-for="(shared, index) in sortedSpaceKbsList"
            :key="'shared-' + (shared.share_id || `agent-${shared.knowledge_base?.id}-${shared.source_from_agent?.agent_id || ''}`)">
            <!-- Shared by me: entries I created and shared into this space; the header hangs off the first is_mine entry -->
            <div v-if="shared.is_mine && index === 0" class="kb-section-header"
              role="button" tabindex="0" :aria-expanded="!isKbSectionCollapsed('sharedByMe')" @click="toggleKbSection('sharedByMe')"
              @keydown.enter.prevent="toggleKbSection('sharedByMe')"
              @keydown.space.prevent="toggleKbSection('sharedByMe')">
              <t-icon name="share" size="14px" />
              <span>{{ 'Shared by me' }}</span>
              <span class="kb-section-count">{{ spaceKbSectionCounts.sharedByMe }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedByMe') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · Can edit: first transition from my own entries into shared + editable -->
            <div v-if="!shared.is_mine
              && isSharedKbEditable(shared.permission)
              && (index === 0 || sortedSpaceKbsList[index - 1].is_mine)" class="kb-section-header"
              role="button" tabindex="0" :aria-expanded="!isKbSectionCollapsed('sharedEditable')" @click="toggleKbSection('sharedEditable')"
              @keydown.enter.prevent="toggleKbSection('sharedEditable')"
              @keydown.space.prevent="toggleKbSection('sharedEditable')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="edit-1" size="12px" class="kb-section-subicon" />
              <span>{{ 'Shared with me · Can edit' }}</span>
              <span class="kb-section-count">{{ spaceKbSectionCounts.sharedEditable }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedEditable') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · View only: first transition from editable shares / my entries into viewer -->
            <div v-if="!shared.is_mine
              && !isSharedKbEditable(shared.permission)
              && (index === 0
                || sortedSpaceKbsList[index - 1].is_mine
                || isSharedKbEditable(sortedSpaceKbsList[index - 1].permission))" class="kb-section-header"
              role="button" tabindex="0" :aria-expanded="!isKbSectionCollapsed('sharedReadonly')" @click="toggleKbSection('sharedReadonly')"
              @keydown.enter.prevent="toggleKbSection('sharedReadonly')"
              @keydown.space.prevent="toggleKbSection('sharedReadonly')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="browse" size="12px" class="kb-section-subicon" />
              <span>{{ 'Shared with me · View only' }}</span>
              <span class="kb-section-count">{{ spaceKbSectionCounts.sharedReadonly }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedReadonly') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <div v-show="!isSpaceKbCollapsed(shared)" class="kb-card shared-kb-card" :class="{
              'kb-type-document': (shared.knowledge_base.type || 'document') === 'document',
              'kb-type-faq': shared.knowledge_base.type === 'faq'
            }" role="link" tabindex="0" @keydown.enter.self.prevent="handleSharedKbClick(shared)" @keydown.space.self.prevent="handleSharedKbClick(shared)" @click="handleSharedKbClick(shared)">
              <div class="card-header">
                <span class="card-title" :title="shared.knowledge_base.name">
                  <KbWikiBadge v-if="isWikiKb(shared.knowledge_base)" />
                  <span class="card-title-text">{{ shared.knowledge_base.name }}</span>
                </span>
                <t-tooltip v-if="!shared.is_mine" :content="'View Details'" placement="top">
                  <button type="button" class="shared-detail-trigger" @click.stop="openSharedDetail(shared)"
                    :aria-label="'View Details'">
                    <t-icon name="info-circle" size="16px" />
                  </button>
                </t-tooltip>
              </div>

              <div class="card-content">
                <div class="card-description" :title="shared.knowledge_base.description || 'No description'">
                  {{ shared.knowledge_base.description || 'No description' }}
                </div>
              </div>

              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="shared.knowledge_base.type === 'faq' ? 'FAQ Q&A' : 'Document-based'"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (shared.knowledge_base.type || 'document') === 'document', 'type-faq': shared.knowledge_base.type === 'faq' }">
                        <t-icon :name="shared.knowledge_base.type === 'faq' ? 'chat-bubble-help' : 'folder'"
                          size="14px" />
                        <span class="badge-count">{{ shared.knowledge_base.type === 'faq' ?
                          (shared.knowledge_base.chunk_count ??
                            '-') : (shared.knowledge_base.knowledge_count ?? '-') }}</span>
                      </div>
                    </t-tooltip>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- Empty "all" state: keeps the create CTA, because it means the space has no knowledge bases at all -->
        <EmptyState v-if="!keyword.trim() && spaceSelection === 'all' && filteredKnowledgeBases.length === 0 && !loading" :title="'No knowledge bases yet'"
          :description="'Click \u0022Create Knowledge Base\u0022 in the top-right corner to add your first one.'">
          <template #icon><ResourceIcon type="knowledge" :size="32" /></template>
          <t-button v-if="authStore.hasRole('contributor')" theme="primary" class="kb-create-btn"
            data-guide="kb-list-create" @click="handleCreateKnowledgeBase">
            <template #icon><t-icon name="folder-add" /></template>
            {{ 'Create Knowledge Base' }}
          </t-button>
        </EmptyState>

        <!-- Empty favorites state: no create button — "no favorites" is not "no knowledge
             bases", so the right nudge is to star one, not to create another. -->
        <EmptyState v-if="!keyword.trim() && spaceSelection === 'favorites' && filteredKnowledgeBases.length === 0 && !loading" icon="star" :title="'No favorites yet'"
          :description="'Star a knowledge base from its card to add it here'" />

        <!-- Empty recents state: same idea, the nudge is to open one. -->
        <EmptyState v-if="!keyword.trim() && spaceSelection === 'recents' && filteredKnowledgeBases.length === 0 && !loading" icon="history" :title="'Nothing here yet'"
          :description="'Knowledge bases you recently opened will show up here'" />

        <EmptyState v-if="!keyword.trim() && spaceSelection === 'mine' && kbs.length === 0 && !loading" :title="'No knowledge bases yet'"
          :description="'Click \u0022Create Knowledge Base\u0022 in the top-right corner to add your first one.'">
          <template #icon><ResourceIcon type="knowledge" :size="32" /></template>
          <t-button v-if="authStore.hasRole('contributor')" theme="primary" class="kb-create-btn"
            data-guide="kb-list-create" @click="handleCreateKnowledgeBase">
            <template #icon><t-icon name="folder-add" /></template>
            {{ 'Create Knowledge Base' }}
          </t-button>
        </EmptyState>

        <EmptyState v-if="!keyword.trim() && spaceSelectionOrgId && !spaceKbsLoading && spaceKbsList.length === 0" :title="'No shared knowledge bases'"
          :description="'You can join a shared space or request others to share knowledge bases with you'">
          <template #icon><ResourceIcon type="knowledge" :size="32" /></template>
        </EmptyState>
      </div>
    </div>

    <KnowledgeBaseEditorModal :visible="uiStore.showKBEditorModal" :mode="uiStore.kbEditorMode"
      :kb-id="uiStore.currentKBId || undefined" :initial-type="uiStore.kbEditorType"
      @update:visible="(val) => val ? null : uiStore.closeKBEditor()" @success="handleKBEditorSuccess" />

    <Teleport to="body">
      <Transition name="shared-detail-drawer">
        <div v-if="sharedDetailPanelVisible && currentSharedKbForDetail" class="shared-detail-drawer-overlay"
          @click.self="closeSharedDetailPanel">
          <div class="shared-detail-drawer">
            <div class="shared-detail-drawer-header">
              <h3 class="shared-detail-drawer-title">{{ 'Shared Knowledge Base' }}</h3>
              <button type="button" class="shared-detail-drawer-close" @click="closeSharedDetailPanel"
                :aria-label="'Close Settings'">
                <t-icon name="close" size="20px" />
              </button>
            </div>
            <div class="shared-detail-drawer-body">
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ 'Name' }}</span>
                <span class="shared-detail-value">{{ currentSharedKbForDetail.knowledge_base.name }}</span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ 'Source' }}</span>
                <span class="shared-detail-value shared-detail-source-type">
                  {{ currentSharedKbForDetail.source_from_agent ? 'Visible via shared agent' :
                    'KB shared directly to this space' }}
                </span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ currentSharedKbForDetail.source_from_agent ?
                  'Agent' : 'Space' }}</span>
                <span class="shared-detail-value shared-detail-org">
                  <img src="@/assets/img/organization-green.svg" class="shared-detail-org-icon" alt=""
                    aria-hidden="true" />
                  {{ currentSharedKbForDetail.source_from_agent ? currentSharedKbForDetail.source_from_agent.agent_name
                    :
                    currentSharedKbForDetail.org_name }}
                </span>
              </div>
              <div v-if="currentSharedKbForDetail.source_from_agent" class="shared-detail-row">
                <span class="shared-detail-label">{{ 'Agent KB strategy' }}</span>
                <span class="shared-detail-value">
                  {{ agentKbStrategyText(currentSharedKbForDetail.source_from_agent?.kb_selection_mode ?? '') }}
                </span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ 'Shared At' }}</span>
                <span class="shared-detail-value">{{ formatStringDate(new Date(currentSharedKbForDetail.shared_at))
                }}</span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ 'My Permission' }}</span>
                <t-tag size="small"
                  :theme="currentSharedKbForDetail.permission === 'admin' ? 'primary' : currentSharedKbForDetail.permission === 'editor' ? 'warning' : 'default'">
                  {{ (ORGANIZATION_ROLE_LABELS[currentSharedKbForDetail.permission] ?? '') }}
                </t-tag>
              </div>
            </div>
            <div class="shared-detail-drawer-footer">
              <t-button theme="default" variant="outline" @click="closeSharedDetailPanel">{{ 'Close'
              }}</t-button>
              <t-button theme="primary" class="go-to-kb-btn" @click="goToSharedKbFromPanel">
                <t-icon name="browse" />
                {{ 'Go to Knowledge Base' }}
              </t-button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <ContextualGuide tour="kbList" :when="showKbListContextualGuide" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { MessagePlugin, Icon as TIcon } from 'tdesign-vue-next'
import EmptyState from '@/components/EmptyState.vue'
import ResourceIcon from '@/components/icons/ResourceIcon.vue'
import { useConfirmDelete } from '@/components/settings/useConfirmDelete'
import { deleteKnowledgeBase, duplicateKnowledgeBase, togglePinKnowledgeBase } from '@/api/knowledge-base'
import { useChatResourcesStore } from '@/stores/chatResources'
import { formatStringDate } from '@/utils/index'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationStore } from '@/stores/organization'
import { listOrganizationSharedKnowledgeBases, type SharedKnowledgeBase, type OrganizationSharedKnowledgeBaseItem, type SourceFromAgentInfo } from '@/api/organization'
import { mergeAllScopeKnowledgeBases, type OwnedKnowledgeBase, type SharedKnowledgeBaseLike } from './kbListMerge'
import KnowledgeBaseEditorModal from './KnowledgeBaseEditorModal.vue'
import KbWikiBadge from './components/KbWikiBadge.vue'
import ResourceListToolbar from '@/components/ResourceListToolbar.vue'
import { matchesResourceQuery } from '@/utils/resourceListSearch'
import ResourceOriginBadge from '@/components/ResourceOriginBadge.vue'
import { shouldShowResourceOriginBadge } from '@/utils/card-list-badge'
import { permissionCanManageKB } from '@/utils/kbPermission'
import ContextualGuide from '@/components/ContextualGuide.vue'
import { isContextualGuideDone, markContextualGuideDone } from '@/config/contextualGuides'
import { useTenantModelReadiness } from '@/composables/useTenantModelReadiness'
import { useListUrlState } from '@/composables/useListUrlState'
import { useResourcePins } from '@/composables/useResourcePins'

const ORGANIZATION_ROLE_LABELS: Record<string, string> = {
  admin: 'Admin',
  editor: 'Editor',
  viewer: 'Viewer',
}

const router = useRouter()
const route = useRoute()
const uiStore = useUIStore()
const authStore = useAuthStore()
const { loaded: modelsReadyLoaded, isReadyForDocumentKb } = useTenantModelReadiness()
const orgStore = useOrganizationStore()
const chatResources = useChatResourcesStore()

// Default space selection depends on the current role.
// A viewer usually owns 0 KBs in the space, so "mine" would show an empty state while
// hiding shared KBs, which is badly misleading; viewers therefore default to "all".
// Contributors and above mainly manage the KBs they created, so they keep "mine".
//
// State lives in `?scope=` so links are shareable/bookmarkable; the
// composable handles two-way sync with the URL. We keep "mine" as the
// stored value (not "workspace") for back-compat with any external link
// that might point at the old query; ResourceListToolbar labels it as the current workspace.
const defaultScope: 'all' | 'mine' = authStore.hasRole('contributor') ? 'mine' : 'all'
const { scope: spaceSelection, creator: creatorFilter, query: keyword } = useListUrlState({
  defaultScope,
  defaultCreator: 'all',
})

// Per-user favorites + recents (localStorage-backed). isFavorite & touchRecent
// are wired into card render and click handlers below.
const pins = useResourcePins()
const kbFavoritesCount = computed(
  () => pins.favorites.value.filter((e) => e.type === 'kb').length
)
const kbRecentsCount = computed(
  () => pins.recents.value.filter((e) => e.type === 'kb').length
)

interface KB {
  id: string;
  name: string;
  description?: string;
  updated_at?: string;
  created_at?: string;
  pinned_at?: string;
  embedding_model_id?: string;
  summary_model_id?: string;
  type?: 'document' | 'faq';
  showMore?: boolean;
  vlm_config?: { enabled?: boolean; model_id?: string };
  extract_config?: { enabled?: boolean };
  storage_provider_config?: { provider?: string };
  storage_config?: { provider?: string; bucket_name?: string }; // legacy
  question_generation_config?: { enabled?: boolean; question_count?: number };
  knowledge_count?: number;
  chunk_count?: number;
  isProcessing?: boolean;
  processing_count?: number;
  share_count?: number;
  is_pinned?: boolean;
  // creator_id is the owner-id matched against authStore.user.id when
  // gating the per-card more-menu (Settings / Delete). Empty for legacy
  // KBs created before PR 5; those fall back to the role gate.
  creator_id?: string;
  // creator_name is backfilled by the backend list endpoint, and is used only for the tooltip on the origin badge.
  creator_name?: string;
}

const kbs = ref<KB[]>([])
const loading = ref(false)
const confirmDelete = useConfirmDelete()
const currentMoreIndex = ref<number>(-1)
const highlightedKbId = ref<string | null>(null)
const highlightedCardRef = ref<HTMLElement | null>(null)
let uploadRefreshTimer: ReturnType<typeof setTimeout> | null = null

// Shared knowledge bases (everything cross-tenant shared to me, including
// viewer-only). Used by the per-space views and the "all" aggregate so
// readers still see read-only shares — those are valid resources, just
// not editable.
const sharedKbs = computed<SharedKnowledgeBase[]>(() => orgStore.sharedKnowledgeBases || [])

const allKnowledgeBases = computed(() => kbs.value.length + sharedKbs.value.length)

// NB: keep the reserved-scope list in sync with ResourceListToolbar's
// non-org buckets — otherwise a new pseudo-scope (e.g. "favorites")
// falls through here and triggers the per-space code paths, which
// renders an extra "no shared KB" empty state on top of the real view.
const RESERVED_SCOPES = new Set(['all', 'mine', 'favorites', 'recents'])
const spaceSelectionOrgId = computed(() => {
  const s = spaceSelection.value
  return !!s && !RESERVED_SCOPES.has(s)
})

const spaceKbsList = ref<OrganizationSharedKnowledgeBaseItem[]>([])
const spaceKbsLoading = ref(false)

// Ordering for the workspace tab:
//   1. pinned KBs (mine or teammate), newest pin first
//   2. my non-pinned KBs
//   3. teammate non-pinned KBs (rendered under the "Workspace · View only" header)
//
// Pin is per-user as of migration 000050, so a teammate-created KB that
// the caller has personally pinned must float into the pinned section
// even though it would otherwise live in the teammate sub-group. The
// previous version only bucketed by isMyKb and silently demoted these
// pinned-but-teammate KBs.
const unsearchedSortedMineKbs = computed<KB[]>(() => {
  return [...kbs.value].sort((a, b) => {
    const ap = a.is_pinned ? 0 : 1
    const bp = b.is_pinned ? 0 : 1
    if (ap !== bp) return ap - bp
    if (a.is_pinned && b.is_pinned) {
      const at = a.pinned_at ? Date.parse(a.pinned_at as string) : 0
      const bt = b.pinned_at ? Date.parse(b.pinned_at as string) : 0
      if (at !== bt) return bt - at
    }
    const am = isMyKb(a) ? 0 : 1
    const bm = isMyKb(b) ? 0 : 1
    if (am !== bm) return am - bm
    const ac = a.created_at ? Date.parse(a.created_at as string) : 0
    const bc = b.created_at ? Date.parse(b.created_at as string) : 0
    return bc - ac
  })
})
const sortedMineKbs = computed(() => unsearchedSortedMineKbs.value.filter(item => matchesResourceQuery(item, keyword.value)))

// Stable ordering in the space view: entries I created (is_mine) come first, then the
// shared ones by editable / view-only — keeping the visual order aligned with the "all" view.
const unsearchedSortedSpaceKbsList = computed(() => {
  return [...spaceKbsList.value].sort((a, b) => {
    const aMine = a.is_mine ? 0 : 1
    const bMine = b.is_mine ? 0 : 1
    if (aMine !== bMine) return aMine - bMine
    const aE = isSharedKbEditable(a.permission) ? 0 : 1
    const bE = isSharedKbEditable(b.permission) ? 0 : 1
    return aE - bE
  })
})
const sortedSpaceKbsList = computed(() => unsearchedSortedSpaceKbsList.value.filter(item => matchesResourceQuery(item.knowledge_base, keyword.value)))
const spaceCountByOrg = ref<Record<string, number>>({})

// Shared KB count per space (shown in the sidebar): prefer the per-space total returned by the API, otherwise the "shared with me" count
const sharedCountByOrg = computed<Record<string, number>>(() => {
  const map: Record<string, number> = {}
  sharedKbs.value.forEach(s => {
    const id = s.organization_id
    if (!id) return
    map[id] = (map[id] || 0) + 1
  })
    ; (orgStore.organizations || []).forEach(org => {
      if (map[org.id] === undefined) map[org.id] = 0
    })
  return map
})
const effectiveSharedCountByOrg = computed<Record<string, number>>(() => {
  const base = sharedCountByOrg.value
  const merged = { ...base }
  Object.keys(spaceCountByOrg.value).forEach(orgId => {
    merged[orgId] = spaceCountByOrg.value[orgId]
  })
  return merged
})

// Favorites / Recents views: hydrate pin entries by id against every KB
// the user can already see in this page (own + cross-tenant shared). KBs
// the user no longer has access to (deleted / share revoked) are dropped
// silently — the pin survives until the next mutation, which keeps the
// composable simple at the cost of harmless ghost entries.
//
// Order:
//   - favorites: most recently starred first (PinEntry.ts desc)
//   - recents: most recently opened first (also ts desc, already sorted)
const kbResourceIndex = computed(() => {
  const map = new Map<string, { kb: any; isMine: boolean; shared?: SharedKnowledgeBase }>()
  for (const kb of kbs.value) {
    map.set(kb.id, { kb, isMine: true })
  }
  for (const shared of sharedKbs.value) {
    if (!shared.knowledge_base) continue
    if (!map.has(shared.knowledge_base.id)) {
      map.set(shared.knowledge_base.id, { kb: shared.knowledge_base, isMine: false, shared })
    }
  }
  return map
})

const favoritesList = computed(() => {
  return pins.favorites.value
    .filter((e) => e.type === 'kb')
    .map((e) => {
      const entry = kbResourceIndex.value.get(e.id)
      if (!entry) return null
      if (entry.isMine) {
        return { ...entry.kb, isMine: true as const, _pinTs: e.ts }
      }
      const s = entry.shared!
      return {
        ...entry.kb,
        isMine: false as const,
        permission: s.permission,
        shared_at: s.shared_at,
        share_id: s.share_id,
        org_name: s.org_name,
        _pinTs: e.ts,
      } as any
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
})

const recentsList = computed(() => {
  return pins.recents.value
    .filter((e) => e.type === 'kb')
    .map((e) => {
      const entry = kbResourceIndex.value.get(e.id)
      if (!entry) return null
      if (entry.isMine) {
        return { ...entry.kb, isMine: true as const, _pinTs: e.ts }
      }
      const s = entry.shared!
      return {
        ...entry.kb,
        isMine: false as const,
        permission: s.permission,
        shared_at: s.shared_at,
        share_id: s.share_id,
        org_name: s.org_name,
        _pinTs: e.ts,
      } as any
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
})

// Editable permission: editor / admin. A viewer lands in the view-only group.
// The check uses the share-level permission, not the space role: a viewer share from another
// space really is read-only even when I own this space, and an editor share really is
// writable even when I am only a contributor here.
const EDITABLE_PERMS = new Set(['admin', 'editor'])
function isSharedKbEditable(perm: string | undefined): boolean {
  return !!perm && EDITABLE_PERMS.has(perm)
}

// Header for KBs created in this workspace by someone other than the current user.
// Contributors / viewers have no write access to them, hence "View only"; admins and
// owners can edit everything in the space, so "View only" would keep misleading them
// into thinking they cannot. This segment really is "KBs other members of the workspace
// created", so labelling it by ownership rather than permission is more accurate.
const tenantSectionLabelKey = computed(() =>
  authStore.hasRole('admin')
    ? 'Workspace · Other members' : 'Workspace · View only'
)

// The icon follows the wording above: admins / owners see "Workspace · Other members",
// an ownership split that suits usergroup; contributors / viewers see "View only", where
// browse (the eye) still conveys read-but-not-write.
const tenantSectionIconName = computed(() =>
  authStore.hasRole('admin') ? 'usergroup' : 'browse'
)

// Section collapsing is ephemeral: it applies to the current session only and is never
// persisted to localStorage or the server. We track collapsed sections rather than
// expanded ones because everything starts expanded, so an empty Set is the initial state.
type KbSectionKey = 'pinned' | 'mine' | 'tenantOthers' | 'sharedByMe' | 'sharedEditable' | 'sharedReadonly'
const collapsedKbSections = ref<Set<KbSectionKey>>(new Set())
const isKbSectionCollapsed = (key: KbSectionKey) => collapsedKbSections.value.has(key)
const toggleKbSection = (key: KbSectionKey) => {
  // Assigning a fresh Set changes the ref identity, which is what triggers the template
  // re-render; .add/.delete does work on a Vue 3 reactive Set, but ref(Set) proxy
  // behaviour varies slightly across versions, so replacing wholesale is the safest.
  const next = new Set(collapsedKbSections.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedKbSections.value = next
}
// Decides which section a KB belongs to — the same rules the template v-ifs use, pulled
// out so the v-show on cards can reuse them instead of restating five sets of v-ifs.
//
// Inputs arrive in two shapes:
//   1. filteredKnowledgeBases entries, which carry an explicit `isMine` flag (see the
//      spread in filteredKnowledgeBases; cross-space shares get isMine=false).
//   2. sortedMineKbs entries are raw KBs, with neither isMine nor permission.
// Cross-space shares always carry `permission` and local entries never do, so "no
// permission" is a safe marker for this space. Check isMine first, then fall back to permission.
const kbSectionOf = (kb: any): KbSectionKey => {
  if (kb?.is_pinned) return 'pinned'
  const isOwnTenant = kb?.isMine === true || (kb?.isMine !== false && kb?.permission == null)
  if (isOwnTenant) return isMyKb(kb) ? 'mine' : 'tenantOthers'
  return isSharedKbEditable(kb?.permission) ? 'sharedEditable' : 'sharedReadonly'
}

// Entries in the space-filtered view (sortedSpaceKbsList) have a different shape: is_mine
// marks "shared out by me", and the rest go to sharedEditable / sharedReadonly by permission.
const spaceKbSectionOf = (shared: any): KbSectionKey => {
  if (shared?.is_mine) return 'sharedByMe'
  return isSharedKbEditable(shared?.permission) ? 'sharedEditable' : 'sharedReadonly'
}
const isSpaceKbCollapsed = (shared: any): boolean => isKbSectionCollapsed(spaceKbSectionOf(shared))

// How many cards each section actually holds — reusing the section resolver. Showing "(N)"
// on the header tells the user what collapsing will hide and helps verify the filters.
const emptyKbCounts = (): Record<KbSectionKey, number> => ({
  pinned: 0, mine: 0, tenantOthers: 0, sharedByMe: 0, sharedEditable: 0, sharedReadonly: 0,
})
const filteredKbSectionCounts = computed<Record<KbSectionKey, number>>(() => {
  const c = emptyKbCounts()
  filteredKnowledgeBases.value.forEach(kb => { c[kbSectionOf(kb)]++ })
  return c
})
const mineKbSectionCounts = computed<Record<KbSectionKey, number>>(() => {
  const c = emptyKbCounts()
  sortedMineKbs.value.forEach(kb => { c[kbSectionOf(kb)]++ })
  return c
})
const spaceKbSectionCounts = computed<Record<KbSectionKey, number>>(() => {
  const c = emptyKbCounts()
  sortedSpaceKbsList.value.forEach(shared => { c[spaceKbSectionOf(shared)]++ })
  return c
})

// Filtered knowledge bases: all = mine + every share; mine = mine only
//
// Favorites / Recents reuse the same render path as `all` — they're just
// pre-filtered, pre-ordered slices, so the existing kb-card / shared
// kb-card templates render them with zero extra markup. Order is
// preserved via the upstream array (pins order is ts-desc).
const unsearchedFilteredKnowledgeBases = computed(() => {
  if (spaceSelection.value === 'favorites') {
    return favoritesList.value
  }
  if (spaceSelection.value === 'recents') {
    return recentsList.value
  }
  if (spaceSelection.value === 'mine') {
    return kbs.value.map(kb => ({ ...kb, isMine: true as const }))
  }
  if (spaceSelection.value !== 'all') {
    return []
  }
  // The "All" scope merges own + shared KBs. The card template keys each
  // row by `kb.id`, so the same KB surfacing twice — owned *and* shared
  // back, or shared into the caller's view through two different orgs —
  // produced duplicate `v-for` keys and blanked the list once there were
  // ≥2 entries (#795). mergeAllScopeKnowledgeBases de-duplicates by KB id
  // (owned wins; most-privileged share kept) while preserving the existing
  // pinned → mine → teammate → shared(editable-first) ordering.
  return mergeAllScopeKnowledgeBases(
    kbs.value as unknown as OwnedKnowledgeBase[],
    sharedKbs.value as unknown as SharedKnowledgeBaseLike[],
    authStore.user?.id,
  ) as unknown as Array<(KB & { isMine: true }) | (SharedKnowledgeBase['knowledge_base'] & { isMine: false; permission: string; shared_at: string; share_id: string } & any)>
})
const filteredKnowledgeBases = computed(() => unsearchedFilteredKnowledgeBases.value.filter(item => matchesResourceQuery(item, keyword.value)))

const showKbListEmpty = computed(() => {
  if (loading.value || keyword.value.trim()) return false
  if (!authStore.hasRole('contributor')) return false
  if (spaceSelection.value === 'all' && filteredKnowledgeBases.value.length === 0) return true
  if (spaceSelection.value === 'mine' && kbs.value.length === 0) return true
  return false
})

const showKbListContextualGuide = computed(
  () => showKbListEmpty.value && !uiStore.showKBEditorModal,
)

const applyKbListData = (data: any[]) => {
  kbs.value = data.map((kb: any) => ({
    ...kb,
    updated_at: kb.updated_at ? formatStringDate(new Date(kb.updated_at)) : '',
    showMore: false,
    isProcessing: kb.is_processing || false,
    processing_count: kb.processing_count || 0
  }))
}

const fetchList = (force = false) => {
  loading.value = true
  // The creator filter only applies to the caller's own tenant KBs (the
  // first call). Shared KBs are inherently "not mine" so we don't filter
  // them server-side; the segmented control is also hidden whenever the
  // user is browsing the shared / per-space scopes.
  return Promise.all([
    chatResources.fetchKnowledgeBasesForList({ creator: creatorFilter.value }, force).then(applyKbListData),
    orgStore.fetchSharedKnowledgeBases({ force }),
    orgStore.fetchOrganizations({ force }),
  ]).finally(() => { loading.value = false }).then(() => {
    // Per-space KB counts already arrive in resource_counts from GET /organizations, stored in orgStore.resourceCounts
    const counts = orgStore.resourceCounts?.knowledge_bases?.by_organization
    if (counts) spaceCountByOrg.value = { ...counts }
  })
}

watch(spaceSelection, (val) => {
  // Stale URL guard: an older "Collaboration" view used scope=shared; that view
  // was removed, so normalize back to "all" instead of letting the
  // value fall through to the per-space fetch branch (which would 404
  // on the string "shared").
  if (val === 'shared') {
    spaceSelection.value = 'all'
    return
  }
  if (val === 'all' || val === 'mine' || val === 'favorites' || val === 'recents' || !val) {
    spaceKbsList.value = []
    return
  }
  spaceKbsLoading.value = true
  listOrganizationSharedKnowledgeBases(val).then((res) => {
    if (res.success && res.data) {
      spaceKbsList.value = res.data
      spaceCountByOrg.value = { ...spaceCountByOrg.value, [val]: res.data.length }
    } else {
      spaceKbsList.value = []
    }
  }).finally(() => {
    spaceKbsLoading.value = false
  })
}, { immediate: true })

// Refetch when the creator filter flips. We re-pull the whole list rather
// than filtering in-memory so the server stays the single source of truth
// (and we don't need to worry about stale share_count or pagination later).
watch(creatorFilter, () => {
  fetchList(true)
})

onMounted(() => {
  fetchList().then(() => {
    const highlightKbId = route.query.highlightKbId as string
    if (highlightKbId) {
      triggerHighlightFlash(highlightKbId)
      // Drop the transient highlight param but preserve other state
      // (scope / creator / q) so refreshing doesn't reset the user's view.
      const { highlightKbId: _drop, ...rest } = route.query
      router.replace({ query: rest })
    }
  })

  window.addEventListener('knowledgeFileUploaded', handleUploadFinishedEvent as EventListener)
})

onUnmounted(() => {
  window.removeEventListener('knowledgeFileUploaded', handleUploadFinishedEvent as EventListener)

  if (uploadRefreshTimer) {
    clearTimeout(uploadRefreshTimer)
    uploadRefreshTimer = null
  }
})

watch(() => route.query.highlightKbId, (newKbId) => {
  if (newKbId && typeof newKbId === 'string' && kbs.value.length > 0) {
    triggerHighlightFlash(newKbId)
    const { highlightKbId: _drop, ...rest } = route.query
    router.replace({ query: rest })
  }
})

const openMore = (index: number) => {
  // Only records which index is open, for the active style;
  // the popup itself is opened and closed by v-model.
  currentMoreIndex.value = index
}

const onVisibleChange = (visible: boolean) => {
  if (!visible) {
    currentMoreIndex.value = -1
  }
}

const handleSettings = (kb: KB) => {
  kb.showMore = false
  goSettings(kb.id)
}

// canManageKBCard mirrors KnowledgeBase.vue's `canManage`, gating the
// destructive items of the per-card menu — Settings, Delete — so a
// Viewer cannot click into them for a KB they don't own. The server
// still rejects the call (PR 5 guards every such mutation with
// OwnedKBOrAdmin) but the UI shouldn't surface buttons the user has
// no authority to use.
//
// The pin item is intentionally NOT gated by this predicate any more:
// pin state is per (user, kb) as of migration 000050 and the backend
// route only requires KB read access, so anyone who can see the card
// should be able to pin it for themselves.
//
// Legacy KBs created before PR 5 have an empty creator_id; treat
// those as tenant-owned (Admin+ may manage) so existing KBs aren't
// suddenly unmanageable for everyone.
function canManageKBCard(kb: KB): boolean {
  // Shared-space cards carry the org-share permission; when it exists it is
  // the only signal that counts. A read-only (viewer) or editor share must
  // not surface Settings/Delete even when the browsing user is an admin of
  // their own personal workspace — the backend 3-D permission cap would 403
  // the call anyway (#3098).
  const sharePermission = (kb as any).permission as string | undefined
  if (sharePermission) return permissionCanManageKB(sharePermission)
  // Shared-card shapes that lost their permission field (pin/recents merges)
  // are marked isMine === false; they must not fall through to the local
  // admin/creator fallbacks either.
  if ((kb as any).isMine === false) return false
  const userId = authStore.user?.id || ''
  if (kb.creator_id && userId && kb.creator_id === userId) return true
  return authStore.hasRole('admin')
}

function canDuplicateKBCard(kb: any): boolean {
  return authStore.hasRole('contributor') && kb.isMine !== false
}

// isMyKb only switches the bottom-right badge between "created by me" and "created by
// another member of this workspace". Unlike canManageKBCard there is no admin fallback:
// the badge matches purely on creator. An empty creator_id (KBs predating the PR 5 RBAC
// migration) is treated as tenant-owned, so old space-wide KBs are not all labelled mine.
function isMyKb(kb: { creator_id?: string }): boolean {
  const userId = authStore.user?.id || ''
  return !!(kb.creator_id && userId && kb.creator_id === userId)
}

// kbOriginVariant decides the shape of the bottom-right badge:
//   - created by me: mine (green "Created by me")
//   - created by someone else in this workspace: the creator variant, showing only the
//     creator name. The user is always browsing inside a workspace (the TenantSelector
//     at the top already names it), so repeating the space name here is redundant, and
//     contributor / admin / owner / viewer all see the same badge. When the creator
//     cannot be resolved the creator variant falls back to the resourceOrigin.tenant wording, so the label is never empty.
function kbOriginVariant(kb: { creator_id?: string }): 'mine' | 'creator' {
  return isMyKb(kb) ? 'mine' : 'creator'
}

function showKbOriginBadge(kb: { creator_id?: string; creator_name?: string }): boolean {
  return shouldShowResourceOriginBadge({
    section: kbSectionOf(kb),
    variant: kbOriginVariant(kb),
    creatorName: kb.creator_name,
    showSectionHeaders: true,
  })
}

const handleSettingsById = (id: string) => {
  goSettings(id)
}

const handleDeleteById = (id: string) => {
  const kb = kbs.value.find(k => k.id === id)
  if (kb) requestDelete(kb)
}

const handleTogglePin = async (kb: KB) => {
  kb.showMore = false
  try {
    const res: any = await togglePinKnowledgeBase(kb.id)
    if (res.success) {
      MessagePlugin.success(
        res.data.is_pinned ? 'Pinned' : 'Unpinned'
      )
      fetchList(true)
    }
  } catch {
    MessagePlugin.error('Operation failed')
  }
}

const handleTogglePinById = async (id: string) => {
  try {
    const res: any = await togglePinKnowledgeBase(id)
    if (res.success) {
      MessagePlugin.success(
        res.data.is_pinned ? 'Pinned' : 'Unpinned'
      )
      fetchList(true)
    }
  } catch {
    MessagePlugin.error('Operation failed')
  }
}

const handleDuplicate = async (kb: KB) => {
  kb.showMore = false
  await duplicateKB(kb.id)
}

const handleDuplicateById = async (id: string) => {
  await duplicateKB(id)
}

const duplicateKB = async (id: string) => {
  try {
    const res: any = await duplicateKnowledgeBase(id)
    if (res?.success) {
      const newKbId = res.data?.target_id || res.data?.knowledge_base?.id
      MessagePlugin.success('Knowledge base duplicate created (content not included)')
      await fetchList(true)
      if (newKbId) {
        triggerHighlightFlash(newKbId)
      }
    } else {
      MessagePlugin.error(res?.message || 'Failed to create knowledge base duplicate')
    }
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Failed to create knowledge base duplicate')
  }
}

const handleSharedKbClick = (sharedKb: SharedKnowledgeBase) => {
  pins.touchRecent('kb', sharedKb.knowledge_base.id)
  router.push(`/platform/knowledge-bases/${sharedKb.knowledge_base.id}`)
}

const handleSharedKbClickFromAll = (kb: any) => {
  pins.touchRecent('kb', kb.id)
  router.push(`/platform/knowledge-bases/${kb.id}`)
}

type SharedKbDetailItem = SharedKnowledgeBase & { is_mine?: boolean; source_from_agent?: SourceFromAgentInfo }
const sharedDetailPanelVisible = ref(false)
const currentSharedKbForDetail = ref<SharedKbDetailItem | null>(null)

const closeSharedDetailPanel = () => {
  sharedDetailPanelVisible.value = false
  currentSharedKbForDetail.value = null
}

const openSharedDetailFromAll = (kb: any) => {
  const sharedKb = sharedKbs.value.find(s => s.knowledge_base.id === kb.id)
  if (sharedKb) {
    currentSharedKbForDetail.value = sharedKb
    sharedDetailPanelVisible.value = true
  }
}

const openSharedDetail = (sharedKb: SharedKbDetailItem) => {
  currentSharedKbForDetail.value = sharedKb
  sharedDetailPanelVisible.value = true
}

const agentKbStrategyText = (mode: string) => {
  if (mode === 'all') return 'All knowledge bases'
  if (mode === 'selected') return 'Selected knowledge bases'
  return 'No knowledge bases'
}

const goToSharedKbFromPanel = () => {
  if (currentSharedKbForDetail.value) {
    router.push(`/platform/knowledge-bases/${currentSharedKbForDetail.value.knowledge_base.id}`)
    closeSharedDetailPanel()
  }
}

const handleDelete = (kb: KB) => {
  kb.showMore = false
  requestDelete(kb)
}

const requestDelete = (kb: KB) => {
  confirmDelete({
    title: 'Delete Confirmation',
    body: `Are you sure you want to delete the knowledge base "${kb.name}"? This action cannot be undone.`,
    onConfirm: async () => {
      try {
        const res: any = await deleteKnowledgeBase(kb.id)
        if (res.success) {
          MessagePlugin.success('Knowledge base deleted')
          fetchList(true)
        } else {
          MessagePlugin.error(res.message || 'Failed to delete knowledge base')
        }
      } catch (e: any) {
        MessagePlugin.error(e?.message || 'Failed to delete knowledge base')
      }
    },
  })
}

const isInitialized = (kb: KB) => {
  // LLM (summary) model is always required
  if (!kb.summary_model_id || kb.summary_model_id === '') return false
  // Embedding model only required when RAG indexing is enabled (vector or keyword)
  const strategy = (kb as any).indexing_strategy
  const needsEmbedding = !strategy || strategy.vector_enabled || strategy.keyword_enabled
  if (needsEmbedding && (!kb.embedding_model_id || kb.embedding_model_id === '')) return false
  return true
}

const isWikiKb = (kb: unknown) =>
  !!(kb as { indexing_strategy?: { wiki_enabled?: boolean } } | null | undefined)?.indexing_strategy?.wiki_enabled

const hasUninitializedKbs = computed(() => {
  return kbs.value.some(kb => !isInitialized(kb))
})

const handleCardClick = (kb: KB) => {
  // Track this open in the per-user "recent" list before navigating —
  // matches the user mental model "this is what I last worked on".
  pins.touchRecent('kb', kb.id)
  if (isInitialized(kb)) {
    goDetail(kb.id)
  } else {
    goSettings(kb.id)
  }
}

// toggleFavoriteKb is the click handler for the star icon rendered on
// each card. Stops propagation so it doesn't bubble into the card's
// own @click which would open the KB.
const toggleFavoriteKb = (kbId: string, evt?: Event) => {
  evt?.stopPropagation()
  pins.toggleFavorite('kb', kbId)
}
const isKbFavorited = (kbId: string) => pins.isFavorite('kb', kbId)

const goDetail = (id: string) => {
  router.push(`/platform/knowledge-bases/${id}`)
}

const goSettings = (id: string) => {
  uiStore.openKBSettings(id)
}

const handleCreateKnowledgeBase = () => {
  markContextualGuideDone('kbList')
  // With no models configured the wizard still opens, landing on the model section: users can add a model inside the wizard without visiting system settings first
  const initialSection =
    modelsReadyLoaded.value && !isReadyForDocumentKb.value ? 'models' : undefined
  uiStore.openCreateKB('document', initialSection)
}

const handleKBEditorSuccess = (kbId: string) => {
  console.log('[KnowledgeBaseList] knowledge operation success:', kbId)
  const shouldOpenDetailForUploadGuide = !isContextualGuideDone('kbDetail')
  // Editing from the list must invalidate the single-KB detail cache too, or the sidebar / detail page keeps showing stale info for 60s
  chatResources.invalidateKnowledgeBaseDetail(kbId)
  fetchList(true).then(() => {
    if (shouldOpenDetailForUploadGuide && kbId && !uiStore.showKBEditorModal) {
      goDetail(kbId)
    }
    if (route.query.highlightKbId === kbId) {
      triggerHighlightFlash(kbId)
      const { highlightKbId: _drop, ...rest } = route.query
      router.replace({ query: rest })
    }
  })
}

const triggerHighlightFlash = (kbId: string) => {
  highlightedKbId.value = kbId
  nextTick(() => {
    if (highlightedCardRef.value) {
      highlightedCardRef.value.scrollIntoView({
        behavior: 'smooth',
        block: 'center'
      })
    }
    setTimeout(() => {
      highlightedKbId.value = null
    }, 3000)
  })
}

const handleUploadFinishedEvent = (event: Event) => {
  const detail = (event as CustomEvent<{ kbId?: string | number; settled?: boolean }>).detail
  if (!detail?.kbId) return
  // Counts only need the batch's final refresh; a forced reload every couple
  // of seconds mid-batch would also close any open card menu.
  if (detail.settled === false) return
  if (uploadRefreshTimer) {
    clearTimeout(uploadRefreshTimer)
  }
  uploadRefreshTimer = setTimeout(() => {
    fetchList(true)
    uploadRefreshTimer = null
  }, 800)
}
const visibleResultCount = computed(() => spaceSelection.value === 'mine' ? sortedMineKbs.value.length : spaceSelectionOrgId.value ? sortedSpaceKbsList.value.length : filteredKnowledgeBases.value.length)
// A new search reveals matching rows even if their group was previously collapsed.
watch(keyword, () => { collapsedKbSections.value = new Set() })
</script>

<style scoped lang="less">
@import (reference) '@/components/css/resource-card.less';

.kb-list-container {
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
  display: flex;
}

.kb-list-content { .resource-list-content(); }

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-right: 28px;

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

.kb-list-main { .resource-list-main(); }

.kb-list-main-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  padding: 12px;
  background: var(--td-bg-color-container);
}

.header-subtitle {
  margin: 0;
  color: var(--td-text-color-placeholder);
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
  :deep(.btn-icon-wrapper) {
    color: var(--td-brand-color);
  }
}

.org-source {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  background: color-mix(in srgb, var(--td-brand-color) 6%, transparent);
  border-radius: var(--app-radius-sm);
  font-size: var(--app-text-sm);
  line-height: 1.4;
  color: var(--td-text-color-secondary);
  max-width: 140px;
  transition: background-color var(--app-motion-fast) ease;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }

  .org-source-icon {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    vertical-align: middle;
  }

  .t-icon {
    color: var(--td-brand-color);
    flex-shrink: 0;
  }
}

.shared-kb-card {
  position: relative;

  .org-tag {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: var(--app-text-sm);
    border-color: color-mix(in srgb, var(--td-brand-color) 15%, transparent);
    color: var(--td-brand-color);
    background: color-mix(in srgb, var(--td-brand-color) 4%, transparent);
    font-weight: 500;
    padding: 2px 8px;
    border-radius: var(--app-radius-xs);
    max-width: fit-content;
  }
}

.warning-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  margin-bottom: 20px;
  background: var(--td-warning-color-light);
  border: 1px solid var(--td-warning-color-focus);
  border-radius: var(--app-radius-sm);
  color: var(--td-warning-color);
  font-family: var(--app-font-family);
  font-size: var(--app-text-base);

  .t-icon {
    color: var(--td-warning-color);
    flex-shrink: 0;
  }
}

.kb-card-wrap {
  .resource-card-grid();
}

.kb-section-header {
  .resource-section-header();
}

.kb-card {
  .resource-card();

  &.uninitialized {
    opacity: 0.9;
  }

  .kb-favorite-star { .resource-favorite-button(); }

}

.bottom-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.bottom-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;

  .card-time {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
  }
}

.feature-badge {
  .resource-feature-badge();

  &.type-document {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    width: auto;
    padding: 0 6px;
    gap: 3px;

    &:hover {
      background: var(--td-bg-color-container-hover);
    }

    .badge-count {
      font-size: var(--app-text-xs);
      font-weight: 500;
    }

    .processing-icon {
      animation: wk-spin 1s linear infinite;
    }
  }

  &.type-faq {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    width: auto;
    padding: 0 6px;
    gap: 3px;

    &:hover {
      background: var(--td-bg-color-container-hover);
    }

    .badge-count {
      font-size: var(--app-text-xs);
      font-weight: 500;
    }

    .processing-icon {
      animation: wk-spin 1s linear infinite;
    }
  }

  &.kg {
    background: color-mix(in srgb, var(--app-accent-purple) 8%, transparent);
    color: var(--td-brand-color);

    &:hover {
      background: color-mix(in srgb, var(--app-accent-purple) 12%, transparent);
    }
  }

  &.multimodal {
    background: color-mix(in srgb, var(--td-warning-color) 8%, transparent);
    color: var(--td-warning-color);

    &:hover {
      background: color-mix(in srgb, var(--td-warning-color) 12%, transparent);
    }
  }

  &.question {
    background: color-mix(in srgb, var(--td-success-color) 8%, transparent);
    color: var(--td-success-color);

    &:hover {
      background: color-mix(in srgb, var(--td-success-color) 12%, transparent);
    }
  }

  &.shared {
    background: color-mix(in srgb, var(--td-brand-color) 8%, transparent);
    color: var(--td-brand-color);

    &:hover {
      background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
    }
  }

  &.role-admin {
    background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
    color: var(--td-brand-color-active);

    &:hover {
      background: color-mix(in srgb, var(--td-brand-color) 15%, transparent);
    }
  }

  &.role-editor {
    background: color-mix(in srgb, var(--td-warning-color) 10%, transparent);
    color: var(--td-warning-color);

    &:hover {
      background: color-mix(in srgb, var(--td-warning-color) 15%, transparent);
    }
  }

  &.role-viewer {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-secondary);

    &:hover {
      background: var(--td-bg-color-component);
    }
  }
}

@keyframes highlightFlash {
  0% {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--td-brand-color) 40%, transparent);
    transform: scale(1);
  }

  50% {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 8px color-mix(in srgb, var(--td-brand-color) 0%, transparent);
    transform: scale(1.02);
  }

  100% {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--td-brand-color) 0%, transparent);
    transform: scale(1);
  }
}

.kb-card.highlight-flash {
  animation: highlightFlash 0.6s ease-in-out 3;
  border-color: var(--td-brand-color) !important;
  box-shadow: 0 0 12px color-mix(in srgb, var(--td-brand-color) 30%, transparent) !important;
}

:deep(.t-dialog__position.t-dialog--top) {
  padding-top: 40vh !important;
}

.resource-list-header();

</style>

<style lang="less">
/* Dropdown menu styles are consolidated in @/assets/dropdown-menu.less */

.shared-detail-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-brand-color);
  font-size: var(--app-text-md);
  font-family: var(--app-font-family);
  cursor: pointer;
  transition: background var(--app-motion-base) ease, color var(--app-motion-base) ease;

  .t-icon {
    flex-shrink: 0;
  }

  &:hover {
    background: color-mix(in srgb, var(--td-brand-color) 8%, transparent);
    color: var(--td-brand-color);
  }
}

.shared-detail-drawer-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
}

.shared-detail-drawer {
  width: 360px;
  max-width: 90vw;
  height: 100%;
  background: var(--td-bg-color-container);
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  font-family: var(--app-font-family);
}

.shared-detail-drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--td-component-stroke);
  flex-shrink: 0;
}

.shared-detail-drawer-title {
  margin: 0;
  font-size: var(--app-text-2xl);
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.shared-detail-drawer-close {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background var(--app-motion-base) ease, color var(--app-motion-base) ease;

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }
}

.shared-detail-drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.shared-detail-drawer-body .shared-detail-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.shared-detail-drawer-body .shared-detail-label {
  font-size: var(--app-text-sm);
  color: var(--td-text-color-secondary);
  line-height: 1.4;
}

.shared-detail-drawer-body .shared-detail-value {
  font-size: var(--app-text-base);
  color: var(--td-text-color-primary);
  line-height: 1.5;
  word-break: break-word;

  &.shared-detail-source-type {
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  &.shared-detail-org {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
}

.shared-detail-drawer-body .shared-detail-org-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.shared-detail-drawer-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
  background: var(--td-bg-color-container);

  .go-to-kb-btn .t-button__text {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
}

.shared-detail-drawer-enter-active,
.shared-detail-drawer-leave-active {
  transition: opacity 0.25s ease;

  .shared-detail-drawer {
    transition: transform 0.25s ease;
  }
}

.shared-detail-drawer-enter-from,
.shared-detail-drawer-leave-to {
  opacity: 0;

  .shared-detail-drawer {
    transform: translateX(100%);
  }
}

</style>
