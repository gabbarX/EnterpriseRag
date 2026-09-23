package chunker

import (
	"testing"
	"unicode/utf8"
)

func TestApproxTokenCount_English(t *testing.T) {
	got := ApproxTokenCount("The quick brown fox jumps over the lazy dog.", LangEnglish)
	// 44 chars / 4 ≈ 11 tokens
	if got < 9 || got > 13 {
		t.Errorf("English token estimate out of range: got %d, want 9..13", got)
	}
}

// TestApproxTokenCount_DevanagariUsesRuneLength is a multibyte tripwire:
// Devanagari is three bytes per rune, so a byte-length estimate over-shoots
// by 3x and every Indic chunk would be cut far below the model budget.
func TestApproxTokenCount_DevanagariUsesRuneLength(t *testing.T) {
	s := "यह एक परीक्षण पाठ है"
	runes := utf8.RuneCountInString(s)
	if runes == len(s) {
		t.Fatal("test requires multi-byte characters")
	}
	got := ApproxTokenCount(s, LangMixed)
	want := int(float64(runes)/3.0 + 0.5)
	if got != want {
		t.Errorf("token estimate: got %d, want %d (runes=%d, bytes=%d)", got, want, runes, len(s))
	}
}

func TestApproxTokenCount_Empty(t *testing.T) {
	if got := ApproxTokenCount("", LangEnglish); got != 0 {
		t.Errorf("empty string should return 0 tokens, got %d", got)
	}
}

func TestApproxTokenCount_UnknownLang(t *testing.T) {
	got := ApproxTokenCount("Hello world hello world", "xx")
	if got <= 0 {
		t.Errorf("unknown lang should fall back to mixed, got %d", got)
	}
}

func TestDetectLanguage_English(t *testing.T) {
	if got := DetectLanguage("The quick brown fox jumps over the lazy dog."); got != LangEnglish {
		t.Errorf("expected English, got %s", got)
	}
}

func TestDetectLanguage_German(t *testing.T) {
	if got := DetectLanguage("Der schnelle braune Fuchs springt über den faulen Hund."); got != LangGerman {
		t.Errorf("expected German, got %s", got)
	}
}

func TestDetectLanguage_GermanByStopwords(t *testing.T) {
	// No umlauts but plenty of German function words.
	if got := DetectLanguage("Das ist ein Test und nicht mit Umlauten."); got != LangGerman {
		t.Errorf("expected German via stopwords, got %s", got)
	}
}

func TestDetectLanguage_DevanagariIsNeutral(t *testing.T) {
	// Devanagari is not a profiled language; it must fall back to the neutral
	// label rather than be mistaken for English and get English-only rules.
	if got := DetectLanguage("यह एक परीक्षण पाठ है"); got != LangMixed {
		t.Errorf("expected Mixed for an unprofiled script, got %s", got)
	}
}

func TestDetectLanguage_Mixed(t *testing.T) {
	got := DetectLanguage("This मिश्रित content with देवनागरी inside")
	if got != LangMixed {
		t.Errorf("expected Mixed, got %s", got)
	}
}

func TestCharsForTokenLimit_AppliesSafetyMargin(t *testing.T) {
	got := CharsForTokenLimit(1000, LangEnglish)
	// 1000 * 4 * 0.9 = 3600
	if got < 3500 || got > 3700 {
		t.Errorf("char budget for 1000 EN tokens out of range: got %d", got)
	}
}

func TestCharsForTokenLimit_ZeroTokens(t *testing.T) {
	if got := CharsForTokenLimit(0, LangEnglish); got != 0 {
		t.Errorf("zero tokens should give zero chars, got %d", got)
	}
}
