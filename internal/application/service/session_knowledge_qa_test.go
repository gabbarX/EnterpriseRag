package service

import (
	"context"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/event"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/asr"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/chat"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/embedding"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/rerank"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vlm"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type captureChatModel struct {
	lastMessages []chat.Message
}

func (m *captureChatModel) Chat(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (*types.ChatResponse, error) {
	return nil, nil
}

func (m *captureChatModel) ChatStream(
	_ context.Context,
	messages []chat.Message,
	_ *chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	m.lastMessages = append([]chat.Message(nil), messages...)

	ch := make(chan types.StreamResponse, 1)
	ch <- types.StreamResponse{
		ResponseType: types.ResponseTypeAnswer,
		Content:      "ok",
		Done:         true,
	}
	close(ch)
	return ch, nil
}

func (m *captureChatModel) GetModelName() string { return "capture" }
func (m *captureChatModel) GetModelID() string   { return "capture" }

type stubModelService struct {
	chatModel       chat.Chat
	modelsByID      map[string]*types.Model
	availableModels []*types.Model
}

func TestEmitKnowledgeReferencesEventIgnoresCitationOutputSetting(t *testing.T) {
	bus := event.NewEventBus()
	var emitted []event.Event
	bus.On(event.EventAgentReferences, func(_ context.Context, evt event.Event) error {
		emitted = append(emitted, evt)
		return nil
	})
	disabled := false
	result := &types.SearchResult{ID: "chunk-1", KnowledgeTitle: "Doc"}
	cm := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{CitationEnabled: &disabled},
		PipelineState: types.PipelineState{
			MergeResult: []*types.SearchResult{result},
		},
		PipelineContext: types.PipelineContext{EventBus: bus.AsEventBusInterface()},
	}

	emitKnowledgeReferencesEvent(context.Background(), cm)
	require.Len(t, emitted, 1)
	require.Equal(t, event.EventAgentReferences, emitted[0].Type)
	require.Equal(t, []*types.SearchResult{result}, emitted[0].Data.(event.AgentReferencesData).References)

	enabled := true
	cm.CitationEnabled = &enabled
	emitKnowledgeReferencesEvent(context.Background(), cm)
	require.Len(t, emitted, 2)
}

func (s *stubModelService) CreateModel(context.Context, *types.Model) error {
	return nil
}

func (s *stubModelService) GetModelByID(_ context.Context, id string) (*types.Model, error) {
	return s.modelsByID[id], nil
}

func (s *stubModelService) ListModels(context.Context) ([]*types.Model, error) {
	return s.availableModels, nil
}

func (s *stubModelService) UpdateModel(context.Context, *types.Model) error {
	return nil
}

func (s *stubModelService) DeleteModel(context.Context, string) error {
	return nil
}

func (s *stubModelService) UpdateModelCredentials(
	context.Context, string, *string, *string,
) (*types.Model, error) {
	return nil, nil
}

func (s *stubModelService) ClearModelCredential(context.Context, string, string) error {
	return nil
}

func (s *stubModelService) GetEmbeddingModel(context.Context, string) (embedding.Embedder, error) {
	return nil, nil
}

func (s *stubModelService) GetEmbeddingModelForTenant(context.Context, string, uint64) (embedding.Embedder, error) {
	return nil, nil
}

func (s *stubModelService) GetRerankModel(context.Context, string) (rerank.Reranker, error) {
	return nil, nil
}

func (s *stubModelService) GetChatModel(context.Context, string) (chat.Chat, error) {
	return s.chatModel, nil
}

func (s *stubModelService) GetVLMModel(context.Context, string) (vlm.VLM, error) {
	return nil, nil
}

func (s *stubModelService) GetASRModel(context.Context, string) (asr.ASR, error) {
	return nil, nil
}

func TestHandleModelFallback_IncludesHistoryMessages(t *testing.T) {
	chatModel := &captureChatModel{}
	svc := &sessionService{
		modelService: &stubModelService{chatModel: chatModel},
	}

	bus := event.NewEventBus()
	cm := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{
			SessionID:      "session-1",
			Query:          "क्या आप आगे बता सकते हैं?",
			ChatModelID:    "chat-model",
			FallbackPrompt: "Answer the latest user question: {{query}}",
			SummaryConfig: types.SummaryConfig{
				Temperature: 0.2,
			},
			Language: "zh-CN",
		},
		PipelineState: types.PipelineState{
			History: []*types.History{
				{
					Query:  "पहले EnterpriseRag के बारे में बताइए",
					Answer: "EnterpriseRag एक नॉलेज बेस प्रश्नोत्तर प्रणाली है।",
				},
			},
		},
		PipelineContext: types.PipelineContext{
			EventBus: bus.AsEventBusInterface(),
		},
	}

	svc.handleModelFallback(context.Background(), cm)

	// Corrected fallback shape: a system message carries the fallback
	// instruction, history is replayed in the middle, and the turn ends on the
	// user's question. Previously the system message was dropped entirely.
	require.Len(t, chatModel.lastMessages, 4)
	assert.Equal(t, "system", chatModel.lastMessages[0].Role)
	assert.Contains(t, chatModel.lastMessages[0].Content, "Answer the latest user question")
	assert.Equal(t, "user", chatModel.lastMessages[1].Role)
	assert.Equal(t, "पहले EnterpriseRag के बारे में बताइए", chatModel.lastMessages[1].Content)
	assert.Equal(t, "assistant", chatModel.lastMessages[2].Role)
	assert.Equal(t, "EnterpriseRag एक नॉलेज बेस प्रश्नोत्तर प्रणाली है।", chatModel.lastMessages[2].Content)
	assert.Equal(t, "user", chatModel.lastMessages[3].Role)
	assert.Contains(t, chatModel.lastMessages[3].Content, "क्या आप आगे बता सकते हैं?")
}
