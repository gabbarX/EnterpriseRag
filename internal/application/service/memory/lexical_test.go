package memory

import (
	"strings"
	"testing"
	"time"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/stretchr/testify/require"
)

func item(kind, content string, importance int) *types.MemoryItem {
	return &types.MemoryItem{
		Kind:          kind,
		Content:       content,
		NormalizedKey: types.NormalizeMemoryKey("", content),
		Importance:    importance,
		ValidFrom:     time.Now(),
	}
}

func topicItem(kind, topic, content string, importance int) *types.MemoryItem {
	entry := item(kind, content, importance)
	entry.Topic = topic
	entry.NormalizedKey = types.NormalizeMemoryKey(topic, content)
	return entry
}

// TestTopicIsPartOfTheRetrievalHandle covers the common shape of an extracted
// memory: the question names the subject while the statement carries only the
// value, so matching on the statement alone would miss it.
func TestTopicIsPartOfTheRetrievalHandle(t *testing.T) {
	items := []*types.MemoryItem{
		topicItem(types.MemoryKindFact, "database in use", "we have moved from MySQL to PostgreSQL", 3),
		topicItem(types.MemoryKindFact, "frontend stack", "we are on Vue 3 with Vite", 3),
	}
	selected := selectRecallItems("write me a snippet that connects to the database", items,
		types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget)
	require.NotEmpty(t, selected)
	require.Equal(t, "we have moved from MySQL to PostgreSQL", selected[0].Content)
	require.NotContains(t, contents(selected), "we are on Vue 3 with Vite")
}

func contents(items []*types.MemoryItem) []string {
	out := make([]string, 0, len(items))
	for _, entry := range items {
		out = append(out, entry.Content)
	}
	return out
}

func TestSelectRecallItemsRanksByTopic(t *testing.T) {
	items := []*types.MemoryItem{
		item(types.MemoryKindFact, "the production database is PostgreSQL 17", 3),
		item(types.MemoryKindFact, "the frontend framework is Vue 3", 3),
		item(types.MemoryKindTask, "refactoring the payment flow in the order service", 3),
	}
	selected := selectRecallItems("how do I debug database timeouts", items,
		types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget)
	require.NotEmpty(t, selected)
	require.Equal(t, "the production database is PostgreSQL 17", selected[0].Content)
	require.NotContains(t, contents(selected), "the frontend framework is Vue 3")
}

func TestSelectRecallItemsMatchesEnglish(t *testing.T) {
	items := []*types.MemoryItem{
		item(types.MemoryKindFact, "The staging cluster runs in Mumbai", 3),
		item(types.MemoryKindFact, "CI is GitHub Actions on self-hosted runners", 3),
	}
	selected := selectRecallItems("where is the staging cluster deployed", items,
		types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget)
	require.NotEmpty(t, selected)
	require.Equal(t, "The staging cluster runs in Mumbai", selected[0].Content)
}

func TestSelectRecallItemsDropsWeakMatches(t *testing.T) {
	items := []*types.MemoryItem{
		item(types.MemoryKindFact, "I usually write services in Go", 3),
	}
	// Filler words share nothing meaningful with the memory: a weakly related
	// memory must not be injected at all.
	require.Empty(t, selectRecallItems("how is the weather today", items,
		types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget))
}

func TestSelectRecallItemsRespectsCountBudget(t *testing.T) {
	var items []*types.MemoryItem
	for i := 0; i < 20; i++ {
		items = append(items, item(types.MemoryKindFact,
			"database note डेटाबेस "+string(rune('a'+i)), 3))
	}
	selected := selectRecallItems("database", items, types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget)
	require.LessOrEqual(t, len(selected), types.MemoryRecallMaxItems)
}

// The budget is counted in runes, so the memory here is deliberately not
// ASCII: a byte-length budget would admit roughly a third of what it should.
func TestSelectRecallItemsRespectsRuneBudget(t *testing.T) {
	var items []*types.MemoryItem
	for i := 0; i < 5; i++ {
		long := "database " + strings.Repeat("बहुत लंबा विवरण ", 20)
		items = append(items, item(types.MemoryKindFact, long, 3))
	}
	selected := selectRecallItems("database", items, types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget)
	require.NotEmpty(t, selected, "one memory still has to fit")
	total := 0
	for _, entry := range selected {
		total += len([]rune(entry.Content))
	}
	require.LessOrEqual(t, total, types.MemoryRecallRuneBudget)
}

func TestSelectRecallItemsEmptyQuery(t *testing.T) {
	items := []*types.MemoryItem{item(types.MemoryKindFact, "any fact at all", 3)}
	require.Empty(t, selectRecallItems("   ", items,
		types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget))
}

func TestASingleSharedFillerWordDoesNotOutrankARealMatch(t *testing.T) {
	// "the" appears in both memories — a word shared with the question must
	// not be enough to put an unrelated memory ahead of the one it is about.
	items := []*types.MemoryItem{
		item(types.MemoryKindFact, "parameter validation lives in the handler layer", 3),
		item(types.MemoryKindFact, "migration scripts live in the migrations directory", 3),
	}
	selected := selectRecallItems("where do the migration scripts live", items,
		types.MemoryRecallMaxItems, types.MemoryRecallRuneBudget)
	require.NotEmpty(t, selected)
	require.Equal(t, "migration scripts live in the migrations directory", selected[0].Content)
}
