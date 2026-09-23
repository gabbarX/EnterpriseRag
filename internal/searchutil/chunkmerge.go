package searchutil

import (
	"sort"
	"strings"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

// JoinChunkContent joins two current chunk bodies without relying on parser
// offsets. Exact containment is collapsed, a real suffix/prefix overlap is
// removed, and otherwise both bodies are retained with separator between
// them. The conservative fallback intentionally prefers small duplication
// over silently dropping edited content.
func JoinChunkContent(acc, next, separator string) string {
	if acc == "" {
		return next
	}
	if next == "" {
		return acc
	}
	if ContainsChunkContent(acc, next) {
		return acc
	}
	if ContainsChunkContent(next, acc) {
		return next
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)
	maxOverlap := minInt(len(accRunes), len(nextRunes))
	// Editable chunks may be much larger than parser-produced chunks. Bound
	// suffix matching so an adversarial 200 KB edit cannot turn retrieval into
	// quadratic work. Parser overlap windows are normally far below this cap;
	// larger unmatched overlap is safely retained as duplication.
	if maxOverlap > defaultSearchSpan {
		maxOverlap = defaultSearchSpan
	}
	for overlap := maxOverlap; overlap >= minOverlapRunes; overlap-- {
		if runeSlicesEqual(accRunes[len(accRunes)-overlap:], nextRunes[:overlap]) {
			return acc + string(nextRunes[overlap:])
		}
	}
	return acc + separator + next
}

// ContainsChunkContent reports whether the complete current body is safely
// represented by another body. Very short substrings are not treated as
// containment because common words and punctuation would create false drops.
func ContainsChunkContent(container, contained string) bool {
	if container == "" || contained == "" {
		return false
	}
	if container == contained {
		return true
	}
	return len([]rune(contained)) >= minOverlapRunes && strings.Contains(container, contained)
}

func runeSlicesEqual(left, right []rune) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// This file implements the shared "overlap join" logic for chunk content, reused
// by document reconstruction (reconstructContent), knowledge-graph content
// merging (graph mergeChunkContents) and similar paths. The chat retrieval path
// allows users to edit chunks, so it uses JoinChunkContent above and avoids
// depending on source position coordinates.
//
// Historically every call site trimmed the overlap "by position" (offset =
// len(content) - (EndAt - lastEndAt) and similar), which assumes
// len([]rune(Content)) == EndAt-StartAt. Two kinds of data break that
// invariant and cause misaligned joins, dropped or duplicated characters:
//  1. the parent-child chunker re-writes the table header onto a split table;
//     the injected header is zero width (start == end), position coordinates
//     cannot express it, and content is longer than EndAt-StartAt;
//  2. content may retain HTML entities (such as &#34; / &gt;), whose character
//     count is longer than the source range.
//
// So the overlap is matched "by text" instead: search a window at the start of
// the next segment for the first occurrence of the merged text's suffix and
// join from just after that position. Position information (StartAt/EndAt) is
// only used to size the search window, never to trim.

const (
	// minOverlapRunes is the shortest suffix length eligible for matching. Very
	// short suffixes (a table separator row such as |---|) match by accident, so
	// they are ignored.
	minOverlapRunes = 12
	// defaultSearchSpan is the lower bound on the search window, so that real
	// overlaps within a certain range are still detected when position
	// information is missing or zero.
	defaultSearchSpan = 400
)

// AppendWithOverlap appends next to acc, removing the overlap between them.
//
// positionOverlap is the overlap estimated from StartAt/EndAt (lastEnd -
// curStart) and is only used to bound the search window; the real overlap is
// matched by text, which tolerates re-written table headers and HTML entity
// length drift. If no textual overlap is found the two are joined as-is (no
// trimming) -- better to keep a duplicate than to destroy content.
//
// When positionOverlap <= 0 the two segments are strictly adjacent or disjoint
// by position and there is nothing to de-duplicate. Running the text match here
// would, because of the headSlack floor of 320, wrongly hit a genuine repeat of
// acc's suffix inside the window at the start of next (the same sentence
// appearing several times in a document), mistake the whole start of next for a
// re-written table header and delete it, causing irreversible content loss. Join
// directly and leave re-written header duplicates to the caller to post-process.
func AppendWithOverlap(acc, next string, positionOverlap int) string {
	if acc == "" {
		return next
	}
	if next == "" {
		return acc
	}
	if positionOverlap <= 0 {
		return acc + next
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)

	span := positionOverlap

	maxK := minInt(len(accRunes), len(nextRunes))
	if cap := maxInt(span*3, defaultSearchSpan); maxK > cap {
		maxK = cap
	}
	// How much prefix may be skipped before the overlap (i.e. synthesised text
	// such as a re-written table header).
	headSlack := maxInt(span*2, 320)

	for k := maxK; k >= minOverlapRunes; k-- {
		needle := accRunes[len(accRunes)-k:]
		if pos := indexRunes(nextRunes, needle, headSlack); pos >= 0 {
			return acc + string(nextRunes[pos+k:])
		}
	}
	return acc + next
}

// AppendWithExactOverlap joins acc and next using the exact overlap given by
// the coordinates, for callers that have already confirmed the position
// coordinates are trustworthy: it checks that the last overlap characters of
// acc equal the first overlap characters of next character by character, trims
// exactly when they do, and joins directly when overlap is 0.
//
// The difference from AppendWithOverlap is that it does not guess: the latter,
// to tolerate re-written table headers and HTML entities, searches a window for
// the longest suffix match, and repetitive periodic text (tables, logs) can be
// mistaken for an overlap and trimmed away. When coordinates are trustworthy the
// overlap is known and no search is needed.
//
// A failed check returns ok=false, leaving the caller to decide whether to fall
// back to AppendWithOverlap.
func AppendWithExactOverlap(acc, next string, overlap int) (string, bool) {
	if acc == "" {
		return next, true
	}
	if next == "" {
		return acc, true
	}
	if overlap < 0 {
		return "", false
	}
	if overlap == 0 {
		return acc + next, true
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)
	if overlap > len(accRunes) || overlap > len(nextRunes) {
		return "", false
	}
	if !runeSlicesEqual(accRunes[len(accRunes)-overlap:], nextRunes[:overlap]) {
		return "", false
	}
	return acc + string(nextRunes[overlap:]), true
}

// MergeTextChunks sorts by StartAt (ties broken by ChunkIndex) and rebuilds the
// full text from several chunks using AppendWithOverlap. gapSep is the separator
// placed between two segments that are not adjacent by position (there is a
// gap), e.g. a newline; pass an empty string to join directly.
//
// The caller is responsible for filtering by type first (for example keeping
// only text chunks); this function is unaware of ChunkType.
func MergeTextChunks(chunks []*types.Chunk, gapSep string) string {
	if len(chunks) == 0 {
		return ""
	}

	sorted := make([]*types.Chunk, len(chunks))
	copy(sorted, chunks)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].StartAt == sorted[j].StartAt {
			return sorted[i].ChunkIndex < sorted[j].ChunkIndex
		}
		return sorted[i].StartAt < sorted[j].StartAt
	})

	merged := ""
	mergedEnd := -1
	for _, c := range sorted {
		if c == nil || c.Content == "" {
			continue
		}
		if merged == "" {
			merged = c.Content
			if c.EndAt > 0 {
				mergedEnd = c.EndAt
			}
			continue
		}

		// Gap, or position information missing (EndAt==0): join as an independent
		// paragraph without overlap trimming.
		if c.StartAt > mergedEnd || c.EndAt == 0 {
			if gapSep != "" {
				merged += gapSep
			}
			merged += c.Content
			if c.EndAt > 0 {
				mergedEnd = c.EndAt
			}
			continue
		}

		// Partial overlap or end-to-end: de-duplicate by text match, then join.
		if c.EndAt > mergedEnd {
			merged = AppendWithOverlap(merged, c.Content, mergedEnd-c.StartAt)
			mergedEnd = c.EndAt
		}
		// Otherwise it is fully covered by the previous segment; skip.
	}

	return merged
}

// indexRunes finds the rune index of the first occurrence of needle in
// haystack, with the start position no greater than maxStart. Returns -1 if not
// found.
func indexRunes(haystack, needle []rune, maxStart int) int {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return -1
	}
	limit := len(haystack) - len(needle)
	if maxStart < limit {
		limit = maxStart
	}
	for i := 0; i <= limit; i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
