package catalog_test

import (
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/catalog"
	_ "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vendors"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

func TestValidateRow(t *testing.T) {
	cases := []struct {
		name      string
		modelName string
		modelType types.ModelType
		params    *types.ModelParameters
		wantErr   bool
	}{
		{
			name: "nil parameters", modelName: "x", modelType: types.ModelTypeKnowledgeQA,
		},
		{
			name: "a catalogued embedding row resolves", modelName: "text-embedding-3-large",
			modelType: types.ModelTypeEmbedding,
			params:    &types.ModelParameters{Provider: "openai"},
		},
		{
			// Embedding is built from Resolve too, so a bad overlay fails
			// every call; it must fail the save instead.
			name: "unknown embedding compat key is rejected", modelName: "text-embedding-3-large",
			modelType: types.ModelTypeEmbedding,
			params: &types.ModelParameters{
				Provider: "openai", Spec: &types.ModelSpecOverride{Compat: map[string]any{"max_tokens_field": "x"}},
			},
			wantErr: true,
		},
		{
			name: "unknown embedding protocol is rejected", modelName: "text-embedding-3-large",
			modelType: types.ModelTypeEmbedding,
			params: &types.ModelParameters{
				Provider: "openai", Spec: &types.ModelSpecOverride{Compat: map[string]any{"api": "openai-completions"}},
			},
			wantErr: true,
		},
		{
			name: "an embedding name inside a chat glob is accepted", modelName: "gpt-5-embed",
			modelType: types.ModelTypeEmbedding,
			params:    &types.ModelParameters{Provider: "openai"},
		},
		{
			name:      "rerank truncation on a vendor without the extension is rejected",
			modelName: "cohere/rerank-v3.5",
			modelType: types.ModelTypeRerank,
			params: &types.ModelParameters{
				Provider: "openrouter", ExtraConfig: map[string]string{"truncate_prompt_tokens": "512"},
			},
			wantErr: true,
		},
		{
			name: "a catalogued asr row resolves", modelName: "whisper-1",
			modelType: types.ModelTypeASR,
			params:    &types.ModelParameters{Provider: "openai"},
		},
		{
			name: "unknown asr compat key is rejected", modelName: "whisper-1",
			modelType: types.ModelTypeASR,
			params: &types.ModelParameters{
				Provider: "openai", Spec: &types.ModelSpecOverride{Compat: map[string]any{"dimensions_field": "x"}},
			},
			wantErr: true,
		},
		{
			name: "a catalogued chat row resolves", modelName: "gpt-5.2",
			modelType: types.ModelTypeKnowledgeQA,
			params:    &types.ModelParameters{Provider: "openai"},
		},
		{
			name: "unknown protocol is rejected", modelName: "claude-opus-5",
			modelType: types.ModelTypeKnowledgeQA,
			params: &types.ModelParameters{
				Provider: "anthropic", Spec: &types.ModelSpecOverride{API: "openai-chat-v9"},
			},
			wantErr: true,
		},
		{
			name: "misspelled thinking level is rejected", modelName: "gpt-5.2",
			modelType: types.ModelTypeKnowledgeQA,
			params: &types.ModelParameters{
				Provider: "openai",
				Spec:     &types.ModelSpecOverride{ThinkingLevels: map[string]*string{"hgih": nil}},
			},
			wantErr: true,
		},
		{
			name: "unknown compat key is rejected", modelName: "gpt-5.2",
			modelType: types.ModelTypeKnowledgeQA,
			params: &types.ModelParameters{
				Provider: "openai",
				Spec:     &types.ModelSpecOverride{Compat: map[string]any{"max_tokens_fields": "max_tokens"}},
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := catalog.ValidateRow(tc.modelName, tc.modelType, tc.params)
			if tc.wantErr && err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
