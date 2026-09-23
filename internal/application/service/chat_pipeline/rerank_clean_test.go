package chatpipeline

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

func TestGetEnrichedPassageKeepsQuestionsFromEarlierRevision(t *testing.T) {
	metadata, err := json.Marshal(types.DocumentChunkMetadata{
		GeneratedQuestionsRevision: 1,
		GeneratedQuestions: []types.GeneratedQuestion{{
			ID: "old", Question: "question generated before the edit",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	passage := getEnrichedPassage(context.Background(), &types.SearchResult{
		Content:       "edited chunk body",
		ChunkMetadata: types.JSON(metadata),
	})
	if !strings.Contains(passage, "question generated before the edit") {
		t.Fatalf("earlier generated question was excluded from rerank passage: %q", passage)
	}
}

func TestGetEnrichedPassageKeepsCodeAndMathCandidates(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "code only",
			content: "```go\nfunc answer() int { return 42 }\n```",
			want:    "func answer() int { return 42 }",
		},
		{
			name:    "math only",
			content: "$$\nE = mc^2\n$$",
			want:    "E = mc^2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passage := getEnrichedPassage(context.Background(), &types.SearchResult{Content: tt.content})
			if strings.TrimSpace(passage) == "" {
				t.Fatal("semantic-only candidate was removed from the rerank passage")
			}
			if !strings.Contains(passage, tt.want) {
				t.Fatalf("rerank passage %q does not preserve %q", passage, tt.want)
			}
		})
	}
}

func TestCleanPassageForRerank(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "plain text unchanged",
			input:  "यह एक सामान्य पाठ सामग्री है",
			expect: "यह एक सामान्य पाठ सामग्री है",
		},
		{
			name:   "remove markdown images",
			input:  "पहले ![चित्र विवरण](https://example.com/img.png) बाद में",
			expect: "पहले  बाद में",
		},
		{
			name:   "convert markdown links to text",
			input:  "कृपया [आधिकारिक दस्तावेज़](https://docs.example.com) देखें",
			expect: "कृपया आधिकारिक दस्तावेज़ देखें",
		},
		{
			name:   "remove standalone URLs",
			input:  "देखें https://example.com/path?q=1&b=2 अधिक जानकारी",
			expect: "देखें  अधिक जानकारी",
		},
		{
			name:   "unwrap code blocks",
			input:  "उदाहरण कोड:\n```python\nprint('hello')\n```\nऊपर उदाहरण है",
			expect: "उदाहरण कोड:\nprint('hello')\nऊपर उदाहरण है",
		},
		{
			name:   "unwrap LaTeX blocks",
			input:  "सूत्र इस प्रकार $$E=mc^2$$ जहाँ E ऊर्जा है",
			expect: "सूत्र इस प्रकार E=mc^2 जहाँ E ऊर्जा है",
		},
		{
			name:   "remove table separator rows and convert data rows",
			input:  "| नाम | मान |\n| --- | --- |\n| A | 1 |",
			expect: "नाम, मान\n\nA, 1",
		},
		{
			name:   "strip heading markers",
			input:  "## अध्याय दो अवलोकन\n### 2.1 पृष्ठभूमि",
			expect: "अध्याय दो अवलोकन\n2.1 पृष्ठभूमि",
		},
		{
			name:   "strip blockquote markers",
			input:  "> यह एक उद्धरण है\n> दूसरी पंक्ति",
			expect: "यह एक उद्धरण है\nदूसरी पंक्ति",
		},
		{
			name:   "unwrap bold and italic",
			input:  "यह **मोटा** और *तिरछा* तथा ***मोटा तिरछा*** पाठ है",
			expect: "यह मोटा और तिरछा तथा मोटा तिरछा पाठ है",
		},
		{
			name:   "strip list markers",
			input:  "- मद एक\n- मद दो\n1. क्रमित एक\n2. क्रमित दो",
			expect: "मद एक\nमद दो\nक्रमित एक\nक्रमित दो",
		},
		{
			name:   "remove HTML tags",
			input:  "पाठ<br>नई पंक्ति<div class=\"test\">सामग्री</div>अंत",
			expect: "पाठनई पंक्तिसामग्रीअंत",
		},
		{
			name:   "collapse excessive newlines",
			input:  "अनुच्छेद एक\n\n\n\n\nअनुच्छेद दो",
			expect: "अनुच्छेद एक\n\nअनुच्छेद दो",
		},
		{
			name: "combined real-world passage",
			input: `## उत्पाद परिचय

यह एक **महत्वपूर्ण** उत्पाद है। विवरण [उत्पाद पृष्ठ](https://example.com/product) पर।

![उत्पाद स्क्रीनशॉट](images/product.png)

> उपयोगकर्ता समीक्षा: बहुत उपयोगी

- सुविधा एक
- सुविधा दो

` + "```json\n{\"key\": \"value\"}\n```",
			expect: "उत्पाद परिचय\n\nयह एक महत्वपूर्ण उत्पाद है। विवरण उत्पाद पृष्ठ पर।\n\nउपयोगकर्ता समीक्षा: बहुत उपयोगी\n\nसुविधा एक\nसुविधा दो\n\n{\"key\": \"value\"}",
		},
		{
			name:   "convert table data rows to plain text",
			input:  "| col1 | col2 | col3 |",
			expect: "col1, col2, col3",
		},
		{
			name:   "multi-row table fully converted",
			input:  "| Header1 | Header2 |\n| --- | --- |\n| data1 | data2 |\n| data3 | data4 |",
			expect: "Header1, Header2\n\ndata1, data2\ndata3, data4",
		},
		{
			name:   "table-only passage becomes empty after separator removal",
			input:  "| --- | --- |",
			expect: "",
		},
		{
			name:   "whitespace-only after cleaning",
			input:  "   \n\n   ",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanPassageForRerank(tt.input)
			if got != tt.expect {
				t.Errorf("cleanPassageForRerank():\ngot:    %q\nexpect: %q", got, tt.expect)
			}
		})
	}
}
