package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

func TestLLMToolOutputsUseMarkdownImages(t *testing.T) {
	imageInfo, err := json.Marshal([]types.ImageInfo{
		{
			URL:     "resource://AbCdEfGhIjKlMnOpQrStUv",
			Caption: "target speaker extraction flow diagram",
			OCRText: "input\noutput",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	chunk := &types.Chunk{
		ID:         "chunk-1",
		ChunkIndex: 0,
		ChunkType:  types.ChunkTypeText,
		Content:    "main flow of the testing phase",
		ImageInfo:  string(imageInfo),
	}

	readOutput := (&ReadDocumentTool{}).buildOutput(
		&types.Knowledge{ID: "knowledge-1", Title: "Test Document"}, 1, []readChunkRow{{chunk: chunk}}, "",
	)
	enrichedOutput := enrichChunkContent(chunk)
	for name, output := range map[string]string{
		"read_document":        readOutput,
		"enrich_chunk_content": enrichedOutput,
	} {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(output, "![target speaker extraction flow diagram](resource://AbCdEfGhIjKlMnOpQrStUv)") {
				t.Fatalf("expected Markdown image in tool output:\n%s", output)
			}
			if strings.Contains(output, "<image") || strings.Contains(output, "<caption>") || strings.Contains(output, "<ocr_text>") {
				t.Fatalf("tool output leaked legacy image XML:\n%s", output)
			}
		})
	}
}
