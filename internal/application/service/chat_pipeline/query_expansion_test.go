package chatpipeline

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestExtractKeywordsSegmentsDevanagariQuery(t *testing.T) {
	keywords := extractKeywords("वेक्टर डेटाबेस को कैसे कॉन्फ़िगर करें?")

	assert.NotContains(t, keywords, "वेक्टर डेटाबेस को कैसे कॉन्फ़िगर करें")
	assert.Contains(t, keywords, "वेक्टर")
	assert.Contains(t, keywords, "डेटाबेस")
	assert.Contains(t, keywords, "कॉन्फ़िगर")
}

func TestTokenizePreservesMixedLanguageBoundaries(t *testing.T) {
	tokens := tokenize("RAG में वेक्टर डेटाबेस, PostgreSQL पर")

	assert.Contains(t, tokens, "RAG")
	assert.Contains(t, tokens, "PostgreSQL")
	// Combining marks (matras, virama, nukta) must stay attached to the
	// syllable they follow instead of cutting the word apart.
	assert.Contains(t, tokens, "वेक्टर")
	assert.Contains(t, tokens, "डेटाबेस")
	assert.NotContains(t, tokens, "व")
	assert.NotContains(t, tokens, "RAG में")
}

func TestExtractKeywordsDropsSingleRuneTokens(t *testing.T) {
	keywords := extractKeywords("क वर्मा की छुट्टी नीति")

	assert.Contains(t, keywords, "वर्मा")
	assert.Contains(t, keywords, "छुट्टी")
	for _, keyword := range keywords {
		assert.Greater(t, utf8.RuneCountInString(keyword), 1, "unexpected single-rune keyword %q", keyword)
	}
}

func TestExpandQueriesBuildsDevanagariKeywordVariant(t *testing.T) {
	expansions := (&PluginSearch{}).expandQueries(context.Background(), &types.ChatManage{
		PipelineState: types.PipelineState{RewriteQuery: "वेक्टर डेटाबेस को कैसे कॉन्फ़िगर करें?"},
	})

	var foundKeywordVariant bool
	for _, expansion := range expansions {
		fields := strings.Fields(expansion)
		if containsToken(fields, "वेक्टर") && containsToken(fields, "डेटाबेस") && containsToken(fields, "कॉन्फ़िगर") {
			foundKeywordVariant = true
			break
		}
	}

	assert.True(t, foundKeywordVariant, "expected a Devanagari keyword expansion with segmented terms, got %v", expansions)
}

func containsToken(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
