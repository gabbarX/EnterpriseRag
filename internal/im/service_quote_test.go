package im

import (
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

func TestFormatQuotedContext(t *testing.T) {
	tests := []struct {
		name  string
		quote *QuotedMessage
		want  string
	}{
		{
			name:  "nil quote",
			quote: nil,
			want:  "",
		},
		{
			name:  "empty content no NonTextType",
			quote: &QuotedMessage{Content: ""},
			want:  "",
		},
		{
			name:  "non-text image quote generates instruction",
			quote: &QuotedMessage{NonTextType: "image"},
			want:  "The user quoted a image message, but you cannot see its contents. Tell the user plainly that you cannot process image messages at present, and ask them to describe the question in text. Do not guess what the message contained.",
		},
		{
			name:  "non-text file quote generates instruction",
			quote: &QuotedMessage{NonTextType: "file"},
			want:  "The user quoted a file message, but you cannot see its contents. Tell the user plainly that you cannot process file messages at present, and ask them to describe the question in text. Do not guess what the message contained.",
		},
		{
			name:  "non-text unknown type uses fallback label",
			quote: &QuotedMessage{NonTextType: "location"},
			want:  "The user quoted a unsupported message, but you cannot see its contents. Tell the user plainly that you cannot process unsupported messages at present, and ask them to describe the question in text. Do not guess what the message contained.",
		},
		{
			name:  "bot message",
			quote: &QuotedMessage{Content: "bot reply text", IsBotMessage: true},
			want:  "The user quoted one of your own earlier replies. It is context only:\n<quoted_message>\nbot reply text\n</quoted_message>",
		},
		{
			name:  "user message",
			quote: &QuotedMessage{Content: "user message text", IsBotMessage: false},
			want:  "The user quoted an earlier message. It is context only:\n<quoted_message>\nuser message text\n</quoted_message>",
		},
		{
			name: "truncation at 500 runes",
			quote: &QuotedMessage{
				Content:      string(make([]rune, 600)),
				IsBotMessage: false,
			},
			want: "The user quoted an earlier message. It is context only:\n<quoted_message>\n" + string(make([]rune, 500)) + "...\n</quoted_message>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatQuotedContext(tt.quote)
			if got != tt.want {
				t.Errorf("formatQuotedContext() length = %d, want length %d", len(got), len(tt.want))
				if len(got) < 200 && len(tt.want) < 200 {
					t.Errorf("got = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestBuildIMQARequest_QuotedContext(t *testing.T) {
	session := &types.Session{ID: "s1"}

	t.Run("nil quote produces empty QuotedContext", func(t *testing.T) {
		req := buildIMQARequest(session, "hello", "a1", "u1", nil, nil, nil)
		if req.QuotedContext != "" {
			t.Errorf("QuotedContext = %q, want empty", req.QuotedContext)
		}
		if req.Query != "hello" {
			t.Errorf("Query = %q, want %q", req.Query, "hello")
		}
	})

	t.Run("bot quote sets QuotedContext with bot label", func(t *testing.T) {
		quote := &QuotedMessage{Content: "bot reply", IsBotMessage: true}
		req := buildIMQARequest(session, "follow up", "a1", "u1", nil, nil, quote)
		if req.Query != "follow up" {
			t.Errorf("Query = %q, want %q", req.Query, "follow up")
		}
		want := "The user quoted one of your own earlier replies. It is context only:\n<quoted_message>\nbot reply\n</quoted_message>"
		if req.QuotedContext != want {
			t.Errorf("QuotedContext = %q, want %q", req.QuotedContext, want)
		}
	})

	t.Run("user quote sets QuotedContext with user label", func(t *testing.T) {
		quote := &QuotedMessage{Content: "user msg", IsBotMessage: false}
		req := buildIMQARequest(session, "question", "a1", "u1", nil, nil, quote)
		want := "The user quoted an earlier message. It is context only:\n<quoted_message>\nuser msg\n</quoted_message>"
		if req.QuotedContext != want {
			t.Errorf("QuotedContext = %q, want %q", req.QuotedContext, want)
		}
	})
}
