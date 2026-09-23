import { get, post, put, del } from "../../utils/request";

export type AgentType = 'rag-qa' | 'wiki-qa' | 'hybrid-rag-wiki' | 'data-analysis' | 'custom';

export interface QuestionSuggestionConfig {
  starters: {
    enabled: boolean;
    mode: 'curated' | 'knowledge' | 'hybrid';
    items: string[];
    count: number;
  };
  follow_ups: {
    enabled: boolean;
    mode: 'generated' | 'knowledge' | 'hybrid';
    count: number;
    model_id?: string;
    additional_instruction?: string;
    categories: Array<'clarify' | 'deepen' | 'action'>;
    max_context_turns: number;
    suppress_on_fallback: boolean;
    suppress_when_answer_asks_question: boolean;
    knowledge_fallback: boolean;
    allow_regenerate: boolean;
  };
}

export interface CustomAgentConfig {
  agent_mode?: 'quick-answer' | 'smart-reasoning';
  agent_type?: AgentType;
  system_prompt?: string;
  system_prompt_id?: string;
  context_template_id?: string;     // Inherit the referenced context template when text is empty
  context_template?: string;

  model_id?: string;
  rerank_model_id?: string;
  temperature?: number;
  max_completion_tokens?: number;
  thinking?: boolean;
  reasoning_effort?: string;
  citation_enabled?: boolean;

  max_iterations?: number;
  llm_call_timeout?: number;
  allowed_tools?: string[];
  reflection_enabled?: boolean;
  mcp_selection_mode?: 'all' | 'selected' | 'none';
  mcp_services?: string[];
  mcp_auth_wait_timeout?: number;

  skills_selection_mode?: 'all' | 'selected' | 'none';
  selected_skills?: string[];

  sandbox_config_id?: string;

  kb_selection_mode?: 'all' | 'selected' | 'none';
  knowledge_bases?: string[];
  retrieve_kb_only_when_mentioned?: boolean;

  image_upload_enabled?: boolean;
  vlm_model_id?: string;
  image_storage_provider?: string;
  audio_upload_enabled?: boolean;
  asr_model_id?: string;
  attachment_image_understanding?: boolean;
  attachment_ocr_max_pages?: number;
  attachment_parse_wait_timeout_sec?: number;

  chat_parser_engine_rules?: { file_types: string[]; engine: string }[];

  supported_file_types?: string[];

  web_search_enabled?: boolean;
  web_search_provider_id?: string;
  web_search_max_results?: number;

  multi_turn_enabled?: boolean;
  history_turns?: number;

  memory_enabled?: boolean;

  embedding_top_k?: number;
  keyword_threshold?: number;
  vector_threshold?: number;
  rerank_top_k?: number;
  rerank_threshold?: number;

  enable_query_expansion?: boolean;
  enable_rewrite?: boolean;
  rewrite_prompt_system?: string;
  rewrite_prompt_user?: string;
  fallback_strategy?: 'fixed' | 'model';
  fallback_response?: string;
  fallback_prompt?: string;
  intent_prompts?: Record<string, string>;

  welcome_message?: string;
  question_suggestions?: QuestionSuggestionConfig;
}

export interface CustomAgent {
  id: string;
  name: string;
  description?: string;
  avatar?: string;
  is_builtin: boolean;
  tenant_id?: number;
  created_by?: string;
  creator_name?: string;
  config: CustomAgentConfig;
  created_at?: string;
  updated_at?: string;
}

export interface CreateAgentRequest {
  name: string;
  description?: string;
  avatar?: string;
  config?: CustomAgentConfig;
}

export interface UpdateAgentRequest {
  name: string;
  description?: string;
  avatar?: string;
  config?: CustomAgentConfig;
}

export const BUILTIN_QUICK_ANSWER_ID = 'builtin-quick-answer';
export const BUILTIN_SMART_REASONING_ID = 'builtin-smart-reasoning';

export const AGENT_MODE_QUICK_ANSWER = 'quick-answer';
export const AGENT_MODE_SMART_REASONING = 'smart-reasoning';

// Deprecated: Use BUILTIN_QUICK_ANSWER_ID instead
export const BUILTIN_AGENT_NORMAL_ID = BUILTIN_QUICK_ANSWER_ID;
// Deprecated: Use BUILTIN_SMART_REASONING_ID instead
export const BUILTIN_AGENT_AGENT_ID = BUILTIN_SMART_REASONING_ID;

export function listAgents(params?: {
  /**
   * Optional creator filter; mirrors listKnowledgeBases. Built-in agents
   * (is_builtin=true) are always returned regardless of this filter so
   * the conversation dropdown never silently loses quick-answer /
   * smart-reasoning when a user picks "Created by me".
   */
  creator?: 'all' | 'mine' | 'others';
}) {
  const qs = params?.creator && params.creator !== 'all' ? `?creator=${params.creator}` : '';
  return get<{ data: CustomAgent[]; disabled_own_agent_ids?: string[] }>(`/api/v1/agents${qs}`);
}

export function getAgentById(id: string) {
  return get<{ data: CustomAgent }>(`/api/v1/agents/${id}`);
}

export function createAgent(data: CreateAgentRequest) {
  return post<{ data: CustomAgent }>('/api/v1/agents', data);
}

export function updateAgent(id: string, data: UpdateAgentRequest) {
  return put<{ data: CustomAgent }>(`/api/v1/agents/${id}`, data);
}

export function deleteAgent(id: string) {
  return del<{ success: boolean }>(`/api/v1/agents/${id}`);
}

export function copyAgent(id: string) {
  return post<{ data: CustomAgent }>(`/api/v1/agents/${id}/copy`);
}

export function isBuiltinAgent(agentId: string): boolean {
  return agentId.startsWith('builtin-');
}

export interface PlaceholderDefinition {
  name: string;
  label: string;
  description: string;
}

export interface PlaceholdersResponse {
  all: PlaceholderDefinition[];
  system_prompt: PlaceholderDefinition[];
  agent_system_prompt: PlaceholderDefinition[];
  context_template: PlaceholderDefinition[];
  rewrite_system_prompt: PlaceholderDefinition[];
  rewrite_prompt: PlaceholderDefinition[];
  fallback_prompt: PlaceholderDefinition[];
}

export function getPlaceholders() {
  return get<{ data: PlaceholdersResponse }>('/api/v1/agents/placeholders');
}


export interface AgentTypeKBFilter {
  any_of?: string[];
  all_of?: string[];
  none_of?: string[];
}

export interface KBCapabilities {
  vector: boolean;
  keyword: boolean;
  wiki: boolean;
  graph: boolean;
  faq: boolean;
}

export interface AgentTypePresetConfig {
  system_prompt_id?: string;
  temperature?: number;
  max_iterations?: number;
  allowed_tools?: string[];
  retain_retrieval_history?: boolean;
  faq_priority_enabled?: boolean;
  web_search_enabled?: boolean;
  supported_file_types?: string[];
  kb_selection_mode?: 'all' | 'selected' | 'none';
}

export interface AgentTypePresetI18n {
  label: string;
  description: string;
}

export interface AgentTypePreset {
  id: AgentType;
  i18n: Record<string, AgentTypePresetI18n>;
  config?: AgentTypePresetConfig;
  kb_filter?: AgentTypeKBFilter;
}

export function getAgentTypePresets() {
  return get<{ data: AgentTypePreset[] }>('/api/v1/agents/type-presets');
}


export interface IMChannel {
  id: string;
  tenant_id?: number;
  agent_id: string;
  platform: 'slack' | 'telegram' | 'mattermost';
  name: string;
  enabled: boolean;
  mode: 'webhook' | 'websocket' | 'longpoll';
  output_mode: 'stream' | 'full';
  session_mode?: 'user' | 'thread';
  knowledge_base_id?: string;
  credentials: Record<string, any>;
  created_at?: string;
  updated_at?: string;
}

export function listIMChannels(agentId: string) {
  return get<{ data: IMChannel[] }>(`/api/v1/agents/${agentId}/im-channels`);
}

// Tenant-wide overview row. Credentials are intentionally omitted — use
// listIMChannels(agentId) when you need to edit a specific channel.
export interface IMChannelOverview {
  id: string;
  tenant_id: number;
  agent_id: string;
  agent_name: string; // localized built-in name when the agent is built-in
  platform: IMChannel['platform'];
  name: string;
  enabled: boolean;
  mode: IMChannel['mode'];
  output_mode: IMChannel['output_mode'];
  session_mode?: IMChannel['session_mode'];
  bot_identity: string;
  created_at: string;
  updated_at: string;
}

export function listAllIMChannels() {
  return get<{ data: IMChannelOverview[] }>('/api/v1/im-channels');
}

export function createIMChannel(agentId: string, data: Partial<IMChannel>) {
  return post<{ data: IMChannel }>(`/api/v1/agents/${agentId}/im-channels`, data);
}

export function updateIMChannel(id: string, data: Partial<IMChannel>) {
  return put<{ data: IMChannel }>(`/api/v1/im-channels/${id}`, data);
}

export function deleteIMChannel(id: string) {
  return del<{ success: boolean }>(`/api/v1/im-channels/${id}`);
}

export function toggleIMChannel(id: string) {
  return post<{ data: IMChannel }>(`/api/v1/im-channels/${id}/toggle`);
}


export interface SuggestedQuestion {
  question: string;
  source: 'faq' | 'document' | 'agent_config' | 'wiki';
  knowledge_base_id?: string;
  knowledge_id?: string;
}

export function getSuggestedQuestions(
  agentId: string,
  params?: {
    knowledge_base_ids?: string[];
    knowledge_ids?: string[];
    tag_scopes?: Array<{ knowledge_base_id: string; tag_ids: string[] }>;
    limit?: number;
  }
) {
  const query = new URLSearchParams();
  if (params?.knowledge_base_ids?.length) query.set('knowledge_base_ids', params.knowledge_base_ids.join(','));
  if (params?.knowledge_ids?.length) query.set('knowledge_ids', params.knowledge_ids.join(','));
  if (params?.tag_scopes?.length) query.set('tag_scopes', JSON.stringify(params.tag_scopes));
  if (params?.limit) query.set('limit', String(params.limit));
  const qs = query.toString();
  return get<{ data: { questions: SuggestedQuestion[] } }>(`/api/v1/agents/${agentId}/suggested-questions${qs ? '?' + qs : ''}`);
}
