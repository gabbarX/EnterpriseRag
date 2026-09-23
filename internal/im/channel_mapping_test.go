package im

import (
	"testing"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
)

// Files uploaded through an IM channel are tagged with a knowledge Channel.
// Only platforms with a dedicated channel constant get their own; everything
// else falls through to the generic "im" channel.
func TestIMPlatformToChannel(t *testing.T) {
	cases := map[string]string{
		"slack": types.ChannelSlack,
		"Slack": types.ChannelSlack, // matching is case-insensitive
		// Platforms without a dedicated channel fall back to the generic one.
		"telegram":   types.ChannelIM,
		"mattermost": types.ChannelIM,
		"":           types.ChannelIM,
	}

	for platform, want := range cases {
		if got := imPlatformToChannel(platform); got != want {
			t.Errorf("imPlatformToChannel(%q) = %q, want %q", platform, got, want)
		}
	}
}
