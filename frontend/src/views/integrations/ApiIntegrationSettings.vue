<template>
  <div class="api-integration">
    <div v-if="loading" class="state-row">
      <t-loading size="small" />
      <span>{{ 'Loading API integration settings...' }}</span>
    </div>

    <t-alert v-else-if="error" theme="error" :message="error">
      <template #operation>
        <t-button size="small" @click="load">{{ 'Retry' }}</t-button>
      </template>
    </t-alert>

    <div v-else class="api-settings">
      <section class="settings-band">
        <div class="row">
          <div class="row-info">
            <label>{{ 'API Base URL' }}</label>
            <p>{{ 'Use this base URL with REST API paths.' }}</p>
          </div>
          <div class="row-control copy-field">
            <t-input :model-value="apiBaseUrl" readonly class="mono-input" />
            <t-button variant="text" :title="'Copy'" @click="copy(apiBaseUrl)">
              <t-icon name="file-copy" />
            </t-button>
          </div>
        </div>

        <template v-if="showDesktopPortSetting || showDesktopBindPublicSetting">
          <div v-if="showDesktopPortSetting" class="row">
            <div class="row-info">
              <label>{{ 'Local API port (desktop)' }}</label>
              <p>{{ 'Use a fixed port (e.g. 37841) so tools like the Chrome extension can keep the same API URL. 0 means a random port each launch. Restart the app after saving.' }}</p>
            </div>
            <div class="row-control desktop-api-control">
              <div class="desktop-port-input-wrap">
                <t-input-number
                  v-model="desktopPortInput"
                  :min="0"
                  :max="65535"
                  theme="normal"
                />
              </div>
              <t-button size="small" variant="text" @click="saveDesktopPort">
                {{ 'Save' }}
              </t-button>
            </div>
          </div>

          <div v-if="showDesktopBindPublicSetting" class="row">
            <div class="row-info">
              <label>{{ 'Allow LAN access to the API' }}</label>
              <p>{{ 'When enabled, the embedded server listens on 0.0.0.0 so other devices on your LAN can reach the API via the address below. This increases exposure—use host firewalls and routing rules as needed. Restart the app after changing this.' }}</p>
            </div>
            <div class="row-control desktop-bind-public-control">
              <t-switch v-model="desktopBindPublicInput" @change="onDesktopBindPublicChange" />
            </div>
          </div>

          <div v-if="wailsApiLanBaseURL" class="row">
            <div class="row-info">
              <label>{{ 'LAN API base URL' }}</label>
              <p>{{ 'Shown after “Allow LAN access” is on and the app has restarted. If the IP is wrong, replace the host with your machine’s IPv4 from system network settings.' }}</p>
            </div>
            <div class="row-control copy-field">
              <t-input :model-value="wailsApiLanBaseURL" readonly class="mono-input" />
              <t-button variant="text" :title="'Copy LAN API URL'" @click="copy(wailsApiLanBaseURL)">
                <t-icon name="file-copy" />
              </t-button>
            </div>
          </div>

          <div v-if="showLanUrlUnavailableHint" class="row row--single">
            <t-alert theme="warning" :message="'The API is listening on all interfaces, but no LAN IPv4 could be detected automatically. Find your IPv4 in network settings and build http://YOUR_IP:PORT/api/v1 manually.'" />
          </div>
        </template>

        <div class="row row--doc">
          <div class="row-info">
            <label>{{ 'API Documentation' }}</label>
            <p>
              {{ 'View complete API documentation and examples,' }}
              <a class="doc-link" @click="openApiDoc">
                {{ 'Open documentation' }}
                <t-icon name="link" class="link-icon" />
              </a>
            </p>
          </div>
        </div>

        <div class="api-key-section">
          <div class="api-key-section__header">
            <div class="api-key-section__title">
              <label>{{ 'API Keys' }}</label>
              <p>{{ 'Create separate keys per integration and restrict operation permissions plus knowledge-base access.' }}</p>
            </div>
            <t-button size="small" variant="outline" @click="openCreateAPIKeyDialog">
              <template #icon><t-icon name="add" /></template>
              {{ 'Create API Key' }}
            </t-button>
          </div>
          <div class="api-key-section__body">
            <div class="api-key-list" :class="{ 'api-key-list--loading': apiKeysLoading }">
              <div v-if="apiKeysLoading" class="api-key-list__empty">
                <t-loading size="small" />
                <span>{{ 'Loading API integration settings...' }}</span>
              </div>
              <div v-else-if="apiKeys.length === 0" class="api-key-list__empty">
                {{ 'No API keys' }}
              </div>
              <div v-else class="api-key-table-wrap">
                <table class="api-key-table">
                  <thead>
                    <tr>
                      <th>{{ 'Name' }}</th>
                      <th>{{ 'API Key' }}</th>
                      <th>{{ 'Access mode' }}</th>
                      <th>{{ 'Knowledge bases' }}</th>
                      <th>{{ 'Created' }}</th>
                      <th class="api-key-table__actions-heading">{{ 'Actions' }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="key in apiKeys" :key="key.id">
                      <td>
                        <span class="api-key-name">{{ key.name }}</span>
                      </td>
                      <td>
                        <code class="api-key-fingerprint">{{ formatKeyMaskedValue(key) }}</code>
                      </td>
                      <td>
                        <div class="api-key-access-cell" :title="formatApiKeyCapabilitiesTitle(key)">
                          <span
                            v-if="key.full_access"
                            class="api-key-access-mode api-key-access-mode--full"
                          >
                            {{ formatApiKeyAccessModeLabel(key) }}
                          </span>
                          <div v-if="!key.full_access" class="api-key-capability-chips">
                            <span
                              v-for="label in keyCapabilityLabels(key)"
                              :key="label"
                              class="api-key-capability-chip"
                            >
                              {{ label }}
                            </span>
                          </div>
                        </div>
                      </td>
                      <td>
                        <span class="api-key-knowledge-scope">
                          {{ formatKeyKnowledgeScope(key.knowledge_base_ids) }}
                        </span>
                      </td>
                      <td>
                        <span class="api-key-created-at">{{ formatDate(key.created_at) }}</span>
                      </td>
                      <td>
                        <div class="api-key-table__actions">
                          <t-button
                            shape="square"
                            variant="text"
                            :title="'Edit API key'"
                            @click="openEditAPIKeyScope(key)"
                          >
                            <t-icon name="edit-1" />
                          </t-button>
                          <t-button
                            shape="square"
                            variant="text"
                            :title="'Copy'"
                            @click="copy(key.api_key)"
                          >
                            <t-icon name="file-copy" />
                          </t-button>
                          <t-button
                            shape="square"
                            variant="text"
                            theme="danger"
                            :title="'Delete'"
                            @click="confirmDeleteAPIKey(key.id)"
                          >
                            <t-icon name="delete" />
                          </t-button>
                        </div>
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="settings-band principal-section">
        <div class="principal-section__header">
          <label>{{ 'User identity mode' }}</label>
          <p>{{ 'Choose how API requests identify the end user. This identity scopes both conversation sessions and MCP tool authorization per user.' }}</p>
          <p class="principal-section__scope">{{ 'End-user identity isolates sessions and MCP OAuth. Operation permissions and knowledge-base scope are controlled by API Keys above.' }}</p>
        </div>

        <t-radio-group v-model="form.mode" class="mode-radio" @change="handlePrincipalModeChange">
          <t-radio-button value="tenant">{{ 'Workspace only' }}</t-radio-button>
          <t-radio-button value="direct_header">{{ 'Direct user ID' }}</t-radio-button>
          <t-radio-button value="signed_token">{{ 'Signed token' }}</t-radio-button>
        </t-radio-group>

        <div v-if="form.mode !== 'tenant'" class="mode-detail">
          <div
            v-if="form.mode === 'direct_header'"
            class="mode-callout mode-callout--warning"
          >
            <div class="mode-callout__body">
              <strong>{{ 'Direct user ID trusts the caller-supplied header. Use only for trusted server-to-server calls.' }}</strong>
              <p>{{ 'Anyone with the API key can change the user ID header to impersonate another external user and reuse or hijack that user\'s MCP OAuth authorization. Do not use this from browsers or untrusted clients; use Signed token for end-user apps.' }}</p>
            </div>
          </div>
          <div
            v-else-if="form.mode === 'signed_token'"
            class="mode-callout"
          >
            <div class="mode-callout__body">
              <strong>{{ 'Recommended for user-facing apps: your backend signs a short-lived HS256 JWT for the external user.' }}</strong>
              <p>{{ 'Keep the HMAC secret only on sustainability.ai and your trusted backend—never put it in requests or ship it to browsers. The request header carries the JWT string signed with that secret (it changes on each issuance or after expiry), not the secret itself. JWT must include sub (external user ID), tenant_id, aud=enterpriserag, and exp (max 24h lifetime).' }}</p>
            </div>
          </div>

          <div v-if="form.mode === 'direct_header'" class="principal-config">
            <div class="config-row">
              <div class="config-row__text">
                <label>{{ 'User ID header' }}</label>
              </div>
              <code class="fixed-header-name">{{ directHeaderName }}</code>
            </div>
            <div class="config-row config-row--switch">
              <div class="config-row__text">
                <label>{{ 'Require user ID header' }}</label>
                <p>{{ 'When enabled, API Key requests without the user ID header are rejected; when disabled, all requests are treated as the whole workspace with no per-user distinction.' }}</p>
              </div>
              <div class="config-row__action">
                <t-switch v-model="form.require_direct_header" size="small" @change="handleRequireDirectHeaderChange" />
              </div>
            </div>
          </div>

          <div v-else-if="form.mode === 'signed_token'" class="principal-config">
            <div class="config-row">
              <div class="config-row__text">
                <label>{{ 'Token header' }}</label>
                <p>{{ 'Clients send the backend-signed JWT in this fixed header.' }}</p>
              </div>
              <code class="fixed-header-name">{{ tokenHeaderName }}</code>
            </div>
            <div class="config-row config-row--secret">
              <div class="config-row__text">
                <label>{{ 'HMAC secret' }}</label>
                <p>{{ 'Same secret stored in sustainability.ai; used only on your backend to sign JWTs—never send it as a request header.' }}</p>
              </div>
              <div class="secret-field">
                <div class="secret-control">
                  <t-input
                    v-model="secretInput"
                    :type="secretInputType"
                    class="mono-input secret-mono-input"
                    :placeholder="config?.has_hmac_secret && !secretInput.trim() ? 'Secret configured (not shown again); enter a new value to rotate' : ''"
                    @blur="triggerAutoSave"
                  />
                  <t-button
                    v-if="secretInput.trim()"
                    size="small"
                    variant="text"
                    @click="showHMACSecret = !showHMACSecret"
                  >
                    <t-icon :name="showHMACSecret ? 'browse-off' : 'browse'" />
                  </t-button>
                  <t-button
                    v-if="secretInput.trim()"
                    size="small"
                    variant="text"
                    :title="'Copy'"
                    @click="copy(secretInput)"
                  >
                    <t-icon name="file-copy" />
                  </t-button>
                  <t-button
                    size="small"
                    variant="text"
                    theme="danger"
                    :title="'Generate secret'"
                    :loading="saving"
                    @click="confirmGenerateSecret"
                  >
                    <t-icon name="refresh" />
                  </t-button>
                </div>
                <p v-if="showSecretSavedHint" class="secret-saved-hint">
                  {{ 'Secret saved. Copy it to your backend now — you will not be able to view it again after leaving this page.' }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div class="examples">
          <t-tabs
            v-if="form.mode === 'signed_token'"
            v-model="exampleTab"
            class="snippet-tabs"
          >
            <t-tab-panel value="jwt" :label="'Backend JWT signing example (Go)'" />
            <t-tab-panel value="curl" :label="'Request example'" />
          </t-tabs>
          <div class="code-panel">
            <div class="code-panel__toolbar">
              <span class="code-panel__label">{{ activeExampleLabel }}</span>
              <t-button size="small" variant="text" class="code-panel__copy" @click="copy(activeExampleText)">
                <template #icon><t-icon name="file-copy" /></template>
                {{ 'Copy' }}
              </t-button>
            </div>
            <pre class="code-panel__pre">{{ activeExampleText }}</pre>
          </div>
        </div>

        <div class="playground-entry">
          <div class="playground-entry__info">
            <label>{{ 'API Playground' }}</label>
            <p>{{ 'Send real requests with the current API key and identity mode to verify session creation, agent chat, and SSE output.' }}</p>
          </div>
          <t-button variant="outline" @click="openPlaygroundDrawer">
            <template #icon><t-icon name="code" /></template>
            {{ 'Open Playground' }}
          </t-button>
        </div>
      </section>
    </div>

    <SettingDrawer
      v-model:visible="playgroundDrawerVisible"
      class="api-playground-drawer"
      :title="'API Playground'"
      :description="'Debug sessions, Agent Chat, and SSE output using the real API integration flow'"
      icon="code"
      width="640px"
      :min-width="560"
      :max-width="960"
      storage-key="setting-drawer:width:api-playground"
      :confirm-text="'Run test'"
      :confirm-loading="playground.running"
      :confirm-disabled="!canRunPlayground"
      @confirm="runPlayground"
      @cancel="handlePlaygroundDrawerCancel"
    >
      <template #footer-left>
        <t-button v-if="playground.running" variant="outline" @click="stopPlayground">
          <template #icon><t-icon name="close-circle" /></template>
          {{ 'Stop' }}
        </t-button>
        <span v-if="playgroundDisabledReason" class="footer-test-message">
          {{ playgroundDisabledReason }}
        </span>
      </template>

      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Request setup' }}</h4>

        <div class="drawer-form-item">
          <label class="drawer-form-label">{{ 'Test agent' }}</label>
          <t-select
            v-model="playground.agent_id"
            :options="agentOptions"
            :loading="agentsLoading"
            filterable
            :placeholder="'Select an agent to test'"
          />
          <p v-if="agentsError" class="drawer-form-desc drawer-form-desc--error">{{ agentsError }}</p>
        </div>

        <div class="drawer-form-item">
          <label class="drawer-form-label">{{ 'External user ID' }}</label>
          <t-input
            v-model="playground.external_user_id"
            :disabled="form.mode === 'tenant'"
            class="mono-input"
            :placeholder="'user_123'"
          />
          <p class="drawer-form-desc">{{ externalUserHint }}</p>
        </div>

        <div class="drawer-form-item">
          <label class="drawer-form-label">{{ 'Test question' }}</label>
          <t-textarea
            v-model="playground.query"
            :autosize="{ minRows: 2, maxRows: 4 }"
            :placeholder="'Enter a question to send to the agent'"
          />
        </div>
      </section>

      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Request preview' }}</h4>
        <div class="code-panel playground-preview">
          <div class="code-panel__toolbar">
            <span class="code-panel__label">{{ 'Request preview (secrets masked)' }}</span>
            <t-button size="small" variant="text" class="code-panel__copy" @click="copy(playgroundRequestPreview)">
              <template #icon><t-icon name="file-copy" /></template>
              {{ 'Copy' }}
            </t-button>
          </div>
          <pre class="code-panel__pre">{{ playgroundRequestPreview }}</pre>
        </div>
      </section>

      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Run result' }}</h4>
        <t-alert v-if="playground.error" theme="error" :message="playground.error" />

        <div v-if="hasPlaygroundResult" class="playground-results">
          <div v-if="form.mode === 'signed_token' && playground.signed_token" class="playground-step">
            <div class="playground-step__header">
              <span>{{ 'Generated test token' }}</span>
              <t-button size="small" variant="text" @click="copy(playground.signed_token)">
                <template #icon><t-icon name="file-copy" /></template>
                {{ 'Copy' }}
              </t-button>
            </div>
            <pre>{{ playground.signed_token }}</pre>
          </div>

          <div class="playground-step">
            <div class="playground-step__header">
              <span>{{ 'Step 1: Create Session' }}</span>
              <t-tag size="small" :theme="playground.session_status === 'success' ? 'success' : 'default'" variant="light">
                {{ playground.session_status || '-' }}
              </t-tag>
            </div>
            <pre>{{ playground.session_response || '-' }}</pre>
          </div>

          <div class="playground-step">
            <div class="playground-step__header">
              <span>{{ 'Step 2: Agent Chat SSE' }}</span>
              <t-tag size="small" :theme="playground.chat_status === 'success' ? 'success' : 'default'" variant="light">
                {{ playground.chat_status || '-' }}
              </t-tag>
            </div>
            <pre>{{ playground.stream_output || '-' }}</pre>
          </div>

          <div v-if="playground.final_answer" class="playground-step">
            <div class="playground-step__header">
              <span>{{ 'Extracted answer' }}</span>
            </div>
            <pre>{{ playground.final_answer }}</pre>
          </div>
        </div>
        <p v-else class="playground-empty">{{ 'Run a test to show the session response, raw SSE output, and extracted answer here.' }}</p>
      </section>
    </SettingDrawer>

    <SettingDrawer
      :visible="apiKeyDialogVisible"
      class="api-key-create-drawer"
      :title="'Create API Key'"
      :description="'Choose capabilities and the knowledge-base scope this key may use.'"
      icon="lock-on"
      width="560px"
      :min-width="480"
      :max-width="920"
      storage-key="setting-drawer:width:api-key-create"
      :close-on-overlay-click="false"
      :confirm-text="'Create API Key'"
      :confirm-loading="apiKeyCreating"
      @update:visible="(v: boolean) => apiKeyDialogVisible = v"
      @confirm="createScopedAPIKey"
    >
      <div class="api-key-dialog">
        <div class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Name' }}</label>
          </div>
          <t-input
            v-model="apiKeyForm.name"
            :placeholder="'Example: MCP read-only access'"
          />
        </div>

        <div class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Access type' }}</label>
          </div>
          <t-radio-group v-model="apiKeyAccessMode" class="mode-radio api-key-access-type-radio">
            <t-radio-button value="scoped">{{ 'Scoped access' }}</t-radio-button>
            <t-radio-button value="full">{{ 'Workspace full access' }}</t-radio-button>
          </t-radio-group>
          <p class="scope-hint">
            {{
              apiKeyFullAccessEnabled
                ? 'Allow every workspace-level API, including models, vector stores, data sources and channels. Knowledge-base scope does not apply.'
                : '\u0022Knowledge base\u0022 operates within the KB scope at the permission level below; \u0022Workspace full\u0022 additionally unlocks every workspace-level API such as models and data sources.'
            }}
          </p>
        </div>

        <div v-if="!apiKeyFullAccessEnabled" class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Capabilities' }}</label>
          </div>
          <div v-if="!apiKeyFullAccessEnabled" class="api-key-capability-list">
            <div
              v-for="group in apiKeyCapabilityGroups"
              :key="group.key"
              class="api-key-capability-group"
            >
              <div class="api-key-capability-group__header">
                <span>{{ group.labelKey }}</span>
                <t-button
                  size="small"
                  variant="text"
                  @click="toggleCapabilityGroup(group, !capabilityGroupAllSelected(group))"
                >
                  {{
                    capabilityGroupAllSelected(group)
                      ? 'Clear'
                      : 'Select all'
                  }}
                </t-button>
              </div>
              <div class="api-key-capability-group__items">
                <div
                  v-for="capability in group.capabilities"
                  :key="capability.value"
                  class="api-key-capability-item"
                >
                  <t-checkbox
                    :model-value="capabilitySelections[capability.value]"
                    @change="handleCapabilityChange(capability.value, $event)"
                  >
                    {{ capability.labelKey }}
                  </t-checkbox>
                  <p class="scope-hint">{{ capability.hintKey }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="apiKeyKnowledgeScopeApplies" class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Knowledge bases' }}</label>
          </div>
          <t-select
            v-model="apiKeyForm.knowledge_base_ids"
            multiple
            filterable
            clearable
            :loading="knowledgeBasesLoading"
            :options="knowledgeBaseOptions"
            :placeholder="'Leave empty to allow all knowledge bases'"
          />
        </div>
      </div>
    </SettingDrawer>

    <SettingDrawer
      :visible="apiKeyScopeDialogVisible"
      class="api-key-edit-drawer"
      :title="'Edit API key'"
      :description="`Change the name and authorization scope of API key \u0022${editingAPIKey?.name || ''}\u0022.`"
      icon="edit-1"
      width="560px"
      :min-width="440"
      :max-width="760"
      storage-key="setting-drawer:width:api-key-scope"
      :confirm-text="'Save'"
      :confirm-loading="apiKeyScopeSaving"
      @update:visible="(v: boolean) => apiKeyScopeDialogVisible = v"
      @confirm="saveAPIKeyConfiguration"
    >
      <div class="api-key-dialog">
        <div class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Name' }}</label>
          </div>
          <t-input
            v-model="editingAPIKeyForm.name"
            :placeholder="'Example: MCP read-only access'"
          />
        </div>

        <div class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Access type' }}</label>
          </div>
          <t-radio-group v-model="editingAPIKeyAccessMode" class="mode-radio api-key-access-type-radio">
            <t-radio-button value="scoped">{{ 'Scoped access' }}</t-radio-button>
            <t-radio-button value="full">{{ 'Workspace full access' }}</t-radio-button>
          </t-radio-group>
          <p class="scope-hint">
            {{
              editingAPIKeyFullAccessEnabled
                ? 'Allow every workspace-level API, including models, vector stores, data sources and channels. Knowledge-base scope does not apply.'
                : '\u0022Knowledge base\u0022 operates within the KB scope at the permission level below; \u0022Workspace full\u0022 additionally unlocks every workspace-level API such as models and data sources.'
            }}
          </p>
        </div>

        <div v-if="!editingAPIKeyFullAccessEnabled" class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Capabilities' }}</label>
          </div>
          <div class="api-key-capability-list">
            <div
              v-for="group in apiKeyCapabilityGroups"
              :key="group.key"
              class="api-key-capability-group"
            >
              <div class="api-key-capability-group__header">
                <span>{{ group.labelKey }}</span>
                <t-button
                  size="small"
                  variant="text"
                  @click="toggleEditingCapabilityGroup(group, !editingCapabilityGroupAllSelected(group))"
                >
                  {{
                    editingCapabilityGroupAllSelected(group)
                      ? 'Clear'
                      : 'Select all'
                  }}
                </t-button>
              </div>
              <div class="api-key-capability-group__items">
                <div
                  v-for="capability in group.capabilities"
                  :key="capability.value"
                  class="api-key-capability-item"
                >
                  <t-checkbox
                    :model-value="editingCapabilitySelections[capability.value]"
                    @change="editingCapabilitySelections[capability.value] = Boolean($event)"
                  >
                    {{ capability.labelKey }}
                  </t-checkbox>
                  <p class="scope-hint">{{ capability.hintKey }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="editingKnowledgeScopeApplies" class="api-key-dialog-row">
          <div class="api-key-dialog-row__label">
            <label>{{ 'Knowledge bases' }}</label>
          </div>
          <t-select
            v-model="editingAPIKeyForm.knowledge_base_ids"
            multiple
            filterable
            clearable
            :loading="knowledgeBasesLoading"
            :options="knowledgeBaseOptions"
            :placeholder="'Leave empty to allow all knowledge bases'"
          />
          <p class="scope-hint">{{ 'Leave empty to allow all knowledge bases in this workspace. All changes take effect immediately.' }}</p>
        </div>
      </div>
    </SettingDrawer>

  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { copyWithToast } from '@/utils/clipboard'
import { getCurrentUser } from '@/api/auth'
import { listAgents, BUILTIN_SMART_REASONING_ID, type CustomAgent } from '@/api/agent'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import {
  createTenantAPIKey,
  deleteTenantAPIKey,
  createAPIPrincipalTestToken,
  getAPIPrincipalConfig,
  listTenantAPIKeys,
  updateTenantAPIKey,
  updateAPIPrincipalConfig,
  type APIPrincipalConfig,
  type APIPrincipalMode,
  type TenantAPIKey,
  type TenantAPIKeyCapability,
} from '@/api/tenant'
import { listKnowledgeBases } from '@/api/knowledge-base'
import { getApiBaseUrl } from '@/utils/api-base'
import {
  DEFAULT_TENANT_API_KEY_CAPABILITIES,
  KB_SCOPED_API_KEY_CAPABILITIES,
  TENANT_API_KEY_CAPABILITIES,
  TENANT_API_KEY_CAPABILITY_GROUPS,
  type ApiKeyCapabilityGroup,
} from '@/config/apiKeyCapabilities'
import { normalizeAPIKeyKnowledgeBaseIDs } from './apiKeyScope'
import { consumeApiPlaygroundSSE } from './apiPlaygroundSSE'


const DEFAULT_DIRECT_HEADER_NAME = 'X-External-User-ID'
const DEFAULT_TOKEN_HEADER_NAME = 'X-External-User-Token'

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const tenantId = ref(0)
const apiKey = ref('')
const config = ref<APIPrincipalConfig | null>(null)
const apiKeys = ref<TenantAPIKey[]>([])
const apiKeysLoading = ref(false)
const apiKeyDialogVisible = ref(false)
const apiKeyCreating = ref(false)
const apiKeyScopeDialogVisible = ref(false)
const apiKeyScopeSaving = ref(false)
const editingAPIKey = ref<TenantAPIKey | null>(null)
const knowledgeBasesLoading = ref(false)
const knowledgeBases = ref<Array<{ id: string; name: string }>>([])
const secretInput = ref('')
/** Plaintext of the last secret successfully saved in this page session. */
const lastSavedSecretInput = ref('')
const exampleTab = ref<'jwt' | 'curl'>('curl')
const agents = ref<CustomAgent[]>([])
const agentsLoading = ref(false)
const agentsError = ref('')
const playgroundDrawerVisible = ref(false)
const playgroundController = ref<AbortController | null>(null)
const showHMACSecret = ref(false)
const wailsApiBaseURL = ref<string | null>(null)
const wailsApiLanBaseURL = ref<string | null>(null)
const showDesktopPortSetting = ref(false)
const showDesktopBindPublicSetting = ref(false)
const desktopPortInput = ref<number | undefined>(0)
const desktopBindPublicInput = ref(false)
const desktopListenPublicActive = ref(false)

const form = reactive({
  mode: 'tenant' as APIPrincipalMode,
  direct_header_name: DEFAULT_DIRECT_HEADER_NAME,
  signed_token_header_name: DEFAULT_TOKEN_HEADER_NAME,
  require_direct_header: false,
})

const API_KEY_CAPABILITIES = TENANT_API_KEY_CAPABILITIES
const DEFAULT_API_KEY_CAPABILITIES = DEFAULT_TENANT_API_KEY_CAPABILITIES
const KB_SCOPED_CAPABILITIES = KB_SCOPED_API_KEY_CAPABILITIES
const apiKeyCapabilityGroups = TENANT_API_KEY_CAPABILITY_GROUPS

const capabilitySelections = reactive<Record<TenantAPIKeyCapability, boolean>>(
  API_KEY_CAPABILITIES.reduce((acc, capability) => {
    acc[capability] = DEFAULT_API_KEY_CAPABILITIES.has(capability)
    return acc
  }, {} as Record<TenantAPIKeyCapability, boolean>),
)

const editingCapabilitySelections = reactive<Record<TenantAPIKeyCapability, boolean>>(
  API_KEY_CAPABILITIES.reduce((acc, capability) => {
    acc[capability] = false
    return acc
  }, {} as Record<TenantAPIKeyCapability, boolean>),
)

const editingAPIKeyForm = reactive({
  name: '',
  knowledge_base_ids: [] as string[],
  tenant_full_enabled: false,
  expires_at_unix: undefined as number | undefined,
})

const editingAPIKeyFullAccessEnabled = computed(() => editingAPIKeyForm.tenant_full_enabled)
const editingAPIKeyAccessMode = computed<'scoped' | 'full'>({
  get: () => (editingAPIKeyForm.tenant_full_enabled ? 'full' : 'scoped'),
  set: (value) => {
    editingAPIKeyForm.tenant_full_enabled = value === 'full'
  },
})
const editingSelectedCapabilities = computed(() => (
  API_KEY_CAPABILITIES.filter((capability) => editingCapabilitySelections[capability])
))
const editingKnowledgeScopeApplies = computed(() => (
  !editingAPIKeyFullAccessEnabled.value
  && editingSelectedCapabilities.value.some((capability) => KB_SCOPED_CAPABILITIES.has(capability))
))

watch(editingKnowledgeScopeApplies, (applies) => {
  if (!applies) editingAPIKeyForm.knowledge_base_ids = []
})

function editingCapabilityGroupAllSelected(group: ApiKeyCapabilityGroup): boolean {
  return group.capabilities.every((capability) => editingCapabilitySelections[capability.value])
}

function toggleEditingCapabilityGroup(group: ApiKeyCapabilityGroup, selected: boolean) {
  group.capabilities.forEach((capability) => {
    editingCapabilitySelections[capability.value] = selected
  })
}

const apiKeyForm = reactive({
  name: '',
  knowledge_base_ids: [] as string[],
  // Tenant-full keys already cover every capability. Scoped keys default to
  // retrieval + chat + agent reads so a fresh integration can ask questions
  // and present an agent picker immediately.
  tenant_full_enabled: false,
})

const apiKeyFullAccessEnabled = computed(() => apiKeyForm.tenant_full_enabled)
const apiKeyAccessMode = computed<'scoped' | 'full'>({
  get: () => (apiKeyForm.tenant_full_enabled ? 'full' : 'scoped'),
  set: (value) => {
    apiKeyForm.tenant_full_enabled = value === 'full'
  },
})
const selectedCapabilityValues = computed(() => API_KEY_CAPABILITIES.filter((capability) => capabilitySelections[capability]))
const apiKeyKnowledgeScopeApplies = computed(() => (
  !apiKeyFullAccessEnabled.value
  && selectedCapabilityValues.value.some((capability) => KB_SCOPED_CAPABILITIES.has(capability))
))

watch(() => apiKeyForm.tenant_full_enabled, (enabled) => {
  if (enabled) {
    apiKeyForm.knowledge_base_ids = []
  }
})

watch(apiKeyKnowledgeScopeApplies, (applies) => {
  if (!applies) {
    apiKeyForm.knowledge_base_ids = []
  }
})

function selectedCapabilities(): TenantAPIKeyCapability[] {
  return selectedCapabilityValues.value
}

function setCapabilitySelected(capability: TenantAPIKeyCapability, selected: boolean) {
  capabilitySelections[capability] = selected
}

function handleCapabilityChange(capability: TenantAPIKeyCapability, checked: unknown) {
  setCapabilitySelected(capability, Boolean(checked))
}

function capabilityGroupAllSelected(group: ApiKeyCapabilityGroup): boolean {
  return group.capabilities.every((capability) => capabilitySelections[capability.value])
}

function toggleCapabilityGroup(group: ApiKeyCapabilityGroup, selected: boolean) {
  group.capabilities.forEach((capability) => {
    capabilitySelections[capability.value] = selected
  })
}

// Full-access keys already cover every capability, so no capability badges for them.
function keyCapabilityLabels(key: TenantAPIKey): string[] {
  if (key.full_access) return []
  const labels: Partial<Record<TenantAPIKeyCapability, string>> = {
    retrieve: 'Retrieve knowledge bases',
    chat: 'Chat',
    read_agents: 'Read agents',
    ingest: 'Write KB content',
    manage_kbs: 'Manage knowledge bases',
    manage_agents: 'Manage agents',
    message_history: 'Message history',
    manage_models: 'Manage models',
    manage_mcp_services: 'Manage MCP services',
    manage_datasources: 'Manage data sources',
    manage_channels: 'Manage channels',
    manage_vector_stores: 'Manage retrieval infrastructure',
    manage_storage_backends: 'Manage storage backends',
    manage_web_search: 'Manage web search',
    run_evaluations: 'Run evaluations',
    manage_members: 'Manage members',
    manage_spaces: 'Manage spaces',
    manage_tenant_settings: 'Manage workspace settings',
  }
  return (key.capabilities ?? [])
    .map((c) => labels[c])
    .filter((label): label is string => Boolean(label))
}

function formatApiKeyCapabilitiesTitle(key: TenantAPIKey): string {
  if (key.full_access) return 'Workspace full access'
  const labels = keyCapabilityLabels(key)
  return labels.length > 0 ? labels.join(' / ') : 'Scoped access'
}

function formatApiKeyAccessModeLabel(key: TenantAPIKey): string {
  return key.full_access
    ? 'Workspace full access'
    : 'Scoped access'
}

type PlaygroundStatus = '' | 'running' | 'success' | 'failed' | 'stopped'

type EnterpriseRagDesktopWindow = Window & {
  __ENTERPRISERAG_API_BASE__?: string
  __ENTERPRISERAG_API_LAN_BASE__?: string
  go?: {
    main?: {
      App?: {
        GetAPIBaseURL?: () => Promise<string> | string
        GetAPILanBaseURL?: () => Promise<string> | string
        GetDesktopHTTPPortSetting?: () => Promise<number> | number
        GetDesktopHTTPBindPublicSetting?: () => Promise<boolean> | boolean
        GetDesktopListenPublicActive?: () => Promise<boolean> | boolean
        SetDesktopHTTPPortSetting?: (port: number) => Promise<void> | void
        SetDesktopHTTPBindPublicSetting?: (v: boolean) => Promise<void> | void
      }
    }
  }
}

const playground = reactive({
  agent_id: '',
  query: 'hello',
  external_user_id: 'user_123',
  signed_token: '',
  running: false,
  session_status: '' as PlaygroundStatus,
  chat_status: '' as PlaygroundStatus,
  session_response: '',
  stream_output: '',
  final_answer: '',
  error: '',
})

watch(() => form.mode, (mode) => {
  if (mode === 'signed_token') {
    exampleTab.value = 'curl'
  }
})

watch(playgroundDrawerVisible, (visible) => {
  if (!visible) {
    stopPlayground()
  }
})

const apiBaseUrl = computed(() => {
  if (wailsApiBaseURL.value) {
    return wailsApiBaseURL.value
  }
  const configured = getApiBaseUrl().trim().replace(/\/$/, '')
  const origin = typeof window !== 'undefined' && window.location.origin !== 'null' ? window.location.origin : ''
  return `${configured || origin}/api/v1`
})

const showLanUrlUnavailableHint = computed(() => (
  showDesktopBindPublicSetting.value
  && desktopListenPublicActive.value
  && !wailsApiLanBaseURL.value
))

const tokenHeaderName = computed(() => DEFAULT_TOKEN_HEADER_NAME)

const directHeaderName = computed(() => DEFAULT_DIRECT_HEADER_NAME)

const secretInputType = computed(() => {
  if (!secretInput.value.trim()) return 'text'
  return showHMACSecret.value ? 'text' : 'password'
})

const canAutoSave = computed(() => {
  if (!tenantId.value) return false
  if (form.mode === 'signed_token') {
    // Either a secret is already stored server-side, or the user has just
    // typed a new one. The plaintext secret is never returned by the API,
    // so we rely on the has_hmac_secret presence flag.
    return config.value?.has_hmac_secret === true || secretInput.value.trim() !== ''
  }
  return true
})

const agentOptions = computed(() => agents.value.map((agent) => ({
  label: `${agent.name}${agent.is_builtin ? ` · ${'Built-in'}` : ''}`,
  value: agent.id,
})))

const knowledgeBaseOptions = computed(() => knowledgeBases.value.map((kb) => ({
  label: kb.name || kb.id,
  value: kb.id,
})))

const hasUnsavedSecretChange = computed(() => {
  const trimmed = secretInput.value.trim()
  if (!trimmed) return false
  return trimmed !== lastSavedSecretInput.value
})

const showSecretSavedHint = computed(() => {
  const trimmed = secretInput.value.trim()
  return trimmed !== '' && trimmed === lastSavedSecretInput.value
})

const hasUnsavedPrincipalChanges = computed(() => {
  const cfg = config.value
  if (!cfg) return false
  return (
    form.mode !== cfg.mode
    || form.require_direct_header !== cfg.require_direct_header
    || hasUnsavedSecretChange.value
  )
})

const externalUserHint = computed(() => {
  if (form.mode === 'tenant') return 'Workspace-only mode does not send an external user identity; all API sessions use the same workspace-level principal.'
  if (form.mode === 'direct_header') {
    return `This user ID is sent with the ${directHeaderName.value} header.`
  }
  return `Signed-token mode uses the JWT sub as the external user. The playground signs a short-lived test JWT for this user ID.`
})

const playgroundRequestPreview = computed(() => {
  const body = {
    query: playground.query || '<query>',
    agent_enabled: true,
    agent_id: playground.agent_id || '<agent_id>',
    channel: 'api',
  }
  const headers = buildPlaygroundHeaders(true)
  return [
    'POST /api/v1/sessions',
    JSON.stringify({ headers: headers.sessionHeaders, body: {} }, null, 2),
    '',
    'POST /api/v1/agent-chat/<session_id>',
    JSON.stringify({ headers: headers.chatHeaders, body }, null, 2),
  ].join('\n')
})

const playgroundDisabledReason = computed(() => {
  if (playground.running) return ''
  if (!apiKey.value) return 'This workspace has no API key.'
  if (!playground.agent_id) return 'Select a test agent.'
  if (!playground.query.trim()) return 'Enter a test question.'
  if (form.mode === 'signed_token' && !playground.external_user_id.trim()) {
    return 'Enter an external user ID.'
  }
  return ''
})

const canRunPlayground = computed(() => !playground.running && !playgroundDisabledReason.value)

const hasPlaygroundResult = computed(() => Boolean(
  playground.signed_token || playground.session_response || playground.stream_output || playground.final_answer,
))

const tokenSignExample = computed(() => {
  const tid = tenantId.value || 10000
  const headerName = tokenHeaderName.value
  return `import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func signExternalUserToken(hmacSecret, externalUserID string, tenantID uint64) (string, error) {
	claims := jwt.MapClaims{
		"sub":       externalUserID, // e.g. "user_123"
		"tenant_id": float64(tenantID),
		"aud":       "enterpriserag",
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(hmacSecret))
}

// Send on each EnterpriseRag API request:
//   ${headerName}: <JWT from signExternalUserToken>
// Tenant ID for this workspace: ${tid}`
})

const requestExample = computed(() => {
  const apiKeyHeader = `  -H "X-API-Key: ${apiKey.value ? '<API_KEY>' : '<YOUR_API_KEY>'}"`
  const contentType = '  -H "Content-Type: application/json"'
  const principalHeaders: string[] = []
  if (form.mode === 'direct_header') {
    principalHeaders.push(`  -H "${directHeaderName.value}: user_123"`)
  }
  if (form.mode === 'signed_token') {
    principalHeaders.push(`  -H "${tokenHeaderName.value}: ${'<JWT signed by your backend>'}"`)
  }
  const commonHeaders = [apiKeyHeader, contentType, ...principalHeaders].join(' \\\n')
  const agentID = playground.agent_id || BUILTIN_SMART_REASONING_ID

  const lines: string[] = []
  if (form.mode === 'signed_token') {
    lines.push(
      `# 0. Sign JWT on your backend (HS256, aud=enterpriserag, sub=user id, tenant_id=${tenantId.value || '<tenant_id>'}, exp<=24h)`,
      `#    Put the JWT in ${tokenHeaderName.value} — not the HMAC secret`,
      '',
    )
  }
  lines.push(
    '# 1. Create a session',
    `curl -X POST ${apiBaseUrl.value}/sessions \\`,
    commonHeaders,
    `  -d '{}'`,
    '',
    '# 2. Agent chat (SSE; replace <session_id> with the id from step 1)',
    `curl -N -X POST ${apiBaseUrl.value}/agent-chat/<session_id> \\`,
    commonHeaders,
    `  -d '{"query":"hello","agent_enabled":true,"agent_id":"${agentID}","channel":"api"}'`,
  )
  return lines.join('\n')
})

const activeExampleText = computed(() => (
  form.mode === 'signed_token' && exampleTab.value === 'jwt'
    ? tokenSignExample.value
    : requestExample.value
))

const activeExampleLabel = computed(() => (
  form.mode === 'signed_token' && exampleTab.value === 'jwt'
    ? 'Backend JWT signing example (Go)'
    : 'Request example'
))

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [userResp] = await Promise.all([
      getCurrentUser(),
      loadAgents(),
    ])
    const tenant = (userResp as any)?.data?.tenant
    if (!tenant?.id) {
      throw new Error('Failed to load API integration settings')
    }
    tenantId.value = Number(tenant.id)
    await Promise.all([
      loadAPIKeys(),
      loadKnowledgeBaseOptions(),
    ])

    const cfgResp = await getAPIPrincipalConfig(tenantId.value)
    if (!cfgResp.success || !cfgResp.data) {
      throw new Error(cfgResp.message || 'Failed to load API integration settings')
    }
    config.value = cfgResp.data
    form.mode = cfgResp.data.mode || 'tenant'
    form.direct_header_name = DEFAULT_DIRECT_HEADER_NAME
    form.signed_token_header_name = DEFAULT_TOKEN_HEADER_NAME
    form.require_direct_header = cfgResp.data.require_direct_header === true
    // The plaintext secret is never returned; start with an empty input and
    // rely on config.has_hmac_secret to reflect whether one is configured.
    secretInput.value = ''
    lastSavedSecretInput.value = ''
    ensurePlaygroundAgent()
  } catch (err: any) {
    error.value = err?.message || 'Failed to load API integration settings'
  } finally {
    loading.value = false
  }
}

async function loadAPIKeys() {
  if (!tenantId.value) return
  apiKeysLoading.value = true
  try {
    const resp = await listTenantAPIKeys(tenantId.value)
    if (!resp.success) {
      throw new Error(resp.message || 'Failed to load API keys')
    }
    apiKeys.value = resp.data || []
    if (!apiKey.value && apiKeys.value.length > 0) {
      apiKey.value = apiKeys.value[0].api_key || ''
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to load API keys')
  } finally {
    apiKeysLoading.value = false
  }
}

async function loadKnowledgeBaseOptions() {
  knowledgeBasesLoading.value = true
  try {
    const resp: any = await listKnowledgeBases({ creator: 'all' })
    const rows = Array.isArray(resp?.data) ? resp.data : []
    knowledgeBases.value = rows.map((item: any) => ({
      id: String(item.id),
      name: item.name || item.id,
    }))
  } catch {
    knowledgeBases.value = []
  } finally {
    knowledgeBasesLoading.value = false
  }
}

async function loadAgents() {
  agentsLoading.value = true
  agentsError.value = ''
  try {
    const resp = await listAgents({ creator: 'all' }) as any
    agents.value = Array.isArray(resp?.data) ? resp.data : []
    ensurePlaygroundAgent()
  } catch (err: any) {
    agentsError.value = err?.message || 'Failed to load agents'
  } finally {
    agentsLoading.value = false
  }
}

function ensurePlaygroundAgent() {
  if (playground.agent_id && agents.value.some((agent) => agent.id === playground.agent_id)) return
  const smartReasoning = agents.value.find((agent) => agent.id === BUILTIN_SMART_REASONING_ID)
  playground.agent_id = smartReasoning?.id || agents.value[0]?.id || ''
}

function openPlaygroundDrawer() {
  ensurePlaygroundAgent()
  playgroundDrawerVisible.value = true
}

function handlePlaygroundDrawerCancel() {
  stopPlayground()
  playgroundDrawerVisible.value = false
}

function handlePrincipalModeChange(mode: APIPrincipalMode) {
  form.mode = mode
  void saveIfNeeded()
}

function handleRequireDirectHeaderChange(checked: boolean) {
  form.require_direct_header = checked
  void saveIfNeeded()
}

function triggerAutoSave() {
  void saveIfNeeded()
}

function confirmGenerateSecret() {
  const dialog = DialogPlugin.confirm({
    header: 'Reset HMAC secret?',
    body: 'After reset, the old HMAC secret is revoked immediately. Every backend service signing JWTs with the old secret must switch to the new secret to keep signed-token calls working. This action cannot be undone.',
    confirmBtn: { content: 'Reset', theme: 'danger' },
    cancelBtn: 'Cancel',
    onConfirm: async () => {
      await generateSecret()
      dialog.destroy()
    },
    onClose: () => dialog.destroy(),
  })
}

async function generateSecret() {
  const bytes = new Uint8Array(32)
  window.crypto.getRandomValues(bytes)
  secretInput.value = btoa(String.fromCharCode(...bytes))
  showHMACSecret.value = true
  await saveIfNeeded({ showSuccess: true })
}

async function saveIfNeeded(options: { showSuccess?: boolean } = {}) {
  if (!hasUnsavedPrincipalChanges.value) return true
  if (!canAutoSave.value) {
    MessagePlugin.error('Signed-token mode needs an HMAC secret before auto-save can run.')
    return false
  }
  saving.value = true
  try {
    const payload: Parameters<typeof updateAPIPrincipalConfig>[1] = {
      mode: form.mode,
      direct_header_name: DEFAULT_DIRECT_HEADER_NAME,
      signed_token_header_name: DEFAULT_TOKEN_HEADER_NAME,
      require_direct_header: form.require_direct_header,
    }
    // Only send the secret when the user entered a new value; otherwise the
    // backend keeps the stored value untouched.
    const secretBeingSaved = hasUnsavedSecretChange.value ? secretInput.value.trim() : ''
    if (secretBeingSaved) {
      payload.hmac_secret = secretBeingSaved
    }
    const resp = await updateAPIPrincipalConfig(tenantId.value, payload)
    if (!resp.success || !resp.data) {
      throw new Error(resp.message || 'Failed to save API integration settings')
    }
    config.value = resp.data
    if (secretBeingSaved) {
      lastSavedSecretInput.value = secretBeingSaved
      showHMACSecret.value = true
    }
    if (options.showSuccess) {
      MessagePlugin.success(
        secretBeingSaved ? 'Secret saved. Copy it to your backend now — you will not be able to view it again after leaving this page.' : 'API integration settings saved',
      )
    }
    return true
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to save API integration settings')
    return false
  } finally {
    saving.value = false
  }
}

async function copy(text: string) {
  await copyWithToast(text, 'Copied')
}

async function tryLoadWailsApiBaseURL() {
  const win = window as EnterpriseRagDesktopWindow
  for (let i = 0; i < 40; i++) {
    const injected = win.__ENTERPRISERAG_API_BASE__
    if (typeof injected === 'string' && injected.trim()) {
      wailsApiBaseURL.value = injected.trim().replace(/\/$/, '')
      await tryLoadWailsLanHints(win)
      return
    }
    const fn = win.go?.main?.App?.GetAPIBaseURL
    if (typeof fn === 'function') {
      try {
        const raw = await Promise.resolve(fn())
        if (typeof raw === 'string' && raw.trim()) {
          wailsApiBaseURL.value = raw.trim().replace(/\/$/, '')
        }
      } catch {
        /* binding error */
      }
      await tryLoadWailsLanHints(win)
      return
    }
    await new Promise((resolve) => setTimeout(resolve, 50))
  }
  await tryLoadWailsLanHints(win)
}

async function tryLoadWailsLanHints(win: EnterpriseRagDesktopWindow) {
  const injectedLan = win.__ENTERPRISERAG_API_LAN_BASE__
  if (typeof injectedLan === 'string' && injectedLan.trim()) {
    wailsApiLanBaseURL.value = injectedLan.trim().replace(/\/$/, '')
  }
  const fnLan = win.go?.main?.App?.GetAPILanBaseURL
  if (typeof fnLan === 'function' && !wailsApiLanBaseURL.value) {
    try {
      const raw = await Promise.resolve(fnLan())
      if (typeof raw === 'string' && raw.trim()) {
        wailsApiLanBaseURL.value = raw.trim().replace(/\/$/, '')
      }
    } catch {
      /* binding error */
    }
  }
  const fnAct = win.go?.main?.App?.GetDesktopListenPublicActive
  if (typeof fnAct === 'function') {
    try {
      desktopListenPublicActive.value = !!(await Promise.resolve(fnAct()))
    } catch {
      desktopListenPublicActive.value = false
    }
  }
}

function desktopPortBindingsAvailable(win: EnterpriseRagDesktopWindow) {
  const app = win.go?.main?.App
  return typeof app?.GetDesktopHTTPPortSetting === 'function' && typeof app?.SetDesktopHTTPPortSetting === 'function'
}

function desktopBindPublicBindingsAvailable(win: EnterpriseRagDesktopWindow) {
  const app = win.go?.main?.App
  return (
    typeof app?.GetDesktopHTTPBindPublicSetting === 'function' &&
    typeof app?.SetDesktopHTTPBindPublicSetting === 'function'
  )
}

async function loadDesktopApiPrefs() {
  const win = window as EnterpriseRagDesktopWindow
  if (desktopPortBindingsAvailable(win)) {
    showDesktopPortSetting.value = true
    try {
      const port = await Promise.resolve(win.go!.main!.App!.GetDesktopHTTPPortSetting!())
      desktopPortInput.value = typeof port === 'number' ? port : 0
    } catch {
      desktopPortInput.value = 0
    }
  }
  if (desktopBindPublicBindingsAvailable(win)) {
    showDesktopBindPublicSetting.value = true
    try {
      const bind = await Promise.resolve(win.go!.main!.App!.GetDesktopHTTPBindPublicSetting!())
      desktopBindPublicInput.value = !!bind
    } catch {
      desktopBindPublicInput.value = false
    }
  }
}

const onDesktopBindPublicChange = async (value: boolean) => {
  const next = value === true
  const fn = (window as EnterpriseRagDesktopWindow).go?.main?.App?.SetDesktopHTTPBindPublicSetting
  if (typeof fn !== 'function') return
  try {
    await Promise.resolve(fn(next))
    MessagePlugin.success('Saved. Restart the app for it to take effect.')
  } catch (err: unknown) {
    MessagePlugin.error(err instanceof Error ? err.message : 'Save failed')
    desktopBindPublicInput.value = !next
  }
}

const saveDesktopPort = async () => {
  const value = desktopPortInput.value
  const port = typeof value === 'number' && !Number.isNaN(value) ? Math.floor(value) : 0
  if (port < 0 || port > 65535) {
    MessagePlugin.warning('Enter a port between 0 and 65535')
    return
  }
  const fn = (window as EnterpriseRagDesktopWindow).go?.main?.App?.SetDesktopHTTPPortSetting
  if (typeof fn !== 'function') return
  try {
    await Promise.resolve(fn(port))
    MessagePlugin.success('Saved. Restart the app for it to take effect.')
  } catch (err: unknown) {
    MessagePlugin.error(err instanceof Error ? err.message : 'Save failed')
  }
}

function openApiDoc() {
  window.open('https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/README.md', '_blank')
}

function openCreateAPIKeyDialog() {
  apiKeyForm.name = ''
  apiKeyForm.knowledge_base_ids = []
  apiKeyForm.tenant_full_enabled = false
  API_KEY_CAPABILITIES.forEach((capability) => {
    capabilitySelections[capability] = DEFAULT_API_KEY_CAPABILITIES.has(capability)
  })
  apiKeyDialogVisible.value = true
  void loadKnowledgeBaseOptions()
}

async function createScopedAPIKey() {
  if (!apiKeyForm.name.trim()) {
    MessagePlugin.error('Enter an API key name')
    return
  }
  if (!apiKeyFullAccessEnabled.value && selectedCapabilities().length === 0) {
    MessagePlugin.error('Select at least one capability for a scoped API key')
    return
  }
  apiKeyCreating.value = true
  try {
    const resp = await createTenantAPIKey(tenantId.value, {
      name: apiKeyForm.name.trim(),
      full_access: apiKeyFullAccessEnabled.value,
      // KB scoping only applies to capabilities that touch knowledge bases.
      knowledge_base_ids: apiKeyKnowledgeScopeApplies.value ? apiKeyForm.knowledge_base_ids : [],
      // Capabilities only matter below full access; full access already covers them all.
      capabilities: apiKeyFullAccessEnabled.value ? [] : selectedCapabilities(),
    })
    if (!resp.success || !resp.data?.api_key) {
      throw new Error(resp.message || 'Failed to create API key')
    }
    apiKeyDialogVisible.value = false
    apiKey.value = resp.data.api_key
    MessagePlugin.success('API key created')
    await loadAPIKeys()
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to create API key')
  } finally {
    apiKeyCreating.value = false
  }
}

// Copy the server-side config when opening the editor, so cancelling does not pollute the original data in the list.
function openEditAPIKeyScope(key: TenantAPIKey) {
  editingAPIKey.value = key
  editingAPIKeyForm.name = key.name
  editingAPIKeyForm.tenant_full_enabled = key.full_access
  editingAPIKeyForm.knowledge_base_ids = normalizeAPIKeyKnowledgeBaseIDs(key.knowledge_base_ids)
  const expiresAt = key.expires_at ? Date.parse(key.expires_at) : Number.NaN
  editingAPIKeyForm.expires_at_unix = Number.isNaN(expiresAt)
    ? undefined
    : Math.floor(expiresAt / 1000)
  const currentCapabilities = new Set(key.capabilities || [])
  API_KEY_CAPABILITIES.forEach((capability) => {
    editingCapabilitySelections[capability] = currentCapabilities.has(capability)
  })
  apiKeyScopeDialogVisible.value = true
  void loadKnowledgeBaseOptions()
}

async function saveAPIKeyConfiguration() {
  const key = editingAPIKey.value
  if (!key) return
  if (!editingAPIKeyForm.name.trim()) {
    MessagePlugin.error('Enter an API key name')
    return
  }
  if (!editingAPIKeyFullAccessEnabled.value && editingSelectedCapabilities.value.length === 0) {
    MessagePlugin.error('Select at least one capability for a scoped API key')
    return
  }
  apiKeyScopeSaving.value = true
  try {
    const resp = await updateTenantAPIKey(tenantId.value, key.id, {
      name: editingAPIKeyForm.name.trim(),
      full_access: editingAPIKeyFullAccessEnabled.value,
      capabilities: editingAPIKeyFullAccessEnabled.value ? [] : editingSelectedCapabilities.value,
      knowledge_base_ids: editingKnowledgeScopeApplies.value ? editingAPIKeyForm.knowledge_base_ids : [],
      expires_at_unix: editingAPIKeyForm.expires_at_unix,
    })
    if (!resp.success || !resp.data) {
      throw new Error(resp.message || 'Failed to update API key')
    }
    apiKeyScopeDialogVisible.value = false
    editingAPIKey.value = null
    MessagePlugin.success('API key updated')
    await loadAPIKeys()
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to update API key')
  } finally {
    apiKeyScopeSaving.value = false
  }
}

function confirmDeleteAPIKey(id: number) {
  const dialog = DialogPlugin.confirm({
    header: 'Delete',
    body: 'After deletion this API key is revoked immediately and deployed integrations using it will stop working.',
    confirmBtn: { content: 'Delete', theme: 'danger' },
    cancelBtn: 'Cancel',
    onConfirm: async () => {
      await deleteScopedAPIKey(id)
      dialog.destroy()
    },
    onClose: () => dialog.destroy(),
  })
}

async function deleteScopedAPIKey(id: number) {
  const resp = await deleteTenantAPIKey(tenantId.value, id)
  if (!resp.success) {
    MessagePlugin.error(resp.message || 'Failed to delete API key')
    return
  }
  MessagePlugin.success('API key deleted')
  await loadAPIKeys()
}

function formatKeyKnowledgeScope(ids: readonly string[] | null | undefined) {
  const normalizedIDs = normalizeAPIKeyKnowledgeBaseIDs(ids)
  if (!normalizedIDs.length) return 'All knowledge bases'
  const names = normalizedIDs.map((id) => knowledgeBases.value.find((kb) => kb.id === id)?.name || id)
  return names.join(', ')
}

function formatKeyMaskedValue(key: TenantAPIKey) {
  const value = key.api_key || ''
  if (!value) return '-'
  return maskAPIKey(value)
}

function maskAPIKey(value: string) {
  if (value.length <= 12) return '*'.repeat(value.length)
  return `${value.slice(0, 8)}${'*'.repeat(8)}${value.slice(-6)}`
}

function formatDate(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function buildPlaygroundHeaders(maskSecrets: boolean) {
  const commonHeaders: Record<string, string> = {
    Accept: 'application/json',
    'Content-Type': 'application/json',
    'X-API-Key': maskSecrets ? '<API_KEY>' : apiKey.value,
  }
  if (form.mode === 'direct_header' && playground.external_user_id.trim()) {
    commonHeaders[directHeaderName.value] = playground.external_user_id.trim()
  }
  if (form.mode === 'signed_token') {
    commonHeaders[tokenHeaderName.value] = maskSecrets ? '<JWT>' : playground.signed_token.trim()
  }
  return {
    sessionHeaders: commonHeaders,
    chatHeaders: { ...commonHeaders, Accept: 'text/event-stream' },
  }
}

function compactText(text: string, max = 12000) {
  if (text.length <= max) return text
  return `${text.slice(0, max)}\n...`
}

function formatJSON(value: unknown) {
  try {
    return compactText(JSON.stringify(value, null, 2))
  } catch {
    return String(value)
  }
}

async function readResponseBody(resp: Response) {
  const text = await resp.text()
  if (!text) return ''
  try {
    return formatJSON(JSON.parse(text))
  } catch {
    return compactText(text)
  }
}

async function ensurePlaygroundSignedToken() {
  if (form.mode !== 'signed_token') return
  if (!tenantId.value) throw new Error('Failed to load API integration settings')
  const resp = await createAPIPrincipalTestToken(tenantId.value, {
    external_user_id: playground.external_user_id.trim(),
    expires_in_seconds: 900,
  })
  if (!resp.success || !resp.data?.token) {
    throw new Error(resp.message || 'Failed to create a short-lived test JWT.')
  }
  playground.signed_token = resp.data.token
}

async function runPlayground() {
  if (!canRunPlayground.value) return
  const controller = new AbortController()
  playgroundController.value = controller
  playground.running = true
  playground.error = ''
  playground.session_status = 'running'
  playground.chat_status = ''
  playground.session_response = ''
  playground.stream_output = ''
  playground.final_answer = ''
  playground.signed_token = ''

  const startedAt = performance.now()
  try {
    await ensurePlaygroundSignedToken()
    const headers = buildPlaygroundHeaders(false).sessionHeaders
    const sessionResp = await fetch(`${apiBaseUrl.value}/sessions`, {
      method: 'POST',
      headers,
      body: '{}',
      signal: controller.signal,
      credentials: 'omit',
    })
    const sessionRaw = await sessionResp.text()
    let sessionPayload: any = null
    try {
      sessionPayload = sessionRaw ? JSON.parse(sessionRaw) : null
    } catch {
      sessionPayload = null
    }
    playground.session_response = sessionPayload ? formatJSON(sessionPayload) : compactText(sessionRaw)
    if (!sessionResp.ok || sessionPayload?.success === false) {
      playground.session_status = 'failed'
      throw new Error(sessionPayload?.message || sessionPayload?.error?.message || `HTTP ${sessionResp.status}`)
    }
    playground.session_status = 'success'
    const sessionID = sessionPayload?.data?.id || sessionPayload?.data?.ID
    if (!sessionID) {
      throw new Error('Session creation succeeded, but the response did not include a session id.')
    }

    playground.chat_status = 'running'
    const chatResp = await fetch(`${apiBaseUrl.value}/agent-chat/${encodeURIComponent(sessionID)}`, {
      method: 'POST',
      headers: buildPlaygroundHeaders(false).chatHeaders,
      body: JSON.stringify({
        query: playground.query.trim(),
        agent_enabled: true,
        agent_id: playground.agent_id,
        channel: 'api',
      }),
      signal: controller.signal,
      credentials: 'omit',
    })
    if (!chatResp.ok) {
      playground.chat_status = 'failed'
      const body = await readResponseBody(chatResp)
      playground.stream_output = body
      throw new Error(body || `HTTP ${chatResp.status}`)
    }
    if (!chatResp.body) {
      throw new Error('The chat API did not return a readable SSE stream.')
    }

    const result = await consumeApiPlaygroundSSE(chatResp.body, ({ raw, answer }) => {
      playground.stream_output = compactText(raw)
      playground.final_answer = answer
    })
    playground.stream_output = compactText(result.raw)
    playground.final_answer = result.answer
    if (result.status === 'failed') {
      playground.chat_status = 'failed'
      throw new Error(result.error || 'API Playground test failed')
    }
    playground.chat_status = 'success'
    MessagePlugin.success(`Test completed (${Math.round(performance.now() - startedAt)}ms)`)
  } catch (err: any) {
    const aborted = err?.name === 'AbortError'
    if (aborted) {
      if (playground.session_status === 'running') playground.session_status = 'stopped'
      if (playground.chat_status === 'running') playground.chat_status = 'stopped'
      playground.error = 'Test stopped'
    } else {
      if (playground.session_status === 'running') playground.session_status = 'failed'
      if (playground.chat_status === 'running') playground.chat_status = 'failed'
      playground.error = err?.message || 'API Playground test failed'
    }
  } finally {
    playground.running = false
    playgroundController.value = null
  }
}

function stopPlayground() {
  playgroundController.value?.abort()
}

onMounted(async () => {
  await tryLoadWailsApiBaseURL()
  await loadDesktopApiPrefs()
  await load()
})
onBeforeUnmount(stopPlayground)
</script>

<style scoped lang="less">
.api-integration {
  width: 100%;
}

.state-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 160px;
  color: var(--td-text-color-secondary);
}

.api-settings,
.settings-band {
  display: flex;
  flex-direction: column;
}

.settings-band {
  border-top: 1px solid var(--td-component-stroke);
}

.row {
  display: grid;
  grid-template-columns: minmax(220px, 0.8fr) minmax(320px, 1fr);
  gap: 24px;
  padding: 20px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.row--single {
  display: block;
}

.row--doc {
  grid-template-columns: 1fr;
}

.doc-link {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--td-brand-color);
  cursor: pointer;
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}

.link-icon {
  font-size: var(--app-text-md);
}

.desktop-api-control {
  display: flex;
  align-items: center;
  gap: 8px;
}

.desktop-port-input-wrap {
  flex: 1;
  min-width: 0;

  :deep(.t-input-number),
  :deep(.t-input__wrap) {
    width: 100%;
  }

  :deep(input) {
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-sm);
  }
}

.desktop-bind-public-control {
  justify-content: flex-end;
  padding-top: 4px;
}

.api-key-section {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 20px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.api-key-section__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.api-key-section__title {
  min-width: 0;

  label {
    display: block;
    margin-bottom: 6px;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-lg);
    font-weight: 600;
    line-height: 1.4;
  }

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    line-height: 1.55;
  }
}

.api-key-section__body {
  min-width: 0;
}

.api-key-list {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container);
  overflow: hidden;
}

.api-key-list__empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 88px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
}

.api-key-table-wrap {
  width: 100%;
  overflow-x: auto;
}

.api-key-table {
  width: 100%;
  min-width: 960px;
  border-collapse: collapse;
  table-layout: fixed;

  th,
  td {
    padding: 13px 14px;
    border-bottom: 1px solid var(--td-component-stroke);
    text-align: left;
    vertical-align: middle;
  }

  th {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-placeholder);
    font-size: var(--app-text-sm);
    font-weight: 500;
    line-height: 1.4;
  }

  td {
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    line-height: 1.45;
  }

  th:nth-child(1),
  td:nth-child(1) {
    width: 18%;
  }

  th:nth-child(2),
  td:nth-child(2) {
    width: 22%;
  }

  th:nth-child(3),
  td:nth-child(3) {
    width: 24%;
  }

  th:nth-child(4),
  td:nth-child(4) {
    width: 16%;
  }

  th:nth-child(5),
  td:nth-child(5) {
    width: 12%;
  }

  th:nth-child(6),
  td:nth-child(6) {
    width: 112px;
  }

  tbody tr:last-child td {
    border-bottom: none;
  }
}

.api-key-table__actions-heading {
  text-align: right !important;
}

.api-key-table__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

.api-key-name {
  display: block;
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-fingerprint {
  display: inline-block;
  max-width: 100%;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-sm);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: top;
  white-space: nowrap;
}

.api-key-access-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  align-items: flex-start;
  gap: 7px;
}

.api-key-access-mode {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  height: 24px;
  padding: 0 9px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  font-size: var(--app-text-sm);
  font-weight: 600;
  line-height: 22px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-access-mode--full {
  border-color: color-mix(in srgb, var(--td-brand-color) 28%, transparent);
  background: color-mix(in srgb, var(--td-brand-color) 10%, var(--td-bg-color-container));
  color: var(--td-brand-color);
}

.api-key-capability-chips {
  display: flex;
  max-width: 100%;
  flex-wrap: wrap;
  gap: 5px;
}

.api-key-capability-chip {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  height: 22px;
  padding: 0 8px;
  border-radius: var(--app-radius-sm);
  background: color-mix(in srgb, var(--td-success-color) 10%, var(--td-bg-color-container));
  color: var(--td-success-color);
  font-size: var(--app-text-sm);
  font-weight: 500;
  line-height: 20px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.api-key-capability-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.api-key-capability-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 0 2px;
}

.api-key-capability-group + .api-key-capability-group {
  border-top: 1px solid var(--td-component-stroke);
}

.api-key-capability-group__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 24px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.api-key-capability-group__items {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.api-key-capability-item .scope-hint {
  margin-top: 2px;
}

.api-key-knowledge-scope,
.api-key-created-at {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}


.api-key-dialog {
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.api-key-dialog-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 0 16px;
  border-bottom: 1px solid var(--td-component-stroke);

  &:first-child {
    padding-top: 0;
  }

  &:last-child {
    border-bottom: none;
  }

  &__label {
    min-width: 0;

    label {
      display: flex;
      align-items: center;
      gap: 8px;
      color: var(--td-text-color-primary);
      font-size: var(--app-text-base);
      font-weight: 600;
      line-height: 1.45;

      &::before {
        content: '';
        flex-shrink: 0;
        width: 3px;
        height: 14px;
        border-radius: 2px;
        background: var(--td-brand-color);
      }
    }

    p {
      margin: 2px 0 0;
      color: var(--td-text-color-placeholder);
      font-size: var(--app-text-sm);
      line-height: 1.5;
    }
  }

  :deep(.t-input),
  :deep(.t-select__wrap) {
    border-radius: var(--app-radius-xs);
  }

  :deep(.t-input) {
    background-color: var(--td-bg-color-secondarycontainer);
    border-color: transparent;
    box-shadow: none !important;
  }

  :deep(.t-input:hover),
  :deep(.t-input.t-is-focused) {
    border-color: var(--td-component-border);
    background-color: var(--td-bg-color-container);
  }
}

.scope-hint {
  margin: 8px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 18px;
}

.principal-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 20px 0;
}

.principal-section__header {
  label {
    display: block;
    margin-bottom: 6px;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-lg);
    font-weight: 600;
  }

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    line-height: 1.55;
  }
}

.principal-section__scope {
  margin-top: 6px !important;
  color: var(--td-text-color-placeholder) !important;
  font-size: var(--app-text-sm) !important;
}

.mode-radio {
  width: fit-content;
  max-width: 100%;
}

.mode-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
  max-width: 760px;
}

.mode-callout {
  position: relative;
  padding: 12px 14px;
  border-radius: var(--app-radius-md);
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
  overflow: hidden;

  &--warning {
    border-color: var(--td-warning-color-3);
    background: var(--td-warning-color-1);
  }

  &__body {
    display: block;

    strong {
      display: block;
      margin-bottom: 5px;
      color: var(--td-text-color-primary);
      font-size: var(--app-text-md);
      font-weight: 600;
      line-height: 1.4;
    }

    margin: 0;
    p {
      margin: 0;
      color: var(--td-text-color-secondary);
      font-size: var(--app-text-sm);
      line-height: 1.6;
    }
  }
}

.principal-config {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container);
  overflow: hidden;
}

.config-row {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) auto;
  align-items: center;
  gap: 16px;
  padding: 14px 16px;

  & + & {
    border-top: 1px solid var(--td-component-stroke);
  }

  &__text {
    min-width: 0;

    label {
      display: block;
      color: var(--td-text-color-primary);
      font-size: var(--app-text-md);
      font-weight: 600;
      line-height: 1.4;
    }

    p {
      margin: 5px 0 0;
      color: var(--td-text-color-placeholder);
      font-size: var(--app-text-sm);
      line-height: 1.5;
    }
  }

  &__action {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    min-width: 52px;
  }

  &--switch {
    align-items: flex-start;
  }

  &--secret {
    grid-template-columns: minmax(220px, 0.55fr) minmax(360px, 1fr);
    align-items: center;
  }
}

.secret-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.secret-control {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;

  .mono-input {
    flex: 1 1 auto;
    min-width: 0;
  }

  .secret-mono-input {
    :deep(.t-input__suffix) {
      display: none;
    }
  }

  :deep(.t-button) {
    flex: 0 0 auto;
  }
}

.secret-saved-hint {
  margin: 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-warning-color);
}

@media (max-width: 640px) {
  .config-row {
    grid-template-columns: 1fr;
    align-items: stretch;
    gap: 10px;

    &__action {
      justify-content: flex-start;
    }
  }

  .fixed-header-name {
    width: 100%;
  }
}

.examples {
  width: 100%;
}

.snippet-tabs {
  margin-bottom: 8px;

  :deep(.t-tabs__nav) {
    min-height: 36px;
  }

  :deep(.t-tabs__nav-item) {
    font-size: var(--app-text-md);
    height: 36px;
    line-height: 36px;
    color: var(--td-text-color-secondary);
  }

  :deep(.t-tabs__nav-item.t-is-active) {
    color: var(--td-text-color-primary);
    font-weight: 500;
  }

  :deep(.t-tabs__bar) {
    background: var(--td-brand-color);
  }
}

.code-panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);
  overflow: hidden;

  &__toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 10px;
    border-bottom: 1px solid var(--td-component-stroke);
    background: var(--td-bg-color-container);
  }

  &__label {
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-text-color-secondary);
  }

  &__copy {
    flex-shrink: 0;

    :deep(.t-button__text) {
      display: inline-flex;
      align-items: center;
    }

    :deep(.t-icon) {
      display: inline-flex;
      align-items: center;
    }
  }

  &__pre {
    margin: 0;
    padding: 10px 12px;
    overflow: auto;
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-sm);
    line-height: 1.5;
    color: var(--td-text-color-primary);
    background: transparent;
  }
}

.mono-input :deep(input) {
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-sm);
}

.fixed-header-name {
  width: fit-content;
  max-width: 100%;
  padding: 7px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-sm);
  line-height: 18px;
  overflow-wrap: anywhere;
}

.mono-textarea :deep(.t-textarea__inner) {
  font-family: var(--app-font-family-mono);
  font-size: var(--app-text-sm);
}

.playground-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);

  &__info {
    min-width: 0;

    label {
      display: block;
      margin-bottom: 4px;
      color: var(--td-text-color-primary);
      font-size: var(--app-text-md);
      font-weight: 500;
    }

    p {
      margin: 0;
      color: var(--td-text-color-secondary);
      font-size: var(--app-text-sm);
      line-height: 1.5;
    }
  }
}

.playground-preview {
  width: 100%;
}

.drawer-form-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.drawer-form-label {
  display: block;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 500;
  line-height: 1.4;
}

.drawer-form-desc {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.5;

  &--error {
    color: var(--td-error-color);
  }
}

.footer-test-message {
  min-width: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.4;
}

.playground-empty {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.6;
}

.playground-results {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}

.playground-step {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);
  overflow: hidden;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 10px;
    border-bottom: 1px solid var(--td-component-stroke);
    background: var(--td-bg-color-container);
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);
    font-weight: 500;
  }

  pre {
    max-height: 240px;
    margin: 0;
    padding: 10px 12px;
    overflow: auto;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family-mono);
    font-size: var(--app-text-sm);
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
  }
}

@media (max-width: 780px) {
  .row {
    grid-template-columns: 1fr;
  }

  .mode-radio {
    width: 100%;

    :deep(.t-radio-group) {
      display: flex;
      width: 100%;
    }

    :deep(.t-radio-button) {
      flex: 1 1 0;
      min-width: 0;
    }
  }

  .mode-detail {
    max-width: none;
  }

  .playground-entry {
    flex-direction: column;
    align-items: stretch;
  }
}

.row-info {
  label {
    display: block;
    margin-bottom: 4px;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-lg);
    font-weight: 600;
  }

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    line-height: 1.5;
  }
}

.row-control,
.copy-field {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
</style>
