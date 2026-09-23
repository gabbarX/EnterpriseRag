package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"text/template"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/agent"
)

// Wiki work is enqueued from background paths (clone/move, reparse, internal
// retries). Persisting an empty locale there strands the language for the whole
// document, because the worker resolves the prompt language from the queued op.
func TestNewWikiIngestPendingOpPersistsLanguage(t *testing.T) {
	pendingOp, err := newWikiIngestPendingOp(context.Background(), 7, "kb-1", "knowledge-1")
	if err != nil {
		t.Fatalf("newWikiIngestPendingOp() error = %v", err)
	}
	var op WikiPendingOp
	if err := json.Unmarshal(pendingOp.Payload, &op); err != nil {
		t.Fatalf("unmarshal pending op payload: %v", err)
	}
	if op.Language != "en-US" {
		t.Fatalf("op.Language = %q, want %q", op.Language, "en-US")
	}
}

func TestResolveSlugUpdateLanguage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		updates []SlugUpdate
		want    string
	}{
		{
			name:    "uses the language carried by the updates",
			updates: []SlugUpdate{{Language: "English"}},
			want:    "English",
		},
		{
			// A page aggregates contributions from several documents; the
			// first one is not guaranteed to carry a language.
			name:    "skips contributors that carry no language",
			updates: []SlugUpdate{{Type: "retract"}, {Language: "English"}},
			want:    "English",
		},
		{
			name:    "falls back to the product language",
			updates: []SlugUpdate{{Type: "retract"}, {Type: "entity"}},
			want:    "English",
		},
		{
			name:    "falls back for an empty update set",
			updates: nil,
			want:    "English",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveSlugUpdateLanguage(ctx, tt.updates); got != tt.want {
				t.Errorf("resolveSlugUpdateLanguage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Guards the observable symptom: an unresolved language renders the editor
// instruction as "Write in ." and leaves the output language to the model.
func TestWikiPromptsNeverRenderAnEmptyLanguage(t *testing.T) {
	prompts := map[string]string{
		"WikiPageModifyUserPrompt":   agent.WikiPageModifyUserPrompt,
		"WikiSummaryPrompt":          agent.WikiSummaryPrompt,
		"WikiIndexIntroPrompt":       agent.WikiIndexIntroPrompt,
		"WikiIndexIntroUpdatePrompt": agent.WikiIndexIntroUpdatePrompt,
		"WikiKnowledgeExtractPrompt": agent.WikiKnowledgeExtractPrompt,
		"WikiCandidateSlugPrompt":    agent.WikiCandidateSlugPrompt,
		"WikiChunkCitationPrompt":    agent.WikiChunkCitationPrompt,
		"WikiTaxonomyPlanPrompt":     agent.WikiTaxonomyPlanPrompt,
	}

	for name, tpl := range prompts {
		t.Run(name, func(t *testing.T) {
			parsed, err := template.New(name).Parse(tpl)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			var buf strings.Builder
			data := map[string]string{
				"HasAdditions": "1",
				"Language":     "English",
			}
			if err := parsed.Execute(&buf, data); err != nil {
				t.Fatalf("execute %s: %v", name, err)
			}
			if strings.Contains(buf.String(), "Write in .") ||
				strings.Contains(buf.String(), "in <no value>") {
				t.Fatalf("%s rendered without a language", name)
			}
		})
	}
}
