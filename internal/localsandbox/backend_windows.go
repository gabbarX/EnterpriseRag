//go:build windows

package localsandbox

import (
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/localsandbox/core"
	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/localsandbox/winhost"
)

func NewBackend() (core.Backend, error) { return winhost.New() }
