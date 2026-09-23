package memory

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/stretchr/testify/require"
)

// This file is the behavioural regression set. Each case is a short sequence
// of turns across separate sessions, ending in an assertion about what the
// next turn's prompt contains. They are written against the service the chat
// path actually calls, so a change that keeps the unit tests green but breaks
// the user-visible behaviour still fails here.
//
// LoCoMo and LongMemEval are deliberately not used: their published scores are
// vendor-run and disagree by tens of points, and neither has a split
// that matches how this feature is used.

type memoryScenario struct {
	name string
	// userTurns are what the user says, in order, as if across sessions.
	userTurns []string
	// extracted is the distillation the model returns for those turns.
	extracted []map[string]any
	// laterQuery is what the user asks in a new session afterwards.
	laterQuery string
	// wantInPrompt must appear in the memory injected into that later turn.
	wantInPrompt []string
	// wantAbsent must not appear.
	wantAbsent []string
}

func TestCrossSessionMemoryScenarios(t *testing.T) {
	scenarios := []memoryScenario{
		{
			name:      "a profile is remembered and carried into a new session",
			userTurns: []string{"I am a backend engineer working on medical imaging, mostly in Go"},
			extracted: []map[string]any{
				{
					"action": "add", "kind": "profile", "topic": "role",
					"content": "backend engineer on medical imaging, mostly writes Go",
				},
			},
			laterQuery:   "help me design an API",
			wantInPrompt: []string{"medical imaging", "backend engineer"},
		},
		{
			name:      "a preference is resident and comes along whatever the question is",
			userTurns: []string{"from now on lead with the conclusion, no long preamble"},
			extracted: []map[string]any{
				{
					"action": "add", "kind": "preference", "topic": "reply style",
					"content": "lead with the conclusion, skip the preamble",
				},
			},
			laterQuery:   "is the weather good for a run today",
			wantInPrompt: []string{"lead with the conclusion"},
		},
		{
			name: "facts are recalled by relevance and the unrelated ones stay out",
			userTurns: []string{
				"our production DB is PostgreSQL 17, running in Mumbai",
				"our frontend is Vue 3 with Vite",
			},
			extracted: []map[string]any{
				{
					"action": "add", "kind": "fact", "topic": "production database",
					"content": "the production DB is PostgreSQL 17, deployed in Mumbai",
				},
				{
					"action": "add", "kind": "fact", "topic": "frontend stack",
					"content": "frontend is Vue 3 with Vite",
				},
			},
			laterQuery:   "how big should the database connection pool be",
			wantInPrompt: []string{"PostgreSQL 17"},
			wantAbsent:   []string{"Vue 3"},
		},
		{
			name: "after a correction only the latest statement survives",
			userTurns: []string{
				"we are on MySQL",
				"correction: we moved to PostgreSQL last month",
			},
			extracted: []map[string]any{
				{
					"action": "add", "kind": "fact", "topic": "database in use",
					"content": "we are on MySQL",
				},
				{
					"action": "update", "kind": "fact", "topic": "database in use",
					"content": "we have moved to PostgreSQL",
				},
			},
			laterQuery:   "write a snippet that connects to the database",
			wantInPrompt: []string{"PostgreSQL"},
			wantAbsent:   []string{"MySQL"},
		},
		{
			name:      "an open task carries across sessions",
			userTurns: []string{"this week I am refactoring the payment flow in the order service, not done yet"},
			extracted: []map[string]any{
				{
					"action": "add", "kind": "task", "topic": "refactor in progress",
					"content": "refactoring the payment flow in the order service, not finished",
				},
			},
			laterQuery:   "how do I carry on with that order service refactor",
			wantInPrompt: []string{"payment flow"},
		},
		{
			name: "a finished task stops being recalled",
			userTurns: []string{
				"refactoring the payment flow in the order service",
				"the payment flow refactor has shipped",
			},
			extracted: []map[string]any{
				{
					"action": "add", "kind": "task", "topic": "refactor in progress",
					"content": "refactoring the payment flow in the order service",
				},
				{
					"action": "delete", "kind": "task", "topic": "refactor in progress",
					"content": "the payment flow refactor is done",
				},
			},
			laterQuery: "what is still open on the order service",
			wantAbsent: []string{"refactoring the payment flow in the order service"},
		},
		{
			name:      "a one-off question must not become a long-term fact",
			userTurns: []string{"is Go's map safe for concurrent use"},
			// A well-behaved extraction returns nothing here, which is the
			// normal outcome; the assertion is that we store nothing either.
			extracted:  nil,
			laterQuery: "how does a Go slice grow under the hood",
			wantAbsent: []string{"map", "concurrent"},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			svc, tenantRepo, messages, models, _ := newExtractionHarness(t)
			tenantRepo.set(1, &types.MemoryConfig{Enabled: true, WriteMode: types.MemoryWriteAuto})

			// Replay the turns one distillation at a time, so an update or a
			// delete sees the state its predecessor left behind.
			for i, turn := range scenario.userTurns {
				messages.messages = []*types.Message{{Role: "user", Content: turn}}
				var decisions []map[string]any
				if i < len(scenario.extracted) {
					decisions = scenario.extracted[i : i+1]
				}
				body, err := json.Marshal(map[string]any{"memories": decisions})
				require.NoError(t, err)
				models.response = string(body)

				require.NoError(t, svc.Handle(context.Background(), extractTask(t, types.MemoryExtractPayload{
					TenantID:    1,
					SubjectID:   "web_user:alice",
					SessionID:   "session-" + string(rune('a'+i)),
					MessageID:   "message-" + string(rune('a'+i)),
					ChatModelID: "conversation-model",
				})))
			}

			// A brand new session: nothing but long-term memory carries over.
			laterCtx := enabledCtx(t, tenantRepo, 1, "alice")
			prompt := svc.Recall(laterCtx, scenario.laterQuery).Prompt

			for _, want := range scenario.wantInPrompt {
				require.Contains(t, prompt, want,
					"expected the later turn to carry %q\nprompt was:\n%s", want, prompt)
			}
			for _, absent := range scenario.wantAbsent {
				require.NotContains(t, prompt, absent,
					"did not expect the later turn to carry %q\nprompt was:\n%s", absent, prompt)
			}
		})
	}
}

// TestReadPathMakesNoModelCall pins the cost promise: recall must not add a
// model call to a turn, no matter how many memories the user has.
func TestReadPathMakesNoModelCall(t *testing.T) {
	svc, tenantRepo, _, models, _ := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	for i := 0; i < 30; i++ {
		_, err := svc.Remember(ctx, types.MemoryItem{
			Kind:    types.MemoryKindFact,
			Topic:   "fact-" + string(rune('a'+i)),
			Content: "a database related fact " + string(rune('a'+i)),
		})
		require.NoError(t, err)
	}

	for i := 0; i < 5; i++ {
		require.NotEmpty(t, svc.Recall(ctx, "how do I tune the database").Prompt)
	}
	require.Zero(t, models.calls, "the read path must not call a model")
}

// TestInjectedMemoryStaysInsideItsBudget keeps a user with a large memory
// space from quietly eating the context window.
func TestInjectedMemoryStaysInsideItsBudget(t *testing.T) {
	svc, tenantRepo, _, _, _ := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	for i := 0; i < 40; i++ {
		_, err := svc.Remember(ctx, types.MemoryItem{
			Kind:    types.MemoryKindPreference,
			Topic:   "pref-" + string(rune('a'+i)),
			Content: strings.Repeat("बहुत लंबा वरीयता विवरण ", 8) + string(rune('a'+i)),
		})
		require.NoError(t, err)
		_, err = svc.Remember(ctx, types.MemoryItem{
			Kind:  types.MemoryKindFact,
			Topic: "fact-" + string(rune('a'+i)),
			Content: "database " + strings.Repeat("बहुत लंबा तथ्य विवरण ", 8) +
				string(rune('a'+i)),
		})
		require.NoError(t, err)
	}

	prompt := svc.Recall(ctx, "how do I tune the database").Prompt
	require.NotEmpty(t, prompt)
	// Envelope wording aside, the memory content itself must fit in the two
	// declared budgets.
	require.LessOrEqual(t, len([]rune(prompt)),
		types.MemoryBlockRuneBudget+types.MemoryRecallRuneBudget+600,
		"injected memory must stay within its budget")
}
