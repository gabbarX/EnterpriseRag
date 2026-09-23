package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types/interfaces"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubKnowledgeBaseService struct {
	kb      *types.KnowledgeBase
	results []*types.SearchResult
}

func (s *stubKnowledgeBaseService) CreateKnowledgeBase(context.Context, *types.KnowledgeBase) (*types.KnowledgeBase, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) GetKnowledgeBaseByID(context.Context, string) (*types.KnowledgeBase, error) {
	return s.kb, nil
}

func (s *stubKnowledgeBaseService) GetKnowledgeBaseByIDOnly(context.Context, string) (*types.KnowledgeBase, error) {
	return s.kb, nil
}

func (s *stubKnowledgeBaseService) GetKnowledgeBasesByIDsOnly(context.Context, []string) ([]*types.KnowledgeBase, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) FillKnowledgeBaseCounts(context.Context, *types.KnowledgeBase) error {
	return nil
}

func (s *stubKnowledgeBaseService) ListKnowledgeBases(context.Context) ([]*types.KnowledgeBase, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) ListKnowledgeBasesByTenantID(context.Context, uint64) ([]*types.KnowledgeBase, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) UpdateKnowledgeBase(
	context.Context,
	string,
	string,
	string,
	*types.KnowledgeBaseConfig,
) (*types.KnowledgeBase, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) DeleteKnowledgeBase(context.Context, string) error {
	return nil
}

func (s *stubKnowledgeBaseService) TogglePinKnowledgeBase(context.Context, string) (*types.KnowledgeBase, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) HybridSearch(context.Context, string, types.SearchParams) ([]*types.SearchResult, error) {
	return s.results, nil
}

func (s *stubKnowledgeBaseService) GetQueryEmbedding(context.Context, string, string) ([]float32, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) ResolveEmbeddingModelKeys(context.Context, []*types.KnowledgeBase) map[string]string {
	return nil
}

func (s *stubKnowledgeBaseService) CopyKnowledgeBase(
	context.Context,
	string,
	string,
) (*types.KnowledgeBase, *types.KnowledgeBase, error) {
	return nil, nil, nil
}

func (s *stubKnowledgeBaseService) DuplicateKnowledgeBase(
	context.Context,
	string,
) (*types.KnowledgeBase, error) {
	return nil, nil
}

func (s *stubKnowledgeBaseService) GetRepository() interfaces.KnowledgeBaseRepository {
	return nil
}

func (s *stubKnowledgeBaseService) ProcessKBDelete(context.Context, *asynq.Task) error {
	return nil
}

func TestQueryKnowledgeGraph_ReportsConfiguredEntityAndRelationTypes(t *testing.T) {
	tool := NewQueryKnowledgeGraphTool(&stubKnowledgeBaseService{
		kb: &types.KnowledgeBase{
			ID: "kb-1",
			ExtractConfig: &types.ExtractConfig{
				Enabled: true,
				Nodes: []*types.GraphNode{
					{Name: "contract"},
					{Name: "legal team"},
					{Name: "approval workflow"},
					{Name: "contract"},
					nil,
					{Name: ""},
				},
				Relations: []*types.GraphRelation{
					{Type: "belongs to"},
					{Type: "owns"},
					{Type: "approves"},
					{Type: "owns"},
					{Type: ""},
					nil,
				},
			},
		},
		results: []*types.SearchResult{
			{
				ID:             "chunk-approval-1",
				Content:        "The contract approval workflow is maintained jointly by the legal and procurement teams; legal owns the compliance review.",
				KnowledgeID:    "doc-approval",
				KnowledgeTitle: "Contract Approval Policy",
				Score:          0.97,
				MatchType:      types.MatchTypeEmbedding,
			},
			{
				ID:             "chunk-approval-2",
				Content:        "Once a purchase request is submitted it enters the contract approval workflow and is filed in the contract register when approved.",
				KnowledgeID:    "doc-procurement",
				KnowledgeTitle: "Procurement and Contract Collaboration Standard",
				Score:          0.89,
				MatchType:      types.MatchTypeKeywords,
			},
			{
				ID:             "chunk-approval-3",
				Content:        "The legal team owns the standard contract templates and maintains the contract risk review checklist.",
				KnowledgeID:    "doc-legal",
				KnowledgeTitle: "Legal Team Responsibilities",
				Score:          0.84,
				MatchType:      types.MatchTypeEmbedding,
			},
		},
	})

	args, err := json.Marshal(QueryKnowledgeGraphInput{
		KnowledgeBaseIDs: []string{"kb-1"},
		Query:            "contract approval and legal collaboration",
	})
	require.NoError(t, err)

	result, err := tool.Execute(context.Background(), args)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
	t.Logf("tool output:\n%s", result.Output)

	assert.Contains(t, result.Output, "Entity Types (3)")
	assert.Contains(t, result.Output, "Relationship Types (3)")
	assert.NotContains(t, result.Output, "No entity types configured")
	assert.NotContains(t, result.Output, "No relationship types configured")
	assert.Contains(t, result.Output, "contract")
	assert.Contains(t, result.Output, "legal team")
	assert.Contains(t, result.Output, "approval workflow")
	assert.Contains(t, result.Output, "owns")
	assert.Contains(t, result.Output, "approves")
	assert.Contains(t, result.Output, "✓ Found 3 relevant results (deduplicated)")
	assert.Contains(t, result.Output, "Result #1:")
	assert.Contains(t, result.Output, "Result #2:")
	assert.Contains(t, result.Output, "Result #3:")
	assert.Contains(t, result.Output, "Contract Approval Policy")

	graphConfig, ok := result.Data["graph_config"].(map[string]interface{})
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"contract", "approval workflow", "legal team"}, graphConfig["nodes"])
	assert.ElementsMatch(t, []string{"belongs to", "approves", "owns"}, graphConfig["relations"])
}
