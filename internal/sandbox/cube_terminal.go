// Package sandbox: stubbed interactive terminal capability for the Cube
// adapter.
//
// The CubeSandbox runtime rode envd's PTY service through the vendor SDK. With
// that runtime removed there is no PTY to attach to, so the capability is kept
// only so the adapter still satisfies RemoteTerminalManager and reports the
// absence explicitly rather than by failing to compile.
package sandbox

import (
	"context"
)

// Compile-time proof that the Cube adapter still serves the terminal
// capability interface.
var _ RemoteTerminalManager = (*CubeRemoteClient)(nil)

// OpenTerminal reports that no execution runtime is configured. It previously
// opened an interactive shell PTY inside the sandbox behind handle.
func (c *CubeRemoteClient) OpenTerminal(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ RemoteTerminalOptions,
) (RemoteTerminalSession, error) {
	return nil, cubeUnavailable("OpenTerminal")
}
