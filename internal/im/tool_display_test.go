package im

import (
	"strings"
	"testing"
)

func TestFormatIMToolLine_pendingWithQuery(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName:  "knowledge_search",
		Pending:   true,
		Arguments: map[string]any{"query": "सभ्यता 6"},
	})
	if line != "Calling Knowledge base search..." {
		t.Fatalf("pending line = %q", line)
	}
}

func TestFormatIMToolLine_searchDoneWithQueryAndSummary(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName: "knowledge_search",
		Success:  true,
		Arguments: map[string]any{
			"query": "सभ्यता 6",
		},
		Data: map[string]interface{}{
			"results":   []interface{}{map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}},
			"kb_counts": map[string]interface{}{"a": 1, "b": 1},
		},
	})
	if !strings.Contains(line, "Searched knowledge base: \"सभ्यता 6\"") {
		t.Fatalf("title missing query: %q", line)
	}
	if !strings.Contains(line, "Found 3 results across 2 files") {
		t.Fatalf("summary missing: %q", line)
	}
}

func TestFormatIMToolLine_grepPatterns(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName: "grep_chunks",
		Success:  true,
		Arguments: map[string]any{
			"patterns": []any{"सभ्यता", "रणनीति"},
		},
		Data: map[string]interface{}{
			"total_matches":  float64(5),
			"document_count": float64(2),
		},
	})
	if line != "Keyword search: \"सभ्यता, रणनीति\" · Found 5 matching snippets across 2 documents" {
		t.Fatalf("grep line = %q", line)
	}
}

func TestFormatIMRagPipelineLine_queryUnderstand(t *testing.T) {
	pending := FormatIMRagPipelineLine(IMToolStep{
		ToolName: "query_understand",
		Pending:  true,
	})
	if pending != "Understanding the question..." {
		t.Fatalf("pending = %q", pending)
	}
	done := FormatIMRagPipelineLine(IMToolStep{
		ToolName: "query_understand",
		Success:  true,
	})
	if done != "Question understood" {
		t.Fatalf("done = %q", done)
	}
}

func TestFormatIMRagPipelineLine_searchWithQuery(t *testing.T) {
	line := FormatIMRagPipelineLine(IMToolStep{
		ToolName:  "knowledge_search",
		Pending:   true,
		Arguments: map[string]any{"query": "भारतीय रेल पोर्टल"},
	})
	if line != "Searching knowledge base: \"भारतीय रेल पोर्टल\"" {
		t.Fatalf("line = %q", line)
	}
}

func TestFormatIMRagPipelineLine_webSearchWithQuery(t *testing.T) {
	line := FormatIMRagPipelineLine(IMToolStep{
		ToolName:  "knowledge_search",
		Pending:   true,
		Arguments: map[string]any{"query": "अरिजीत सिंह कॉन्सर्ट", "search_source": "web"},
	})
	if line != "Searching the web: \"अरिजीत सिंह कॉन्सर्ट\"" {
		t.Fatalf("line = %q", line)
	}
}

func TestIMGetQueryText_joinsUniqueQueries(t *testing.T) {
	got := imGetQueryText(map[string]any{
		"query":   "foo",
		"queries": []any{"foo", "bar"},
	})
	if got != "foo, bar" {
		t.Fatalf("query text = %q", got)
	}
}

func TestFormatIMToolLine_writeSandboxPendingShowsDiffStat(t *testing.T) {
	line := FormatIMToolLine(IMToolStep{
		ToolName: "write_sandbox_file",
		Pending:  true,
		Arguments: map[string]any{
			"path":          "/workspace/output/a.py",
			"added_lines":   12,
			"removed_lines": 0,
		},
	})
	if line != "Write sandbox file: \"/workspace/output/a.py\"... +12" {
		t.Fatalf("pending write line = %q", line)
	}
}
