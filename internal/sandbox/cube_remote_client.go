// Package sandbox: stubbed Cube adapter for the provider-neutral
// RemoteSandboxClient.
//
// The CubeSandbox runtime and its vendor SDK were removed from this product.
// CubeRemoteClient is retained so the package's exported surface, the
// SandboxTypeCube configuration shape and every caller that dispatches on it
// keep compiling, but no execution runtime sits behind it any more: every
// operation reports ErrSandboxUnavailable. A replacement runtime is tracked as
// a separate follow-up spec.
package sandbox

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

// ErrSandboxUnavailable is returned by every sandbox operation while no
// execution runtime is configured. The CubeSandbox runtime was removed; a
// replacement is tracked as a separate follow-up spec.
var ErrSandboxUnavailable = errors.New("sandbox: no execution runtime configured")

// sandboxRuntimeEnabled reports whether an operator has explicitly opted in to
// the (currently absent) execution runtime. It defaults to false, so a
// deployment that never sets ENTERPRISERAG_SANDBOX_ENABLED gets the unavailable path.
func sandboxRuntimeEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ENTERPRISERAG_SANDBOX_ENABLED")), "true")
}

// cubeUnavailable wraps ErrSandboxUnavailable in the package's neutral remote
// error so callers that branch on RemoteErrorKind keep working.
func cubeUnavailable(op string) error {
	return NewRemoteError(
		SandboxTypeCube, op, RemoteErrorKindUnavailable,
		ErrSandboxUnavailable.Error(), ErrSandboxUnavailable,
	)
}

// CubeRemoteClient is the retained shape of the Cube backend. It carries only
// its configuration: the SDK-backed control and data planes are gone, so the
// struct holds no client, transport or token registry any more.
type CubeRemoteClient struct {
	config *Config
}

// NewCubeRemoteClient constructs a Cube-backed RemoteSandboxClient. With the
// CubeSandbox runtime removed it only validates the configuration and the
// ENTERPRISERAG_SANDBOX_ENABLED opt-in; the returned client reports every operation as
// unavailable.
func NewCubeRemoteClient(config *Config) (*CubeRemoteClient, error) {
	return NewCubeRemoteClientWithPool(config, nil)
}

// NewCubeRemoteClientWithPool matches the pooled constructor callers still use.
// The pool argument is accepted and ignored: with no data plane to dial there
// are no connections to share.
func NewCubeRemoteClientWithPool(
	config *Config,
	_ *SandboxGatewayTransportPool,
) (*CubeRemoteClient, error) {
	if config == nil {
		return nil, errors.New("cube remote client config is required")
	}
	if !sandboxRuntimeEnabled() {
		return nil, ErrSandboxUnavailable
	}
	return &CubeRemoteClient{config: config}, nil
}

// cubeRemoteHandle is the RemoteSandboxHandle the stub would return. No
// operation produces one any more, so it carries only the identity fields the
// interface requires.
type cubeRemoteHandle struct {
	id       string
	metadata map[string]string
	token    string
}

func (h *cubeRemoteHandle) ID() string {
	if h == nil {
		return ""
	}
	return h.id
}

func (h *cubeRemoteHandle) Provider() RemoteProvider { return SandboxTypeCube }

func (h *cubeRemoteHandle) Metadata() map[string]string {
	if h == nil || len(h.metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(h.metadata))
	for k, v := range h.metadata {
		out[k] = v
	}
	return out
}

func (h *cubeRemoteHandle) TrafficAccessToken() string {
	if h == nil {
		return ""
	}
	return h.token
}

func (c *CubeRemoteClient) Provider() RemoteProvider { return SandboxTypeCube }

// Capabilities advertises nothing: without a runtime the client can honour no
// optional capability.
func (c *CubeRemoteClient) Capabilities() RemoteSandboxCapabilities {
	return RemoteSandboxCapabilities{}
}

func (c *CubeRemoteClient) Health(_ context.Context) error {
	return cubeUnavailable("Health")
}

func (c *CubeRemoteClient) ListTemplates(_ context.Context) ([]RemoteTemplate, error) {
	return nil, cubeUnavailable("ListTemplates")
}

func (c *CubeRemoteClient) EnsureStandardTemplate(_ context.Context) (*RemoteTemplate, error) {
	return nil, cubeUnavailable("EnsureStandardTemplate")
}

func (c *CubeRemoteClient) ReplaceStandardTemplate(_ context.Context) (*RemoteTemplate, error) {
	return nil, cubeUnavailable("ReplaceStandardTemplate")
}

func (c *CubeRemoteClient) EnsureDesktopTemplate(_ context.Context) (*RemoteTemplate, error) {
	return nil, cubeUnavailable("EnsureDesktopTemplate")
}

func (c *CubeRemoteClient) ReplaceDesktopTemplate(_ context.Context) (*RemoteTemplate, error) {
	return nil, cubeUnavailable("ReplaceDesktopTemplate")
}

func (c *CubeRemoteClient) DeleteSupersededDesktopTemplates(_ context.Context, _ string) error {
	return cubeUnavailable("DeleteSupersededDesktopTemplates")
}

func (c *CubeRemoteClient) DeleteSupersededStandardTemplates(_ context.Context, _ string) error {
	return cubeUnavailable("DeleteSupersededStandardTemplates")
}

func (c *CubeRemoteClient) Create(
	_ context.Context,
	_ RemoteCreateRequest,
) (RemoteSandboxHandle, error) {
	return nil, cubeUnavailable("Create")
}

func (c *CubeRemoteClient) Connect(
	_ context.Context,
	_ RemoteConnectRequest,
) (RemoteSandboxHandle, error) {
	return nil, cubeUnavailable("Connect")
}

func (c *CubeRemoteClient) ConnectSession(
	_ context.Context,
	_ RemoteConnectRequest,
) (RemoteSandboxHandle, error) {
	return nil, cubeUnavailable("ConnectSession")
}

func (c *CubeRemoteClient) Get(
	_ context.Context,
	_ string,
) (*RemoteSandboxSummary, error) {
	return nil, cubeUnavailable("Get")
}

func (c *CubeRemoteClient) List(
	_ context.Context,
	_ RemoteListFilter,
) ([]RemoteSandboxSummary, error) {
	return nil, cubeUnavailable("List")
}

func (c *CubeRemoteClient) Delete(_ context.Context, _ string) error {
	return cubeUnavailable("Delete")
}

func (c *CubeRemoteClient) Exec(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ RemoteExecRequest,
) (*RemoteExecResult, error) {
	return nil, cubeUnavailable("Exec")
}

func (c *CubeRemoteClient) WriteFile(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ string,
	_ []byte,
) error {
	return cubeUnavailable("WriteFile")
}

func (c *CubeRemoteClient) ReadFile(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ string,
) ([]byte, error) {
	return nil, cubeUnavailable("ReadFile")
}

func (c *CubeRemoteClient) ListDir(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ string,
) ([]RemoteDirEntry, error) {
	return nil, cubeUnavailable("ListDir")
}

func (c *CubeRemoteClient) MakeDir(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ string,
) error {
	return cubeUnavailable("MakeDir")
}

func (c *CubeRemoteClient) Remove(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ string,
) error {
	return cubeUnavailable("Remove")
}

func (c *CubeRemoteClient) Stat(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ string,
) (*RemoteStatEntry, error) {
	return nil, cubeUnavailable("Stat")
}

func (c *CubeRemoteClient) CreateSnapshot(
	_ context.Context,
	_ string,
	_ string,
) (RemoteSnapshotRef, error) {
	return RemoteSnapshotRef{}, cubeUnavailable("CreateSnapshot")
}

func (c *CubeRemoteClient) DeleteSnapshot(_ context.Context, _ string) error {
	return cubeUnavailable("DeleteSnapshot")
}

func (c *CubeRemoteClient) ListSnapshots(
	_ context.Context,
	_ string,
) ([]RemoteSnapshotRef, error) {
	return nil, cubeUnavailable("ListSnapshots")
}

func (c *CubeRemoteClient) DialDesktop(
	_ context.Context,
	_ RemoteSandboxHandle,
	_ RemoteDesktopOptions,
) (*websocket.Conn, error) {
	return nil, cubeUnavailable("DialDesktop")
}

// StartDesktopTTLRefresh is a no-op: there is no sandbox whose timeout could
// be extended.
func (c *CubeRemoteClient) StartDesktopTTLRefresh(_ context.Context, _ RemoteSandboxHandle) {}

// StateMatches reports whether candidate is in allowed, treating an empty
// allow-list as "any state". It is provider-neutral and shared with the E2B
// adapter.
func StateMatches(candidate RemoteSandboxState, allowed []RemoteSandboxState) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, state := range allowed {
		if candidate == state {
			return true
		}
	}
	return false
}

// parseProxyURL splits a gateway URL into host, port and scheme, defaulting
// the port from the scheme when the URL carries none. It is provider-neutral
// and shared with the gateway transport pool and the websocket dialer.
func parseProxyURL(raw string) (host string, port int, scheme string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", 0, "", false
	}
	scheme = strings.ToLower(parsed.Scheme)
	if scheme == "" {
		scheme = "http"
	}
	h, p, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		h = parsed.Host
		if scheme == "https" {
			p = "443"
		} else {
			p = "80"
		}
	}
	portInt, err := strconv.Atoi(p)
	if err != nil || portInt <= 0 {
		return "", 0, "", false
	}
	return h, portInt, scheme, true
}

// buildShellLine turns argv into a single shell-safe command line, relying on
// the sandbox image's shell to resolve the command against $PATH. It is
// provider-neutral and shared with the E2B adapter.
func buildShellLine(cmd string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, ShellQuote(cmd))
	for _, a := range args {
		parts = append(parts, ShellQuote(a))
	}
	return strings.Join(parts, " ")
}

// wrapWithStdin funnels a caller-supplied stdin payload into the child process
// by prepending a heredoc, for SDKs whose Run contract takes no explicit stdin
// argument. It is provider-neutral and shared with the E2B adapter.
func wrapWithStdin(line, stdin string) string {
	// Use a heredoc delimiter unlikely to appear in caller data.
	const delim = "ENTERPRISERAG_STDIN_EOF"
	// Escape lines containing the delimiter defensively.
	safe := strings.ReplaceAll(stdin, delim, "")
	return "cat <<'" + delim + "' | " + line + "\n" + safe + "\n" + delim
}

var (
	_ RemoteSandboxClient          = (*CubeRemoteClient)(nil)
	_ RemoteSnapshotManager        = (*CubeRemoteClient)(nil)
	_ RemoteTemplateCatalog        = (*CubeRemoteClient)(nil)
	_ RemoteDesktopTemplateCatalog = (*CubeRemoteClient)(nil)
	_ RemoteDesktopManager         = (*CubeRemoteClient)(nil)
	_ RemoteDesktopTTLRefresher    = (*CubeRemoteClient)(nil)
	_ RemoteSandboxHandle          = (*cubeRemoteHandle)(nil)
	_ RemoteInboundTokenCarrier    = (*cubeRemoteHandle)(nil)
)
