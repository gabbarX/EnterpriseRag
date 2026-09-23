package tools

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

func TestWriteFAQMetadataXML_IncludesAnswers(t *testing.T) {
	var b strings.Builder
	meta := &types.FAQChunkMetadata{
		StandardQuestion: "How do I create a knowledge base?",
		SimilarQuestions: []string{"How is a knowledge base set up?"},
		Answers:          []string{"Click New Knowledge Base in the console."},
	}
	writeFAQMetadataXML(&b, meta)
	out := b.String()

	for _, want := range []string{
		"<faq>", "<question>How do I create a knowledge base?</question>",
		"<similar_question>How is a knowledge base set up?</similar_question>",
		"<answer>Click New Knowledge Base in the console.</answer>", "</faq>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestWriteSimilarQuestionsXML_TruncatesWhenTooMany(t *testing.T) {
	var b strings.Builder
	questions := make([]string, 8)
	for i := range questions {
		questions[i] = fmt.Sprintf("similar question %d", i+1)
	}
	writeSimilarQuestionsXML(&b, questions)
	out := b.String()

	if strings.Count(out, "<similar_question>") != faqMaxSimilarQuestionsDisplay {
		t.Fatalf("want %d similar_question tags, got:\n%s", faqMaxSimilarQuestionsDisplay, out)
	}
	if !strings.Contains(out, `<similar_questions_omitted count="3"`) {
		t.Fatalf("want omitted marker, got:\n%s", out)
	}
}

func TestWriteFAQEntryXML_UsesFaqNotChunk(t *testing.T) {
	chunk := &types.Chunk{
		ID:          "faq-chunk-1",
		ChunkType:   types.ChunkTypeFAQ,
		ChunkIndex:  0,
		KnowledgeID: "kb-doc-1",
		Content:     "Q: test\nSimilar Questions:\n- alt",
	}
	meta := &types.FAQChunkMetadata{
		StandardQuestion: "How do I create a knowledge base?",
		Answers:          []string{"Just click New."},
	}
	if err := chunk.SetFAQMetadata(meta); err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	writeFAQEntryXML(&b, chunk)
	out := b.String()

	if strings.Contains(out, "<chunk") {
		t.Fatalf("FAQ list output must not use <chunk>, got:\n%s", out)
	}
	for _, want := range []string{
		`<faq faq_id="faq-chunk-1"`,
		"<question>How do I create a knowledge base?</question>",
		"<answer>Just click New.</answer>",
		"</faq>",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
