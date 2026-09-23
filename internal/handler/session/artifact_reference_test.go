package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/stretchr/testify/require"
)

// handleFor builds a syntactically valid 22-character resource handle for
// artifact i, so tests exercise the same ParseResourcePath path as production.
func handleFor(i int) string {
	base := fmt.Sprintf("art%d", i)
	return base + strings.Repeat("x", types.ResourceHandleLength-len(base))
}

func refFor(i int) string {
	return types.BuildResourcePath(handleFor(i))
}

// artifactsFixture builds artifacts whose storage URL is a catalog handle —
// the normal deployment. Use artifactsWithoutCatalog for the degraded case.
func artifactsFixture(names ...string) types.MessageArtifacts {
	list := make(types.MessageArtifacts, 0, len(names))
	for i, name := range names {
		list = append(list, types.MessageArtifact{FileName: name, URL: refFor(i)})
	}
	return list
}

func artifactsWithoutCatalog(names ...string) types.MessageArtifacts {
	list := make(types.MessageArtifacts, 0, len(names))
	for _, name := range names {
		list = append(list, types.MessageArtifact{FileName: name, URL: "local://7/exports/" + name})
	}
	return list
}

func TestRewriteArtifactReferences(t *testing.T) {
	artifacts := artifactsFixture(
		"बाज़ारस्कोर_e7edba.html",
		"concept_ranking.csv",
		"trend.png",
		"रिलायंस(500325) वॉल्यूम_838ccc.html",
	)

	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "file name with spaces and parentheses",
			content: "![volume](sandbox:रिलायंस(500325) वॉल्यूम_838ccc.html)",
			want:    "![volume](" + refFor(3) + ")",
		},
		{
			name:    "file name with spaces and parentheses, no prefix",
			content: "![volume](रिलायंस(500325) वॉल्यूम_838ccc.html)",
			want:    "![volume](" + refFor(3) + ")",
		},
		{
			name:    "bare file name in image",
			content: "![market score](बाज़ारस्कोर_e7edba.html)",
			want:    "![market score](" + refFor(0) + ")",
		},
		{
			name:    "sandbox prefix",
			content: "![score](sandbox:बाज़ारस्कोर_e7edba.html)",
			want:    "![score](" + refFor(0) + ")",
		},
		{
			name:    "sandbox scheme with slashes",
			content: "![score](sandbox://trend.png)",
			want:    "![score](" + refFor(2) + ")",
		},
		{
			name:    "directory prefix is dropped",
			content: "[ranking](/workspace/output/concept_ranking.csv)",
			want:    "[ranking](" + refFor(1) + ")",
		},
		{
			name:    "percent-encoded name",
			content: "![score](%E0%A4%AC%E0%A4%BE%E0%A4%9C%E0%A4%BC%E0%A4%BE%E0%A4%B0%E0%A4%B8%E0%A5%8D%E0%A4%95%E0%A5%8B%E0%A4%B0_e7edba.html)",
			want:    "![score](" + refFor(0) + ")",
		},
		{
			name:    "title is preserved",
			content: `![score](trend.png "trend")`,
			want:    `![score](` + refFor(2) + ` "trend")`,
		},
		{
			name:    "ordinary link with a colliding name is not rewritten",
			content: "See [note](trend.png)",
			want:    "See [note](trend.png)",
		},
		{
			name:    "sandbox-prefixed link is rewritten even when not an image",
			content: "Data: [table](sandbox:concept_ranking.csv)",
			want:    "Data: [table](" + refFor(1) + ")",
		},
		{
			name:    "already-rewritten reference is left alone",
			content: "![score](" + refFor(0) + ")",
			want:    "![score](" + refFor(0) + ")",
		},
		{
			name:    "prose parentheses are not link destinations",
			content: "रिलायंस(500325) का वॉल्यूम नीचे देखें।",
			want:    "रिलायंस(500325) का वॉल्यूम नीचे देखें।",
		},
		{
			name:    "unknown file name untouched",
			content: "![other](missing.html)",
			want:    "![other](missing.html)",
		},
		{
			name:    "http url untouched",
			content: "![remote](https://example.com/trend.png)",
			want:    "![remote](https://example.com/trend.png)",
		},
		{
			name:    "knowledge base image untouched",
			content: "![resource](resource://abcdefghijklmnopqrstuv)",
			want:    "![resource](resource://abcdefghijklmnopqrstuv)",
		},
		{
			name:    "fenced code untouched",
			content: "```\n![score](trend.png)\n```",
			want:    "```\n![score](trend.png)\n```",
		},
		{
			name:    "inline code untouched",
			content: "Write `![score](trend.png)` instead",
			want:    "Write `![score](trend.png)` instead",
		},
		{
			name:    "plain prose untouched",
			content: "Generated the two files trend.png and concept_ranking.csv.",
			want:    "Generated the two files trend.png and concept_ranking.csv.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rewriteArtifactReferences(tc.content, artifacts); got != tc.want {
				t.Fatalf("rewriteArtifactReferences() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRewriteArtifactReferencesMixedContent(t *testing.T) {
	artifacts := artifactsFixture("chart.html", "data.csv")
	content := "## Charts\n\n![chart](chart.html)\n\nData: [table](sandbox:data.csv); " +
		"the external [document](https://example.com/chart.html) link is unaffected."
	want := "## Charts\n\n![chart](" + refFor(0) + ")\n\nData: [table](" + refFor(1) + "); " +
		"the external [document](https://example.com/chart.html) link is unaffected."

	if got := rewriteArtifactReferences(content, artifacts); got != want {
		t.Fatalf("rewriteArtifactReferences() = %q, want %q", got, want)
	}
}

// A knowledge-base image and a skill artifact routinely appear in the same
// answer. Only the artifact is rebound; the existing reference must survive
// byte for byte, since it already is the canonical form.
func TestRewriteArtifactReferencesKeepsExistingResourceImages(t *testing.T) {
	artifacts := artifactsFixture("chart.html")
	kbImage := types.BuildResourcePath(strings.Repeat("Z", types.ResourceHandleLength))
	content := "![retrieved image](" + kbImage + ")\n\n![chart](chart.html)"
	want := "![retrieved image](" + kbImage + ")\n\n![chart](" + refFor(0) + ")"

	if got := rewriteArtifactReferences(content, artifacts); got != want {
		t.Fatalf("rewriteArtifactReferences() = %q, want %q", got, want)
	}
}

// Without a resource catalog there is no durable handle, so references are
// normalized to the chat-only sandbox form rather than leaking a storage path.
func TestRewriteArtifactReferencesWithoutCatalog(t *testing.T) {
	artifacts := artifactsWithoutCatalog("chart.html")
	got := rewriteArtifactReferences("![chart](chart.html)", artifacts)
	if want := "![chart](sandbox:chart.html)"; got != want {
		t.Fatalf("rewriteArtifactReferences() = %q, want %q", got, want)
	}
	if strings.Contains(got, "local://") {
		t.Fatalf("storage path leaked into content: %q", got)
	}
}

func TestRewriteArtifactReferencesNoArtifacts(t *testing.T) {
	content := "![chart](chart.html)"
	if got := rewriteArtifactReferences(content, nil); got != content {
		t.Fatalf("rewriteArtifactReferences() = %q, want unchanged", got)
	}
}

func TestArtifactRefByNameKeepsFirstDuplicate(t *testing.T) {
	byName := artifactRefByName(artifactsFixture("a.html", "a.html", "b.html"))
	if byName["a.html"] != refFor(0) {
		t.Fatalf("duplicate name resolved to %q, want %q", byName["a.html"], refFor(0))
	}
	if byName["b.html"] != refFor(2) {
		t.Fatalf("b.html resolved to %q, want %q", byName["b.html"], refFor(2))
	}
}

func TestReferencedArtifactsMatchesNamesAndHandles(t *testing.T) {
	artifacts := artifactsFixture("report.pptx", "chart.html", "data.csv")

	cases := []struct {
		name    string
		content string
		want    types.MessageArtifacts
	}{
		{
			name:    "sandbox-prefixed name",
			content: "Generated ![report](sandbox:report.pptx)",
			want:    types.MessageArtifacts{artifacts[0]},
		},
		{
			name:    "bare name in an image",
			content: "![chart](chart.html)",
			want:    types.MessageArtifacts{artifacts[1]},
		},
		{
			name:    "output path in an ordinary link",
			content: "[data](./output/data.csv)",
			want:    types.MessageArtifacts{artifacts[2]},
		},
		{
			name:    "canonical handle",
			content: "![report](" + refFor(0) + ")",
			want:    types.MessageArtifacts{artifacts[0]},
		},
		{
			name:    "prose mention is not a reference",
			content: "Generated the two files report.pptx and chart.html.",
			want:    nil,
		},
		{
			name:    "bare name in an ordinary link is not a reference",
			content: "See [note](report.pptx)",
			want:    nil,
		},
		{
			name:    "unknown name",
			content: "![other](missing.pptx)",
			want:    nil,
		},
		{
			name:    "code sample is not a reference",
			content: "```\n![report](sandbox:report.pptx)\n```",
			want:    nil,
		},
		{
			// Go regexp.Split does not interleave the matched fences, so
			// walking parts with i+=2 would skip the segment after the first
			// fence and miss a real citation that rewrite still rewrites.
			name:    "reference after a code fence",
			content: "![report](sandbox:report.pptx)\n\n```\n![ignored](sandbox:chart.html)\n```\n\n![chart](sandbox:chart.html)",
			want:    types.MessageArtifacts{artifacts[0], artifacts[1]},
		},
		{
			name:    "multiple references keep candidate order",
			content: "![data](data.csv)\n\n![report](sandbox:report.pptx)",
			want:    types.MessageArtifacts{artifacts[0], artifacts[2]},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, referencedArtifacts(tc.content, artifacts))
		})
	}
}

// A regenerated file shadows the older version of itself: the name resolves to
// the first candidate, while an explicit handle still reaches the old one.
func TestReferencedArtifactsPrefersTheFirstCandidate(t *testing.T) {
	old := types.MessageArtifact{FileName: "deck.pptx", URL: refFor(0)}
	fresh := types.MessageArtifact{FileName: "deck.pptx", URL: refFor(1)}
	candidates := types.MessageArtifacts{fresh, old}

	require.Equal(t, types.MessageArtifacts{fresh},
		referencedArtifacts("![deck](sandbox:deck.pptx)", candidates))
	require.Equal(t, types.MessageArtifacts{old},
		referencedArtifacts("![deck]("+old.URL+")", candidates))
}

// KnownArtifacts is oldest-first. Reverse it before merging so a hash-skipped
// name binds the latest file; this turn still shadows that latest file.
func TestMergeArtifactListsPrefersThisTurnThenLatestKnown(t *testing.T) {
	old := types.MessageArtifact{FileName: "deck.pptx", URL: refFor(0)}
	latest := types.MessageArtifact{FileName: "deck.pptx", URL: refFor(1)}
	fresh := types.MessageArtifact{FileName: "deck.pptx", URL: refFor(2)}
	known := types.MessageArtifacts{old, latest}

	require.Equal(t, types.MessageArtifacts{latest},
		referencedArtifacts("![deck](sandbox:deck.pptx)",
			mergeArtifactLists(nil, artifactsNewestFirst(known))))
	require.Equal(t, types.MessageArtifacts{fresh},
		referencedArtifacts("![deck](sandbox:deck.pptx)",
			mergeArtifactLists(types.MessageArtifacts{fresh}, artifactsNewestFirst(known))))
}

func TestArtifactsNewestFirstDoesNotMutateInput(t *testing.T) {
	in := artifactsFixture("a.html", "b.html")
	got := artifactsNewestFirst(in)
	require.Equal(t, "b.html", got[0].FileName)
	require.Equal(t, "a.html", got[1].FileName)
	require.Equal(t, "a.html", in[0].FileName)
}
