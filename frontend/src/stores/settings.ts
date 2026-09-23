import { defineStore } from "pinia";
import { nextTick } from "vue";
import { BUILTIN_QUICK_ANSWER_ID, BUILTIN_SMART_REASONING_ID } from "@/api/agent";
import { getApiBaseUrl } from "@/utils/api-base";
import { isAgentStreamAgentId } from "@/utils/agent-mode";
import { loadAndReconcileSettings } from "@/stores/settingsStorage";

interface Settings {
  endpoint: string;
  apiKey: string;
  knowledgeBaseId: string;
  isAgentEnabled: boolean;
  agentConfig: AgentConfig;
  selectedKnowledgeBases: string[];
  selectedFiles: string[];
  selectedFileKbMap: Record<string, string>;
  selectedTags: Array<{ id: string; name: string; kbId: string; kbName?: string }>;
  selectedMCPServices: string[];
  selectedSkills: string[];
  selectedTools?: string[];
  modelConfig: ModelConfig;
  ollamaConfig: OllamaConfig;
  localBrowserEnabled: boolean; // Explicit source preference; composer activates it only while the extension is online
  webSearchEnabled: boolean;
  conversationModels: ConversationModels;
  selectedAgentId: string;
  selectedAgentSourceTenantId: string | null;
  autoCheckUpdate?: boolean;
}

interface AgentConfig {
  maxIterations: number;
  temperature: number;
  allowedTools: string[];
  system_prompt?: string;  // Unified system prompt (uses {{web_search_status}} placeholder)
}

interface ConversationModels {
  summaryModelId: string;
  rerankModelId: string;
  selectedChatModelId: string;
}

interface ModelItem {
  id: string;
  name: string;
  source: 'local' | 'remote';
  modelName: string;
  baseUrl?: string;
  apiKey?: string;
  dimension?: number;
  interfaceType?: 'ollama' | 'openai';
  isDefault?: boolean;
}

interface ModelConfig {
  chatModels: ModelItem[];
  embeddingModels: ModelItem[];
  rerankModels: ModelItem[];
  vllmModels: ModelItem[];
}

interface OllamaConfig {
  baseUrl: string;
  enabled: boolean;
}

const defaultSettings: Settings = {
  endpoint: getApiBaseUrl(),
  apiKey: "",
  knowledgeBaseId: "",
  isAgentEnabled: false,
  agentConfig: {
    maxIterations: 5,
    temperature: 0.7,
    allowedTools: [],
    system_prompt: "",
  },
  selectedKnowledgeBases: [],
  selectedFiles: [],
  selectedFileKbMap: {},
  selectedTags: [],
  selectedMCPServices: [],
  selectedSkills: [],
  modelConfig: {
    chatModels: [],
    embeddingModels: [],
    rerankModels: [],
    vllmModels: []
  },
  ollamaConfig: {
    baseUrl: "http://localhost:11434",
    enabled: true
  },
  localBrowserEnabled: false,
  webSearchEnabled: false,
  conversationModels: {
    summaryModelId: "",
    rerankModelId: "",
    selectedChatModelId: "",
  },
  selectedAgentId: BUILTIN_QUICK_ANSWER_ID,
  selectedAgentSourceTenantId: null as string | null,
  autoCheckUpdate: true,
};

export const useSettingsStore = defineStore("settings", {
  state: () => ({
    settings: loadAndReconcileSettings(defaultSettings),
    _defaultsSnapshot: null as Settings | null,
    _isApplyingSessionState: false,
  }),

  getters: {
    isAgentEnabled: (state) => state.settings.isAgentEnabled || false,

    isQuickAnswerMode: (state) =>
      (state.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID) === BUILTIN_QUICK_ANSWER_ID,

    isAgentStreamMode: (state) =>
      isAgentStreamAgentId(
        state.settings.selectedAgentId,
        state.settings.isAgentEnabled || false,
      ),
    
    isAgentReady: (state) => {
      const config = state.settings.agentConfig || defaultSettings.agentConfig
      const models = state.settings.conversationModels || defaultSettings.conversationModels
      return Boolean(
        config.allowedTools && config.allowedTools.length > 0 &&
        models.summaryModelId && models.summaryModelId.trim() !== '' &&
        models.rerankModelId && models.rerankModelId.trim() !== ''
      )
    },
    
    isNormalModeReady: (state) => {
      const models = state.settings.conversationModels || defaultSettings.conversationModels
      return Boolean(
        models.summaryModelId && models.summaryModelId.trim() !== '' &&
        models.rerankModelId && models.rerankModelId.trim() !== ''
      )
    },
    
    agentConfig: (state) => state.settings.agentConfig || defaultSettings.agentConfig,

    conversationModels: (state) => state.settings.conversationModels || defaultSettings.conversationModels,
    
    modelConfig: (state) => state.settings.modelConfig || defaultSettings.modelConfig,
    
    isLocalBrowserEnabled: (state) => state.settings.localBrowserEnabled === true,
    isWebSearchEnabled: (state) => state.settings.webSearchEnabled || false,
    
    isAutoCheckUpdateEnabled: (state) => state.settings.autoCheckUpdate ?? true,

    selectedAgentId: (state) => state.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID,
    selectedAgentSourceTenantId: (state) => state.settings.selectedAgentSourceTenantId ?? null,
  },

  actions: {
    saveSettings(settings: Settings) {
      this.settings = { ...settings };
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    getSettings(): Settings {
      return this.settings;
    },

    getEndpoint(): string {
      return this.settings.endpoint || defaultSettings.endpoint;
    },

    getApiKey(): string {
      return this.settings.apiKey;
    },

    getKnowledgeBaseId(): string {
      return this.settings.knowledgeBaseId;
    },
    
    toggleAgent(enabled: boolean) {
      this.settings.isAgentEnabled = enabled;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    updateAgentConfig(config: Partial<AgentConfig>) {
      this.settings.agentConfig = { ...this.settings.agentConfig, ...config };
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    updateConversationModels(models: Partial<ConversationModels>) {
      const current = this.settings.conversationModels || defaultSettings.conversationModels;
      this.settings.conversationModels = { ...current, ...models };
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    updateModelConfig(config: Partial<ModelConfig>) {
      this.settings.modelConfig = { ...this.settings.modelConfig, ...config };
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    addModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', model: ModelItem) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      if (model.isDefault) {
        models.forEach(m => m.isDefault = false);
      }
      if (models.length === 0) {
        model.isDefault = true;
      }
      models.push(model);
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    updateModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string, updates: Partial<ModelItem>) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      const index = models.findIndex(m => m.id === modelId);
      if (index !== -1) {
        if (updates.isDefault) {
          models.forEach(m => m.isDefault = false);
        }
        models[index] = { ...models[index], ...updates };
        this.settings.modelConfig[key] = models as any;
        localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
      }
    },
    
    deleteModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string) {
      const key = `${type}Models` as keyof ModelConfig;
      let models = [...this.settings.modelConfig[key]] as ModelItem[];
      const deletedModel = models.find(m => m.id === modelId);
      models = models.filter(m => m.id !== modelId);
      if (deletedModel?.isDefault && models.length > 0) {
        models[0].isDefault = true;
      }
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    setDefaultModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      models.forEach(m => m.isDefault = (m.id === modelId));
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    updateOllamaConfig(config: Partial<OllamaConfig>) {
      this.settings.ollamaConfig = { ...this.settings.ollamaConfig, ...config };
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    selectKnowledgeBases(kbIds: string[]) {
      this.settings.selectedKnowledgeBases = kbIds;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    addKnowledgeBase(kbId: string) {
      if (!this.settings.selectedKnowledgeBases.includes(kbId)) {
        this.settings.selectedKnowledgeBases.push(kbId);
        localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
      }
    },
    
    removeKnowledgeBase(kbId: string) {
      this.settings.selectedKnowledgeBases = 
        this.settings.selectedKnowledgeBases.filter((id: string) => id !== kbId);
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    clearKnowledgeBases() {
      this.settings.selectedKnowledgeBases = [];
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    getSelectedKnowledgeBases(): string[] {
      return this.settings.selectedKnowledgeBases || [];
    },
    
    toggleLocalBrowser(enabled: boolean) {
      this.settings.localBrowserEnabled = enabled;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    toggleWebSearch(enabled: boolean) {
      this.settings.webSearchEnabled = enabled;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    toggleAutoCheckUpdate(enabled: boolean) {
      this.settings.autoCheckUpdate = enabled;
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    // File selection actions
    addFile(fileId: string) {
      if (!this.settings.selectedFiles) this.settings.selectedFiles = [];
      if (!this.settings.selectedFiles.includes(fileId)) {
        this.settings.selectedFiles.push(fileId);
        localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
      }
    },

    removeFile(fileId: string) {
      if (!this.settings.selectedFiles) return;
      this.settings.selectedFiles = this.settings.selectedFiles.filter((id: string) => id !== fileId);
      if (this.settings.selectedFileKbMap) delete this.settings.selectedFileKbMap[fileId];
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    clearFiles() {
      this.settings.selectedFiles = [];
      this.settings.selectedFileKbMap = {};
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    addTag(tag: { id: string; name: string; kbId: string; kbName?: string }) {
      if (!this.settings.selectedTags) this.settings.selectedTags = [];
      if (!this.settings.selectedTags.some(t => t.id === tag.id && t.kbId === tag.kbId)) {
        this.settings.selectedTags.push(tag);
        localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
      }
    },

    removeTag(tagId: string, kbId?: string) {
      if (!this.settings.selectedTags) return;
      this.settings.selectedTags = this.settings.selectedTags.filter(t => !(t.id === tagId && (!kbId || t.kbId === kbId)));
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    clearTags() {
      this.settings.selectedTags = [];
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    addMCPService(serviceId: string) {
      if (!this.settings.selectedMCPServices) this.settings.selectedMCPServices = [];
      if (!this.settings.selectedMCPServices.includes(serviceId)) {
        this.settings.selectedMCPServices.push(serviceId);
        localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
      }
    },

    removeMCPService(serviceId: string) {
      if (!this.settings.selectedMCPServices) return;
      this.settings.selectedMCPServices = this.settings.selectedMCPServices.filter(id => id !== serviceId);
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    addSkill(skillName: string) {
      if (!this.settings.selectedSkills) this.settings.selectedSkills = [];
      if (!this.settings.selectedSkills.includes(skillName)) {
        this.settings.selectedSkills.push(skillName);
        localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
      }
    },

    removeSkill(skillName: string) {
      if (!this.settings.selectedSkills) return;
      this.settings.selectedSkills = this.settings.selectedSkills.filter(name => name !== skillName);
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    setFileKbMap(updates: Record<string, string>) {
      if (!this.settings.selectedFileKbMap) this.settings.selectedFileKbMap = {};
      Object.assign(this.settings.selectedFileKbMap, updates);
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },

    removeFileKbId(fileId: string) {
      if (this.settings.selectedFileKbMap) delete this.settings.selectedFileKbMap[fileId];
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    getSelectedFiles(): string[] {
      return this.settings.selectedFiles || [];
    },

    /**
     * Scope for suggested-questions API (KB / file / tag @mentions).
     * Leave `limit` undefined to let the backend apply the agent's configured
     * starter count; pass a number only to request a specific count.
     */
    getSuggestedQuestionsParams(limit?: number) {
      const selectedKBs = this.getSelectedKnowledgeBases();
      const selectedFiles = this.getSelectedFiles();
      const tags = this.settings.selectedTags || [];
      const tagScopes = Object.entries(tags.reduce<Record<string, string[]>>((scopes, tag) => {
        if (!tag.id || !tag.kbId) return scopes;
        (scopes[tag.kbId] ||= []).push(tag.id);
        return scopes;
      }, {})).map(([knowledge_base_id, ids]) => ({
        knowledge_base_id,
        tag_ids: [...new Set(ids)],
      }));
      return {
        // A tag's parent KB is only an ownership hint, not an explicit whole-KB
        // selection. Keep it in tag_scopes so the backend cannot widen a tag to
        // every document in that KB.
        knowledge_base_ids: selectedKBs.length > 0 ? selectedKBs : undefined,
        knowledge_ids: selectedFiles.length > 0 ? selectedFiles : undefined,
        tag_scopes: tagScopes.length > 0 ? tagScopes : undefined,
        limit,
      };
    },
    
    selectAgent(agentId: string, sourceTenantId?: string | null) {
      this.settings.selectedAgentId = agentId;
      this.settings.selectedAgentSourceTenantId = (sourceTenantId != null && sourceTenantId !== "") ? sourceTenantId : null;
      this.settings.webSearchEnabled = false;
      this.settings.localBrowserEnabled = false;
      if (agentId === BUILTIN_QUICK_ANSWER_ID) {
        this.settings.isAgentEnabled = false;
      } else if (agentId === BUILTIN_SMART_REASONING_ID) {
        this.settings.isAgentEnabled = true;
      }
      
      this.settings.selectedKnowledgeBases = [];
      this.settings.selectedFiles = [];
      this.settings.selectedFileKbMap = {};
      this.settings.selectedTags = [];
      this.settings.selectedMCPServices = [];
      this.settings.selectedSkills = [];
      localStorage.setItem("EnterpriseRag_settings", JSON.stringify(this.settings));
    },
    
    getSelectedAgentId(): string {
      return this.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID;
    },

    //

    // The composer's agent / model / KB / web-search / MCP choices live in this store and
    // are shared across sessions. But what the user wants is: opening an old session shows
    // the state that session's requests were actually made with.
    // Strategy: on entering a session, stash "the current global defaults" in a
    // non-persisted `_defaultsSnapshot` field, then overwrite the store from
    // session.last_request_state; on leaving, restore from the snapshot.
    // The snapshot is never written to localStorage, because it only means anything while
    // the route sits inside an old session; a page refresh is equivalent to re-entering the
    // session (snapshot again, then overwrite), so the user's global defaults are not lost.

    // Take a snapshot of the current settings as "the defaults to restore on leaving".
    // An existing snapshot is never overwritten, so switching between sessions (B -> B')
    // cannot mistake an already-restored store for the defaults.
    snapshotAsDefaultsIfNeeded() {
      if (this._defaultsSnapshot) return;
      this._defaultsSnapshot = JSON.parse(JSON.stringify(this.settings));
    },

    restoreDefaultsIfSnapshotted() {
      if (!this._defaultsSnapshot) return;
      this.settings = this._defaultsSnapshot;
      this._defaultsSnapshot = null;
    },

    hydrateSessionInputState(state: SessionLastRequestStatePayload | null | undefined, preserveDraft = false) {
      if (!state || preserveDraft) return;
      this.snapshotAsDefaultsIfNeeded();
      this.applyLastRequestState(state);
    },

    applyLastRequestState(state: SessionLastRequestStatePayload | null | undefined) {
      if (!state) return;
      this._isApplyingSessionState = true;
      try {
        if (typeof state.agent_enabled === "boolean") {
          this.settings.isAgentEnabled = state.agent_enabled;
        }
        if (typeof state.agent_id === "string" && state.agent_id) {
          this.settings.selectedAgentId = state.agent_id;
        }
        if (state.model_id !== undefined) {
          const current = this.settings.conversationModels || defaultSettings.conversationModels;
          this.settings.conversationModels = { ...current, selectedChatModelId: state.model_id || "" };
        }
        if (Array.isArray(state.knowledge_base_ids)) {
          this.settings.selectedKnowledgeBases = [...state.knowledge_base_ids];
        }
        if (Array.isArray(state.knowledge_ids)) {
          this.settings.selectedFiles = [...state.knowledge_ids];
        }
        if (Array.isArray(state.mentioned_items)) {
          const fromMentions = state.mentioned_items
            .filter(item => item.type === "tag" && item.id && item.kb_id)
            .map(item => ({ id: item.id, name: item.name || item.id, kbId: item.kb_id!, kbName: item.kb_name }));
          const covered = new Set(fromMentions.map(t => t.id));
          const orphanTagIds = (state.tag_ids || []).filter(id => id && !covered.has(id));
          if (orphanTagIds.length > 0 && Array.isArray(state.knowledge_base_ids) && state.knowledge_base_ids.length === 1) {
            const kbId = state.knowledge_base_ids[0];
            orphanTagIds.forEach(id => {
              fromMentions.push({ id, name: id, kbId, kbName: undefined });
            });
          }
          this.settings.selectedTags = fromMentions;
        } else if (Array.isArray(state.tag_ids)) {
          const existing = this.settings.selectedTags || [];
          this.settings.selectedTags = existing.filter(tag => state.tag_ids?.includes(tag.id));
        }
        if (Array.isArray(state.mcp_service_ids)) {
          this.settings.selectedMCPServices = [...state.mcp_service_ids];
        } else if (Array.isArray(state.mentioned_items)) {
          this.settings.selectedMCPServices = state.mentioned_items
            .filter(item => item.type === "mcp" && item.id)
            .map(item => item.id);
        }
        if (Array.isArray(state.skill_names)) {
          this.settings.selectedSkills = [...state.skill_names];
        } else if (Array.isArray(state.mentioned_items)) {
          this.settings.selectedSkills = state.mentioned_items
            .filter(item => item.type === "skill" && item.id)
            .map(item => item.skill_name || item.id);
        }
        this.settings.localBrowserEnabled = state.local_browser_enabled === true;
        if (typeof state.web_search_enabled === "boolean") {
          this.settings.webSearchEnabled = state.web_search_enabled;
        }
      } finally {
        nextTick(() => {
          this._isApplyingSessionState = false;
        });
      }
    },
  },
});

export interface SessionLastRequestStatePayload {
  agent_id?: string;
  agent_enabled?: boolean;
  model_id?: string;
  knowledge_base_ids?: string[];
  knowledge_ids?: string[];
  tag_ids?: string[];
  mcp_service_ids?: string[];
  skill_names?: string[];
  mentioned_items?: Array<{
    id: string;
    name?: string;
    type: string;
    kb_id?: string;
    kb_name?: string;
    skill_name?: string;
  }>;
  local_browser_enabled?: boolean;
  web_search_enabled?: boolean;
}
