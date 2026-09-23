package chatpipeline

import (
	"context"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/config"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

// Phase two's claim is that memory changes what gets retrieved, not only what
// the answer prompt says. These tests hold the two places that has to be true:
// the query the retriever is given, and the order documents come back in.

type stubRetrievalMemory struct {
	stubMemoryService

	retrieval interfaces.RetrievalContext
	affinity  map[string]int
	askedFor  []string
}

func (s *stubRetrievalMemory) RetrievalContextFor(context.Context) interfaces.RetrievalContext {
	return s.retrieval
}

func (s *stubRetrievalMemory) DocumentAffinity(_ context.Context, ids []string) map[string]int {
	s.askedFor = ids
	return s.affinity
}

func TestWhoIsAskingReachesTheQueryRewriter(t *testing.T) {
	memoryService := &stubRetrievalMemory{
		retrieval: interfaces.RetrievalContext{
			Background: "चिकित्सा इमेजिंग का बैकएंड बना रहे हैं",
			Interests:  []string{"चिकित्सा इमेज सेगमेंटेशन"},
			Documents:  []string{"सेगमेंटेशन मॉडल ट्यूनिंग पुस्तिका"},
			Items: []*types.MemoryItem{
				{ID: "m1", Kind: types.MemoryKindProfile, Content: "चिकित्सा इमेजिंग का बैकएंड बना रहे हैं"},
			},
		},
	}
	plugin := &PluginQueryUnderstand{
		memoryService: memoryService,
		config: &config.Config{Conversation: &config.ConversationConfig{
			RewritePromptSystem: "उपयोगकर्ता के प्रश्न को फिर से लिखें।",
			RewritePromptUser:   "{{query}}",
		}},
	}

	chatManage := &types.ChatManage{}
	chatManage.Query = "सेगमेंटेशन की ट्यूनिंग कैसे करें"

	_, userPrompt := plugin.buildPrompts(t.Context(), chatManage, nil)

	require.Contains(t, userPrompt, "चिकित्सा इमेजिंग का बैकएंड बना रहे हैं",
		"the same question means different things to different people, and only "+
			"the rewriter can act on that before retrieval runs")
	require.Contains(t, userPrompt, "चिकित्सा इमेज सेगमेंटेशन")
	require.Contains(t, userPrompt, "सेगमेंटेशन मॉडल ट्यूनिंग पुस्तिका")
	require.Contains(t, userPrompt, "सेगमेंटेशन की ट्यूनिंग कैसे करें", "the question itself must survive")

	// Conditioning the rewriter is not a recall. The background is fed in
	// whole, relevant or not, so counting it as "memories this answer used"
	// would report unrelated memories on every single turn. That list is
	// MEMORY_RECALL's to build, from what the question actually matched.
	require.Empty(t, chatManage.UsedMemories)
}

func TestQueryRewriterIsUnchangedWithoutMemory(t *testing.T) {
	plugin := &PluginQueryUnderstand{
		memoryService: &stubRetrievalMemory{},
		config: &config.Config{Conversation: &config.ConversationConfig{
			RewritePromptSystem: "उपयोगकर्ता के प्रश्न को फिर से लिखें।",
			RewritePromptUser:   "{{query}}",
		}},
	}
	chatManage := &types.ChatManage{}
	chatManage.Query = "सेगमेंटेशन की ट्यूनिंग कैसे करें"

	_, userPrompt := plugin.buildPrompts(t.Context(), chatManage, nil)
	require.NotContains(t, userPrompt, "asker_background")
	require.Empty(t, chatManage.UsedMemories)
}

func TestFamiliarDocumentsRankHigher(t *testing.T) {
	memoryService := &stubRetrievalMemory{affinity: map[string]int{"doc-familiar": 8}}
	plugin := &PluginMemoryAffinity{memoryService: memoryService}

	chatManage := &types.ChatManage{
		PipelineState: types.PipelineState{RerankResult: []*types.SearchResult{
			{ID: "c1", KnowledgeID: "doc-stranger", Score: 0.80},
			{ID: "c2", KnowledgeID: "doc-familiar", Score: 0.78},
		}},
	}

	err := plugin.OnEvent(t.Context(), types.CHUNK_RERANK, chatManage, func() *PluginError {
		return nil
	})
	require.Nil(t, err)
	require.Equal(t, "c2", chatManage.RerankResult[0].ID,
		"between two comparable passages, prefer the document this person works from")
}

func TestAnUnrelatedDocumentIsNotDraggedToTheTop(t *testing.T) {
	// The signal is weak — it says the retriever kept picking a document, not
	// that the user found it useful — so it must never overturn a clear
	// relevance gap.
	memoryService := &stubRetrievalMemory{affinity: map[string]int{"doc-familiar": 1000}}
	plugin := &PluginMemoryAffinity{memoryService: memoryService}

	chatManage := &types.ChatManage{
		PipelineState: types.PipelineState{RerankResult: []*types.SearchResult{
			{ID: "c1", KnowledgeID: "doc-relevant", Score: 0.90},
			{ID: "c2", KnowledgeID: "doc-familiar", Score: 0.40},
		}},
	}

	err := plugin.OnEvent(t.Context(), types.CHUNK_RERANK, chatManage, func() *PluginError {
		return nil
	})
	require.Nil(t, err)
	require.Equal(t, "c1", chatManage.RerankResult[0].ID)
}

func TestRerankIsUntouchedWithoutAffinity(t *testing.T) {
	plugin := &PluginMemoryAffinity{memoryService: &stubRetrievalMemory{}}
	chatManage := &types.ChatManage{
		PipelineState: types.PipelineState{RerankResult: []*types.SearchResult{
			{ID: "c1", KnowledgeID: "doc-a", Score: 0.80},
			{ID: "c2", KnowledgeID: "doc-b", Score: 0.78},
		}},
	}
	err := plugin.OnEvent(t.Context(), types.CHUNK_RERANK, chatManage, func() *PluginError {
		return nil
	})
	require.Nil(t, err)
	require.Equal(t, 0.80, chatManage.RerankResult[0].Score)
	require.Equal(t, 0.78, chatManage.RerankResult[1].Score)
}
