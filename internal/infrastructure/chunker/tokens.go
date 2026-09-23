// Package chunker - tokens.go provides language-aware token count approximation.
//
// We avoid pulling in a tokenizer dependency (e.g. tiktoken) and instead use
// per-language chars-per-token ratios derived from common embedding model
// vocabularies. The numbers are conservative — they tend to slightly
// over-estimate token counts so that chunks stay safely under model limits.
package chunker

import (
	"unicode"
	"unicode/utf8"
)

// Language identifiers used by the token estimator and the heuristic splitter.
const (
	LangEnglish = "en"
	LangGerman  = "de"
	LangMixed   = "mixed"
)

// charsPerToken holds approximate chars/token ratios per language.
// Numbers err on the conservative side so estimates over-shoot a little.
var charsPerToken = map[string]float64{
	LangEnglish: 4.0,
	LangGerman:  4.5,
	LangMixed:   3.0,
}

// ApproxTokenCount returns a conservative token estimate for s in the given
// language. An empty or unknown lang falls back to "mixed".
func ApproxTokenCount(s string, lang string) int {
	if s == "" {
		return 0
	}
	return ApproxTokenCountFromRuneLen(utf8.RuneCountInString(s), lang)
}

// ApproxTokenCountFromRuneLen is the allocation-free variant of
// ApproxTokenCount when the caller has already computed the rune length.
// Use this in hot loops where the same content's rune count would
// otherwise be recomputed multiple times (e.g. preview endpoint emitting
// per-chunk stats).
func ApproxTokenCountFromRuneLen(runeLen int, lang string) int {
	if runeLen <= 0 {
		return 0
	}
	ratio, ok := charsPerToken[lang]
	if !ok {
		ratio = charsPerToken[LangMixed]
	}
	approx := float64(runeLen) / ratio
	if approx < 1 {
		return 1
	}
	return int(approx + 0.5)
}

// DetectLanguage returns a coarse language label. The result is one of
// LangGerman, LangEnglish or LangMixed. Detection is cheap and meant only for
// heuristic dispatch — it is NOT a replacement for proper language
// identification.
//
// Text written in a script we do not profile (Devanagari, Tamil, Bengali, …)
// carries no Latin-letter signal, so it reports LangMixed and the caller falls
// back to the script-agnostic defaults rather than to English-only rules.
func DetectLanguage(s string) string {
	if s == "" {
		return LangMixed
	}
	var latin, other, umlaut int
	for _, r := range s {
		switch {
		case isGermanUmlaut(r):
			umlaut++
			latin++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			latin++
		case unicode.IsLetter(r), unicode.Is(unicode.Mn, r), unicode.Is(unicode.Mc, r):
			// Any other script, including the combining marks (matras, virama)
			// that carry a large share of the code points in Indic text.
			other++
		}
	}
	total := latin + other
	if total == 0 {
		return LangMixed
	}
	// Meaningful presence of both a Latin and a non-Latin script, or mostly
	// non-Latin: stay neutral.
	if float64(latin)/float64(total) < 0.7 {
		return LangMixed
	}
	if umlaut > 0 || hasGermanWords(s) {
		return LangGerman
	}
	return LangEnglish
}

func isGermanUmlaut(r rune) bool {
	switch r {
	case 'ä', 'ö', 'ü', 'Ä', 'Ö', 'Ü', 'ß':
		return true
	}
	return false
}

// hasGermanWords does a tiny stop-word check to bias towards "de" when the
// text uses common German function words. Cheap heuristic — false positives
// on borrowed terms are acceptable.
func hasGermanWords(s string) bool {
	const sample = 512
	if len(s) > sample {
		s = s[:sample]
	}
	for _, w := range []string{" der ", " die ", " das ", " und ", " ist ", " nicht ", " mit ", " auf "} {
		if containsLower(s, w) {
			return true
		}
	}
	return false
}

func containsLower(haystack, needle string) bool {
	if len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			h := haystack[i+j]
			if h >= 'A' && h <= 'Z' {
				h += 'a' - 'A'
			}
			if h != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// CharsForTokenLimit converts a token limit into an approximate character
// budget for a given language. Used to size chunks so they fit within an
// embedding model's max-token window with a small safety margin.
func CharsForTokenLimit(tokens int, lang string) int {
	if tokens <= 0 {
		return 0
	}
	ratio, ok := charsPerToken[lang]
	if !ok {
		ratio = charsPerToken[LangMixed]
	}
	// 0.9 safety factor so we under-shoot the model limit instead of overshooting.
	return int(float64(tokens) * ratio * 0.9)
}
