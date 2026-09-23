package parity

import (
	"strings"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/api"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/catalog"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEveryRerankVendorResolvesToAKnownProtocol walks the catalog, so a newly
// added vendor is covered without touching this file.
func TestEveryRerankVendorResolvesToAKnownProtocol(t *testing.T) {
	vendors := catalog.ListByType(types.ModelTypeRerank)
	require.NotEmpty(t, vendors, "no vendor serves rerank")

	for _, v := range vendors {
		t.Run(v.ID, func(t *testing.T) {
			model := "some-rerank-model"
			if catalogued := v.ModelsByType(types.ModelTypeRerank); len(catalogued) > 0 {
				model = catalogued[0].ID
			}
			resolved, err := catalog.Resolve(catalog.Ref{
				Provider: v.ID, Model: model, ModelType: types.ModelTypeRerank,
			})
			require.NoError(t, err)
			assert.True(t, resolved.RerankAPI.Known(),
				"unknown rerank protocol %q", resolved.RerankAPI)
			// The catch-all vendor has no endpoint of its own: the operator
			// supplies one, and Validate requires it.
			if v.ID != catalog.GenericID {
				assert.NotEmpty(t, resolved.BaseURL, "rerank vendors need a default endpoint")
			}

			// A vendor that declares a ceiling must declare a usable one.
			assert.GreaterOrEqual(t, resolved.Rerank.MaxDocuments, 0)
			assert.GreaterOrEqual(t, resolved.Rerank.MaxDocumentChars, 0)
			if resolved.Rerank.ScoreScale != "" {
				assert.Contains(t,
					[]api.ScoreScale{api.ScoreProbability, api.ScoreLogit},
					resolved.Rerank.ScoreScale)
			}
		})
	}
}

// TestRerankProtocolAssignment pins which dialect each vendor speaks. The
// wire shapes are mutually incompatible, so a silent reassignment breaks
// every rerank call for that vendor.
func TestRerankProtocolAssignment(t *testing.T) {
	for id, want := range map[string]api.RerankAPI{
		"generic":    api.RerankCohere,
		"openrouter": api.RerankCohere,
	} {
		t.Run(id, func(t *testing.T) {
			resolved, err := catalog.Resolve(catalog.Ref{
				Provider: id, Model: "m", ModelType: types.ModelTypeRerank,
			})
			require.NoError(t, err)
			assert.Equal(t, want, resolved.RerankAPI)
		})
	}
}

// TestRerankOutboundShapePerProtocol builds the real client through
// rerank.NewReranker and asserts what each dialect puts on the wire.
// TestTruncatePromptTokensOnlyReachesVLLMClassVendors pins the gate on the
// vLLM extension restored from the pre-catalog client. It is opt-in per row,
// but the row can only opt into it on a runtime that implements it: no
// managed vendor documents the field, and sending it to one is how an
// undocumented parameter ends up on every request.
func TestTruncatePromptTokensOnlyReachesVLLMClassVendors(t *testing.T) {
	optIn := map[string]string{catalog.ExtraTruncatePromptTokens: "512"}

	for _, id := range []string{"generic"} {
		t.Run(id+" accepts it", func(t *testing.T) {
			resolved, err := catalog.Resolve(catalog.Ref{
				Provider: id, Model: "m", ModelType: types.ModelTypeRerank,
				BaseURL: "http://127.0.0.1:9/v1", Extra: optIn,
			})
			require.NoError(t, err)
			assert.Equal(t, 512, resolved.Rerank.TruncatePromptTokens)
		})
	}

	for _, id := range []string{"openrouter"} {
		t.Run(id+" rejects it", func(t *testing.T) {
			_, err := catalog.Resolve(catalog.Ref{
				Provider: id, Model: "m", ModelType: types.ModelTypeRerank, Extra: optIn,
			})
			require.Error(t, err, "a managed vendor must not silently accept a vLLM extension")
			assert.Contains(t, err.Error(), "vLLM extension")
		})
	}
}

func TestTruncatePromptTokensRejectsAnInvalidValue(t *testing.T) {
	for _, raw := range []string{"0", "-1", "abc"} {
		_, err := catalog.Resolve(catalog.Ref{
			Provider: "generic", Model: "m", ModelType: types.ModelTypeRerank,
			BaseURL: "http://127.0.0.1:9/v1",
			Extra:   map[string]string{catalog.ExtraTruncatePromptTokens: raw},
		})
		require.Error(t, err, "value %q", raw)
	}
}

// TestTruncatePromptTokensIsReachableFromTheEditor closes the loop between the
// vendor accepting the extension and an operator being able to turn it on.
// Before the catalog it was readable from extra_config but had no input in the
// model editor, so the only way to set it was the API or the database.
// GET /models/providers renders extra fields dynamically, so declaring one is
// all it takes — and it must be declared exactly where the extension is
// accepted, or the form offers a switch the vendor will reject.
func TestTruncatePromptTokensIsReachableFromTheEditor(t *testing.T) {
	for _, v := range catalog.ListByType(types.ModelTypeRerank) {
		var field *catalog.ExtraField
		for i := range v.ExtraFields {
			if v.ExtraFields[i].Key == catalog.ExtraTruncatePromptTokens {
				field = &v.ExtraFields[i]
			}
		}
		resolved, err := catalog.Resolve(catalog.Ref{
			Provider: v.ID, Model: "m", ModelType: types.ModelTypeRerank,
		})
		require.NoError(t, err)

		if !resolved.Rerank.AcceptsTruncatePromptTokens {
			assert.Nil(t, field, "%s does not accept the extension, so it must not offer the input", v.ID)
			continue
		}
		require.NotNil(t, field, "%s accepts the extension but offers no way to set it", v.ID)
		assert.Equal(t, "number", field.Type)
		assert.False(t, field.Required, "the extension is opt-in")
		assert.Equal(t, []types.ModelType{types.ModelTypeRerank}, field.ModelTypes,
			"the input belongs to the rerank form only")
	}
}

// TestRerankScoreScalesMatchTheVendorDocs pins which vendors return something
// other than a 0..1 relevance score. Getting this wrong is silent: the number
// still looks like a score, and the retrieval threshold still compares it, so
// a logit-scaled vendor simply loses every negatively scored document.
func TestRerankScoreScalesMatchTheVendorDocs(t *testing.T) {
	// No vendor in this build answers on a logit scale; a new one that does
	// has to be named here as well as in its own vendor.go.
	logitScaled := map[string]bool{}

	for _, v := range catalog.ListByType(types.ModelTypeRerank) {
		resolved, err := catalog.Resolve(catalog.Ref{
			Provider: v.ID, Model: "m", ModelType: types.ModelTypeRerank,
		})
		require.NoError(t, err)
		want := api.ScoreProbability
		if logitScaled[v.ID] {
			want = api.ScoreLogit
		}
		assert.Equal(t, want, resolved.Rerank.ScoreScale, "%s score scale", v.ID)
	}
}

// TestGatewayRerankEndpoints pins the URL each gateway's rerank rows reach.
// They serve the Cohere dialect the protocol package already speaks, so
// opening the type was a declaration.
func TestGatewayRerankEndpoints(t *testing.T) {
	for _, tc := range []struct {
		provider string
		wantURL  string
	}{
		// https://openrouter.ai/docs/api/api-reference/rerank/submit-a-rerank-request
		{provider: "openrouter", wantURL: "https://openrouter.ai/api/v1/rerank"},
	} {
		t.Run(tc.provider, func(t *testing.T) {
			resolved, err := catalog.Resolve(catalog.Ref{
				Provider: tc.provider, Model: "m", ModelType: types.ModelTypeRerank,
			})
			require.NoError(t, err)
			require.Equal(t, api.RerankCohere, resolved.RerankAPI)

			// Rebuild the URL the way cohererank does.
			got := strings.TrimRight(resolved.BaseURL, "/")
			if !strings.HasSuffix(got, "/rerank") {
				got += "/rerank"
			}
			assert.Equal(t, tc.wantURL, got)
		})
	}
}

// TestScoreScaleIsOverridablePerRow covers the gateways, where the vendor
// cannot know the answer: a generic endpoint serves whatever reranker was
// deployed behind it, and the two families disagree — BGE
// answers a 0..1 probability, Qwen3-Reranker an unbounded score. Converting a
// probability as if it were a logit squeezes every score into [0.5, 0.73],
// which makes a relevance threshold meaningless, so the operator needs a way
// to say which one is actually there.
func TestScoreScaleIsOverridablePerRow(t *testing.T) {
	for _, id := range []string{"generic"} {
		t.Run(id, func(t *testing.T) {
			base := "http://127.0.0.1:9/v1"
			overridden, err := catalog.Resolve(catalog.Ref{
				Provider: id, Model: "bge-reranker-v2-m3", ModelType: types.ModelTypeRerank,
				BaseURL: base, Extra: map[string]string{catalog.ExtraScoreScale: "probability"},
			})
			require.NoError(t, err)
			assert.Equal(t, api.ScoreProbability, overridden.Rerank.ScoreScale)

			// And the input to set it is rendered by the editor.
			v, ok := catalog.Get(id)
			require.True(t, ok)
			var field *catalog.ExtraField
			for i := range v.ExtraFields {
				if v.ExtraFields[i].Key == catalog.ExtraScoreScale {
					field = &v.ExtraFields[i]
				}
			}
			require.NotNil(t, field, "%s must offer the override it needs", id)
			assert.Equal(t, "select", field.Type)
			assert.Len(t, field.Options, 2)
		})
	}

	_, err := catalog.Resolve(catalog.Ref{
		Provider: "generic", Model: "m", ModelType: types.ModelTypeRerank,
		BaseURL: "http://127.0.0.1:9/v1",
		Extra:   map[string]string{catalog.ExtraScoreScale: "sigmoid"},
	})
	require.Error(t, err, "an unknown scale must not be treated as a probability")
}

// TestGatewayVendorsOfferTheScoreScaleOverride keeps the override where the
// question is open. A gateway cannot know which reranker is deployed behind
// it, so it offers the input; the other vendors' scales are settled by their
// own documentation and must not offer a switch that only lets an operator
// get it wrong.
func TestGatewayVendorsOfferTheScoreScaleOverride(t *testing.T) {
	gateways := map[string]bool{"generic": true}

	for _, v := range catalog.ListByType(types.ModelTypeRerank) {
		var field *catalog.ExtraField
		for i := range v.ExtraFields {
			if v.ExtraFields[i].Key == catalog.ExtraScoreScale {
				field = &v.ExtraFields[i]
			}
		}
		if !gateways[v.ID] {
			assert.Nil(t, field, "%s: its documentation settles the scale", v.ID)
			continue
		}
		require.NotNil(t, field, "%s is a gateway and needs the override", v.ID)
		require.Len(t, field.Options, 2)
		for _, opt := range field.Options {
			assert.NotEmpty(t, opt.Label, "%s: option %q has no label", v.ID, opt.Value)
		}
		assert.NotEmpty(t, field.Placeholder,
			"%s: the override needs a placeholder saying how to choose", v.ID)
	}
}

// An operator-facing select is only usable if every choice says what it means,
// so a bare value is never enough.
func TestEveryExtraFieldProseIsPresent(t *testing.T) {
	for _, v := range catalog.List() {
		for _, field := range v.ExtraFields {
			assert.NotEmpty(t, field.Label, "%s/%s: field has no label", v.ID, field.Key)
			for _, opt := range field.Options {
				assert.NotEmpty(t, opt.Label,
					"%s/%s: option %q has no label", v.ID, field.Key, opt.Value)
			}
		}
	}
}
