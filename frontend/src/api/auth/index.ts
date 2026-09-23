import { post, get, put } from '@/utils/request'

const t = (key: string) => key

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  success: boolean
  message?: string
  user?: {
    id: string
    username: string
    email: string
    avatar?: string
    tenant_id: number
    can_access_all_tenants?: boolean
    is_system_admin?: boolean
    is_active: boolean
    created_at: string
    updated_at: string
  }
  tenant?: {
    id: number
    name: string
    description: string
    status: string
    business: string
    storage_quota: number
    storage_used: number
    created_at: string
    updated_at: string
  } | null
  // active_tenant mirrors `tenant` for endpoints that distinguish home
  // tenant from current tenant (e.g. /auth/register-by-invite). Only
  // one of `tenant` / `active_tenant` is populated by any given endpoint.
  active_tenant?: {
    id: number
    name: string
    description?: string
    status?: string
    business?: string
    storage_quota?: number
    storage_used?: number
    created_at?: string
    updated_at?: string
  } | null
  memberships?: MembershipInfo[]
  token?: string
  refresh_token?: string
}

export interface OIDCAuthURLResponse {
  success: boolean
  authorization_url?: string
  state?: string
  message?: string
}

export interface OIDCConfigResponse {
  success: boolean
  enabled: boolean
  provider_display_name?: string
  message?: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface RegisterResponse {
  success: boolean
  message?: string
  data?: {
    user: {
      id: string
      username: string
      email: string
    }
    tenant: {
      id: string
      name: string
    }
  }
}

// User preferences (mirrors the backend types.UserPreferences; an optional field means
// it was never explicitly set). When adding a key, remember the backend's
// service.UpdateUserPreferences also needs a merge branch for it; frontend callers read
// it as needed and fall back to a default.
export interface UserPreferences {
  browser_search_instructions?: string | null
  // last_active_tenant_id persists the "come back to the space I was last in after a
  // refresh / new device / re-login" preference. The backend only honours it on Login /
  // RefreshToken once it has verified the membership is still valid; otherwise it falls
  // back to home and clears the field. PATCH with 0 to clear the preference.
  last_active_tenant_id?: number | null
  oidc_only_login?: boolean
}

export interface UserInfo {
  id: string
  username: string
  email: string
  avatar?: string
  tenant_id: string
  can_access_all_tenants?: boolean
  preferences?: UserPreferences
  is_system_admin?: boolean
  created_at: string
  updated_at: string
}

/**
 * Normalises the user JSON the backend returns into the frontend UserInfo.
 *
 * Historically there were four independent setUser call sites (Login, autoSetup, token
 * rehydrate, an explicit /auth/me refresh), each with its own hand-written field
 * allowlist, so every new user field had to be added in four places - otherwise it was
 * silently filtered out. When is_system_admin shipped, one missed copy meant the
 * "System administration" entry never appeared. This factory exists so that class of
 * omission cannot happen again. **Add new user fields here only.**
 *
 * fallbackTenantId is the fallback source when tenant_id is missing:
 *   - the autoSetup response has tenant.id at the top level but no tenant_id on the user
 *   - /auth/me occasionally returns only user, with no tenant
 * Callers pass it when they have it; otherwise it stays an empty string (matching the
 * historical behaviour).
 *
 * Fields are read with `=== true` rather than `|| false`, to strictly narrow the
 * occasional non-boolean value (should the backend ever send 1/0 or a string) instead of
 * treating a truthy string as a granted permission.
 */
export function userInfoFromApi(
  u: any,
  fallbackTenantId?: string | number | null,
): UserInfo {
  const rawTenantId =
    u?.tenant_id !== undefined && u?.tenant_id !== null && u.tenant_id !== ''
      ? u.tenant_id
      : fallbackTenantId ?? ''
  const tid = Number(rawTenantId) > 0 ? rawTenantId : ''
  return {
    id: u?.id || '',
    username: u?.username || '',
    email: u?.email || '',
    avatar: u?.avatar,
    tenant_id: String(tid) || '',
    can_access_all_tenants: u?.can_access_all_tenants === true,
    is_system_admin: u?.is_system_admin === true,
    preferences: u?.preferences,
    created_at: u?.created_at || new Date().toISOString(),
    updated_at: u?.updated_at || new Date().toISOString(),
  }
}

export interface TenantInfo {
  id: string
  name: string
  description?: string
  status?: string
  business?: string
  owner_id: string
  storage_quota?: number
  storage_used?: number
  created_at: string
  updated_at: string
  knowledge_bases?: KnowledgeBaseInfo[]
}

export interface KnowledgeBaseInfo {
  id: string
  name: string
  description: string
  tenant_id: string
  // creator_id is the user id of whoever originally created the KB.
  // Set by PR 5 of the multi-tenant RBAC series; nullable for legacy
  // KBs created before that migration backfilled the column.
  creator_id?: string
  creator_name?: string
  created_at: string
  updated_at: string
  document_count?: number
  chunk_count?: number
}

export interface ModelInfo {
  id: string
  name: string
  type: string
  source: string
  description?: string
  is_default?: boolean
  created_at: string
  updated_at: string
}

export async function login(data: LoginRequest): Promise<LoginResponse> {
  try {
    const response = await post('/api/v1/auth/login', data)
    return response as unknown as LoginResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Login failed'
    }
  }
}

export async function getOIDCAuthorizationURL(redirectURI: string): Promise<OIDCAuthURLResponse> {
  try {
    const response = await get(`/api/v1/auth/oidc/url?redirect_uri=${encodeURIComponent(redirectURI)}`)
    return response as unknown as OIDCAuthURLResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Login failed'
    }
  }
}

export async function getOIDCConfig(): Promise<OIDCConfigResponse> {
  try {
    const response = await get('/api/v1/auth/oidc/config')
    return response as unknown as OIDCConfigResponse
  } catch (error: any) {
    return {
      success: false,
      enabled: false,
      message: error.message || 'Login failed'
    }
  }
}

/**
 * Fetches the auth config (only the public fields the frontend needs to render, such as
 * the registration mode).
 *
 * The backend controls whether self-service sign-up is allowed via
 * `auth.registration_mode`:
 *   - "self_serve"  keeps the existing self-service sign-up entry (default)
 *   - "invite_only" closes sign-up and requires an admin invitation
 *
 * On failure this falls back to self_serve, so an API error does not make the sign-up
 * entry disappear entirely.
 */
export interface AuthConfigResponse {
  success: boolean
  registration_mode: 'self_serve' | 'invite_only' | string
  complex_password_enabled: boolean
}

export async function getAuthConfig(): Promise<AuthConfigResponse> {
  try {
    const response = await get('/api/v1/auth/config')
    return response as unknown as AuthConfigResponse
  } catch {
    return { success: false, registration_mode: 'self_serve', complex_password_enabled: false }
  }
}

export async function register(data: RegisterRequest): Promise<RegisterResponse> {
  try {
    const response = await post('/api/v1/auth/register', data)
    return response as unknown as RegisterResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Registration failed'
    }
  }
}

export async function autoSetup(): Promise<LoginResponse> {
  try {
    const nativeApp = (window as any).go?.main?.App
    if (!nativeApp?.GetAutoSetupToken) return { success: false, message: 'Desktop authentication required' }
    const token = await nativeApp.GetAutoSetupToken()
    const response = await post('/api/v1/auth/auto-setup', {}, {
      headers: { 'X-EnterpriseRag-Desktop-Token': token },
    })
    return response as unknown as LoginResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Auto-setup unavailable'
    }
  }
}

/**
 * Membership row returned alongside /auth/me. Mirrors the LoginResponse
 * shape so the frontend can refresh `currentTenantRole` on every page
 * load — without it, role changes after login (e.g. an Owner demoting
 * us in a peer tenant) stay invisible until the user logs out and back
 * in.
 */
export interface MembershipInfo {
  tenant_id: number
  tenant_name?: string
  role: string
}

export interface AuthCapabilities {
  can_create_tenant: boolean
  auto_accept_invitation: boolean
}

export async function getCurrentUser(): Promise<{ success: boolean; data?: { user: UserInfo; tenant?: TenantInfo | null; memberships?: MembershipInfo[]; tenant_required?: boolean; capabilities?: AuthCapabilities; preference_defaults?: { browser_search_instructions: string } }; message?: string }> {
  try {
    const response = await get('/api/v1/auth/me')
    return response as unknown as { success: boolean; data?: { user: UserInfo; tenant?: TenantInfo | null; memberships?: MembershipInfo[]; tenant_required?: boolean; capabilities?: AuthCapabilities; preference_defaults?: { browser_search_instructions: string } }; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Failed to get user info'
    }
  }
}

/**
 * Updates the current user's preferences (PATCH semantics: send only the keys you want to
 * change, and the backend overwrites only the keys that were sent, leaving the rest
 * untouched). The backend returns the full updated preferences object.
 */
export async function updateMyPreferences(
  patch: Partial<UserPreferences>,
): Promise<{ success: boolean; data?: UserPreferences; message?: string }> {
  try {
    const response = await put('/api/v1/auth/me/preferences', patch)
    return response as unknown as { success: boolean; data?: UserPreferences; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Failed to update preferences',
    }
  }
}

export async function getCurrentTenant(): Promise<{ success: boolean; data?: TenantInfo; message?: string }> {
  try {
    const response = await get('/api/v1/auth/tenant')
    return response as unknown as { success: boolean; data?: TenantInfo; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Failed to get workspace info'
    }
  }
}

export async function refreshToken(refreshToken: string): Promise<{ success: boolean; data?: { token: string; refreshToken: string }; message?: string }> {
  try {
    const response: any = await post('/api/v1/auth/refresh', { refreshToken })
    if (response && response.success) {
      if (response.access_token || response.refresh_token) {
        return {
          success: true,
          data: {
            token: response.access_token,
            refreshToken: response.refresh_token,
          }
        }
      }
    }

    return {
      success: false,
      message: response?.message || 'Token refresh failed'
    }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Token refresh failed'
    }
  }
}

export async function logout(): Promise<{ success: boolean; message?: string }> {
  try {
    await post('/api/v1/auth/logout', {})
    return {
      success: true
    }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || 'Logout failed'
    }
  }
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

/** Map change-password API failures to localized UI strings. */
export function resolveChangePasswordError(error: any): string {
  const details =
    typeof error?.error?.details === 'string'
      ? error.error.details
      : typeof error?.details === 'string'
        ? error.details
        : ''
  switch (details) {
    case 'invalid_old_password':
      return 'Failed to change password. Check that your current password is correct.'
    case 'password_policy':
      return 'New password must be 8-32 characters and include letters and numbers'
    case 'same_password':
      return 'New password must differ from your current password'
    default:
      return error?.message || 'Failed to change password. Check that your current password is correct.'
  }
}

/**
 * Self-service password rotation. On success the backend revokes every
 * outstanding session for the caller, so the client should clear local
 * auth state and send the user back to /login.
 */
export async function changePassword(
  data: ChangePasswordRequest,
): Promise<{ success: boolean; message?: string }> {
  try {
    const response = await post('/api/v1/auth/change-password', data)
    return response as unknown as { success: boolean; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: resolveChangePasswordError(error),
    }
  }
}

export async function validateToken(): Promise<{ success: boolean; valid?: boolean; message?: string }> {
  try {
    const response = await get('/api/v1/auth/validate')
    return response as unknown as { success: boolean; valid?: boolean; message?: string }
  } catch (error: any) {
    return {
      success: false,
      valid: false,
      message: error.message || 'Token validation failed'
    }
  }
}





// ---- share-link registration --------------------------------------------

// InviteLookup is the public projection of a share-link row used by
// /register?token=xxx — enough to render the registration page header
// ("X invited you to Y") without leaking sensitive inviter fields.
export interface InviteLookup {
  tenant_id: number
  tenant_name?: string
  role: string
  expires_at: string
}

export interface InviteLookupResponse {
  success: boolean
  data?: InviteLookup
  message?: string
}

export interface RegisterByInviteRequest {
  token: string
  email: string
  username: string
  password: string
}

/**
 * Resolve a share-link token (no auth) into the context the
 * registration page needs (tenant name, role, expiry). Returns 410
 * when the link is invalid / revoked / expired.
 *
 * Uses POST + body (rather than GET + path) so the plaintext token
 * never appears in access logs, browser history, or tracing spans.
 */
export async function getInvitationByToken(token: string): Promise<InviteLookupResponse> {
  try {
    const response = await post(`/api/v1/auth/invitations/lookup`, { token })
    return response as unknown as InviteLookupResponse
  } catch (error: any) {
    return { success: false, message: error.message || '' }
  }
}

/**
 * Complete registration via a share-link token. The invitee supplies
 * their own email — the token is the authorisation, not an identity
 * lock.
 */
export async function registerByInvite(data: RegisterByInviteRequest): Promise<LoginResponse> {
  try {
    const response = await post('/api/v1/auth/register-by-invite', data)
    return response as unknown as LoginResponse
  } catch (error: any) {
    return { success: false, message: error.message || 'Registration failed' }
  }
}
