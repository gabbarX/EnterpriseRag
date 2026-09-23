package embedding

import (
	"context"
	"fmt"
	"strings"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/logger"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/utils/ollama"
	// catalog.Resolve answers from the vendor catalog, which is empty until
	// the vendor packages have run their init. Without this import every row
	// resolves to the generic vendor — silently, and only in builds that do
	// not already link the container (leaf tests, future tools).
	_ "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vendors"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/tracing/langfuse"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

// Embedder defines the interface for text vectorization
type Embedder interface {
	// Embed converts text to vector
	Embed(ctx context.Context, text string) ([]float32, error)

	// BatchEmbed converts multiple texts to vectors in batch
	BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)

	// GetModelName returns the model name
	GetModelName() string

	// GetDimensions returns the vector dimensions
	GetDimensions() int

	// GetModelID returns the model ID
	GetModelID() string

	EmbedderPooler
}

type EmbedderPooler interface {
	BatchEmbedWithPool(ctx context.Context, model Embedder, texts []string) ([][]float32, error)
}

// EmbedderType represents the embedder type
type EmbedderType string

// Config represents the embedder configuration
type Config struct {
	Source                    types.ModelSource `json:"source"`
	BaseURL                   string            `json:"base_url"`
	ModelName                 string            `json:"model_name"`
	APIKey                    string            `json:"api_key"`
	TruncatePromptTokens      int               `json:"truncate_prompt_tokens"`
	Dimensions                int               `json:"dimensions"`
	SupportsDimensionOverride bool              `json:"supports_dimension_override"`
	ModelID                   string            `json:"model_id"`
	Provider                  string            `json:"provider"`
	// MaxConcurrency caps concurrent background calls to this model; 0 falls
	// back to the process-wide default (see limiter.GateN).
	MaxConcurrency int               `json:"max_concurrency"`
	ExtraConfig    map[string]string `json:"extra_config"`
	// CustomHeaders adds custom HTTP request headers when calling the remote API
	// (the equivalent of extra_headers in the OpenAI Python SDK).
	CustomHeaders map[string]string `json:"custom_headers"`
	AppID         string
	AppSecret     string
}

// ConfigFromModel builds an embedding.Config from a types.Model.
// The production path (loaded from the DB) and the connection-test path
// (temporary form) share this mapping.
// appID / appSecret are already-decrypted application-level credentials; the
// caller is responsible for passing them in.
func ConfigFromModel(m *types.Model, appID, appSecret string) Config {
	if m == nil {
		return Config{}
	}
	return Config{
		Source:                    m.Source,
		BaseURL:                   m.Parameters.BaseURL,
		APIKey:                    m.Parameters.APIKey,
		ModelID:                   m.ID,
		ModelName:                 m.Name,
		Dimensions:                m.Parameters.EmbeddingParameters.Dimension,
		SupportsDimensionOverride: m.Parameters.EmbeddingParameters.SupportsDimensionOverride,
		TruncatePromptTokens:      m.Parameters.EmbeddingParameters.TruncatePromptTokens,
		Provider:                  m.Parameters.Provider,
		MaxConcurrency:            m.Parameters.MaxConcurrency,
		ExtraConfig:               m.Parameters.ExtraConfig,
		CustomHeaders:             m.Parameters.CustomHeaders,
		AppID:                     appID,
		AppSecret:                 appSecret,
	}
}

// NewEmbedder creates an embedder based on the configuration
func NewEmbedder(config Config, pooler EmbedderPooler, ollamaService *ollama.OllamaService) (Embedder, error) {
	e, err := newEmbedder(config, pooler, ollamaService)
	if err != nil {
		return e, err
	}
	if setter, ok := e.(interface{ SetSupportsDimensionOverride(bool) }); ok {
		setter.SetSupportsDimensionOverride(config.SupportsDimensionOverride)
	}
	// Innermost: gate the real provider round-trips (including the per-sub-batch
	// pool callbacks) before debug/langfuse wrap for logging/tracing. See
	// concurrencyEmbedder for why this sits below the observability decorators.
	e = wrapEmbeddingConcurrency(e, config.MaxConcurrency)
	if logger.LLMDebugEnabled() {
		e = &debugEmbedder{inner: e}
	}
	if langfuse.GetManager().Enabled() {
		e = &langfuseEmbedder{inner: e}
	}
	return e, nil
}

func newEmbedder(config Config, pooler EmbedderPooler, ollamaService *ollama.OllamaService) (Embedder, error) {
	switch strings.ToLower(string(config.Source)) {
	case string(types.ModelSourceLocal):
		return NewOllamaEmbedder(config.BaseURL,
			config.ModelName, config.TruncatePromptTokens, config.Dimensions, config.ModelID, pooler, ollamaService)
	case string(types.ModelSourceRemote):
		return newRemoteEmbedder(config, pooler)
	default:
		return nil, fmt.Errorf("unsupported embedder source: %s", config.Source)
	}
}
