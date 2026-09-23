package types

import (
	"context"
	"testing"
)

func TestApplyBuiltinAgentLocalizationOverlaysYAMLCopy(t *testing.T) {
	restore := OverrideBuiltinAgentEntriesForTest(map[string]*BuiltinAgentEntry{
		BuiltinQuickAnswerID: {
			ID:     BuiltinQuickAnswerID,
			Avatar: "quick.png",
			I18n: map[string]BuiltinAgentI18n{
				"default": {Name: "Quick Answer", Description: "Knowledge base RAG Q&A"},
			},
		},
	})
	t.Cleanup(restore)

	agent := &CustomAgent{
		ID:          BuiltinQuickAnswerID,
		Name:        "Seeded name",
		Description: "Seeded description",
		Avatar:      "",
		IsBuiltin:   true,
		TenantID:    1,
	}
	ApplyBuiltinAgentLocalization(context.Background(), agent)

	if agent.Name != "Quick Answer" {
		t.Fatalf("Name = %q, want Quick Answer", agent.Name)
	}
	if agent.Description != "Knowledge base RAG Q&A" {
		t.Fatalf("Description = %q, want the YAML copy", agent.Description)
	}
	if agent.Avatar != "quick.png" {
		t.Fatalf("Avatar = %q, want quick.png", agent.Avatar)
	}
}

func TestApplyBuiltinAgentLocalizationNilSafe(t *testing.T) {
	ApplyBuiltinAgentLocalization(context.Background(), nil)
}

func TestApplyBuiltinAgentLocalizationLeavesUnknownAgents(t *testing.T) {
	agent := &CustomAgent{ID: "custom-agent-1", Name: "Mine", Description: "keep me"}
	ApplyBuiltinAgentLocalization(context.Background(), agent)
	if agent.Name != "Mine" || agent.Description != "keep me" {
		t.Fatalf("custom agent was rewritten: %+v", agent)
	}
}
