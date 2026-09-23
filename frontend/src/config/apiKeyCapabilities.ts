import type { TenantAPIKeyCapability } from '@/api/tenant'

export type ApiKeyCapabilityOption = {
  value: TenantAPIKeyCapability
  labelKey: string
  hintKey: string
}

export type ApiKeyCapabilityGroup = {
  key: string
  labelKey: string
  capabilities: ApiKeyCapabilityOption[]
}

export const TENANT_API_KEY_CAPABILITIES: TenantAPIKeyCapability[] = [
  'retrieve', 'chat', 'read_agents', 'ingest', 'manage_kbs',
  'message_history', 'manage_agents', 'manage_mcp_services',
  'manage_datasources', 'manage_models', 'manage_vector_stores',
  'manage_storage_backends', 'manage_web_search', 'manage_channels',
  'run_evaluations', 'manage_members', 'manage_spaces',
  'manage_tenant_settings',
]

export const SYSTEM_API_KEY_CAPABILITIES: TenantAPIKeyCapability[] = [
  'system_tenants_read', 'system_tenants_manage',
  'system_settings_read', 'system_settings_manage',
  'system_runtime_read', 'system_runtime_manage', 'system_audit_read',
]

export const DEFAULT_TENANT_API_KEY_CAPABILITIES = new Set<TenantAPIKeyCapability>([
  'retrieve', 'chat', 'read_agents',
])

export const KB_SCOPED_API_KEY_CAPABILITIES = new Set<TenantAPIKeyCapability>([
  'retrieve', 'chat', 'ingest', 'manage_kbs', 'manage_agents', 'manage_datasources',
])

export const TENANT_API_KEY_CAPABILITY_GROUPS: ApiKeyCapabilityGroup[] = [
  {
    key: 'knowledge',
    labelKey: 'Knowledge-base data',
    capabilities: [
      { value: 'retrieve', labelKey: 'Retrieve knowledge bases', hintKey: 'Read, query, and search data inside the selected knowledge-base scope. Does not create sessions or modify content.' },
      { value: 'chat', labelKey: 'Chat', hintKey: 'Let this key hold conversations and manage its own sessions. Does not modify knowledge-base content.' },
      { value: 'ingest', labelKey: 'Write KB content', hintKey: 'Let this key write content into its allowed knowledge bases (upload documents, edit chunks/FAQ/tags/wiki). It cannot create knowledge bases or agents, cannot clear a knowledge base, and stays bounded to the selected knowledge bases.' },
      { value: 'manage_kbs', labelKey: 'Manage knowledge bases', hintKey: 'Let this key manage the full knowledge-base lifecycle: create, copy, update, and delete knowledge bases and change their initialization/configuration. Operations on existing knowledge bases (copy/update/delete) stay bounded by the selected scope; creating a new knowledge base is unrestricted (the new KB belongs to this space).' },
      { value: 'message_history', labelKey: 'Message history', hintKey: 'Let this key search workspace chat history and read chat-history stats. It does not grant workspace configuration access.' },
    ],
  },
  {
    key: 'automation',
    labelKey: 'Agents and integrations',
    capabilities: [
      { value: 'read_agents', labelKey: 'Read agents', hintKey: 'List agents, read agent details, presets, and suggested questions. Does not allow chat or agent edits.' },
      { value: 'manage_agents', labelKey: 'Manage agents', hintKey: 'Let this key create, update, delete and copy agents. Agent config can carry sensitive model/MCP bindings, so this is off by default — enable only when needed.' },
      { value: 'manage_mcp_services', labelKey: 'Manage MCP services', hintKey: 'Manage MCP services, credentials, tool approval policies, and OAuth state for this principal.' },
      { value: 'manage_datasources', labelKey: 'Manage data sources', hintKey: 'Manage data-source connectors, credentials, resource selection, and sync jobs. Knowledge-base scope still applies when a source is bound to a KB.' },
    ],
  },
  {
    key: 'collaboration',
    labelKey: 'Members and spaces',
    capabilities: [
      { value: 'manage_members', labelKey: 'Manage members', hintKey: 'List and manage workspace members, roles, invitations, and invite links. Does not include API key management, workspace deletion, or ownership transfer.' },
      { value: 'manage_spaces', labelKey: 'Manage spaces', hintKey: 'Manage organization spaces, join flows, space membership, invitations, and shared-space visibility. Does not grant KB or agent share management.' },
    ],
  },
  {
    key: 'tenant',
    labelKey: 'Workspace configuration',
    capabilities: [
      { value: 'manage_models', labelKey: 'Manage models', hintKey: 'Manage model definitions, credentials, and connectivity checks.' },
      { value: 'manage_vector_stores', labelKey: 'Manage retrieval infrastructure', hintKey: 'Manage vector-store configuration plus parser, document reader, and storage engine connectivity checks.' },
      { value: 'manage_storage_backends', labelKey: 'Manage storage backends', hintKey: 'Manage object/file storage backend instances (e.g. S3-compatible or local file storage): their CRUD lifecycle, connectivity tests, and the workspace default selection.' },
      { value: 'manage_web_search', labelKey: 'Manage web search', hintKey: 'Manage web-search provider configurations, credentials, and connection tests.' },
      { value: 'manage_channels', labelKey: 'Manage channels', hintKey: 'Manage agent embed channels and IM channels.' },
      { value: 'run_evaluations', labelKey: 'Run evaluations', hintKey: 'Run evaluation jobs and read evaluation results.' },
      { value: 'manage_tenant_settings', labelKey: 'Manage workspace settings', hintKey: 'Read and update workspace-level integration settings such as API end-user identity mode, request header configuration, and workspace KV settings. Does not include API key management, member management, workspace deletion, or ownership transfer.' },
    ],
  },
]

export const SYSTEM_API_KEY_CAPABILITY_GROUP: ApiKeyCapabilityGroup = {
  key: 'system',
  labelKey: 'Platform control plane',
  capabilities: [
    { value: 'system_tenants_read', labelKey: 'Read workspaces', hintKey: 'List, search, and inspect every workspace.' },
    { value: 'system_tenants_manage', labelKey: 'Manage workspaces', hintKey: 'Create, update, delete workspaces and apply global workspace settings.' },
    { value: 'system_settings_read', labelKey: 'Read system settings', hintKey: 'Read platform runtime settings.' },
    { value: 'system_settings_manage', labelKey: 'Manage system settings', hintKey: 'Update and reset platform runtime settings.' },
    { value: 'system_runtime_read', labelKey: 'Read runtime', hintKey: 'Inspect task queues and task details.' },
    { value: 'system_runtime_manage', labelKey: 'Manage runtime', hintKey: 'Retry, run, cancel, or delete runtime tasks.' },
    { value: 'system_audit_read', labelKey: 'Read system audit', hintKey: 'Read platform audit events.' },
  ],
}

export const PLATFORM_API_KEY_CAPABILITY_GROUPS: ApiKeyCapabilityGroup[] = [
  SYSTEM_API_KEY_CAPABILITY_GROUP,
  ...TENANT_API_KEY_CAPABILITY_GROUPS,
]
