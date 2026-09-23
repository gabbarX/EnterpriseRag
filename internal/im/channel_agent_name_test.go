package im

import (
	"context"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/stretchr/testify/require"
)

func TestRelocalizeBuiltinChannelAgentNamesUsesYAMLCopy(t *testing.T) {
	restore := types.OverrideBuiltinAgentEntriesForTest(map[string]*types.BuiltinAgentEntry{
		types.BuiltinQuickAnswerID: {
			ID:     types.BuiltinQuickAnswerID,
			Avatar: "quick.png",
			I18n: map[string]types.BuiltinAgentI18n{
				"default": {Name: "Quick Answer"},
			},
		},
	})
	t.Cleanup(restore)

	rows := []ChannelWithAgent{
		{AgentID: "custom-1", TenantID: 1, AgentName: "Mine"},
		{AgentID: types.BuiltinQuickAnswerID, TenantID: 1, AgentName: ""},
		{AgentID: types.BuiltinQuickAnswerID, TenantID: 1, AgentName: "seeded-name"},
	}
	ctx := context.Background()
	relocalizeBuiltinChannelAgentNames(ctx, rows)

	require.Equal(t, "Mine", rows[0].AgentName)
	require.Equal(t, "Quick Answer", rows[1].AgentName)
	require.Equal(t, "Quick Answer", rows[2].AgentName)
}
