package service

import (
	"regexp"
	"strings"
	"testing"
)

func TestSanitizeOCRText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   \n\t  ",
			want:  "",
		},
		{
			name:  "pure HTML skeleton with no text",
			input: `<html><body><div class="image"><img/></div></body></html>`,
			want:  "",
		},
		{
			name:  "HTML with only whitespace text",
			input: "<html><body>  \n  </body></html>",
			want:  "",
		},
		{
			name:  "valid markdown passes through",
			input: "# शीर्षक\n\nयह एक सामान्य अनुच्छेद है, जिसमें कुछ सामग्री है।\n\n| स्तंभ1 | स्तंभ2 |\n| --- | --- |\n| डेटा1 | डेटा2 |",
			want:  "# शीर्षक\n\nयह एक सामान्य अनुच्छेद है, जिसमें कुछ सामग्री है।\n\n| स्तंभ1 | स्तंभ2 |\n| --- | --- |\n| डेटा1 | डेटा2 |",
		},
		{
			name:  "code block wrapper stripped",
			input: "```markdown\n# दस्तावेज़ शीर्षक\n\nमूल सामग्री यहाँ है।\n```",
			want:  "# दस्तावेज़ शीर्षक\n\nमूल सामग्री यहाँ है।",
		},
		{
			name:  "html code block wrapper stripped",
			input: "```html\n<p>यह एक अनुच्छेद है</p>\n```",
			want:  "यह एक अनुच्छेद है",
		},
		{
			name:  "HTML document converted to markdown",
			input: "<html><body><h1>शीर्षक</h1><p>यह एक लंबा मूल पाठ है, जो HTML से Markdown रूपांतरण की जाँच के लिए है।</p></body></html>",
			want:  "# शीर्षक\n\nयह एक लंबा मूल पाठ है, जो HTML से Markdown रूपांतरण की जाँच के लिए है।",
		},
		{
			name:  "known empty reply - Chinese",
			input: "无文字内容",
			want:  "",
		},
		{
			name:  "known empty reply - no text",
			input: "No text",
			want:  "",
		},
		{
			name:  "known empty reply - 图片中没有文字",
			input: "图片中没有文字",
			want:  "",
		},
		{
			name:  "plain text with minimal HTML not converted",
			input: "यह सामान्य पाठ है, कीमत <100 रुपये।",
			want:  "यह सामान्य पाठ है, कीमत <100 रुपये।",
		},
		{
			name:  "multiple blank lines collapsed",
			input: "अनुच्छेद एक\n\n\n\n\nअनुच्छेद दो",
			want:  "अनुच्छेद एक\n\nअनुच्छेद दो",
		},
		{
			name:  "HTML with substantial text content is converted",
			input: "<div><h2>रिपोर्ट सारांश</h2><p>इस तिमाही राजस्व 15% बढ़ा, शुद्ध लाभ ₹2.3 करोड़ रहा।</p><table><tr><th>संकेतक</th><th>मान</th></tr><tr><td>राजस्व</td><td>₹10 करोड़</td></tr></table></div>",
			want:  "", // placeholder; will be checked for non-empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeOCRText(tt.input)

			if tt.name == "HTML with substantial text content is converted" {
				if got == "" {
					t.Errorf("sanitizeOCRText() returned empty for substantial HTML content")
				}
				if got == tt.input {
					t.Errorf("sanitizeOCRText() did not convert HTML, got original")
				}
				return
			}

			if got != tt.want {
				t.Errorf("sanitizeOCRText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripMarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no code block",
			input: "just normal text",
			want:  "just normal text",
		},
		{
			name:  "markdown code block",
			input: "```markdown\n# Title\nContent here\n```",
			want:  "# Title\nContent here",
		},
		{
			name:  "html code block",
			input: "```html\n<p>hello</p>\n```",
			want:  "<p>hello</p>",
		},
		{
			name:  "plain code block",
			input: "```\nsome text\n```",
			want:  "some text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdownCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLooksLikeHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "HTML document",
			input: "<html><body><p>text</p></body></html>",
			want:  true,
		},
		{
			name:  "DOCTYPE",
			input: "<!DOCTYPE html><html><body></body></html>",
			want:  true,
		},
		{
			name:  "body tag",
			input: "<body><p>content</p></body>",
			want:  true,
		},
		{
			name:  "plain markdown",
			input: "# Title\n\nSome paragraph text",
			want:  false,
		},
		{
			name:  "text with minor HTML",
			input: "This is mostly text with a <b>bold</b> word.",
			want:  false,
		},
		{
			name:  "heavy HTML tags",
			input: "<div><p><span>x</span></p></div><div><p><span>y</span></p></div>",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeHTML(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeHTML() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSanitizeOCRText_ConvertsInlineHTMLTable guards against a regression where
// an HTML <table> embedded in a markdown body survived sanitization untouched:
// the mixed content does not satisfy looksLikeHTML, so the old HTML-to-markdown
// path never ran and the raw <table> markup reached the chunker.
func TestSanitizeOCRText_ConvertsInlineHTMLTable(t *testing.T) {
	para := "इस तिमाही कंपनी का समग्र कारोबार स्थिर रहा और राजस्व तथा लाभ दोनों प्रमुख संकेतकों में वार्षिक वृद्धि दर्ज हुई। " +
		"प्रबंधन को परिणाम शीघ्र समझ आ सकें, इसके लिए नीचे दी गई तालिका में रिपोर्ट अवधि के मुख्य वित्तीय आँकड़े संकलित हैं, " +
		"जिनका उपयोग आगे के कारोबारी विश्लेषण, बजट निर्माण और वार्षिक समीक्षा में किया जा सकता है। " +
		"कृपया वास्तविक कारोबारी परिस्थितियों के साथ ही इनका आकलन करें और किसी भी आँकड़े को संदर्भ से अलग न पढ़ें।"
	tail := "ऊपर दिए गए सभी आँकड़े वित्त विभाग द्वारा सत्यापित औपचारिक विवरणों से लिए गए हैं, " +
		"लेखांकन नीति में कोई बदलाव नहीं हुआ है। प्रश्न के लिए वित्त विभाग से संपर्क करें। " +
		"अंतिम व्याख्या का अधिकार वित्त विभाग के पास है।"
	input := "# रिपोर्ट\n\n" + para + "\n\n" +
		`<table><tr><th>संकेतक</th><th>मान</th></tr>` +
		`<tr><td>राजस्व</td><td>₹10 करोड़</td></tr>` +
		`<tr><td>लाभ</td><td>₹2.3 करोड़</td></tr></table>` +
		"\n\n" + tail + "\n\n"

	if looksLikeHTML(input) {
		t.Fatalf("test input unexpectedly looks like HTML; the regression test must exercise the mixed-content path")
	}

	got := sanitizeOCRText(input)

	if strings.Contains(got, "<table") {
		t.Fatalf("expected inline HTML table to be converted, got:\n%s", got)
	}
	if !strings.Contains(got, "# रिपोर्ट") || !strings.Contains(got, "अंतिम व्याख्या का अधिकार वित्त विभाग के पास है।") {
		t.Fatalf("expected surrounding markdown to be preserved, got:\n%s", got)
	}
	for _, want := range []string{"संकेतक", "मान", "राजस्व", "₹10 करोड़", "लाभ", "₹2.3 करोड़"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected converted table to retain %q, got:\n%s", want, got)
		}
	}
	if !regexp.MustCompile(`\|[-:]{3,}\|`).MatchString(got) {
		t.Fatalf("expected a GFM table separator row, got:\n%s", got)
	}
}

func TestIsKnownEmptyReply(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"无文字内容", true},
		{"无法识别", true},
		{"no text", true},
		{"No Text", true},
		{"NO CONTENT", true},
		{"empty", true},
		{"यह सामान्य सामग्री है", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isKnownEmptyReply(tt.input)
			if got != tt.want {
				t.Errorf("isKnownEmptyReply(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
