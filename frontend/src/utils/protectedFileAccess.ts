/**
 * Access context for protected files (provider:// / resource:// and friends).
 *
 * The backend splits the file proxy by calling subject, and each one has a different
 * authorisation model:
 *   - `/files`                                    -> logged-in Bearer + X-Tenant-ID
 *   - `/api/v1/knowledge-bases/:id/files`         -> knowledge-base access (cross-tenant shared KBs)
 *   - `/api/v1/sessions/:id/messages/:mid/files`  -> session message ownership + shared agent rights
 *   - `/api/v1/embed/:channel_id/files`           -> the embed visitor's embed token
 *
 * Which proxy to use depends on the authentication plane of the current request, not on
 * which component happens to be rendering the image. This module is the single source of
 * truth for that decision, so rendering components only declare a scope instead of each
 * assembling their own URL.
 */
export const PROVIDER_SCHEME_PATTERN = 'resource|local|minio|cos|tos|s3|oss|ks3|obs';

const PROVIDER_FILE_SCHEME_RE = new RegExp(`^(${PROVIDER_SCHEME_PATTERN}):\\/\\/\\S+$`, 'i');
const STORAGE_BACKEND_FILE_SCHEME_RE = new RegExp(
  `^storage:\\/\\/[0-9A-Za-z_-]+\\/(${PROVIDER_SCHEME_PATTERN}):\\/\\/\\S+$`,
  'i',
);

const KB_FILE_PROXY_PATH_RE = /^\/api\/v1\/knowledge-bases\/[^/]+\/files$/;
const EMBED_FILE_PROXY_PATH_RE = /^\/api\/v1\/embed\/[^/]+\/files$/;
const MESSAGE_FILE_PROXY_PATH_RE = /^\/api\/v1\/sessions\/[^/]+\/messages\/[^/]+\/files$/;

export type ProtectedFileAccessContext =
  | { mode: 'tenant' }
  | { mode: 'embed'; channelId: string; token: string }
  | { mode: 'knowledgeBase'; kbId: string }
  | { mode: 'message'; sessionId: string; messageId: string };

export interface ProtectedFileRequest {
  url: string;
  headers: Record<string, string>;
}

const TENANT_ACCESS: ProtectedFileAccessContext = { mode: 'tenant' };

interface ProtectedFileAccessState {
  current: ProtectedFileAccessContext;
}

const accessState: ProtectedFileAccessState = (() => {
  const fresh = (): ProtectedFileAccessState => ({ current: TENANT_ACCESS });
  if (typeof window === 'undefined') return fresh();
  const scope = window as typeof window & {
    __enterpriseragProtectedFileAccessV1__?: ProtectedFileAccessState;
  };
  scope.__enterpriseragProtectedFileAccessV1__ ||= fresh();
  return scope.__enterpriseragProtectedFileAccessV1__;
})();

/**
 * Registers the default access context for this document. Called once by the application
 * entry point (the embed app registers it after it has the channelId/token); from then on
 * every protected-file request automatically uses the matching proxy.
 */
export function setDefaultProtectedFileAccess(
  access: ProtectedFileAccessContext | null,
): void {
  accessState.current = access ?? TENANT_ACCESS;
}

export function getDefaultProtectedFileAccess(): ProtectedFileAccessContext {
  return accessState.current;
}

/**
 * Merges the default context with the scope a component passes in.
 *
 * The default context carries the authentication plane (embed visitor vs logged-in user);
 * a component-level override may only refine the scope within that same plane. An embed
 * visitor has no Bearer, so letting a `knowledgeBase` override replace the embed plane
 * would send the request to a proxy that needs a login and get a 401.
 */
export function resolveProtectedFileAccess(
  override?: ProtectedFileAccessContext | null,
): ProtectedFileAccessContext {
  const fallback = accessState.current;
  if (fallback.mode === 'embed') return fallback;
  if (!override) return fallback;
  if (override.mode === 'knowledgeBase' && !override.kbId.trim()) return fallback;
  if (override.mode === 'message' && (!override.sessionId.trim() || !override.messageId.trim())) {
    return fallback;
  }
  return override;
}

export function isProviderFileURL(url: string): boolean {
  const trimmed = url.trim();
  return PROVIDER_FILE_SCHEME_RE.test(trimmed) || STORAGE_BACKEND_FILE_SCHEME_RE.test(trimmed);
}

export function isProtectedFileProxyPath(pathname: string): boolean {
  return (
    pathname === '/files'
    || KB_FILE_PROXY_PATH_RE.test(pathname)
    || MESSAGE_FILE_PROXY_PATH_RE.test(pathname)
    || EMBED_FILE_PROXY_PATH_RE.test(pathname)
  );
}

function tenantRequestHeaders(): Record<string, string> {
  const headers: Record<string, string> = {};
  try {
    const token = (localStorage.getItem('enterpriserag_token') || '').trim();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const selectedTenantId = (localStorage.getItem('enterpriserag_selected_tenant_id') || '').trim();
    if (selectedTenantId) {
      // Always attach when a selected tenant is set. Same rationale as
      // utils/request.ts / api/chat/streame.ts: the
      // "selectedTenantId === defaultTenantId → skip" short-circuit
      // silently drops the header whenever any code path writes the
      // active tenant into enterpriserag_tenant, leaving authenticated file
      // fetches landing on the home tenant.
      headers['X-Tenant-ID'] = selectedTenantId;
    }
  } catch {
    // ignore localStorage read errors
  }
  return headers;
}

/**
 * Builds the proxy request for a storage path. null means the current context cannot make
 * that request (not a storage path, or an embed context that has no token yet) and the
 * caller should skip it and retry later.
 */
export function buildProtectedFileRequest(
  sourceURL: string,
  access: ProtectedFileAccessContext,
): ProtectedFileRequest | null {
  const filePath = sourceURL.trim();
  if (!isProviderFileURL(filePath)) return null;

  const query = new URLSearchParams({ file_path: filePath }).toString();

  if (access.mode === 'embed') {
    const channelId = access.channelId.trim();
    const token = access.token.trim();
    if (!channelId || !token) return null;
    return {
      url: `/api/v1/embed/${encodeURIComponent(channelId)}/files?${query}`,
      headers: { Authorization: `Embed ${token}` },
    };
  }

  if (access.mode === 'knowledgeBase') {
    return {
      url: `/api/v1/knowledge-bases/${encodeURIComponent(access.kbId.trim())}/files?${query}`,
      headers: tenantRequestHeaders(),
    };
  }

  if (access.mode === 'message') {
    return {
      url: `/api/v1/sessions/${encodeURIComponent(access.sessionId.trim())}/messages/${encodeURIComponent(access.messageId.trim())}/files?${query}`,
      headers: tenantRequestHeaders(),
    };
  }

  return { url: `/files?${query}`, headers: tenantRequestHeaders() };
}
