package rerank

import (
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/catalog"
)

// TestDetectByURLSeesVendorCatalog guards the blank import of
// internal/models/vendors in reranker.go.
//
// catalog.DetectByURL returns "generic" for every URL while the catalog is
// empty. Without the vendor packages linked in, a stored rerank row that
// carries no provider id resolves to the generic Cohere client at the wrong
// endpoint.
func TestDetectByURLSeesVendorCatalog(t *testing.T) {
	cases := map[string]string{
		"https://openrouter.ai/api/v1":                         "openrouter",
		"https://api.openai.com/v1":                            "openai",
		"https://some-self-hosted-gateway.example.internal/v1": catalog.GenericID,
	}
	for baseURL, want := range cases {
		if got := catalog.DetectByURL(baseURL); got != want {
			t.Errorf("DetectByURL(%q) = %q, want %q", baseURL, got, want)
		}
	}
}
