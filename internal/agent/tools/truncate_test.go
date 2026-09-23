package tools

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncateToolOutput(t *testing.T) {
	t.Run("short output unchanged", func(t *testing.T) {
		input := "hello world"
		result := TruncateToolOutput(input, 1000)
		assert.Equal(t, input, result)
	})

	t.Run("exact limit unchanged", func(t *testing.T) {
		input := strings.Repeat("a", 16000)
		result := TruncateToolOutput(input, 16000)
		assert.Equal(t, input, result)
	})

	t.Run("zero maxChars returns unchanged", func(t *testing.T) {
		input := strings.Repeat("a", 100)
		result := TruncateToolOutput(input, 0)
		assert.Equal(t, input, result)
	})

	t.Run("negative maxChars returns unchanged", func(t *testing.T) {
		input := strings.Repeat("a", 100)
		result := TruncateToolOutput(input, -1)
		assert.Equal(t, input, result)
	})

	t.Run("large output truncated with marker", func(t *testing.T) {
		input := strings.Repeat("H", 10000) + strings.Repeat("T", 10000)
		result := TruncateToolOutput(input, 5000)

		runeCount := utf8.RuneCountInString(result)
		assert.LessOrEqual(t, runeCount, 5000+truncationMarkerReserve,
			"truncated output rune count should be within limits")
		assert.True(t, strings.HasPrefix(result, "HHHH"),
			"should preserve head content")
		assert.True(t, strings.HasSuffix(result, "TTTT"),
			"should preserve tail content")
		assert.Contains(t, result, "output truncated")
		assert.Contains(t, result, "20000 → 5000 chars")
	})

	t.Run("50k output truncated to 16k default", func(t *testing.T) {
		input := strings.Repeat("x", 50000)
		result := TruncateToolOutput(input, DefaultMaxToolOutput)

		runeCount := utf8.RuneCountInString(result)
		assert.Less(t, runeCount, DefaultMaxToolOutput+truncationMarkerReserve)
		assert.Contains(t, result, "output truncated")
	})

	t.Run("head and tail are from correct positions", func(t *testing.T) {
		head := strings.Repeat("A", 5000)
		mid := strings.Repeat("B", 5000)
		tail := strings.Repeat("C", 5000)
		input := head + mid + tail
		result := TruncateToolOutput(input, 5000)

		assert.True(t, strings.HasPrefix(result, "AAAA"))
		assert.True(t, strings.HasSuffix(result, "CCCC"))
	})

	t.Run("Devanagari text truncated by rune count not bytes", func(t *testing.T) {
		// Each Devanagari character is 3 bytes in UTF-8 but 1 rune.
		// 10000 Devanagari chars = 30000 bytes. With maxChars=5000 runes,
		// the result should have ~5000 runes, not be cut at 5000 bytes.
		head := strings.Repeat("क", 5000)
		tail := strings.Repeat("ख", 5000)
		input := head + tail
		result := TruncateToolOutput(input, 5000)

		resultRunes := utf8.RuneCountInString(result)
		assert.LessOrEqual(t, resultRunes, 5000+truncationMarkerReserve,
			"should truncate by rune count, not byte count")
		assert.True(t, strings.HasPrefix(result, "ककक"),
			"should preserve Devanagari head content")
		assert.True(t, strings.HasSuffix(result, "खखख"),
			"should preserve Devanagari tail content")
		assert.Contains(t, result, "10000 → 5000 chars")
	})

	t.Run("mixed Devanagari and ASCII", func(t *testing.T) {
		// Mix of 1-byte and 3-byte chars
		input := strings.Repeat("a", 3000) + strings.Repeat("ग", 3000) +
			strings.Repeat("b", 3000) + strings.Repeat("घ", 3000)
		result := TruncateToolOutput(input, 5000)

		resultRunes := utf8.RuneCountInString(result)
		assert.LessOrEqual(t, resultRunes, 5000+truncationMarkerReserve)
		assert.True(t, strings.HasPrefix(result, "aaa"))
		assert.True(t, strings.HasSuffix(result, "घघघ"))
	})

	t.Run("Devanagari within limit unchanged", func(t *testing.T) {
		input := strings.Repeat("च", 100)
		result := TruncateToolOutput(input, 200)
		assert.Equal(t, input, result)
	})
}
