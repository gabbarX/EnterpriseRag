package im

import (
	"strings"
	"testing"
)

func TestStripThinkBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no think blocks", input: "Hello, world!", want: "Hello, world!"},
		{name: "empty", input: "", want: ""},
		{
			name:  "single block before answer",
			input: "<think>reasoning</think>The answer is 42.",
			want:  "The answer is 42.",
		},
		{
			name:  "multiline think with tools",
			input: "<think>\nपहले ज्ञान भंडार खोजता हूँ\nकॉल कर रहा हूँ खोज कीवर्ड...\nखोज कीवर्ड: “सभ्यता”\n</think>\n\nसभ्यता 6 एक रणनीति गेम है।",
			want:  "सभ्यता 6 एक रणनीति गेम है।",
		},
		{
			name:  "multiple blocks",
			input: "<think>first</think>Part 1. <think>second</think>Part 2.",
			want:  "Part 1. Part 2.",
		},
		{name: "only think block", input: "<think>just thinking</think>", want: ""},
		{
			name:  "unclosed think block",
			input: "<think>still streaming",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripThinkBlocks(tt.input)
			if got != tt.want {
				t.Fatalf("StripThinkBlocks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatIMDisplayContent_intermediate_showsThinkingStyled(t *testing.T) {
	raw := "<think>\nउपयोगकर्ता का प्रश्न समझ रहा हूँ\nकॉल कर रहा हूँ ज्ञान भंडार खोज...\n</think>\n\n"
	got := FormatIMDisplayContent(raw, StreamDisplayIntermediate)

	if !strings.Contains(got, "Reasoning") {
		t.Fatalf("intermediate display should include thinking header, got: %q", got)
	}
	if !strings.Contains(got, "उपयोगकर्ता का प्रश्न समझ रहा हूँ") {
		t.Fatalf("intermediate display should include thinking body, got: %q", got)
	}
	if !strings.Contains(got, "ज्ञान भंडार खोज") {
		t.Fatalf("intermediate display should include tool progress, got: %q", got)
	}
	if strings.Contains(got, "<think>") {
		t.Fatalf("intermediate display must not leak raw think tags, got: %q", got)
	}
}

func TestFormatIMDisplayContent_intermediate_inProgressThink(t *testing.T) {
	raw := "<think>\nतर्क कर रहा हूँ\nकॉल कर रहा हूँ खोज कीवर्ड...\n"
	got := FormatIMDisplayContent(raw, StreamDisplayIntermediate)

	if !strings.Contains(got, "Reasoning") {
		t.Fatalf("open think block should show thinking header, got: %q", got)
	}
	if strings.Contains(got, "<think>") {
		t.Fatalf("must not leak raw tags, got: %q", got)
	}
}

func TestFormatIMDisplayContent_intermediate_showsAnswerPreview(t *testing.T) {
	raw := "<think>\nखोज जारी\n</think>\n\nसभ्यता 6 एक टर्न-बेस्ड रणनीति गेम है।"
	got := FormatIMDisplayContent(raw, StreamDisplayIntermediate)

	if !strings.Contains(got, "सभ्यता 6 एक टर्न-बेस्ड रणनीति गेम है") {
		t.Fatalf("intermediate display should preview answer after think block, got: %q", got)
	}
}

func TestFormatIMDisplayContent_final_stripsThinkingAndTools(t *testing.T) {
	raw := "<think>\nपहले ज्ञान भंडार खोजता हूँ\nकॉल कर रहा हूँ खोज कीवर्ड...\nखोज कीवर्ड: “सभ्यता”\n</think>\n\nसभ्यता 6 एक रणनीति गेम है।"
	got := FormatIMDisplayContent(raw, StreamDisplayFinal)

	want := "सभ्यता 6 एक रणनीति गेम है।"
	if got != want {
		t.Fatalf("final display = %q, want %q", got, want)
	}
}

func TestFormatIMDisplayContent_final_plainAnswerUnchanged(t *testing.T) {
	raw := "यह अंतिम उत्तर है।"
	got := FormatIMDisplayContent(raw, StreamDisplayFinal)
	if got != raw {
		t.Fatalf("final display = %q, want %q", got, raw)
	}
}

func TestFormatIMDisplayContent_final_ragPipelineHidden(t *testing.T) {
	raw := "<think>\nप्रश्न समझ रहा हूँ...\nप्रश्न समझ पूरी हुई\nज्ञान भंडार खोज रहा हूँ...\nज्ञान भंडार खोज: “query” · 3 परिणाम मिले\n</think>\n\nज्ञान भंडार के अनुसार, उत्तर A है।"
	got := FormatIMDisplayContent(raw, StreamDisplayFinal)

	if strings.Contains(got, "प्रश्न समझ") || strings.Contains(got, "ज्ञान भंडार खोज") {
		t.Fatalf("final display must not contain RAG pipeline steps, got: %q", got)
	}
	if got != "ज्ञान भंडार के अनुसार, उत्तर A है।" {
		t.Fatalf("final display = %q", got)
	}
}

func TestFormatIMAgentIntermediate_answerFirstBeforeTools(t *testing.T) {
	parts := IMStreamParts{
		Mode:       IMStreamModeAgent,
		LiveAnswer: "ठीक है, पहले ज्ञान भंडार खोजता हूँ।",
	}
	got := FormatIMIntermediateFromParts(parts, true)
	if got != "ठीक है, पहले ज्ञान भंडार खोजता हूँ।" {
		t.Fatalf("should stream as plain answer, got: %q", got)
	}
	if strings.Contains(got, "Reasoning") {
		t.Fatal("think header must not appear while answer is live")
	}
}

func TestFormatIMAgentIntermediate_retractIntoThinkOnTools(t *testing.T) {
	parts := IMStreamParts{
		Mode:       IMStreamModeAgent,
		AgentInner: "ठीक है, पहले ज्ञान भंडार खोजता हूँ।\n",
		AgentToolSteps: []IMToolStep{
			{ToolName: "grep_chunks", Pending: true},
			{ToolName: "knowledge_search", Success: true, Arguments: map[string]any{"query": "सभ्यता 6"}},
		},
	}
	got := FormatIMIntermediateFromParts(parts, true)
	if !strings.Contains(got, "Reasoning") {
		t.Fatalf("after tool retract should show think block, got: %q", got)
	}
	if !strings.Contains(got, "ठीक है, पहले ज्ञान भंडार खोजता हूँ") {
		t.Fatalf("retracted preamble should be inside think, got: %q", got)
	}
	if !strings.Contains(got, "Keyword search") {
		t.Fatalf("tool lines should be inside think, got: %q", got)
	}
	if !strings.Contains(got, "सभ्यता 6") {
		t.Fatalf("tool query should be inside think, got: %q", got)
	}
}

func TestFormatIMAgentIntermediate_newAnswerAfterTools(t *testing.T) {
	parts := IMStreamParts{
		Mode:       IMStreamModeAgent,
		AgentInner: "ठीक है, मैं खोजता हूँ\n",
		AgentToolSteps: []IMToolStep{
			{ToolName: "knowledge_search", Success: true, Arguments: map[string]any{"query": "सभ्यता 6"}},
		},
		LiveAnswer: "खोज परिणाम के अनुसार, सभ्यता 6…",
	}
	got := FormatIMIntermediateFromParts(parts, true)
	if !strings.Contains(got, "खोज परिणाम के अनुसार, सभ्यता 6…") {
		t.Fatalf("should still stream live answer, got: %q", got)
	}
	if !strings.Contains(got, "Reasoning") {
		t.Fatalf("think block should stay visible above answer, got: %q", got)
	}
	if !strings.Contains(got, "सभ्यता 6") {
		t.Fatalf("tool query should remain in think block, got: %q", got)
	}
}

func TestBuildIMStreamRaw_agentInProgress_mergesToolsAndNarrativeIntoThink(t *testing.T) {
	parts := IMStreamParts{
		Mode:       IMStreamModeAgent,
		AgentInner: "उपयोगकर्ता ने फिर सभ्यता 6 पूछा\n",
		AgentToolSteps: []IMToolStep{
			{ToolName: "grep_chunks", Pending: true},
			{ToolName: "knowledge_search", Success: true},
		},
	}
	got := FormatIMIntermediateFromParts(parts, true)

	if !strings.Contains(got, "Keyword search") {
		t.Fatalf("tool progress should be inside think block, got: %q", got)
	}
	if !strings.Contains(got, "Reasoning") {
		t.Fatalf("agent tooling phase should show Reasoning, got: %q", got)
	}
}

func TestFormatIMQuickQA_separatesPipelineAndThinking(t *testing.T) {
	parts := IMStreamParts{
		Mode: IMStreamModeQuickQA,
		PipelineToolSteps: []IMToolStep{
			{ToolName: "query_understand", Pending: true},
			{ToolName: "knowledge_search", Success: true, Arguments: map[string]any{"query": "सभ्यता 6"}},
		},
		ReasoningInner: "प्रश्न का आशय समझ रहा हूँ…",
	}
	got := FormatIMIntermediateFromParts(parts, false)

	if strings.Contains(got, "Reasoning") {
		t.Fatalf("quick QA should not use agent Reasoning header, got: %q", got)
	}
	if !strings.Contains(got, "> 💭 **Thoughts**") {
		t.Fatalf("quick QA reasoning should use separate Thoughts section, got: %q", got)
	}
	if !strings.Contains(got, "प्रश्न का आशय समझ रहा हूँ") {
		t.Fatalf("reasoning body missing, got: %q", got)
	}
	if !strings.Contains(got, "Understanding the question") {
		t.Fatalf("pipeline steps missing, got: %q", got)
	}
	if !strings.Contains(got, "सभ्यता 6") {
		t.Fatalf("pipeline query missing, got: %q", got)
	}
}

func TestFormatIMQuickQA_collapsesToAnswerWhenStreaming(t *testing.T) {
	parts := IMStreamParts{
		Mode: IMStreamModeQuickQA,
		PipelineToolSteps: []IMToolStep{
			{ToolName: "query_understand", Success: true},
			{ToolName: "knowledge_search", Success: true},
		},
		Answer: "सभ्यता 6 एक टर्न-बेस्ड रणनीति गेम है।",
	}
	got := FormatIMIntermediateFromParts(parts, false)
	if got != "सभ्यता 6 एक टर्न-बेस्ड रणनीति गेम है।" {
		t.Fatalf("quick QA should collapse to answer preview, got: %q", got)
	}
}

func TestFormatIMFinalFromParts_agentAnswerOnly(t *testing.T) {
	parts := IMStreamParts{
		Mode:       IMStreamModeAgent,
		AgentInner: "ठीक है, मैं खोजता हूँ\n",
		AgentToolSteps: []IMToolStep{
			{ToolName: "grep_chunks", Pending: true},
			{ToolName: "knowledge_search", Success: true},
		},
		LiveAnswer: "अंतिम संदेश में नहीं आना चाहिए",
		Answer:     "सभ्यता 6 एक रणनीति गेम है।",
	}
	got := FormatIMFinalFromParts(parts)
	if got != "सभ्यता 6 एक रणनीति गेम है।" {
		t.Fatalf("final should be answer-only, got: %q", got)
	}
	if strings.Contains(got, "Reasoning") {
		t.Fatalf("final must not include collapsed think header, got: %q", got)
	}
}

func TestFormatIMFinalFromParts_usesAnswerOnly(t *testing.T) {
	parts := IMStreamParts{
		Mode:              IMStreamModeQuickQA,
		PipelineToolSteps: []IMToolStep{{ToolName: "query_understand", Success: true}},
		ReasoningInner:    "तर्क जारी",
		AgentToolSteps:    []IMToolStep{{ToolName: "grep_chunks", Pending: true}},
		Answer:            "सभ्यता 6 एक रणनीति गेम है।",
	}
	got := FormatIMFinalFromParts(parts)
	if got != "सभ्यता 6 एक रणनीति गेम है।" {
		t.Fatalf("final display = %q", got)
	}
}

func TestIsRAGPipelineToolName_matchesWeb(t *testing.T) {
	for _, name := range []string{"query_understand", "knowledge_search"} {
		if !IsRAGPipelineToolName(name) {
			t.Fatalf("%q should be a RAG pipeline tool", name)
		}
	}
	if IsRAGPipelineToolName("grep_chunks") {
		t.Fatal("grep_chunks is agent tool, not RAG pipeline progress tool")
	}
}
