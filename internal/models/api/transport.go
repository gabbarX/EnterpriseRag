package api

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	secutils "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/utils"
)

// LLM call timeout configuration. This is only a fallback for when the caller
// has not set a deadline, so that a hung request cannot block a worker forever.
// If the caller's ctx already carries a deadline (whether shorter or longer
// than the default), it is honoured as-is and no default timeout is layered on
// top. Both values can be overridden with environment variables:
//   - ENTERPRISERAG_LLM_CHAT_TIMEOUT_SECONDS    fallback timeout for non-streaming calls (default 600s)
//   - ENTERPRISERAG_LLM_STREAM_TIMEOUT_SECONDS  fallback timeout for streaming calls (default 1800s)
var (
	DefaultChatTimeout   = envDurationSeconds("ENTERPRISERAG_LLM_CHAT_TIMEOUT_SECONDS", 300*time.Second)
	DefaultStreamTimeout = envDurationSeconds("ENTERPRISERAG_LLM_STREAM_TIMEOUT_SECONDS", 600*time.Second)
)

// envDurationSeconds reads an environment variable expressed in seconds, and
// falls back to fallback when it cannot be parsed or is not positive.
func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}

// WithLLMTimeout attaches a fallback timeout only when the caller's ctx has no
// deadline; if the caller has set one explicitly (whether shorter or longer) the
// ctx is returned unchanged, so the caller keeps the final say over its own
// timeout policy.
func WithLLMTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// HTTPClient is a shared HTTP client for raw HTTP LLM calls with connection-level timeouts.
// Per-request timeout is enforced via context deadline (see DefaultChatTimeout / DefaultStreamTimeout)
// rather than http.Client.Timeout, so streaming calls are not prematurely terminated.
// Uses SSRFSafeDialContext to prevent DNS rebinding attacks at the connection layer.
var httpTransport = &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	DialContext:         secutils.SSRFSafeDialContext,
	TLSHandshakeTimeout: 10 * time.Second,
	IdleConnTimeout:     90 * time.Second,
	MaxIdleConnsPerHost: 5,
}

// HTTPClient is the shared SSRF-safe client every protocol uses.
var HTTPClient = secutils.NewSSRFSafeHTTPClientWithTransport(
	secutils.SSRFSafeHTTPClientConfig{Timeout: 0, MaxRedirects: 10},
	httpTransport,
)
