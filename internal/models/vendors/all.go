// Package vendors links every built-in vendor package so their init
// functions register with the catalog. Import it for side effects from the
// binary's entry point (and from tests that need the full catalog).
package vendors

import (
	// Each vendor registers itself with the catalog in init.
	_ "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vendors/anthropic"
	_ "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vendors/gemini"
	_ "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vendors/generic"
	_ "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vendors/openai"
	_ "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/models/vendors/openrouter"
)
