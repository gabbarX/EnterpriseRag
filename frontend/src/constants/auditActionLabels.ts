/**
 * Display labels for the dot-namespaced audit action strings the API returns.
 *
 * Action values mirror `internal/types/audit_log.go` (e.g. `kb.created`,
 * `rbac.member_added`). Keep these maps in sync when a new AuditAction
 * constant is added in Go; an unknown action falls back to its raw value.
 */

/** Workspace membership / invitation audit events (workspace members drawer). */
export const TENANT_MEMBER_AUDIT_ACTION_LABELS: Record<string, string> = {
  'rbac.member_added': 'Member added',
  'rbac.member_removed': 'Member removed',
  'rbac.member_role_changed': 'Role changed',
  'rbac.member_left': 'Member left',
  'rbac.access_denied': 'Access denied',
  'rbac.invitation_sent': 'Invitation sent',
  'rbac.invitation_accepted': 'Invitation accepted',
  'rbac.invitation_declined': 'Invitation declined',
  'rbac.invitation_revoked': 'Invitation revoked',
  'rbac.invitation_expired': 'Invitation expired',
}

/** Platform control-plane audit events (system settings, audit tab). */
export const SYSTEM_AUDIT_ACTION_LABELS: Record<string, string> = {
  'system.setting_changed': 'System setting changed',
  'system.admin_promoted': 'System admin granted',
  'system.api_key_created': 'Platform API key created',
  'system.api_key_revoked': 'Platform API key revoked',
  'system.admin_revoked': 'System admin revoked',
  'system.user_password_reset': 'User password reset',
  'system.user_created': 'User created',
  'system.queue_task_retried': 'Failed task run again',
  'system.queue_task_deleted': 'Failed task record cleared',
  'system.queue_task_run_now': 'Queue task run now',
  'system.queue_task_cancelled': 'Queue task cancelled',
  'system.queue_archived_purged': 'All failed tasks cleared',
}

/** Knowledge-base activity feed events (knowledge base settings, activity tab). */
export const KB_ACTIVITY_ACTION_LABELS: Record<string, string> = {
  'kb.created': 'Knowledge base created',
  'kb.updated': 'Knowledge base updated',
  'kb.deleted': 'Knowledge base deleted',
  'kb.duplicated': 'Knowledge base duplicated',
  'kb.clone_started': 'Clone started',
  'kb.clone_completed': 'Clone completed',
  'kb.clone_failed': 'Clone failed',
  'knowledge.created': 'Knowledge added',
  'knowledge.updated': 'Knowledge updated',
  'knowledge.deleted': 'Knowledge deleted',
  'knowledge.batch_deleted': 'Knowledge batch deleted',
  'knowledge.reparse_started': 'Reparse started',
  'knowledge.parse_canceled': 'Parsing canceled',
  'knowledge.move_started': 'Knowledge move started',
  'knowledge.move_completed': 'Knowledge move completed',
  'knowledge.move_failed': 'Knowledge move failed',
  'tag.created': 'Tag created',
  'tag.updated': 'Tag updated',
  'tag.deleted': 'Tag deleted',
  'datasource.created': 'Data source created',
  'datasource.updated': 'Data source updated',
  'datasource.deleted': 'Data source deleted',
  'datasource.sync_started': 'Data source sync started',
  'datasource.sync_completed': 'Data source sync completed',
  'datasource.sync_failed': 'Data source sync failed',
  'datasource.paused': 'Data source paused',
  'datasource.resumed': 'Data source resumed',
  'kb.share_added': 'Share added',
  'kb.share_permission_changed': 'Share permission changed',
  'kb.share_removed': 'Share removed',
  'wiki.content_changed': 'Wiki content changed',
  'faq.import_started': 'FAQ import started',
  'faq.import_completed': 'FAQ import completed',
  'faq.import_failed': 'FAQ import failed',
}

/** Resolve an audit action to its display label, falling back to the raw value. */
export function auditActionLabel(labels: Record<string, string>, action: string): string {
  return labels[action] ?? action
}
