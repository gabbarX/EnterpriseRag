import { get, post, postUpload, put, del } from '../../utils/request';
import { ModelInUseError, modelInUseErrorFromRequest } from './modelUsage'

export * from './modelUsage'

const t = (key: string) => key

// Protocol-neutral thinking level. Mirrors internal/models/api.ReasoningEffort.
export type ReasoningEffortLevel =
  | 'off'
  | 'auto'
  | 'minimal'
  | 'low'
  | 'medium'
  | 'high'
  | 'xhigh'
  | 'max'

// Catalog view of a saved chat/VLM model. Mirrors internal/models/catalog.Capabilities.
// thinking_levels: empty array = the model cannot be asked to think.
export interface ModelCapabilities {
  provider: string;
  api: string;
  cataloged: boolean;
  reasoning: boolean;
  thinking_levels: ReasoningEffortLevel[];
  thinking_format: string;
  input?: string[];
  context_window?: number;
  max_output_tokens?: number;
  max_tokens_field?: string;
}

// Per-row override of a catalog entry. Mirrors internal/types.ModelSpecOverride.
// compat is the flat, protocol-specific object documented in
// internal/models/catalog/compat.go (free-form JSON).
export interface ModelSpecOverride {
  api?: string;
  reasoning?: boolean;
  input?: string[];
  context_window?: number;
  max_output_tokens?: number;
  thinking_levels?: Record<string, string | null>;
  compat?: Record<string, unknown>;
}

export interface ModelConfig {
  id?: string;
  tenant_id?: number;
  name: string;
  display_name?: string;
  type: 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM' | 'ASR';
  source: 'local' | 'remote';
  description?: string;
  parameters: {
    base_url?: string;
    api_key?: string;
    provider?: string; // Provider identifier: openai, anthropic, gemini, openrouter, generic
    embedding_parameters?: {
      dimension?: number;
      truncate_prompt_tokens?: number;
      supports_dimension_override?: boolean;
    };
    interface_type?: 'ollama' | 'openai';
    parameter_size?: string;
    extra_config?: Record<string, string>; // Provider-specific configuration
    custom_headers?: Record<string, string>;
    supports_vision?: boolean; // Whether the model accepts image/multimodal input
    context_window?: number;
    max_output_tokens?: number;
    max_concurrency?: number;
    app_id?: string;
    // Secret fields (api_key, app_secret) are never returned by the server in
    // this shape — they live behind the /credentials subresource. They are
    // kept on the type so create-mode payloads can still carry them in the
    // initial POST body.
    app_secret?: string;
    // Per-model catalog override (protocol, limits, protocol compat knobs).
    spec?: ModelSpecOverride;
  };
  // Catalog view (chat / VLM remote models only): protocol, thinking levels,
  // context window. Computed by the backend from provider + name + overrides.
  capabilities?: ModelCapabilities;
  is_default?: boolean;
  is_builtin?: boolean;
  status?: string;
  // Per-field configured? metadata from the main response. For builtin
  // models it is returned only to system administrators.
  credentials?: Record<ModelCredentialField, { configured: boolean }>;
  created_at?: string;
  updated_at?: string;
  deleted_at?: string | null;
}

export function createModel(data: ModelConfig): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    post('/api/v1/models', data)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || 'Failed to create model'));
        }
      })
      .catch((error: any) => {
        console.error('Failed to create model:', error);
        reject(error);
      });
  });
}

export function listModels(type?: string): Promise<ModelConfig[]> {
  return new Promise((resolve, reject) => {
    const url = `/api/v1/models`;
    get(url)
      .then((response: any) => {
        if (response.success && response.data) {
          if (type) {
            response.data = response.data.filter((item: ModelConfig) => item.type === type);
          }
          resolve(response.data);
        } else {
          resolve([]);
        }
      })
      .catch((error: any) => {
        console.error('Failed to list models:', error);
        reject(error);
      });
  });
}

export function getModel(id: string): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    get(`/api/v1/models/${id}`)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || 'Failed to get model'));
        }
      })
      .catch((error: any) => {
        console.error('Failed to get model:', error);
        reject(error);
      });
  });
}

export function updateModel(id: string, data: Partial<ModelConfig>): Promise<ModelConfig> {
  return new Promise((resolve, reject) => {
    put(`/api/v1/models/${id}`, data)
      .then((response: any) => {
        if (response.success && response.data) {
          resolve(response.data);
        } else {
          reject(new Error(response.message || 'Failed to update model'));
        }
      })
      .catch((error: any) => {
        console.error('Failed to update model:', error);
        reject(error);
      });
  });
}

export function deleteModel(id: string): Promise<void> {
  return new Promise((resolve, reject) => {
    del(`/api/v1/models/${id}`)
      .then((response: any) => {
        if (response.success) {
          resolve();
        } else {
          const conflict = modelInUseErrorFromRequest(response)
          if (conflict) {
            reject(conflict)
            return
          }
          reject(new Error(response.message || 'Failed to delete model'));
        }
      })
      .catch((error: any) => {
        console.error('Failed to delete model:', error);
        if (error instanceof ModelInUseError) {
          reject(error)
          return
        }
        const conflict = modelInUseErrorFromRequest(error)
        if (conflict) {
          reject(conflict)
          return
        }
        reject(error);
      });
  });
}

export interface ModelDebugOptions {
  system_prompt?: string
  temperature?: number
  top_p?: number
  max_tokens?: number
  thinking?: boolean
  // Graded thinking level; takes precedence over the boolean when set.
  reasoning_effort?: ReasoningEffortLevel | string
}

export interface ModelDebugResult {
  ok: boolean
  elapsed_ms: number
  request: Record<string, unknown>
  raw_response: unknown
  observations: Record<string, unknown>
  error?: string
}

export async function debugModel(
  id: string,
  data: {
    input?: string
    documents?: string[]
    options?: ModelDebugOptions
    file?: File | null
  },
): Promise<ModelDebugResult> {
  const form = new FormData()
  form.append('input', data.input || '')
  form.append('documents', JSON.stringify(data.documents || []))
  form.append('options', JSON.stringify(data.options || {}))
  if (data.file) form.append('file', data.file)
  const response: any = await postUpload(
    `/api/v1/models/${id}/debug`,
    form,
    undefined,
    { timeout: 300000 },
  )
  if (response?.success && response?.data) return response.data
  throw new Error(response?.message || 'Failed to get model')
}

// ----------------------------------------------------------------------------
// Model credential subresource. See mcp-service.ts for the matching MCP API
// shape and the design notes in internal/handler/dto/mcp.go.
// ----------------------------------------------------------------------------

export type ModelCredentialField = 'api_key' | 'app_secret'

export interface ModelCredentialsResponse {
  fields: Record<ModelCredentialField, { configured: boolean }>
}

export async function putModelCredentials(
  id: string,
  body: Partial<Record<ModelCredentialField, string>>,
): Promise<ModelCredentialsResponse> {
  const response: any = await put(`/api/v1/models/${id}/credentials`, body)
  return (response.data ?? response) as ModelCredentialsResponse
}

export async function deleteModelCredentialField(
  id: string,
  field: ModelCredentialField,
): Promise<void> {
  await del(`/api/v1/models/${id}/credentials/${field}`)
}
