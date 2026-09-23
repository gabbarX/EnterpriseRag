package utils

import (
	"net/http"
	"strings"
)

// reservedHeaderKeys lists the critical request headers that user-supplied
// custom headers are not allowed to override. These headers are controlled by
// each provider's signing, authentication or SSE flow, and overriding them can
// make the call fail outright.
var reservedHeaderKeys = map[string]struct{}{
	"authorization":     {},
	"api-key":           {},
	"x-api-key":         {},
	"x-goog-api-key":    {},
	"content-type":      {},
	"content-length":    {},
	"accept-encoding":   {},
	"host":              {},
	"connection":        {},
	"transfer-encoding": {},
}

// IsReservedHeader reports whether a header key is reserved. Reserved headers
// cannot be overridden by custom headers.
func IsReservedHeader(key string) bool {
	_, ok := reservedHeaderKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

// ApplyCustomHeaders writes user-supplied custom headers onto an http.Request.
// Reserved headers (Authorization, api-key, Content-Type and so on) are skipped
// so authentication and signing are not broken. Other headers overwrite any
// entry of the same name, letting the user replace a default (such as Accept).
func ApplyCustomHeaders(req *http.Request, headers map[string]string) {
	if req == nil || len(headers) == 0 {
		return
	}
	for k, v := range headers {
		name := strings.TrimSpace(k)
		if name == "" {
			continue
		}
		if IsReservedHeader(name) {
			continue
		}
		req.Header.Set(name, v)
	}
}
