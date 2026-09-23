package chunker

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitText_BasicASCII(t *testing.T) {
	text := "Hello world. This is a test."
	cfg := SplitterConfig{ChunkSize: 100, ChunkOverlap: 0, Separators: []string{". "}}
	chunks := SplitText(text, cfg)
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	combined := ""
	for _, c := range chunks {
		combined += c.Content
	}
	if combined != text {
		t.Errorf("combined content mismatch:\n  got:  %q\n  want: %q", combined, text)
	}
}

func TestSplitText_DevanagariText_StartEndAreRuneOffsets(t *testing.T) {
	// Each Devanagari code point is 3 bytes in UTF-8 but 1 rune.
	// This test ensures Start/End are rune offsets, not byte offsets.
	text := "नमस्ते दुनिया यह एक परीक्षण पाठ है"
	runeCount := utf8.RuneCountInString(text)
	byteCount := len(text)
	if runeCount == byteCount {
		t.Fatal("test requires multi-byte characters")
	}

	cfg := SplitterConfig{ChunkSize: 100, ChunkOverlap: 0, Separators: []string{"\n"}}
	chunks := SplitText(text, cfg)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	c := chunks[0]
	if c.Start != 0 {
		t.Errorf("Start: got %d, want 0", c.Start)
	}
	if c.End != runeCount {
		t.Errorf("End: got %d, want %d (runeCount); byteCount would be %d",
			c.End, runeCount, byteCount)
	}
}

func TestSplitText_DevanagariMultiChunk_StartEndConsistency(t *testing.T) {
	// Build a long Devanagari text that will be split into multiple chunks.
	line := "यह एक परीक्षण वाक्य है जो विभाजन की जाँच करता है।"
	text := strings.Repeat(line+"\n\n", 20)
	text = strings.TrimRight(text, "\n")

	cfg := SplitterConfig{ChunkSize: 30, ChunkOverlap: 5, Separators: []string{"\n\n", "\n", "।"}}
	chunks := SplitText(text, cfg)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	textRunes := []rune(text)
	for i, c := range chunks {
		contentRunes := []rune(c.Content)
		contentRuneLen := len(contentRunes)

		// End - Start must equal the rune length of the content
		spanLen := c.End - c.Start
		if spanLen != contentRuneLen {
			t.Errorf("chunk[%d]: End(%d) - Start(%d) = %d, but rune len of content = %d",
				i, c.End, c.Start, spanLen, contentRuneLen)
		}

		// Start must be non-negative and End must not exceed total rune count
		if c.Start < 0 {
			t.Errorf("chunk[%d]: Start is negative: %d", i, c.Start)
		}
		if c.End > len(textRunes) {
			t.Errorf("chunk[%d]: End %d exceeds total rune count %d", i, c.End, len(textRunes))
		}

		// Content from rune slice must match the chunk content
		if c.Start >= 0 && c.End <= len(textRunes) {
			sliced := string(textRunes[c.Start:c.End])
			if sliced != c.Content {
				t.Errorf("chunk[%d]: content mismatch via rune slice:\n  got:  %q\n  want: %q",
					i, sliced, c.Content)
			}
		}
	}
}

func TestSplitText_MixedDevanagariAndASCII(t *testing.T) {
	text := "Helloनमस्तेWorldदुनियाTestपरीक्षण"
	cfg := SplitterConfig{ChunkSize: 100, ChunkOverlap: 0, Separators: []string{"\n"}}
	chunks := SplitText(text, cfg)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	c := chunks[0]
	expectedRuneLen := utf8.RuneCountInString(text)
	if c.End-c.Start != expectedRuneLen {
		t.Errorf("End(%d) - Start(%d) = %d, want rune len %d (byte len would be %d)",
			c.End, c.Start, c.End-c.Start, expectedRuneLen, len(text))
	}
}

func TestSplitText_ProtectedPattern_DevanagariContext(t *testing.T) {
	// Test protected markdown images in a Devanagari context.
	text := "यह पहले का पाठ है।![चित्र विवरण](http://example.com/img.png)यह बाद का पाठ है।"
	cfg := SplitterConfig{ChunkSize: 200, ChunkOverlap: 0, Separators: []string{"।"}}
	chunks := SplitText(text, cfg)

	textRunes := []rune(text)
	for i, c := range chunks {
		if c.Start < 0 || c.End > len(textRunes) {
			t.Errorf("chunk[%d]: out of rune range [%d, %d), total runes %d",
				i, c.Start, c.End, len(textRunes))
			continue
		}
		sliced := string(textRunes[c.Start:c.End])
		if sliced != c.Content {
			t.Errorf("chunk[%d]: rune-slice mismatch:\n  sliced: %q\n  content: %q",
				i, sliced, c.Content)
		}
	}
}

func TestSplitText_SimulateMergeSlicing(t *testing.T) {
	// Simulate what merge.go:104-106 does to ensure it won't panic.
	// This is the exact pattern that caused the production crash.
	line := "यह पहला अनुच्छेद है जो ज्ञान आधार का अनुकरण करता है"
	text := line + "\n\n" + line + "\n\n" + line

	cfg := SplitterConfig{ChunkSize: 25, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)
	if len(chunks) < 2 {
		t.Fatalf("need at least 2 chunks for overlap test, got %d", len(chunks))
	}

	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1]
		curr := chunks[i]

		if curr.Start > prev.End {
			continue // non-overlapping, no merge needed
		}

		// This is the exact merge.go logic:
		contentRunes := []rune(curr.Content)
		offset := len(contentRunes) - (curr.End - prev.End)

		if offset < 0 {
			t.Fatalf("chunk[%d] merge panic: offset=%d < 0 (contentRunes=%d, curr.End=%d, prev.End=%d)",
				i, offset, len(contentRunes), curr.End, prev.End)
		}
		if offset > len(contentRunes) {
			t.Fatalf("chunk[%d] merge panic: offset=%d > len(contentRunes)=%d",
				i, offset, len(contentRunes))
		}

		_ = string(contentRunes[offset:])
	}
}

// TestSplitText_RecursiveSeparators_NoOversizeChunks exposes the regression
// where after picking the first separator that yields >1 piece, sub-pieces
// that are still larger than ChunkSize were not split further with the next
// separator. Real-world docs with one paragraph break followed by a long
// run of newline-separated lines must still be honored.
func TestSplitText_RecursiveSeparators_NoOversizeChunks(t *testing.T) {
	// One paragraph break, then 50 short newline-separated lines forming
	// ~1500 chars in the second paragraph.
	body := strings.Repeat("This is one fairly short line of text.\n", 50)
	text := "lead paragraph that is short.\n\n" + body
	cfg := SplitterConfig{
		ChunkSize:    300,
		ChunkOverlap: 30,
		Separators:   []string{"\n\n", "\n", ". "},
	}
	chunks := SplitText(text, cfg)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	// No chunk should exceed roughly 1.5x ChunkSize — recursive splitting
	// at the next-priority separator should keep this bounded.
	maxAllowed := cfg.ChunkSize * 3 / 2
	for i, c := range chunks {
		l := len([]rune(c.Content))
		if l > maxAllowed {
			t.Errorf("chunk %d is %d runes, > 1.5x ChunkSize (%d) — recursive split missing", i, l, maxAllowed)
		}
	}
}

func TestSplitText_Empty(t *testing.T) {
	chunks := SplitText("", DefaultConfig())
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestSplitText_SingleMultibyteRune(t *testing.T) {
	text := "क"
	cfg := SplitterConfig{ChunkSize: 10, ChunkOverlap: 0, Separators: []string{"\n"}}
	chunks := SplitText(text, cfg)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Start != 0 || chunks[0].End != 1 {
		t.Errorf("expected [0,1), got [%d,%d)", chunks[0].Start, chunks[0].End)
	}
}

func TestSplitText_LaTeXBlockInDevanagari(t *testing.T) {
	text := "पहले का पाठ$$E=mc^2$$बाद का पाठ"
	cfg := SplitterConfig{ChunkSize: 200, ChunkOverlap: 0, Separators: []string{"\n"}}
	chunks := SplitText(text, cfg)

	textRunes := []rune(text)
	for i, c := range chunks {
		spanLen := c.End - c.Start
		contentRuneLen := utf8.RuneCountInString(c.Content)
		if spanLen != contentRuneLen {
			t.Errorf("chunk[%d]: span %d != rune len %d", i, spanLen, contentRuneLen)
		}
		if c.End > len(textRunes) {
			t.Errorf("chunk[%d]: End %d > total runes %d", i, c.End, len(textRunes))
		}
	}
}

func TestSplitText_CodeBlockInDevanagari(t *testing.T) {
	text := "देवनागरी विवरण\n```python\nprint('hello')\n```\nजारी देवनागरी"
	cfg := SplitterConfig{ChunkSize: 200, ChunkOverlap: 0, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	textRunes := []rune(text)
	for i, c := range chunks {
		if c.Start < 0 || c.End > len(textRunes) {
			t.Errorf("chunk[%d]: out of range [%d,%d), total %d", i, c.Start, c.End, len(textRunes))
			continue
		}
		sliced := string(textRunes[c.Start:c.End])
		if sliced != c.Content {
			t.Errorf("chunk[%d]: rune-slice mismatch:\n  sliced: %q\n  content: %q",
				i, sliced, c.Content)
		}
	}
}

func TestSplitText_OverlapChunks_NonNegativeStart(t *testing.T) {
	// When overlap is used, start of the next chunk could go before 0 if broken.
	text := strings.Repeat("परीक्षण सामग्री, ", 50)
	cfg := SplitterConfig{ChunkSize: 20, ChunkOverlap: 5, Separators: []string{", "}}
	chunks := SplitText(text, cfg)

	for i, c := range chunks {
		if c.Start < 0 {
			t.Errorf("chunk[%d]: negative Start %d", i, c.Start)
		}
		if c.End < c.Start {
			t.Errorf("chunk[%d]: End %d < Start %d", i, c.End, c.Start)
		}
	}
}

func TestFindSemanticOverlapBoundary_PriorityThenEarliest(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "paragraph outranks earlier sentence and line",
			text: "पहला वाक्य।दूसरा वाक्य\nसामान्य पंक्ति\n\nअंतिम अनुच्छेद",
			want: "अंतिम अनुच्छेद",
		},
		{
			name: "earliest sentence wins within same priority",
			text: "पहला।दूसरा।तीसरा",
			want: "दूसरा।तीसरा",
		},
		{
			name: "windows paragraph break",
			text: "पहला अनुच्छेद।\r\n\r\nदूसरा अनुच्छेद",
			want: "दूसरा अनुच्छेद",
		},
		{
			name: "line outranks earlier sentence",
			text: "पहला वाक्य।पहली पंक्ति\nदूसरी पंक्ति",
			want: "दूसरी पंक्ति",
		},
		{
			name: "windows line break",
			text: "पहला वाक्य।पहली पंक्ति\r\nदूसरी पंक्ति",
			want: "दूसरी पंक्ति",
		},
		{
			name: "english period requires following space",
			text: "First sentence. Second sentence",
			want: "Second sentence",
		},
		{
			name: "english question mark requires following space",
			text: "Question? Answer",
			want: "Answer",
		},
		{
			name: "english exclamation mark requires following space",
			text: "Warning! Continue",
			want: "Continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			end, ok := findSemanticOverlapBoundary(tt.text)
			if !ok {
				t.Fatal("expected semantic overlap boundary")
			}
			got := string([]rune(tt.text)[end:])
			if got != tt.want {
				t.Fatalf("overlap tail = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindSemanticOverlapBoundary_NoConfiguredSemanticSeparator(t *testing.T) {
	for _, text := range []string{
		"निरंतर पाठ जिसमें कोई अर्थपूर्ण विभाजक नहीं",
		"अल्पविराम, अर्धविराम; और कोलन: केवल",
		"version1.2 remains one unit",
		"address 192.168.1.1 remains one unit",
		"see https://ex.com?q=1&foo=bar",
	} {
		if end, ok := findSemanticOverlapBoundary(text); ok {
			t.Errorf("findSemanticOverlapBoundary(%q) returned boundary at %d", text, end)
		}
	}
}

func TestFindSemanticOverlapBoundary_IgnoresProtectedContent(t *testing.T) {
	for _, text := range []string{
		"कोड `fmt.Println(\"hello. world\")` शेष सामग्री",
		"कोड ब्लॉक ```go\nfmt.Println(\"hello. world\")\n``` शेष सामग्री",
		"सूत्र $$x. y$$ शेष सामग्री",
	} {
		if end, ok := findSemanticOverlapBoundary(text); ok {
			t.Errorf("findSemanticOverlapBoundary(%q) returned protected boundary at %d", text, end)
		}
	}
}

func TestFindSemanticOverlapBoundary_FiltersEligibilityBeforePriorityAndPosition(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		minEnd int
		want   string
	}{
		{
			name:   "earlier ineligible sentence does not hide eligible sentence",
			text:   "a।bc।XYZ",
			minEnd: 4,
			want:   "XYZ",
		},
		{
			name:   "ineligible paragraph does not outrank eligible sentence",
			text:   "\n\nx? tail",
			minEnd: 3,
			want:   "tail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			end, ok := findSemanticOverlapBoundaryEndingAtOrAfter(tt.text, tt.minEnd)
			if !ok {
				t.Fatal("expected eligible semantic overlap boundary")
			}
			if got := string([]rune(tt.text)[end:]); got != tt.want {
				t.Fatalf("overlap tail = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComputeOverlap_FindsBoundaryInsideLargeUnit(t *testing.T) {
	text := "abcdefgh।पूंछ"
	unit := splitUnit{text: text, start: 100, end: 100 + len([]rune(text))}

	overlap, overlapLen := computeOverlap([]splitUnit{unit}, 8, 32, 8)
	if got := unitsText(overlap); got != "पूंछ" {
		t.Fatalf("overlap = %q, want %q", got, "पूंछ")
	}
	if overlapLen != len([]rune("पूंछ")) {
		t.Fatalf("overlapLen = %d, want %d", overlapLen, len([]rune("पूंछ")))
	}
	if len(overlap) != 1 || overlap[0].start != 109 || overlap[0].end != 113 {
		t.Fatalf("overlap source span = %+v, want [109,113)", overlap)
	}
}

func TestComputeOverlap_NoBoundaryMeansNoOverlap(t *testing.T) {
	text := "abcdefghपूंछ"
	unit := splitUnit{text: text, start: 0, end: len([]rune(text))}

	overlap, overlapLen := computeOverlap([]splitUnit{unit}, 8, 32, 8)
	if len(overlap) != 0 || overlapLen != 0 {
		t.Fatalf("expected no overlap, got %q (%d)", unitsText(overlap), overlapLen)
	}
}

func TestComputeOverlap_RespectsNextChunkCapacity(t *testing.T) {
	text := "abcdefgh।है"
	unit := splitUnit{text: text, start: 0, end: len([]rune(text))}

	overlap, overlapLen := computeOverlap([]splitUnit{unit}, 10, 8, 5)
	if got := unitsText(overlap); got != "है" {
		t.Fatalf("overlap = %q, want %q", got, "है")
	}
	if overlapLen+5 > 8 {
		t.Fatalf("overlap plus next content exceeds chunk size: %d + %d > %d", overlapLen, 5, 8)
	}
}

func TestComputeOverlap_LookbehindBoundaryEligibility(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "single rune separator ending at minus one is eligible",
			text: "abcd।WXYZ",
			want: "WXYZ",
		},
		{
			name: "longest separator ending at minus one is eligible",
			text: "\r\n\r\nWXYZ",
			want: "WXYZ",
		},
		{
			name: "separator ending before minus one is ineligible",
			text: "a।bcWXYZ",
			want: "",
		},
		{
			name: "separator crossing original window start is eligible",
			text: "abc. WXY",
			want: "WXY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unit := splitUnit{text: tt.text, start: 0, end: len([]rune(tt.text))}
			overlap, overlapLen := computeOverlap([]splitUnit{unit}, 4, 20, 4)
			if got := unitsText(overlap); got != tt.want {
				t.Fatalf("overlap = %q, want %q", got, tt.want)
			}
			if overlapLen != len([]rune(tt.want)) {
				t.Fatalf("overlapLen = %d, want %d", overlapLen, len([]rune(tt.want)))
			}
			if overlapLen > 4 {
				t.Fatalf("overlap exceeds configured limit: %d > 4", overlapLen)
			}
		})
	}
}

func TestSplitText_OverlapDoesNotBreakURLQuery(t *testing.T) {
	line1 := strings.Repeat("a", 35)
	text := line1 + "\n" + "https://ex.com?q=1 tail\n" + "more content here"
	cfg := SplitterConfig{ChunkSize: 40, ChunkOverlap: 10, Separators: []string{"\n\n", "\n"}}

	chunks := SplitText(text, cfg)
	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d: %#v", len(chunks), chunks)
	}
	if strings.HasPrefix(chunks[2].Content, "q=1") {
		t.Fatalf("overlap broke at URL query string: chunk[2]=%q", chunks[2].Content)
	}
}

func TestBuildUnitsWithProtection_InlineCodeNotSplit(t *testing.T) {
	text := "intro `code.here` suffix"
	units := buildUnitsWithProtection(text, protectedSpans(text), []string{"\n"}, 0)

	for _, u := range units {
		if strings.Contains(u.text, "code.here") && !strings.Contains(u.text, "`code.here`") {
			t.Fatalf("inline code period split unit: %q", u.text)
		}
	}
}

func TestSplitText_SemanticOverlapInsideParagraph(t *testing.T) {
	first := strings.Repeat("क", 15) + "।है"
	text := first + "\n\n" + "अगला"
	cfg := SplitterConfig{
		ChunkSize:    20,
		ChunkOverlap: 10,
		Separators:   []string{"\n\n", "\n", "।"},
	}

	chunks := SplitText(text, cfg)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %#v", len(chunks), chunks)
	}
	if !strings.HasPrefix(chunks[1].Content, "है\n\n") {
		t.Fatalf("second chunk should start at the sentence boundary inside the large unit, got %q", chunks[1].Content)
	}
	if got := chunks[1].End - chunks[1].Start; got != len([]rune(chunks[1].Content)) {
		t.Fatalf("position invariant broken: span=%d content=%d", got, len([]rune(chunks[1].Content)))
	}
}

func TestBuildUnitsWithProtection_RuneOffsets(t *testing.T) {
	text := "नमस्ते"
	units := buildUnitsWithProtection(text, nil, []string{"\n"}, 0)

	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}

	u := units[0]
	expectedRuneLen := 6 // 6 Devanagari code points, incl. the virama and matra
	byteLen := len(text) // 18 bytes

	if u.start != 0 {
		t.Errorf("start: got %d, want 0", u.start)
	}
	if u.end != expectedRuneLen {
		t.Errorf("end: got %d, want %d (rune len); byte len is %d", u.end, expectedRuneLen, byteLen)
	}
}

func TestBuildUnitsWithProtection_WithProtectedSpan(t *testing.T) {
	text := "आगे![alt](url)पीछे"
	protected := protectedSpans(text)
	units := buildUnitsWithProtection(text, protected, []string{"\n"}, 0)

	textRunes := []rune(text)
	for i, u := range units {
		contentRuneLen := utf8.RuneCountInString(u.text)
		spanLen := u.end - u.start
		if spanLen != contentRuneLen {
			t.Errorf("unit[%d] %q: span %d != rune len %d (byte len %d)",
				i, u.text, spanLen, contentRuneLen, len(u.text))
		}
		if u.start < 0 || u.end > len(textRunes) {
			t.Errorf("unit[%d]: out of range [%d,%d), total runes %d",
				i, u.start, u.end, len(textRunes))
		}
	}
}

func TestSplitBySeparators(t *testing.T) {
	tests := []struct {
		text       string
		separators []string
		wantParts  int
	}{
		{"a\n\nb\n\nc", []string{"\n\n"}, 5},
		{"abc", []string{"\n"}, 1},
		{"a\nb\nc", []string{"\n"}, 5},
		{"", []string{"\n"}, 1},
	}

	for _, tt := range tests {
		parts := splitBySeparators(tt.text, tt.separators, 0)
		if len(parts) != tt.wantParts {
			t.Errorf("splitBySeparators(%q, %v): got %d parts %v, want %d",
				tt.text, tt.separators, len(parts), parts, tt.wantParts)
		}
	}
}

func TestExtractImageRefs(t *testing.T) {
	text := "hello ![alt1](url1) world ![alt2](url2) end"
	refs := ExtractImageRefs(text)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0].OriginalRef != "url1" || refs[0].AltText != "alt1" {
		t.Errorf("ref[0] mismatch: %+v", refs[0])
	}
	if refs[1].OriginalRef != "url2" || refs[1].AltText != "alt2" {
		t.Errorf("ref[1] mismatch: %+v", refs[1])
	}
}

func TestSplitText_LargeDevanagariDocument(t *testing.T) {
	// Simulate a real document with paragraphs of Devanagari text.
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString(fmt.Sprintf("अनुच्छेद %d: यह परीक्षण के लिए देवनागरी सामग्री है।", i))
		sb.WriteString("\n\n")
	}
	text := sb.String()

	cfg := SplitterConfig{ChunkSize: 50, ChunkOverlap: 10, Separators: []string{"\n\n", "\n", "।"}}
	chunks := SplitText(text, cfg)

	textRunes := []rune(text)
	for i, c := range chunks {
		contentRuneLen := utf8.RuneCountInString(c.Content)
		spanLen := c.End - c.Start
		if spanLen != contentRuneLen {
			t.Errorf("chunk[%d]: End(%d)-Start(%d)=%d != runeLen(%d)",
				i, c.End, c.Start, spanLen, contentRuneLen)
		}
		if c.Start < 0 {
			t.Errorf("chunk[%d]: negative Start %d", i, c.Start)
		}
		if c.End > len(textRunes) {
			t.Errorf("chunk[%d]: End %d > total runes %d", i, c.End, len(textRunes))
		}
		if c.Start >= 0 && c.End <= len(textRunes) {
			sliced := string(textRunes[c.Start:c.End])
			if sliced != c.Content {
				t.Errorf("chunk[%d]: content mismatch via rune-slice", i)
			}
		}
	}

	// Simulate merge.go logic on all overlapping chunk pairs
	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1]
		curr := chunks[i]
		if curr.Start > prev.End {
			continue
		}
		contentRunes := []rune(curr.Content)
		offset := len(contentRunes) - (curr.End - prev.End)
		if offset < 0 || offset > len(contentRunes) {
			t.Fatalf("chunk[%d] merge would panic: offset=%d, contentRunes=%d, curr.End=%d, prev.End=%d",
				i, offset, len(contentRunes), curr.End, prev.End)
		}
	}
}

// ---------------------------------------------------------------------------
// Table header prepending tests
// ---------------------------------------------------------------------------

func TestSplitText_TableHeaderPrependedToChunks(t *testing.T) {
	// A markdown table large enough to span multiple chunks.
	// Each chunk after the first should have the header row + separator prepended.
	text := "" +
		"Text before the table.\n\n" +
		"| Name | Age | City |\n" +
		"| --- | --- | --- |\n" +
		"| Asha | 25 | Mumbai |\n" +
		"| Vijay | 30 | Delhi |\n" +
		"| Meera | 28 | Chennai |\n" +
		"| Rahul | 35 | Kolkata |\n" +
		"| Sunita | 22 | Pune |\n" +
		"| Arjun | 40 | Jaipur |\n" +
		"\nText after the table."

	tableHeader := "| Name | Age | City |\n| --- | --- | --- |\n"

	cfg := SplitterConfig{ChunkSize: 70, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	if len(chunks) < 3 {
		t.Fatalf("expected at least 3 chunks, got %d", len(chunks))
	}

	// Find chunks that contain table row data but not the original header position.
	// These should have the header prepended.
	headerPrependCount := 0
	for _, c := range chunks {
		if strings.Contains(c.Content, "| Vijay") || strings.Contains(c.Content, "| Meera") ||
			strings.Contains(c.Content, "| Rahul") || strings.Contains(c.Content, "| Sunita") ||
			strings.Contains(c.Content, "| Arjun") {
			if !strings.Contains(c.Content, "| Asha") {
				// This is a chunk with table rows but not the first row;
				// it should have the header prepended.
				if !strings.HasPrefix(c.Content, tableHeader) {
					t.Errorf("chunk (seq=%d) has table rows but is missing prepended header:\n%s",
						c.Seq, c.Content)
				} else {
					headerPrependCount++
				}
			}
		}
	}

	if headerPrependCount == 0 {
		t.Error("expected at least one chunk to have prepended table header, found none")
		for i, c := range chunks {
			t.Logf("chunk[%d] (seq=%d, start=%d, end=%d):\n%s", i, c.Seq, c.Start, c.End, c.Content)
		}
	}
}

func TestSplitText_NoHeaderForNonTableContent(t *testing.T) {
	// Ensure header prepending doesn't affect non-table content.
	text := strings.Repeat("यह एक सामान्य अनुच्छेद है, इसमें कोई तालिका नहीं।\n\n", 10)

	cfg := SplitterConfig{ChunkSize: 30, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	textRunes := []rune(text)
	for i, c := range chunks {
		contentRuneLen := utf8.RuneCountInString(c.Content)
		spanLen := c.End - c.Start
		if spanLen != contentRuneLen {
			t.Errorf("chunk[%d]: span %d != rune len %d (no table, should be exact)", i, spanLen, contentRuneLen)
		}
		if c.End > len(textRunes) {
			t.Errorf("chunk[%d]: End %d exceeds total runes %d", i, c.End, len(textRunes))
		}
	}
}

func TestSplitText_TableHeaderEndedByEmptyLine(t *testing.T) {
	// After the table ends (empty line), subsequent chunks should NOT have the header.
	text := "" +
		"| A | B |\n" +
		"| --- | --- |\n" +
		"| 1 | 2 |\n" +
		"| 3 | 4 |\n" +
		"\n" +
		"यह तालिका के बाद का पाठ है।\n" +
		"भरने के लिए और सामान्य पाठ।"

	cfg := SplitterConfig{ChunkSize: 40, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	for _, c := range chunks {
		hasTableRow := strings.Contains(c.Content, "| A | B |") || strings.Contains(c.Content, "|")
		hasPlainText := strings.Contains(c.Content, "यह तालिका के बाद") || strings.Contains(c.Content, "भरने के लिए")
		if hasPlainText && !hasTableRow {
			// This chunk is purely post-table text; should NOT have table header
			if strings.Contains(c.Content, "| --- |") {
				t.Errorf("post-table chunk should not contain table header:\n%s", c.Content)
			}
		}
	}
}

func TestHeaderTracker_BasicLifecycle(t *testing.T) {
	ht := newHeaderTracker()

	// Before table: no headers
	ht.update("Some regular text")
	if h := ht.getHeaders(); h != "" {
		t.Errorf("expected no headers before table, got %q", h)
	}

	// Table header unit
	ht.update("| A | B |\n| --- | --- |\n")
	if h := ht.getHeaders(); h == "" {
		t.Error("expected active header after table header unit")
	}

	// Table row: header should stay active
	ht.update("| 1 | 2 |\n")
	if h := ht.getHeaders(); h == "" {
		t.Error("header should remain active during table rows")
	}

	// Empty line: header should end
	ht.update("\n")
	if h := ht.getHeaders(); h != "" {
		t.Errorf("header should be cleared after empty line, got %q", h)
	}

	// New table can be tracked after the old one ended
	ht.update("| X | Y |\n| --- | --- |\n")
	if h := ht.getHeaders(); h == "" {
		t.Error("expected new header to be tracked after previous table ended")
	}
}

func TestHeaderTracker_EmptyHeaderRowRewrite(t *testing.T) {
	// Some converters (e.g., MarkItDown) produce tables with empty header rows:
	//   ||
	//   | --- | --- |
	//   | real col A | real col B |
	// The tracker should rewrite the header to be a proper Markdown table header:
	//   | real col A | real col B |
	//   | --- | --- |
	ht := newHeaderTracker()

	// Empty header row + separator
	ht.update("||\n| --- | --- | --- |\n")
	h := ht.getHeaders()
	if h == "" {
		t.Fatal("expected active header after empty header unit")
	}
	t.Logf("after empty header unit (pending): %q", h)

	// First data row → becomes the real column names
	ht.update("| Test Case ID | Module | Notes |\n")
	h = ht.getHeaders()
	t.Logf("after rewrite: %q", h)

	if !strings.Contains(h, "Test Case ID") {
		t.Errorf("rewritten header should contain column names, got:\n%s", h)
	}
	if strings.Contains(h, "||") {
		t.Errorf("rewritten header should NOT contain empty '||' row, got:\n%s", h)
	}
	if !strings.Contains(h, "---") {
		t.Errorf("rewritten header should contain separator, got:\n%s", h)
	}
	// Column names should come BEFORE the separator
	colIdx := strings.Index(h, "Test Case ID")
	sepIdx := strings.Index(h, "---")
	if colIdx > sepIdx {
		t.Errorf("column names should appear before separator in rewritten header:\n%s", h)
	}

	// Subsequent data rows should NOT be absorbed
	ht.update("| TC-001 | Module A | Note 1 |\n")
	h2 := ht.getHeaders()
	if strings.Contains(h2, "TC-001") {
		t.Errorf("header should NOT include subsequent data rows, got:\n%s", h2)
	}

	// Table end
	ht.update("\n")
	if ht.getHeaders() != "" {
		t.Error("header should be cleared after empty line")
	}
}

func TestHeaderTracker_NormalHeaderNoExtension(t *testing.T) {
	// A table with proper column names in the header row should NOT be extended.
	ht := newHeaderTracker()

	ht.update("| Name | Age |\n| --- | --- |\n")
	h := ht.getHeaders()
	if h == "" {
		t.Fatal("expected active header")
	}

	// First data row should NOT be absorbed into the header
	ht.update("| Asha | 25 |\n")
	h2 := ht.getHeaders()
	if strings.Contains(h2, "Asha") {
		t.Errorf("normal header should not absorb data rows, got:\n%s", h2)
	}
}

func TestSplitText_EmptyHeaderRowPrepend(t *testing.T) {
	// Simulate MarkItDown output: empty header row, real column names in first data row.
	text := "" +
		"Preface\n\n" +
		"||\n" +
		"| --- | --- | --- |\n" +
		"| Case ID | Module | Step |\n" +
		"| TC-001 | A | Step 1 |\n" +
		"| TC-002 | B | Step 2 |\n" +
		"| TC-003 | C | Step 3 |\n" +
		"| TC-004 | D | Step 4 |\n" +
		"\n" +
		"End"

	cfg := SplitterConfig{ChunkSize: 90, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	t.Logf("total chunks: %d", len(chunks))
	for i, c := range chunks {
		t.Logf("chunk[%d] seq=%d start=%d end=%d:\n%s", i, c.Seq, c.Start, c.End, c.Content)
	}

	for _, c := range chunks {
		hasLaterRow := strings.Contains(c.Content, "TC-002") ||
			strings.Contains(c.Content, "TC-003") ||
			strings.Contains(c.Content, "TC-004")
		if hasLaterRow && !strings.Contains(c.Content, "TC-001") {
			// Should have column names prepended
			if !strings.Contains(c.Content, "Case ID") {
				t.Errorf("chunk with data rows should have real column names prepended:\n%s", c.Content)
			}
			// Should NOT have the empty || row
			lines := strings.Split(c.Content, "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				isOnlyPipes := trimmed != "" && func() bool {
					for _, r := range trimmed {
						if r != '|' && r != ' ' {
							return false
						}
					}
					return true
				}()
				if isOnlyPipes {
					t.Errorf("chunk should NOT contain empty pipe row %q:\n%s", trimmed, c.Content)
					break
				}
			}
		}

		// No line should appear as a duplicate in any chunk
		lines := strings.Split(strings.TrimRight(c.Content, "\n"), "\n")
		seen := make(map[string]int)
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.Contains(trimmed, "---") {
				continue
			}
			seen[trimmed]++
			if seen[trimmed] > 1 {
				t.Errorf("line appears %d times in chunk (seq=%d): %q", seen[trimmed], c.Seq, trimmed)
			}
		}
	}

	// Verify restoration still works
	restored := restoreTextFromChunks(chunks)
	if restored != text {
		t.Errorf("restoration failed for empty-header table\n  original: %q\n  restored: %q", text, restored)
	}
}

func TestHeaderTracker_ColumnMismatchEndsTable(t *testing.T) {
	ht := newHeaderTracker()
	ht.update("| Name | Game | Fame | Blame |\n| --- | --- | --- | --- |\n")
	if ht.getHeaders() == "" {
		t.Fatal("expected active table header")
	}
	ht.update("| Sinple | Table |\n")
	if h := ht.getHeaders(); h != "" {
		t.Fatalf("2-col row should end 4-col table header, still active:\n%s", h)
	}
}

func TestHeaderTracker_ParagraphBreakEndsOnNextUnit(t *testing.T) {
	ht := newHeaderTracker()
	ht.update("| Name | Game | Fame | Blame |\n| --- | --- | --- | --- |\n")
	ht.update("| Russell Wilson | Football | High | Tacky uniform |\n\n")
	if h := ht.getHeaders(); h == "" {
		t.Fatal("paragraph break alone should not clear header yet")
	}
	if !ht.pendingTableBreak {
		t.Fatal("expected pendingTableBreak after row ending with \\n\\n")
	}
	ht.update("| Sinple | Table |\n")
	if h := ht.getHeaders(); h != "" {
		t.Fatalf("next table row should clear previous header, got %q", h)
	}
	if !ht.headerEndedThisUnit {
		t.Fatal("expected flush signal when new table starts after paragraph break")
	}
}

func TestSplitText_EnTablesNoCrossTableHeader(t *testing.T) {
	text := "## A table, with and without a header row\n\n" +
		"| Name | Game | Fame | Blame |\n" +
		"| --- | --- | --- | --- |\n" +
		"| Lebron James | Basketball | Very High | Leaving Cleveland |\n" +
		"| Ryan Braun | Baseball | Moderate | Steroids |\n" +
		"| Russell Wilson | Football | High | Tacky uniform |\n\n" +
		"| Sinple | Table |\n" +
		"| Without | Header |\n\n" +
		"| Simple  Multiparagraph | Table  Full |\n" +
		"| Of  Paragraphs | In each  Cell. |\n"

	cfg := SplitterConfig{ChunkSize: 200, ChunkOverlap: 20, Separators: []string{"\n\n", "\n", "।"}}
	chunks := SplitText(text, cfg)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	for i, c := range chunks {
		hasSinple := strings.Contains(c.Content, "| Sinple | Table |")
		hasSimple := strings.Contains(c.Content, "| Simple  Multiparagraph |")
		if hasSinple || hasSimple {
			if strings.Contains(c.Content, "| Name | Game | Fame | Blame |") {
				t.Errorf("chunk[%d] must not carry table-1 header into later tables:\n%s", i, c.Content)
			}
		}
	}
}

func TestSplitText_MultipleTablesInDocument(t *testing.T) {
	text := "" +
		"First table:\n\n" +
		"| Name | Value |\n" +
		"| --- | --- |\n" +
		"| A | 1 |\n" +
		"| B | 2 |\n" +
		"| C | 3 |\n" +
		"\n" +
		"Middle text\n\n" +
		"| Item | Status |\n" +
		"| --- | --- |\n" +
		"| X | Done |\n" +
		"| Y | Running |\n" +
		"| Z | Pending |\n" +
		"\n" +
		"Closing text"

	cfg := SplitterConfig{ChunkSize: 60, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	// Verify that if a chunk has rows from table 2, it has table 2's header, not table 1's.
	for _, c := range chunks {
		if strings.Contains(c.Content, "| Y |") && !strings.Contains(c.Content, "| X |") {
			if !strings.Contains(c.Content, "| Item | Status |") {
				t.Errorf("chunk with table-2 rows should have table-2 header:\n%s", c.Content)
			}
			if strings.Contains(c.Content, "| Name | Value |") {
				t.Errorf("chunk with table-2 rows should NOT have table-1 header:\n%s", c.Content)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Start/End restoration tests — verify original text can be reconstructed
// ---------------------------------------------------------------------------

// restoreTextFromChunks reconstructs the original text using only chunk
// Start/End positions. For chunks with prepended headers, the header is a
// "virtual" prefix whose length = runeLen(Content) - (End - Start).
// The original text portion is the last (End-Start) runes of Content.
func restoreTextFromChunks(chunks []Chunk) string {
	if len(chunks) == 0 {
		return ""
	}

	// Sort by End (ascending), then Start (ascending) — same order as Python restore_text
	sorted := make([]Chunk, len(chunks))
	copy(sorted, chunks)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0; j-- {
			if sorted[j].End < sorted[j-1].End ||
				(sorted[j].End == sorted[j-1].End && sorted[j].Start < sorted[j-1].Start) {
				sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			} else {
				break
			}
		}
	}

	var result []rune
	lastEnd := 0

	for _, c := range sorted {
		if c.End <= lastEnd {
			continue // fully contained in a previously processed chunk
		}

		contentRunes := []rune(c.Content)
		spanLen := c.End - c.Start
		headerLen := len(contentRunes) - spanLen
		if headerLen < 0 {
			headerLen = 0
		}
		// originalPortion is the text[Start:End] part, excluding any prepended header
		originalPortion := contentRunes[headerLen:]

		// Only take the portion after lastEnd (skip overlap)
		newStart := 0
		if lastEnd > c.Start {
			newStart = lastEnd - c.Start
		}
		if newStart < len(originalPortion) {
			result = append(result, originalPortion[newStart:]...)
		}

		lastEnd = c.End
	}

	return string(result)
}

func TestSplitText_RestoreTextNoTable(t *testing.T) {
	// Plain Devanagari text without any tables — baseline restoration check.
	var sb strings.Builder
	for i := 0; i < 30; i++ {
		sb.WriteString(fmt.Sprintf("अनुच्छेद %d: यह परीक्षण के लिए देवनागरी सामग्री है।", i))
		sb.WriteString("\n\n")
	}
	text := sb.String()

	cfg := SplitterConfig{ChunkSize: 50, ChunkOverlap: 10, Separators: []string{"\n\n", "\n", "।"}}
	chunks := SplitText(text, cfg)

	restored := restoreTextFromChunks(chunks)
	if restored != text {
		t.Errorf("restoration failed for plain text\n  original len: %d\n  restored len: %d",
			len([]rune(text)), len([]rune(restored)))
		// Find first difference
		orig := []rune(text)
		rest := []rune(restored)
		minLen := len(orig)
		if len(rest) < minLen {
			minLen = len(rest)
		}
		for i := 0; i < minLen; i++ {
			if orig[i] != rest[i] {
				t.Errorf("first diff at rune %d: orig=%q rest=%q", i, string(orig[i]), string(rest[i]))
				break
			}
		}
	}
}

func TestSplitText_RestoreTextWithTable(t *testing.T) {
	// Document with a table large enough to span multiple chunks.
	// Use a 2-column table (shorter header ~24 runes) + ChunkSize 80 so the
	// header can be prepended (header 24 + row ~16 = 40 < 80).
	text := "" +
		"This is the preface of the document.\n\n" +
		"| Name | City |\n" +
		"| --- | --- |\n" +
		"| Asha | Mumbai |\n" +
		"| Vijay | Delhi |\n" +
		"| Meera | Chennai |\n" +
		"| Rahul | Kolkata |\n" +
		"| Sunita | Pune |\n" +
		"| Arjun | Jaipur |\n" +
		"| Kavita | Indore |\n" +
		"| Deepak | Nagpur |\n" +
		"\n" +
		"This is the text after the table.\n" +
		"There are more ordinary paragraphs here."

	cfg := SplitterConfig{ChunkSize: 80, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	t.Logf("total chunks: %d", len(chunks))
	for i, c := range chunks {
		contentRunes := []rune(c.Content)
		spanLen := c.End - c.Start
		headerLen := len(contentRunes) - spanLen
		t.Logf("chunk[%d] seq=%d start=%d end=%d span=%d contentLen=%d headerPrepend=%d\n  content: %q",
			i, c.Seq, c.Start, c.End, spanLen, len(contentRunes), headerLen, c.Content)
	}

	// 1. Verify basic position invariants
	textRunes := []rune(text)
	for i, c := range chunks {
		if c.Start < 0 {
			t.Errorf("chunk[%d]: Start %d < 0", i, c.Start)
		}
		if c.End > len(textRunes) {
			t.Errorf("chunk[%d]: End %d > total runes %d", i, c.End, len(textRunes))
		}
		if c.End < c.Start {
			t.Errorf("chunk[%d]: End %d < Start %d", i, c.End, c.Start)
		}
	}

	// 2. Verify text[Start:End] matches the non-header portion of Content
	for i, c := range chunks {
		contentRunes := []rune(c.Content)
		spanLen := c.End - c.Start
		headerLen := len(contentRunes) - spanLen
		if headerLen < 0 {
			t.Errorf("chunk[%d]: Content rune len (%d) < span len (%d) — impossible",
				i, len(contentRunes), spanLen)
			continue
		}

		if c.Start >= 0 && c.End <= len(textRunes) {
			originalSlice := string(textRunes[c.Start:c.End])
			contentSuffix := string(contentRunes[headerLen:])
			if originalSlice != contentSuffix {
				t.Errorf("chunk[%d]: text[%d:%d] != Content[%d:]"+
					"\n  text slice:     %q"+
					"\n  content suffix: %q",
					i, c.Start, c.End, headerLen, originalSlice, contentSuffix)
			}
		}
	}

	// 3. Restore original text and compare
	restored := restoreTextFromChunks(chunks)
	if restored != text {
		t.Errorf("restoration FAILED"+
			"\n  original rune len: %d"+
			"\n  restored rune len: %d",
			len(textRunes), len([]rune(restored)))
		orig := []rune(text)
		rest := []rune(restored)
		minLen := len(orig)
		if len(rest) < minLen {
			minLen = len(rest)
		}
		for i := 0; i < minLen; i++ {
			if orig[i] != rest[i] {
				lo := i - 20
				if lo < 0 {
					lo = 0
				}
				hi := i + 20
				if hi > minLen {
					hi = minLen
				}
				t.Errorf("first diff at rune %d:\n  orig context: %q\n  rest context: %q",
					i, string(orig[lo:hi]), string(rest[lo:hi]))
				break
			}
		}
	} else {
		t.Log("restoration OK — original text perfectly reconstructed from Start/End")
	}

	// 4. Verify full coverage — Start/End spans must cover [0, len(textRunes))
	covered := make([]bool, len(textRunes))
	for _, c := range chunks {
		for p := c.Start; p < c.End && p < len(textRunes); p++ {
			covered[p] = true
		}
	}
	for i, v := range covered {
		if !v {
			t.Errorf("rune position %d is not covered by any chunk", i)
			break
		}
	}
}

func TestSplitText_RestoreTextWithMultipleTables(t *testing.T) {
	text := "" +
		"Preface\n\n" +
		"| A | B |\n| --- | --- |\n" +
		"| 1 | 2 |\n| 3 | 4 |\n| 5 | 6 |\n| 7 | 8 |\n" +
		"\nMiddle text\n\n" +
		"| X | Y |\n| --- | --- |\n" +
		"| a | b |\n| c | d |\n| e | f |\n" +
		"\nEnd"

	cfg := SplitterConfig{ChunkSize: 50, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	chunks := SplitText(text, cfg)

	// Restore and compare
	restored := restoreTextFromChunks(chunks)
	if restored != text {
		t.Errorf("multi-table restoration failed\n  original: %q\n  restored: %q", text, restored)
	}

	// Verify text[Start:End] matches content suffix for every chunk
	textRunes := []rune(text)
	for i, c := range chunks {
		contentRunes := []rune(c.Content)
		spanLen := c.End - c.Start
		headerLen := len(contentRunes) - spanLen
		if headerLen < 0 {
			t.Errorf("chunk[%d]: headerLen < 0", i)
			continue
		}
		if c.End <= len(textRunes) {
			if string(textRunes[c.Start:c.End]) != string(contentRunes[headerLen:]) {
				t.Errorf("chunk[%d]: text[Start:End] mismatch with content suffix", i)
			}
		}
	}
}

func TestSplitText_RestoreTextWithOverlap(t *testing.T) {
	// Larger overlap to stress the overlap+header interaction.
	text := "" +
		"| C1 | C2 | C3 |\n" +
		"| --- | --- | --- |\n" +
		"| A1 | A2 | A3 |\n" +
		"| B1 | B2 | B3 |\n" +
		"| C1 | C2 | C3 |\n" +
		"| D1 | D2 | D3 |\n" +
		"| E1 | E2 | E3 |\n" +
		"| F1 | F2 | F3 |\n" +
		"\n" +
		"Text after the table."

	for _, overlap := range []int{0, 3, 10, 20} {
		t.Run(fmt.Sprintf("overlap=%d", overlap), func(t *testing.T) {
			cfg := SplitterConfig{ChunkSize: 60, ChunkOverlap: overlap, Separators: []string{"\n\n", "\n"}}
			chunks := SplitText(text, cfg)

			restored := restoreTextFromChunks(chunks)
			if restored != text {
				t.Errorf("restoration failed with overlap=%d\n  orig len=%d  rest len=%d",
					overlap, len([]rune(text)), len([]rune(restored)))
				for i, c := range chunks {
					t.Logf("  chunk[%d] start=%d end=%d content=%q", i, c.Start, c.End, c.Content)
				}
			}
		})
	}
}

func TestSplitTextParentChild_WithTableHeaders(t *testing.T) {
	text := "" +
		"Preface\n\n" +
		"| CA | CB |\n" +
		"| --- | --- |\n" +
		"| D1 | D2 |\n" +
		"| D3 | D4 |\n" +
		"| D5 | D6 |\n" +
		"| D7 | D8 |\n" +
		"\n" +
		"End"

	parentCfg := SplitterConfig{ChunkSize: 200, ChunkOverlap: 0, Separators: []string{"\n\n", "\n"}}
	childCfg := SplitterConfig{ChunkSize: 40, ChunkOverlap: 5, Separators: []string{"\n\n", "\n"}}
	result := SplitTextParentChild(text, parentCfg, childCfg)

	if len(result.Children) == 0 {
		t.Fatal("expected child chunks")
	}

	// Verify child chunk positions don't exceed parent document
	textRunes := []rune(text)
	for i, child := range result.Children {
		if child.Start < 0 {
			t.Errorf("child[%d]: negative Start %d", i, child.Start)
		}
		if child.End > len(textRunes) {
			t.Errorf("child[%d]: End %d exceeds text rune count %d", i, child.End, len(textRunes))
		}
	}
}
