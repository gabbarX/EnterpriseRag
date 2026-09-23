<template>
  <span class="resource-origin-badge" :class="variantClass" :title="tooltipText">
    <t-icon :name="iconName" size="12px" class="badge-icon" />
    <span class="badge-text">{{ displayText }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Icon as TIcon } from 'tdesign-vue-next'
import { useAuthStore } from '@/stores/auth'

/**
 * ResourceOriginBadge – a unified, compact label that explains *where* a
 * / org_name pills scattered across KnowledgeBaseList and AgentList. The
 * variants below cover the five origin shapes the list views actually
 * surface; future origins (e.g. "system" / "imported") should add a new
 * variant rather than re-using one of these.
 *
 * Variants:
 *  - mine        : created by the current user in the current tenant
 *  - tenant      : owned by the current tenant but created by someone else
 *                  — label shows tenant name; use when context doesn't say
 *  - creator     : same data shape as `tenant`, but the surrounding section
 *                  the badge only carries the creator name to avoid the
 *                  every card. Falls back to the i18n label when the
 *                  creator name is unknown.
 *  - space       : reached through a cross-tenant space (organization)
 *  - shared      : cross-tenant share without a useful org name to show
 *
 * variant, or to drive the visible label of the `creator` variant; omit it
 * for the `mine` / `space` / `shared` variants where the subject is implicit.
 */
const props = withDefaults(
  defineProps<{
    variant: 'mine' | 'tenant' | 'creator' | 'space' | 'shared'
    /** Used in `space` variant — the organization (space) display name. */
    spaceName?: string
    /** Optional creator display name, surfaces in tooltip for `tenant` variant. */
    creatorName?: string
    /** Optional source tenant name, surfaces in tooltip for cross-tenant. */
    sourceTenantName?: string
  }>(),
  { spaceName: '', creatorName: '', sourceTenantName: '' }
)

const authStore = useAuthStore()

const iconName = computed(() => {
  switch (props.variant) {
    case 'mine':
      return 'user'
    case 'tenant':
      return 'usergroup'
    case 'creator':
      return 'user'
    case 'space':
      return 'building'
    case 'shared':
      return 'share'
    default:
      return 'usergroup'
  }
})

const variantClass = computed(() => `origin-${props.variant}`)

const displayText = computed(() => {
  switch (props.variant) {
    case 'mine':
      return 'Mine'
    case 'tenant':
      // Prefer the tenant name when known so the badge says where the
      // resource lives, not a vague "tenant" label. Falls back to i18n.
      return authStore.currentTenantName || 'Workspace'
    case 'creator':
      // show who created it. Fall back to a generic label when the user
      return props.creatorName || 'Workspace'
    case 'space':
      return props.spaceName || 'Space'
    case 'shared':
      return props.sourceTenantName || 'External'
    default:
      return ''
  }
})

const tooltipText = computed(() => {
  switch (props.variant) {
    case 'mine':
      return 'Created by you'
    case 'tenant':
      if (props.creatorName) {
        return `Created by ${props.creatorName}`
      }
      return 'Created by another member of this workspace'
    case 'creator':
      if (props.creatorName) {
        return `Created by ${props.creatorName}`
      }
      return 'Created by another member of this workspace'
    case 'space':
      if (props.sourceTenantName) {
        return `Shared via space "${props.spaceName}" · from ${props.sourceTenantName}`
      }
      return `Shared via space "${props.spaceName}"`
    case 'shared':
      return 'Accessed from an external workspace via a shared space'
    default:
      return ''
  }
})
</script>

<style scoped lang="less">
.resource-origin-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 6px;
  border-radius: var(--app-radius-md);
  font-size: var(--app-text-xs);
  line-height: 1.4;
  font-weight: 500;
  max-width: 140px;

  .badge-icon {
    flex-shrink: 0;
  }

  .badge-text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &.origin-mine {
    color: var(--td-brand-color);
    background: var(--td-success-color-light);
  }

  &.origin-tenant {
    color: var(--td-text-color-secondary);
    background: var(--td-bg-color-secondarycontainer);
  }

  &.origin-creator {
    color: var(--td-text-color-secondary);
    background: var(--td-bg-color-secondarycontainer);
  }

  &.origin-space {
    color: var(--td-warning-color-7);
    background: var(--td-warning-color-1);
  }

  &.origin-shared {
    color: var(--td-text-color-secondary);
    background: var(--td-bg-color-secondarycontainer);
  }
}
</style>
