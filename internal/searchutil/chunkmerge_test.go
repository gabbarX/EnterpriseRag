package searchutil

import (
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

func TestAppendWithOverlap_ContiguousNoTrim(t *testing.T) {
	// The two pieces meet end to end (no overlap), and next carries HTML entities so
	// its character count exceeds EndAt-StartAt. The old position-based formula cut
	// too much off the head; the whole segment must survive here.
	acc := "## दूसरा भाग\n\n"
	next := "| स्तंभ A | स्तंभ B |\n| मान1 | इकाई&#34;उद्धरण&#34;वाली सामग्री |\n"
	got := AppendWithOverlap(acc, next, 0)
	want := acc + next
	if got != want {
		t.Fatalf("contiguous merge mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestAppendWithOverlap_PrependedTableHeaderSkipped(t *testing.T) {
	// Simulates the chunker re-adding a header to a split table: next starts with an
	// extra copy of the header (zero width, invisible to positions) and the real
	// overlapping rows follow it. Trimming by position misaligns; matching by text
	// must deduplicate correctly.
	header := "| स्तंभ1 | स्तंभ2 | स्तंभ3 |\n|:---|:---|:---|\n"
	overlapRows := "| पंक्ति5 | सामग्री5A | सामग्री5B |\n| पंक्ति6 | सामग्री6A | सामग्री6B |\n"
	accTail := "| पंक्ति4 | सामग्री4A | सामग्री4B |\n" + overlapRows
	acc := header + "| पंक्ति1 | x | — |\n" + accTail

	newRows := "| पंक्ति7 | सामग्री7A | सामग्री7B |\n| पंक्ति8 | सामग्री8A | सामग्री8B |\n"
	next := header + overlapRows + newRows

	// The positional overlap is roughly two rows long; an approximation is enough,
	// since it only sizes the search window.
	got := AppendWithOverlap(acc, next, len([]rune(overlapRows)))
	want := acc + newRows
	if got != want {
		t.Fatalf("prepended-header merge mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestAppendWithOverlap_PlainOverlap(t *testing.T) {
	acc := "abcdefghijklmnopqrstuvwxyz0123"
	// next overlaps acc's tail "klmnopqrstuvwxyz0123" and then adds new content
	next := "klmnopqrstuvwxyz0123ABCDEFG"
	got := AppendWithOverlap(acc, next, 20)
	want := "abcdefghijklmnopqrstuvwxyz0123ABCDEFG"
	if got != want {
		t.Fatalf("plain overlap mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestAppendWithOverlap_NoOverlap(t *testing.T) {
	acc := "hello world"
	next := "completely different"
	got := AppendWithOverlap(acc, next, 0)
	want := acc + next
	if got != want {
		t.Fatalf("no overlap mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestAppendWithExactOverlap_TrimsKnownOverlap(t *testing.T) {
	overlap := "shared boundary text"
	got, ok := AppendWithExactOverlap("before "+overlap, overlap+" after", len([]rune(overlap)))
	if !ok {
		t.Fatal("exact overlap should be accepted")
	}
	want := "before " + overlap + " after"
	if got != want {
		t.Fatalf("exact overlap mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestAppendWithExactOverlap_ZeroOverlapConcatenatesRepeatedText(t *testing.T) {
	// Both abutting segments are made of the same repeating text (table rows, log
	// lines). With an overlap of 0 they must be concatenated verbatim rather than
	// suffix-matched, or the repeated rows at the head of next are eaten.
	row := "| cell | cell |\n"
	acc := "प्रस्तावना\n" + row + row
	next := row + row + row + "समापन\n"
	got, ok := AppendWithExactOverlap(acc, next, 0)
	if !ok {
		t.Fatal("zero overlap should be accepted")
	}
	if got != acc+next {
		t.Fatalf("zero overlap must concatenate verbatim:\n got=%q\nwant=%q", got, acc+next)
	}
}

func TestAppendWithExactOverlap_RejectsMismatchedOverlap(t *testing.T) {
	// The length invariant still holds but the text no longer matches (HTML entities,
	// a re-added header). This must be rejected so the caller falls back to matching
	// by text.
	if _, ok := AppendWithExactOverlap("abcdefghijkl", "XYZdefghijkl", 6); ok {
		t.Fatal("mismatched overlap should be rejected")
	}
}

func TestAppendWithExactOverlap_RejectsOverlapLongerThanBodies(t *testing.T) {
	if _, ok := AppendWithExactOverlap("short", "shorter", 99); ok {
		t.Fatal("overlap exceeding body length should be rejected")
	}
	if _, ok := AppendWithExactOverlap("short", "shorter", -1); ok {
		t.Fatal("negative overlap should be rejected")
	}
}

func TestJoinChunkContentUsesCurrentTextInsteadOfSourceOffsets(t *testing.T) {
	first := "first edited body with no original overlap"
	second := "second independently edited body"
	got := JoinChunkContent(first, second, "\n\n")
	want := first + "\n\n" + second
	if got != want {
		t.Fatalf("edited join mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestJoinChunkContentRemovesRealBoundaryOverlap(t *testing.T) {
	overlap := "shared boundary text"
	got := JoinChunkContent("before "+overlap, overlap+" after", "\n\n")
	want := "before " + overlap + " after"
	if got != want {
		t.Fatalf("overlap join mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestJoinChunkContentCollapsesExactContainment(t *testing.T) {
	outer := "prefix complete current body suffix"
	if got := JoinChunkContent(outer, "complete current body", "\n\n"); got != outer {
		t.Fatalf("contained body should be collapsed: %q", got)
	}
}

func TestMergeTextChunks_OrdersFiltersAndStitches(t *testing.T) {
	header := "| a | b |\n|:--|:--|\n"
	chunks := []*types.Chunk{
		{
			Content: header + "| r1 | x |\n| r2 | y |\n", ChunkType: types.ChunkTypeText,
			StartAt: 0, EndAt: 20, ChunkIndex: 0,
		},
		{
			// re-added header + overlap with the previous segment's r2 + the new row r3
			Content: header + "| r2 | y |\n| r3 | z |\n", ChunkType: types.ChunkTypeText,
			StartAt: 10, EndAt: 40, ChunkIndex: 1,
		},
	}
	got := MergeTextChunks(chunks, "\n")
	want := header + "| r1 | x |\n| r2 | y |\n" + "| r3 | z |\n"
	if got != want {
		t.Fatalf("MergeTextChunks mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestMergeTextChunks_GapSeparator(t *testing.T) {
	chunks := []*types.Chunk{
		{Content: "first", ChunkType: types.ChunkTypeText, StartAt: 0, EndAt: 5, ChunkIndex: 0},
		{Content: "second", ChunkType: types.ChunkTypeText, StartAt: 100, EndAt: 106, ChunkIndex: 1},
	}
	got := MergeTextChunks(chunks, "\n")
	want := "first\nsecond"
	if got != want {
		t.Fatalf("gap separator mismatch:\n got=%q\nwant=%q", got, want)
	}
}

// TestAppendWithOverlap_ContiguousRealContentRepeat is a regression test:
// two chunks are strictly contiguous (positionOverlap==0), but a boilerplate
// sentence at the tail of acc reappears inside the head window of next as
// real content (the same sentence is written multiple times in the document).
// The old algorithm would enter text matching and, due to the headSlack
// floor of 320, mistake the head of next for a prepended table header and
// delete it, causing irreversible content loss. After the fix it should
// concatenate directly without trimming.
func TestAppendWithOverlap_ContiguousRealContentRepeat(t *testing.T) {
	repeat := "The system shall maintain a complete audit trail of all transactions."
	acc := "3.2 Logging Requirements\n\n" + repeat
	next := "\n\n5.1 Security Controls\n\n* Role-based access\n* Encryption at rest\n\n5.2 Compliance\n\n" + repeat + " This satisfies SOC 2."
	got := AppendWithOverlap(acc, next, 0)
	want := acc + next
	if got != want {
		t.Fatalf("contiguous real-content repeat must not trim:\n got.len=%d\nwant.len=%d\n got.tail=%q\nwant.tail=%q",
			len(got), len(want), got[len(acc)-50:], want[len(acc)-50:])
	}
}
