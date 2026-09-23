package service

import (
	"strings"
	"testing"
)

func TestLinkifyContent_BasicDevanagari(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
	}
	got, changed := linkifyContent("मैं मुंबई शहर में रहता हूँ", refs, "")
	if !changed {
		t.Fatalf("expected change")
	}
	want := "मैं [[mumbai|मुंबई]] शहर में रहता हूँ"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLinkifyContent_LongerNameWinsOverSubstring(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
		{slug: "iitb", matchText: "मुंबई प्रौद्योगिकी संस्थान"},
	}
	got, changed := linkifyContent("मैं मुंबई प्रौद्योगिकी संस्थान में पढ़ता हूँ", refs, "")
	if !changed {
		t.Fatalf("expected change")
	}
	// The longer match should win; "मुंबई" must not swallow the prefix.
	if !strings.Contains(got, "[[iitb|मुंबई प्रौद्योगिकी संस्थान]]") {
		t.Fatalf("longer match not preferred: %q", got)
	}
	if strings.Contains(got, "[[mumbai|") {
		t.Fatalf("shorter substring should not have linked: %q", got)
	}
}

func TestLinkifyContent_SkipsSingleHanRuneAutoLink(t *testing.T) {
	refs := []linkRef{
		{slug: "entity/feng", matchText: "风"},
	}
	// isSingleHanRune is a deliberately Han-only guard in wiki_linkify.go, so
	// the surface form under test has to stay a Han character.
	in := "The manual mentions 风 as a loanword and nothing else."
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("single Han rune must not be auto-linked: %q", got)
	}
	if got != in {
		t.Fatalf("content changed: got %q, want %q", got, in)
	}
}

func TestLinkifyContent_PreservesExplicitSingleHanRuneLink(t *testing.T) {
	refs := []linkRef{
		{slug: "entity/feng", matchText: "风"},
	}
	in := "In the fable, [[entity/feng|风]] envies the eye."
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("explicit single-character link must be preserved without reinjection: %q", got)
	}
	if got != in {
		t.Fatalf("content changed: got %q, want %q", got, in)
	}
}

func TestLinkifyContent_ASCIIWordBoundary(t *testing.T) {
	refs := []linkRef{
		{slug: "ai", matchText: "AI"},
	}
	// Should NOT match "AI" inside "TRAINING" or "PAINT".
	in := "TRAINING and PAINT are words."
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("should not change, got %q", got)
	}

	// SHOULD match standalone "AI".
	in2 := "AI is cool."
	got2, changed2 := linkifyContent(in2, refs, "")
	if !changed2 {
		t.Fatalf("expected change, got %q", got2)
	}
	if !strings.HasPrefix(got2, "[[ai|AI]] ") {
		t.Fatalf("standalone AI should link: %q", got2)
	}
}

func TestLinkifyContent_SkipsExistingWikiLink(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
	}
	// Already linked — should not double-wrap.
	in := "मैं [[mumbai|मुंबई]] शहर में रहता हूँ"
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("should not change already-linked content: %q", got)
	}
}

func TestLinkifyContent_SkipsInsideFencedCode(t *testing.T) {
	refs := []linkRef{
		{slug: "go", matchText: "Go"},
	}
	in := "Prose mentions Go.\n\n```go\nfunc Go() {}\n```\nAfter code."
	got, _ := linkifyContent(in, refs, "")
	// The prose occurrence should link.
	if !strings.Contains(got, "Prose mentions [[go|Go]].") {
		t.Fatalf("prose occurrence not linked: %q", got)
	}
	// The occurrence inside the fenced block must be untouched. Since linkifyContent
	// only wraps the first eligible match, the code-block one is skipped anyway,
	// but double-check that no link was injected inside the code fence.
	if strings.Contains(got, "func [[go|Go]]") {
		t.Fatalf("should not link inside fenced code: %q", got)
	}
}

func TestLinkifyContent_SkipsInsideInlineCode(t *testing.T) {
	refs := []linkRef{
		{slug: "foo", matchText: "Foo"},
	}
	// Only occurrence is inside `Foo` — must NOT be linked.
	in := "Use the `Foo` type."
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("should not link inside inline code: %q", got)
	}

	// Second case: prose before the inline code — should link the prose one.
	in2 := "Foo is a type. Use `Foo` here."
	got2, changed2 := linkifyContent(in2, refs, "")
	if !changed2 {
		t.Fatalf("expected change, got %q", got2)
	}
	if !strings.HasPrefix(got2, "[[foo|Foo]] is") {
		t.Fatalf("prose occurrence not linked: %q", got2)
	}
	if strings.Contains(got2, "`[[foo|Foo]]`") {
		t.Fatalf("inline code occurrence should not be linked: %q", got2)
	}
}

func TestLinkifyContent_SkipsInsideMarkdownLink(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
	}
	// Occurrence inside [text](url) should be skipped.
	in := "देखें [मुंबई](https://example.com/mumbai)"
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("should not link inside markdown link text: %q", got)
	}
}

func TestLinkifyContent_OnlyFirstOccurrenceWrapped(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
	}
	in := "मुंबई बड़ी है। मुंबई में भीड़ है। मुंबई बंदरगाह है।"
	got, changed := linkifyContent(in, refs, "")
	if !changed {
		t.Fatalf("expected change")
	}
	// Only first match should be wrapped.
	count := strings.Count(got, "[[mumbai|मुंबई]]")
	if count != 1 {
		t.Fatalf("expected exactly 1 link, got %d: %q", count, got)
	}
	// Later occurrences remain bare.
	if strings.Count(got, "मुंबई") != 3 {
		t.Fatalf("expected 3 total मुंबई (incl. one inside the link), got content %q", got)
	}
}

func TestLinkifyContent_SkipsSelfSlug(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
	}
	// Rendering on the mumbai page itself — must not self-link.
	got, changed := linkifyContent("मुंबई बंदरगाह है", refs, "mumbai")
	if changed {
		t.Fatalf("should not self-link: %q", got)
	}
}

func TestLinkifyContent_SkipsWhenSlugAlreadyUsed(t *testing.T) {
	// Title matches by alias, but slug already appears via a different mention.
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
		{slug: "mumbai", matchText: "बंबई"},
	}
	in := "मैं [[mumbai|बंबई]] में था, बाद में मुंबई गया।"
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("should skip when slug already linked: %q", got)
	}
}

func TestLinkifyContent_MultipleRefsDisjoint(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
		{slug: "pune", matchText: "पुणे"},
	}
	in := "मैं आज मुंबई जाऊँगा, कल पुणे जाऊँगा।"
	got, changed := linkifyContent(in, refs, "")
	if !changed {
		t.Fatalf("expected change")
	}
	if !strings.Contains(got, "[[mumbai|मुंबई]]") {
		t.Fatalf("missing mumbai link: %q", got)
	}
	if !strings.Contains(got, "[[pune|पुणे]]") {
		t.Fatalf("missing pune link: %q", got)
	}
}

func TestLinkifyContent_EmptyOrNoMatch(t *testing.T) {
	if _, c := linkifyContent("", []linkRef{{slug: "x", matchText: "y"}}, ""); c {
		t.Fatalf("empty content must not change")
	}
	if _, c := linkifyContent("hello", nil, ""); c {
		t.Fatalf("nil refs must not change")
	}
	if _, c := linkifyContent("hello world", []linkRef{{slug: "x", matchText: "zzz"}}, ""); c {
		t.Fatalf("no match must not change")
	}
}

func TestFindFirstSafeMatch_BoundaryCases(t *testing.T) {
	// Case where the first occurrence is unsafe (in code), second is safe.
	s := "See `AI` in docs. AI rocks."
	forb, _ := computeForbiddenSpans(s)
	idx := findFirstSafeMatch(s, "AI", forb)
	// First AI is inside `...`, second is safe.
	want := strings.Index(s, "AI rocks")
	if idx != want {
		t.Fatalf("expected safe match at %d, got %d", want, idx)
	}
}

func TestComputeForbiddenSpans_FencedCode(t *testing.T) {
	s := "before\n```\nhidden\n```\nafter"
	spans, _ := computeForbiddenSpans(s)
	if len(spans) == 0 {
		t.Fatalf("expected at least one span")
	}
	// Ensure "hidden" is inside a forbidden span.
	h := strings.Index(s, "hidden")
	if !spanContains(spans, h, h+len("hidden")) {
		t.Fatalf("hidden should be forbidden, spans=%v", spans)
	}
	// "before" and "after" must NOT be forbidden.
	b := strings.Index(s, "before")
	if spanContains(spans, b, b+len("before")) {
		t.Fatalf("before should not be forbidden")
	}
	a := strings.Index(s, "after")
	if spanContains(spans, a, a+len("after")) {
		t.Fatalf("after should not be forbidden")
	}
}

func TestComputeForbiddenSpans_CollectsUsedSlugs(t *testing.T) {
	s := "Refer to [[mumbai|मुंबई]] and [[pune]] and [[nope|x]]."
	_, used := computeForbiddenSpans(s)
	for _, slug := range []string{"mumbai", "pune", "nope"} {
		if _, ok := used[slug]; !ok {
			t.Fatalf("expected slug %q in used set, got %v", slug, used)
		}
	}
}

func TestLinkifyContent_SkipsInsideReferenceStyleLink(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
	}
	// [text][label] form — occurrence should not be linkified.
	in := "See [मुंबई][metro] for details.\n\n[metro]: https://example.com/mumbai"
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("should not link inside reference-style link or definition: %q", got)
	}
}

func TestLinkifyContent_SkipsReferenceDefinitionLine(t *testing.T) {
	refs := []linkRef{
		{slug: "example", matchText: "example"},
	}
	// The `example` inside the definition URL must not be wrapped.
	in := "[cap]: https://example.com/x"
	got, changed := linkifyContent(in, refs, "")
	if changed {
		t.Fatalf("should not link inside reference definition: %q", got)
	}
}

func TestLinkifyContent_RepeatedLinkifyIsIdempotent(t *testing.T) {
	refs := []linkRef{
		{slug: "mumbai", matchText: "मुंबई"},
		{slug: "pune", matchText: "पुणे"},
	}
	in := "आज मुंबई, कल पुणे।"
	once, _ := linkifyContent(in, refs, "")
	twice, changed := linkifyContent(once, refs, "")
	if changed {
		t.Fatalf("second run should be a no-op, got %q", twice)
	}
	if once != twice {
		t.Fatalf("second run altered content:\nbefore: %q\nafter:  %q", once, twice)
	}
}
