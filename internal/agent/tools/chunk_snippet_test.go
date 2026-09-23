package tools

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

func faqChunkForSnippetTest(t *testing.T) *types.Chunk {
	t.Helper()
	chunk := &types.Chunk{
		ChunkType: types.ChunkTypeFAQ,
		Content: `Q: Can the resignation reason be customised
Similar Questions:
- Customise resignation reason
- Set resignation reason`,
	}
	meta := &types.FAQChunkMetadata{
		StandardQuestion: "Can the resignation reason be customised",
		SimilarQuestions: []string{"Customise resignation reason", "Set resignation reason"},
		Answers:          []string{"Yes, you can edit the reason list under system settings."},
	}
	if err := chunk.SetFAQMetadata(meta); err != nil {
		t.Fatalf("SetFAQMetadata: %v", err)
	}
	return chunk
}

func TestFaqMatchSnippet_StandardQuestionAndMetadataAnswer(t *testing.T) {
	chunk := faqChunkForSnippetTest(t)
	re := regexp.MustCompile(`(?i)be customised`)
	snippet := faqMatchSnippet(chunk, []*regexp.Regexp{re})

	if !strings.Contains(snippet, "Q: Can the resignation reason be customised") {
		t.Fatalf("want standard question in snippet, got: %s", snippet)
	}
	if !strings.Contains(snippet, "A: Yes, you can edit the reason list") {
		t.Fatalf("want metadata answer in snippet, got: %s", snippet)
	}
	for _, unwanted := range []string{"Similar Questions", "Customise resignation reason", "Set resignation reason"} {
		if strings.Contains(snippet, unwanted) {
			t.Errorf("snippet should not include similar-question noise %q: %s", unwanted, snippet)
		}
	}
}

func TestFaqMatchSnippetFromQueries_StandardQuestionAndMetadataAnswer(t *testing.T) {
	meta := &types.FAQChunkMetadata{
		StandardQuestion: "Can the resignation reason be customised",
		SimilarQuestions: []string{"Customise resignation reason", "Set resignation reason"},
		Answers:          []string{"Yes, you can edit the reason list under system settings."},
	}
	snippet := faqMatchSnippetFromQueries(meta, []string{"be customised"})

	if !strings.Contains(snippet, "Q: Can the resignation reason be customised") {
		t.Fatalf("want standard question, got: %s", snippet)
	}
	if !strings.Contains(snippet, "A: Yes, you can edit the reason list") {
		t.Fatalf("want metadata answer, got: %s", snippet)
	}
	if strings.Contains(snippet, "Similar Questions") {
		t.Fatalf("should not include similar-question list: %s", snippet)
	}
}

func TestFaqMatchSnippetFromQueries_SimilarQuestionHitShowsMatchedVariant(t *testing.T) {
	meta := &types.FAQChunkMetadata{
		StandardQuestion: "Can the resignation reason be customised",
		SimilarQuestions: []string{"Customise resignation reason", "Set resignation reason"},
		Answers:          []string{"Yes, it can be edited."},
	}
	snippet := faqMatchSnippetFromQueries(meta, []string{"Customise resignation reason"})

	if !strings.Contains(snippet, "Q: Customise resignation reason") {
		t.Fatalf("want matched similar question, got: %s", snippet)
	}
}

func TestFaqMatchSnippet_SimilarQuestionHitShowsMatchedVariant(t *testing.T) {
	chunk := faqChunkForSnippetTest(t)
	re := regexp.MustCompile(`(?i)Customise resignation reason`)
	snippet := faqMatchSnippet(chunk, []*regexp.Regexp{re})

	if !strings.Contains(snippet, "Q: Customise resignation reason") {
		t.Fatalf("want matched similar question, got: %s", snippet)
	}
	if strings.Contains(snippet, "be customised") {
		t.Fatalf("should not show standard question when similar question matched: %s", snippet)
	}
}

func TestExtractChunkMatchSnippet_NonFAQUsesBodyContext(t *testing.T) {
	content := strings.Repeat("पूर्व। ", 50) + "TARGET" + strings.Repeat("बाद। ", 50)
	chunk := &types.Chunk{ChunkType: types.ChunkTypeText, Content: content}
	re := regexp.MustCompile(`TARGET`)
	snippet := extractChunkMatchSnippet(chunk, []*regexp.Regexp{re})

	if !strings.Contains(snippet, "TARGET") {
		t.Fatalf("missing match: %s", snippet)
	}
	if !strings.Contains(snippet, "पूर्व") || !strings.Contains(snippet, "बाद") {
		t.Fatalf("expected expanded body context: %s", snippet)
	}
}
