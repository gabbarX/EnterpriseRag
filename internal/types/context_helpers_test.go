package types

import (
	"context"
	"testing"
)

func TestIsSyntheticUserID(t *testing.T) {
	cases := []struct {
		name string
		id   string
		want bool
	}{
		{"matches system-<digits>", "system-1", true},
		{"matches large tenant id", "system-1234567890", true},
		{"empty string", "", false},
		{"prefix only", "system-", false},
		{"missing prefix", "1", false},
		{"non-digit suffix", "system-abc", false},
		{"mixed suffix", "system-1a2", false},
		{"prefix with space", "system- 1", false},
		{"uppercase prefix", "SYSTEM-1", false},
		{"normal uuid user", "550e8400-e29b-41d4-a716-446655440000", false},
		{"system uuid trap", "system-550e8400", false}, // contains '-'
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			got := IsSyntheticUserID(c.id)
			if got != c.want {
				t.Fatalf("IsSyntheticUserID(%q) = %v, want %v", c.id, got, c.want)
			}
		})
	}
}

func TestMCPOAuthNonInteractive(t *testing.T) {
	if IsMCPOAuthNonInteractive(nil) {
		t.Fatal("nil context should not be non-interactive")
	}
	if IsMCPOAuthNonInteractive(context.Background()) {
		t.Fatal("background context should not be non-interactive")
	}

	ctx := WithMCPOAuthNonInteractive(context.Background())
	if !IsMCPOAuthNonInteractive(ctx) {
		t.Fatal("marked context should be non-interactive")
	}
	child := context.WithValue(ctx, TenantIDContextKey, uint64(7))
	if !IsMCPOAuthNonInteractive(child) {
		t.Fatal("child context should inherit non-interactive flag")
	}
}

func TestLLMCallMetadataContext(t *testing.T) {
	ctx := WithLLMCallMetadata(context.Background(), "wiki_page_modify", "abc123")
	purpose, prefix := LLMCallMetadataFromContext(ctx)
	if purpose != "wiki_page_modify" || prefix != "abc123" {
		t.Fatalf("metadata = (%q, %q)", purpose, prefix)
	}
}

func TestTaskRetryMetadataContext(t *testing.T) {
	if _, _, ok := TaskRetryMetadataFromContext(nil); ok {
		t.Fatal("nil context should not contain task retry metadata")
	}
	if _, _, ok := TaskRetryMetadataFromContext(context.Background()); ok {
		t.Fatal("background context should not contain task retry metadata")
	}

	ctx := WithTaskRetryMetadata(context.Background(), 2, 3)
	retried, maxRetry, ok := TaskRetryMetadataFromContext(ctx)
	if !ok || retried != 2 || maxRetry != 3 {
		t.Fatalf("retry metadata = (%d, %d, %v), want (2, 3, true)", retried, maxRetry, ok)
	}
}
