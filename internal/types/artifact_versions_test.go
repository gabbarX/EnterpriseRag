package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactVersionsPreserveBothFiles(t *testing.T) {
	old := MessageArtifact{URL: "resource://dHZ_fFslfs0GgJGaJZGjGA", FileName: "प्रतियोगिता.pptx", SourcePath: "/workspace/output/प्रतियोगिता.pptx"}
	next := old
	next.URL = "resource://4N1nAo-FZZoDEExDQz2yoA"
	body := "A new version was generated. ![PPT](" + old.URL + ")"
	got := ClarifyArtifactVersions(body, MessageArtifacts{next}, MessageArtifacts{old})
	require.True(t, strings.HasPrefix(got, body), "must not silently redirect an intentional historical reference")
	require.Contains(t, got, "Referenced previous version")
	require.Contains(t, got, "File generated this turn: ![प्रतियोगिता.pptx]("+next.URL+")")
	require.Equal(t, got, ClarifyArtifactVersions(got, MessageArtifacts{next}, MessageArtifacts{old}))
	require.Equal(t, body, ClarifyArtifactVersions(body, MessageArtifacts{old}, MessageArtifacts{old}))
	// An explicit comparison already contains the new file; leave it alone.
	comparison := body + "\n![new version](" + next.URL + ")"
	require.Equal(t, comparison, ClarifyArtifactVersions(comparison, MessageArtifacts{next}, MessageArtifacts{old}))
	next.SourcePath = "/workspace/output/other/प्रतियोगिता.pptx"
	require.Equal(t, body, ClarifyArtifactVersions(body, MessageArtifacts{next}, MessageArtifacts{old}), "matching file names alone cannot establish a version relationship")
}

func TestArtifactVersionsEscapeMarkdownDelimitersInFileName(t *testing.T) {
	old := MessageArtifact{URL: "resource://dHZ_fFslfs0GgJGaJZGjGA", FileName: "रिपोर्ट(अंतिम).pptx", SourcePath: "/workspace/output/रिपोर्ट(अंतिम).pptx"}
	next := old
	next.URL = "resource://4N1nAo-FZZoDEExDQz2yoA"
	got := ClarifyArtifactVersions("![old]("+old.URL+")", MessageArtifacts{next}, MessageArtifacts{old})
	require.Contains(t, got, "![रिपोर्ट\\(अंतिम\\).pptx]("+old.URL+")")
	require.Contains(t, got, "![रिपोर्ट\\(अंतिम\\).pptx]("+next.URL+")")
	require.True(t, strings.HasSuffix(got, "]("+next.URL+")"))
}
