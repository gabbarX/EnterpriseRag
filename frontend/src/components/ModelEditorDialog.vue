<template>
  <SettingDrawer :visible="dialogVisible" :title="isEdit ? 'Edit Model' : 'Add Model'"
    :description="getModalDescription()" :icon="modelTypeIcon" :confirm-loading="saving" :cancel-disabled="saving"
    :confirm-text="'Save and close'"
    :close-on-overlay-click="!saving" :close-on-esc-keydown="!saving"
    @update:visible="(v: boolean) => dialogVisible = v" @confirm="handleConfirm" @cancel="handleCancel">

    <template v-if="formData.source === 'remote'" #footer-left>
      <t-button variant="outline" @click="checkRemoteAPI" :loading="checking"
        :disabled="saving || !formData.modelName || !formData.baseUrl">
        <template #icon>
          <t-icon v-if="!checking && remoteChecked && remoteAvailable" name="check-circle-filled"
            class="status-icon available" />
          <t-icon v-else-if="!checking && remoteChecked && !remoteAvailable" name="close-circle-filled"
            class="status-icon unavailable" />
        </template>
        {{ checking ? 'Testing...' : 'Test Connection' }}
      </t-button>
      <span v-if="remoteChecked" :class="['connection-status', remoteAvailable ? 'success' : 'error']">
        {{ remoteAvailable ? 'Connection succeeded' : 'Connection failed' }}
      </span>
    </template>

    <template #footer-extra>
      <div v-if="saveError" class="connection-result error" role="alert">
        <strong>{{ 'Failed to save model' }}</strong>
        <div class="connection-result__details" tabindex="0">{{ saveError }}</div>
      </div>
      <div v-if="formData.source === 'remote'" class="connection-feedback" aria-live="polite">
        <p class="connection-hint">{{ (isEdit ? 'Test the current connection settings using the API key saved separately.' : 'Test the current connection settings without saving first.') }}</p>
        <p v-if="remoteStale" class="connection-hint">{{ 'Settings changed. Run the test again.' }}</p>
        <div v-if="remoteChecked && !remoteAvailable" class="connection-result error">
          <div class="connection-result__header">
            <strong>{{ 'Connection failed' }}</strong>
            <t-button size="small" variant="text" @click="copyWithToast(remoteMessage, 'Copied')">
              {{ 'Copy' }}
            </t-button>
          </div>
          <div class="connection-result__details" tabindex="0">{{ remoteMessage }}</div>
        </div>
      </div>
    </template>

    <t-form :inert="saving || undefined" ref="formRef" :data="formData" :rules="rules" layout="vertical">

      <section v-if="!isEdit" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Model Type' }}</h4>
        <div class="model-type-options" role="radiogroup" :aria-label="'Model Type'">
          <button
            v-for="opt in modelTypeChoices"
            :key="opt.value"
            type="button"
            class="model-type-option"
            :class="{ 'is-active': activeModelType === opt.value }"
            role="radio"
            :aria-checked="activeModelType === opt.value"
            @click="selectModelType(opt.value)"
          >
            <t-icon :name="opt.icon" class="model-type-option__icon" />
            <span class="model-type-option__label">{{ opt.label }}</span>
          </button>
        </div>
      </section>

      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Source' }}</h4>

        <div class="form-item">
          <div class="source-options" role="radiogroup" :aria-label="'Model Source'">
            <button
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.source === 'remote' }"
              role="radio"
              :aria-checked="formData.source === 'remote'"
              @click="formData.source = 'remote'"
            >
              <t-icon name="cloud" class="source-option__icon" />
              <span class="source-option__label">{{ 'API' }}</span>
            </button>
            <button
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.source === 'local', 'is-disabled': ollamaServiceStatus === false || activeModelType === 'rerank' }"
              :disabled="ollamaServiceStatus === false || activeModelType === 'rerank'"
              role="radio"
              :aria-checked="formData.source === 'local'"
              @click="formData.source = 'local'"
            >
              <t-icon name="server" class="source-option__icon" />
              <span class="source-option__label">{{ 'Ollama' }}</span>
            </button>
          </div>

          <div v-if="activeModelType === 'rerank'" class="ollama-unavailable-tip rerank-tip">
            <t-icon name="info-circle-filled" class="tip-icon info" />
            <span class="tip-text">{{ 'Ollama does not support ReRank models, please use a remote API instead' }}</span>
          </div>

          <div v-else-if="shouldShowOllamaUnavailableTip(formData.source, activeModelType, ollamaServiceStatus)"
            class="ollama-unavailable-tip">
            <t-icon name="error-circle-filled" class="tip-icon" />
            <span class="tip-text">{{ 'Ollama service is unavailable, local models cannot be selected' }}</span>
            <t-button variant="text" size="small" @click="goToOllamaSettings" class="tip-link">
              <template #icon><t-icon name="jump" /></template>
              {{ 'Open Settings' }}
            </t-button>
          </div>
        </div>

        <div v-if="formData.source === 'local'" class="form-item">
          <label class="form-label required">{{ 'Model Name' }}</label>
          <div class="model-select-row">
            <t-select v-model="formData.modelName" :loading="loadingOllamaModels" :class="{ 'downloading': downloading }"
              :style="downloading ? `--progress: ${downloadProgress}%` : ''" filterable :filter="handleModelFilter"
              :placeholder="'Search models...'" @focus="loadOllamaModels"
              @visible-change="handleDropdownVisibleChange">
              <t-option v-for="model in filteredOllamaModels" :key="model.name" :value="model.name" :label="model.name">
                <div class="model-option">
                  <t-icon name="check-circle-filled" class="downloaded-icon" />
                  <span class="model-name">{{ model.name }}</span>
                  <span class="model-size">{{ formatModelSize(model.size) }}</span>
                </div>
              </t-option>

              <t-option v-if="showDownloadOption" :value="`__download__${searchKeyword}`"
                :label="`Download: ${searchKeyword}`" class="download-option">
                <div class="model-option download">
                  <t-icon name="download" class="download-icon" />
                  <span class="model-name">{{ `Download: ${searchKeyword}` }}</span>
                </div>
              </t-option>

              <template v-if="downloading" #suffix>
                <div class="download-suffix">
                  <t-icon name="loading" class="spinning" />
                  <span class="progress-text">{{ downloadProgress.toFixed(1) }}%</span>
                </div>
              </template>
            </t-select>

            <t-button variant="text" size="small" :loading="loadingOllamaModels" @click="refreshOllamaModels"
              class="refresh-btn">
              <t-icon name="refresh" />
              {{ 'Refresh List' }}
            </t-button>
          </div>
        </div>
      </section>

      <template v-if="formData.source === 'remote'">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Provider Settings' }}</h4>

          <div class="form-item">
            <label class="form-label">{{ 'Provider' }}</label>
            <t-select v-model="formData.provider" :placeholder="'Select model provider'"
              :loading="loadingProviders" filterable
              @change="handleProviderChange"
              :popup-props="{ overlayClassName: 'wk-popover provider-select-popup', overlayInnerStyle: matchTriggerWidth }">
              <!--
                Selected value: icon + localised name (the description only shows inside the
                dropdown, to keep the input from getting too long). When the catalogue no
                longer has this provider (vendor retired / legacy data) fall back to the raw
                id, otherwise the input goes entirely blank and looks like "no provider
                selected" even though it is still submitted on save.
              -->
              <template #valueDisplay>
                <span v-if="formData.provider" class="provider-value">
                  <img v-if="selectedProviderIcon" :src="selectedProviderIcon" class="provider-icon" alt="" />
                  <span class="provider-value__name">{{ selectedProviderDisplayLabel }}</span>
                </span>
              </template>
              <!--
                show-overflow-tooltip=false: by default TDesign floats a small bubble with
                the full label on hover, but these options are already two lines (name +
                description) and never truncate, so the tooltip only fights the highlight
                background. Turn it off.
              -->
              <t-option v-for="opt in providerOptions" :key="opt.value" :value="opt.value"
                :label="providerDisplayLabel(opt)" :show-overflow-tooltip="false">
                <div class="provider-option">
                  <img v-if="providerIcon(opt)" :src="providerIcon(opt)" class="provider-icon" alt="" />
                  <span v-else class="provider-icon provider-icon--placeholder" aria-hidden="true">
                    {{ providerDisplayLabel(opt).slice(0, 1) }}
                  </span>
                  <div class="provider-option__text">
                    <span class="provider-name">{{ providerDisplayLabel(opt) }}</span>
                    <span class="provider-desc">{{ providerDisplayDescription(opt) }}</span>
                  </div>
                </div>
              </t-option>
            </t-select>
            <p v-if="vendorDocLink" class="form-desc provider-doc-link">
              <a :href="vendorDocLink" target="_blank" rel="noopener noreferrer">
                {{ `Read the ${selectedProviderDisplayLabel} model docs` }}
                <t-icon name="jump" size="12px" />
              </a>
            </p>
          </div>

          <div class="form-item">
            <label class="form-label required">{{ 'Model Name' }}</label>
            <!--
              A plain input when there is no built-in catalogue: with zero options TDesign's
              select hides the whole popup (hideEmptyPopup), including the creatable "create"
              row, so the user can type but has no way to commit and the input is simply lost
              on blur. Custom (OpenAI-compatible endpoint) is exactly that case, which is why
              a model name could not be entered there at all.
            -->
            <t-input
              v-if="catalogModelOptions.length === 0"
              v-model="formData.modelName"
              :placeholder="getModelNamePlaceholder()"
              clearable
              autocomplete="off"
              spellcheck="false"
            />
            <t-select
              v-else
              v-model="formData.modelName"
              filterable
              creatable
              clearable
              :placeholder="getModelNamePlaceholder()"
              :popup-props="{ overlayClassName: 'wk-popover catalog-model-select-popup', overlayInnerStyle: matchTriggerWidth }"
              @create="handleCatalogModelCreate"
              @change="handleCatalogModelChange"
            >
              <t-option v-for="opt in catalogModelOptions" :key="opt.value" :value="opt.value" :label="opt.label"
                :show-overflow-tooltip="false">
                <div class="catalog-model-option">
                  <span class="catalog-model-option__name">{{ opt.label }}</span>
                  <span v-if="opt.value !== opt.label" class="catalog-model-option__id">{{ opt.value }}</span>
                  <span class="catalog-model-option__badges">
                    <span v-if="opt.contextWindow" class="catalog-badge">{{ opt.contextWindow }}</span>
                    <span v-if="opt.dimension" class="catalog-badge">{{ 'Vector Dimension' }} {{ opt.dimension }}</span>
                    <span v-if="opt.reasoning" class="catalog-badge catalog-badge--accent">{{ 'Reasoning' }}</span>
                    <span v-if="opt.vision" class="catalog-badge catalog-badge--accent">{{ 'Vision' }}</span>
                  </span>
                </div>
              </t-option>
            </t-select>
            <p v-if="catalogModelOptions.length > 0" class="form-desc">{{ 'Pick a model from the vendor catalog or type a custom model name.' }}</p>
          </div>

          <div class="form-item">
            <label class="form-label">{{ 'Display name (optional)' }}</label>
            <t-input v-model="formData.displayName" :placeholder="'e.g. Support QA model'" />
            <p class="form-desc">{{ 'Used only in the UI. Runtime calls still use the model name above.' }}</p>
          </div>

          <div class="form-item">
            <label class="form-label required">{{ 'Base URL' }}</label>
            <t-input v-model="formData.baseUrl" :placeholder="getBaseUrlPlaceholder()" />
          </div>

          <div class="form-item">
            <label class="form-label" :class="{ required: apiKeyRequired }">{{ apiKeyLabel }}</label>
            <!--
              Edit mode: credentials live behind the /credentials subresource
              of the model — managed by the shared CredentialResource card,
              which now renders an INPUT-LOOKING row (32px tall, same border
              + radius as t-input) so it sits flush with the Base URL field
              above and the Custom headers controls below - no more
              "card inside a card" feel.
              Create mode: the resource doesn't exist yet, so we render a
              plain password input with a leading lock icon; TDesign's password
              input provides the show/hide toggle.
            -->
            <CredentialResource v-if="isEdit && props.modelData?.id" :api="credentialApi" :fields="credentialFields"
              :meta="credentialMeta" @changed="invalidateConnectionTest()" />
            <t-input v-else v-model="formData.apiKey" type="password"
              :placeholder="apiKeyPlaceholder"
              class="api-key-input" autocomplete="off" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <p v-if="apiKeyHint" class="form-desc">{{ apiKeyHint }}</p>
          </div>

          <div v-if="secretExtraField && !isEdit" class="form-item">
            <label class="form-label" :class="{ required: secretExtraField.required }">{{ extraFieldDisplayLabel(secretExtraField) }}</label>
            <t-input v-model="formData.appSecret" type="password"
              :placeholder="secretExtraField.placeholder || ''" autocomplete="off" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <p v-if="secretExtraField.placeholder" class="form-desc">{{ secretExtraField.placeholder }}</p>
          </div>

          <div v-for="field in plainExtraFields" :key="field.key" class="form-item">
            <label class="form-label" :class="{ required: field.required }">{{ extraFieldDisplayLabel(field) }}</label>
            <div v-if="field.type === 'boolean'" class="vision-toggle">
              <t-switch :model-value="extraConfigBool(field.key)"
                @update:model-value="(v: boolean) => setExtraConfig(field.key, v ? 'true' : 'false')" />
              <span v-if="extraFieldDisplayPlaceholder(field)" class="form-desc form-desc--inline">{{ extraFieldDisplayPlaceholder(field) }}</span>
            </div>
            <t-select v-else-if="field.type === 'select'" :model-value="formData.extraConfig[field.key] || ''"
              :placeholder="extraFieldDisplayPlaceholder(field)" clearable
              @update:model-value="(v: string) => setExtraConfig(field.key, v)">
              <t-option v-for="opt in (field.options || [])" :key="opt.value" :value="opt.value"
                :label="extraFieldDisplayOptionLabel(opt)" />
            </t-select>
            <t-input v-else-if="field.type === 'number'" :model-value="formData.extraConfig[field.key] || ''"
              type="number" :placeholder="extraFieldDisplayPlaceholder(field)"
              @update:model-value="(v: string | number) => setExtraConfig(field.key, String(v ?? ''))" />
            <t-input v-else-if="field.type === 'password'" :model-value="formData.extraConfig[field.key] || ''"
              type="password" :placeholder="extraFieldDisplayPlaceholder(field)" autocomplete="off" spellcheck="false"
              @update:model-value="(v: string) => setExtraConfig(field.key, v)">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <t-input v-else :model-value="formData.extraConfig[field.key] || ''"
              :placeholder="extraFieldDisplayPlaceholder(field)"
              @update:model-value="(v: string) => setExtraConfig(field.key, v)" />
          </div>

          <div class="form-item">
            <div class="custom-headers-header">
              <label class="form-label" style="margin-bottom: 0;">{{ 'Custom Request Headers (optional)' }}</label>
              <t-button variant="text" size="small" theme="primary" @click="addCustomHeader">
                <template #icon><t-icon name="add" /></template>
                {{ 'Add Header' }}
              </t-button>
            </div>
            <p class="form-desc custom-headers-desc">{{ 'Extra HTTP headers appended to requests to the remote model API (e.g. for enterprise gateway auth or tracing). Reserved headers like Authorization / Content-Type are ignored.' }}</p>
            <div v-if="formData.customHeaders && formData.customHeaders.length > 0" class="custom-headers-list">
              <div v-for="(item, idx) in formData.customHeaders" :key="idx" class="custom-header-row">
                <t-input v-model="item.key" :placeholder="'Header name'"
                  class="custom-header-key" />
                <t-input v-model="item.value" :placeholder="'Header value'"
                  class="custom-header-value" />
                <t-button variant="text" shape="square" size="small" class="custom-header-remove"
                  @click="removeCustomHeader(idx)" :aria-label="'Delete'">
                  <t-icon name="close" />
                </t-button>
              </div>
            </div>
          </div>

          <!--
            Connection diagnostics: the effective protocol / thinking format / support level /
            whether it is in the catalogue / context window, resolved live by the backend
            catalogue, read-only. Refreshed 400ms after the provider, model name or Base URL
            changes. Only meaningful for chat / VLM - Embedding, ReRank and ASR have neither a
            thinking level nor a context window, and showing them would suggest those values
            apply.
          -->
          <div v-if="showResolvedPanel" class="form-item">
            <div class="resolved-panel" aria-live="polite">
              <div class="resolved-panel__header">
                <t-icon name="system-code" class="resolved-panel__icon" />
                <span class="resolved-panel__title">{{ 'How this model is called' }}</span>
                <t-icon v-if="resolving" name="loading" class="spinning resolved-panel__loading" />
              </div>
              <p v-if="!resolved && !resolving && !resolveFailed" class="form-desc">{{ 'Fill in the vendor and model name to see how this model will be called' }}</p>
              <p v-else-if="resolveFailed" class="form-desc form-desc--warn">
                {{ 'Resolve failed' }}<template v-if="resolveError">: {{ resolveError }}</template>
              </p>
              <dl v-else-if="resolved" class="resolved-panel__grid">
                <dt>{{ 'Request protocol' }}</dt>
                <dd><code>{{ resolved.api }}</code></dd>
                <dt>{{ 'Capabilities from' }}</dt>
                <dd>
                  <span class="catalog-badge" :class="{ 'catalog-badge--accent': resolved.cataloged }">
                    {{ resolved.cataloged ? 'Built-in model profile' : 'Vendor defaults (model not in the catalog)' }}
                  </span>
                </dd>
                <template v-if="resolved.url">
                  <dt>{{ 'Request endpoint' }}</dt>
                  <dd><code class="resolved-endpoint">{{ resolved.url }}</code></dd>
                </template>
                <template v-if="resolved.remote_model && resolved.remote_model !== formData.modelName">
                  <dt>{{ 'Remote model name' }}</dt>
                  <dd><code>{{ resolved.remote_model }}</code></dd>
                </template>
                <template v-if="isChatLike">
                  <dt>{{ 'Thinking switch sent as' }}</dt>
                  <dd><code>{{ resolved.capabilities?.thinking_format || '-' }}</code></dd>
                  <dt>{{ 'Selectable effort' }}</dt>
                  <dd>
                    <template v-if="resolvedThinkingLevels.length > 0">
                      <span v-for="level in resolvedThinkingLevels" :key="level" class="catalog-badge">
                        {{ levelLabel(level) }}
                      </span>
                    </template>
                    <span v-else class="form-desc form-desc--inline">{{ 'This model cannot think' }}</span>
                  </dd>
                  <template v-if="resolved.capabilities?.context_window">
                    <dt>{{ 'Context Window' }}</dt>
                    <dd>{{ formatTokenCount(resolved.capabilities.context_window) }}</dd>
                  </template>
                  <template v-if="resolved.capabilities?.max_output_tokens">
                    <dt>{{ 'Max output tokens' }}</dt>
                    <dd>{{ formatTokenCount(resolved.capabilities.max_output_tokens) }}</dd>
                  </template>
                </template>
              </dl>
            </div>
          </div>

          <!--
            Connection test action moved to the drawer footer (footer-left
            slot above) so primary actions live in one row at the bottom.
          -->
        </section>
      </template>

      <section v-if="['embedding', 'chat', 'vllm'].includes(activeModelType) || formData.source === 'remote'" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Advanced Options' }}</h4>

        <div v-if="activeModelType === 'embedding'" class="form-item">
          <label class="form-label">{{ 'Vector Dimension' }}</label>
          <div class="dimension-control">
            <t-input v-model.number="formData.dimension" type="number" :min="128" :max="4096"
              :placeholder="'e.g. 1536'"
              :disabled="!formData.supportsDimensionOverride || (formData.source === 'local' && checking)" />
            <t-button v-if="formData.source === 'local' && formData.modelName" variant="text" size="small"
              :loading="checking" @click="checkOllamaDimension" class="dimension-check-btn">
              <t-icon name="refresh" />
              {{ 'Detect Dimension' }}
            </t-button>
          </div>
          <p v-if="dimensionChecked && dimensionMessage" class="dimension-hint" :class="{ success: dimensionSuccess }">
            {{ dimensionMessage }}
          </p>
        </div>

        <div v-if="activeModelType === 'embedding'" class="form-item">
          <label class="form-label">{{ 'Custom Output Dimension' }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.supportsDimensionOverride" />
            <span class="form-desc form-desc--inline">{{ 'Enable only if the provider documentation says this model accepts a dimensions parameter.' }}</span>
          </div>
        </div>

        <!-- Chat / VLM: context window. Agent compaction sizes itself from this. -->
        <div v-if="activeModelType === 'chat' || activeModelType === 'vllm'" class="form-item">
          <label class="form-label">{{ 'Context Window' }}</label>
          <t-input v-model.number="formData.contextWindow" type="number" :min="1024" :max="10000000"
            :placeholder="`Default ${DEFAULT_MODEL_CONTEXT_WINDOW}`" />
          <p class="form-desc">{{ 'How many tokens this model can take in one request. Agent history compaction uses this limit. Leave empty for the default 200000 (200K). Use the provider’s real window — a larger guess means compaction never fires and the provider rejects the request.' }}</p>
        </div>

        <!-- Chat: supports vision toggle (VLLM models are inherently multimodal) -->
        <div v-if="activeModelType === 'chat'" class="form-item">
          <label class="form-label">{{ 'Supports Vision / Multimodal' }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.supportsVision" />
            <span class="form-desc form-desc--inline">{{ 'Whether the model accepts image and multimodal input' }}</span>
          </div>
        </div>

        <!-- Chat / VLM: max output tokens (catalog default when empty). -->
        <div v-if="activeModelType === 'chat' || activeModelType === 'vllm'" class="form-item">
          <label class="form-label">{{ 'Max output tokens' }}</label>
          <t-input v-model.number="formData.maxOutputTokens" type="number" :min="1" :max="10000000"
            :placeholder="'Leave empty for the catalog default'" />
          <p class="form-desc">{{ 'Output cap per response. Leave empty to use the catalog default for this model.' }}</p>
        </div>

        <!--
          Background concurrency cap for this model. Only chat / embedding / vllm
          are gated by the governor (see internal/models/limiter), so we surface
          it just for those three. 0 = fall back to the global default.
        -->
        <div v-if="['embedding', 'chat', 'vllm'].includes(activeModelType)" class="form-item">
          <label class="form-label">{{ 'Background concurrency limit' }}</label>
          <t-input v-model.number="formData.maxConcurrency" type="number" :min="0" :max="4096"
            :placeholder="'0 = use global default'" />
          <p class="form-desc">{{ 'Caps concurrent background (ingestion/enrichment) calls to this model, shared per model across all replicas. 0 or empty falls back to the global default; interactive chat is never affected.' }}</p>
        </div>

        <template v-if="formData.source === 'remote'">
          <button type="button" class="advanced-toggle" :aria-expanded="advancedOpen" @click="advancedOpen = !advancedOpen">
            <t-icon name="chevron-right" class="toggle-arrow" :class="{ open: advancedOpen }" />
            <span>{{ 'Advanced' }}</span>
          </button>

          <template v-if="advancedOpen">
            <!--
              extra_config.api only selects the chat protocol. The embedding / rerank rows do
              not read it: an embedding protocol override goes in the compat JSON below
              ("api"), where the value is the vector protocol.
            -->
            <div v-if="isChatLike" class="form-item">
              <label class="form-label">{{ 'Protocol override' }}</label>
              <t-select :model-value="formData.extraConfig.api || ''" clearable
                @update:model-value="(v: string) => setExtraConfig('api', v)">
                <t-option value="" :label="'Auto (from vendor / URL)'" />
                <t-option v-for="api in PROTOCOL_OPTIONS" :key="api" :value="api" :label="api" />
              </t-select>
              <p class="form-desc">{{ 'Force a request protocol; normally unnecessary.' }}</p>
            </div>

            <div class="form-item">
              <label class="form-label">{{ 'Remote model name' }}</label>
              <t-input :model-value="formData.extraConfig.remote_model_name || ''"
                :placeholder="'Leave empty to use the model name'"
                @update:model-value="(v: string) => setExtraConfig('remote_model_name', v)" />
              <p class="form-desc">{{ 'Model id actually sent to the vendor when it differs from the name above.' }}</p>
            </div>

            <div v-if="showLegacyThinkingControl" class="form-item">
              <label class="form-label">{{ 'Thinking parameter format (legacy)' }}</label>
              <t-select :model-value="formData.thinkingControl || ''"
                @update:model-value="(v: string) => formData.thinkingControl = v">
                <t-option value="" :label="'Follow catalog default (recommended)'" />
                <t-option v-for="opt in LEGACY_THINKING_CONTROL_VALUES" :key="opt" :value="opt"
                  :label="opt === 'none' ? 'Do not send thinking fields' : opt" />
              </t-select>
              <p class="form-desc form-desc--warn">{{ 'This model still carries a legacy thinking_control setting. Choose \u0022Follow catalog default\u0022 to let the catalog decide.' }}</p>
            </div>

            <div class="form-item">
              <label class="form-label">{{ 'Protocol compat override (JSON)' }}</label>
              <t-textarea v-model="formData.specCompat" :autosize="{ minRows: 3, maxRows: 10 }"
                :placeholder="'{\'{\'} \u0022max_tokens_field\u0022: \u0022max_tokens\u0022 {\'}\'}'" class="compat-textarea"
                :status="specCompatError ? 'error' : undefined" />
              <p v-if="specCompatError" class="form-desc form-desc--error">{{ 'Invalid JSON' }}: {{ specCompatError }}</p>
              <p v-else class="form-desc">{{ 'Compat switches merged over the catalog defaults for the resolved protocol; see catalog/compat.go on the backend. Leave empty for no override.' }}</p>
            </div>
          </template>
        </template>
      </section>

    </t-form>
  </SettingDrawer>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import {
  checkOllamaModels, checkRemoteModel, testEmbeddingModel, checkRerankModel, checkASRModel, listOllamaModels,
  downloadOllamaModel, getDownloadProgress, checkOllamaStatus, resolveModelCatalog,
  type OllamaModelInfo, type ModelProviderOption, type ModelProviderExtraField,
  type ModelProviderExtraFieldOption, type ModelCatalogEntry,
  type ResolvedModelCatalog,
} from '@/api/initialization'
import {
  putModelCredentials,
  deleteModelCredentialField,
  type ModelCredentialField,
  type ModelSpecOverride,
} from '@/api/model'
import { useUIStore } from '@/stores/ui'
import { useModelProvidersStore } from '@/stores/modelProviders'
import {
  credentialLabelForModelType,
  extraFieldLabel,
  extraFieldOptionLabel,
  extraFieldPlaceholder,
  extraFieldsForModelType,
  pickLocalized,
  providerDescription,
  providerIcon,
  providerLabel,
} from '@/stores/modelProvidersState'
import { levelLabel, supportedLevels } from '@/utils/reasoningEffort'
import { DEFAULT_MODEL_CONTEXT_WINDOW, formatTokenCount } from '@/utils/contextWindow'
import { copyWithToast } from '@/utils/clipboard'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import CredentialResource, {
  type CredentialFieldDef,
  type CredentialResourceApi,
} from '@/components/credentials/CredentialResource.vue'
import { shouldShowOllamaUnavailableTip } from '@/components/modelEditorSourceState'

interface CustomHeaderItem {
  key: string
  value: string
}

interface ModelFormData {
  id: string
  name: string
  source: 'local' | 'remote'
  provider?: string // Provider identifier: openai, anthropic, gemini, openrouter, generic
  modelName: string
  displayName?: string
  baseUrl?: string
  apiKey?: string
  dimension?: number
  supportsDimensionOverride?: boolean
  interfaceType?: 'ollama' | 'openai'
  isDefault: boolean
  supportsVision?: boolean
  contextWindow?: number
  maxConcurrency?: number
  maxOutputTokens?: number
  /**
   * Legacy extra_config.thinking_control (none | enable_thinking | thinking_type
   * | chat_template_kwargs). Only rows saved by older UIs carry it; the catalog
   * decides the encoding otherwise. Empty string = drop the key on save.
   */
  thinkingControl?: string
  /**
   * Provider-specific extra_config entries: vendor-declared extra fields
   * (api_version, region, ...) plus the advanced overrides `api` and
   * `remote_model_name`. thinking_control lives in its own field above.
   */
  extraConfig: Record<string, string>
  /** parameters.spec.compat as JSON text (validated before save). */
  specCompat?: string
  /** Other parameters.spec fields preserved verbatim from the loaded row. */
  spec?: ModelSpecOverride | null
  customHeaders?: CustomHeaderItem[]
  appSecret?: string
}

/** Protocols the backend accepts in extra_config.api (internal/models/api.API). */
const PROTOCOL_OPTIONS = [
  'openai-completions',
  'openai-responses',
  'anthropic-messages',
  'google-generative-ai',
] as const

/** Legacy thinking_control values still honoured by catalog.Resolve. */
const LEGACY_THINKING_CONTROL_VALUES = ['none', 'enable_thinking', 'thinking_type', 'chat_template_kwargs'] as const

/** Keys of extra_config that are edited by dedicated controls, not the generic renderer. */
const RESERVED_EXTRA_CONFIG_KEYS = new Set(['thinking_control'])

type EditorModelType = 'chat' | 'embedding' | 'rerank' | 'vllm' | 'asr'

interface Props {
  visible: boolean
  modelType: EditorModelType
  modelData?: ModelFormData | null
  saveModel: (data: ModelFormData & { modelType?: EditorModelType }) => Promise<void>
}

const uiStore = useUIStore()
const providersStore = useModelProvidersStore()

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  modelData: null
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
}>()

const draftModelType = ref<EditorModelType>(props.modelType)

const isEdit = computed(() => !!props.modelData)

const activeModelType = computed(() => (
  isEdit.value ? props.modelType : draftModelType.value
))

const modelTypeChoices = computed(() => ([
  { value: 'chat' as const, label: 'Chat', icon: 'chat' },
  { value: 'embedding' as const, label: 'Embedding', icon: 'chart-bubble' },
  { value: 'rerank' as const, label: 'ReRank', icon: 'filter-sort' },
  { value: 'vllm' as const, label: 'Vision', icon: 'image' },
  { value: 'asr' as const, label: 'Speech', icon: 'sound' },
]))

const loadingProviders = computed(() => providersStore.isLoading(activeModelType.value))

const loadProviders = async () => {
  try {
    await providersStore.ensureLoaded(activeModelType.value)
  } catch (error) {
    console.error('Failed to load providers from API', error)
  }
}

const providerOptions = computed<ModelProviderOption[]>(() => providersStore.providersFor(activeModelType.value))

const selectedProvider = computed<ModelProviderOption | undefined>(() => {
  const id = formData.value.provider
  if (!id) return undefined
  return providerOptions.value.find(p => p.value === id) || providersStore.providerById(id)
})

const currentLocale = computed(() => 'en-US')
const providerDisplayLabel = (p: ModelProviderOption) => providerLabel(p, currentLocale.value)
const providerDisplayDescription = (p: ModelProviderOption) => providerDescription(p, currentLocale.value)
const selectedProviderIcon = computed(() => providerIcon(selectedProvider.value))
/** Localized vendor name, or the raw stored id when the catalog no longer has it. */
const selectedProviderDisplayLabel = computed(() => (
  selectedProvider.value
    ? providerDisplayLabel(selectedProvider.value)
    : (formData.value.provider || '')
))
/**
 * Pin a select's dropdown to the width of its input.
 *
 * TDesign sizes a select popup as max(popup content, trigger)
 * (select-input/hooks/useOverlayInnerStyle), so one long option — a vendor
 * whose description lists half a dozen model ids — stretches the whole menu
 * past the field it belongs to, leaving a wide band of empty space and
 * pushing the tick mark far from the text. Passing a function here replaces
 * that matching outright (the hook keeps a function as-is), and the option
 * rows already ellipsize, so the long ones simply truncate.
 */
const matchTriggerWidth = (triggerElement: HTMLElement) => ({
  width: `${triggerElement.offsetWidth}px`,
})

const extraFieldDisplayLabel = (field: ModelProviderExtraField) => extraFieldLabel(field, currentLocale.value)
const extraFieldDisplayPlaceholder = (field: ModelProviderExtraField) =>
  extraFieldPlaceholder(field, currentLocale.value)
const extraFieldDisplayOptionLabel = (option: ModelProviderExtraFieldOption) =>
  extraFieldOptionLabel(option, currentLocale.value)

/**
 * Vendors whose API is not a bearer-token API name their first credential
 * themselves (LKEAP rerank takes a SecretId, Volcengine rerank an Access Key
 * ID). Falling back to the generic "API Key" wording is what led operators to
 * paste an `sk-` token into a signature field.
 */
const credentialLabel = computed(() =>
  credentialLabelForModelType(selectedProvider.value?.credentialLabels, activeModelType.value),
)
const apiKeyLabel = computed(() => {
  const label = credentialLabel.value
  return label ? pickLocalized(label.labels, currentLocale.value, label.label) : 'API Key (optional)'
})
const apiKeyRequired = computed(
  () => credentialLabel.value?.required === true || (!!selectedProvider.value?.requiresAuth && !isEdit.value),
)
const apiKeyHint = computed(() => {
  const label = credentialLabel.value
  if (!label?.hint) return ''
  return pickLocalized(label.hints, currentLocale.value, label.hint)
})

const visibleExtraFields = computed<ModelProviderExtraField[]>(() =>
  extraFieldsForModelType(selectedProvider.value?.extraFields, activeModelType.value)
    .filter(field => !RESERVED_EXTRA_CONFIG_KEYS.has(field.key)),
)
/**
 * A credential-bearing vendor field. `secret` is the backend's own marker;
 * `type: 'password'` is treated the same way so a vendor that forgets the flag
 * still cannot get its value echoed back from GET /models into extra_config.
 */
const isSecretExtraField = (field: ModelProviderExtraField) => field.secret === true || field.type === 'password'
// Only one credential slot (app_secret) exists per model, so the first such
// field wins; vendors today declare at most one (LKEAP / Volcengine rerank).
const secretExtraField = computed<ModelProviderExtraField | undefined>(() =>
  visibleExtraFields.value.find(isSecretExtraField),
)
const plainExtraFields = computed<ModelProviderExtraField[]>(() =>
  visibleExtraFields.value.filter(field => !isSecretExtraField(field)),
)

/** extra_config keys that belong to the connection, not to the vendor. */
const VENDOR_NEUTRAL_EXTRA_CONFIG_KEYS = ['api', 'remote_model_name'] as const

/** What survives a vendor (or model-type) switch: everything else is vendor-specific. */
const keepVendorNeutralExtraConfig = (): Record<string, string> => {
  const keep: Record<string, string> = {}
  for (const key of VENDOR_NEUTRAL_EXTRA_CONFIG_KEYS) {
    const value = formData.value.extraConfig?.[key]
    if (value) keep[key] = value
  }
  return keep
}

const setExtraConfig = (key: string, value: string | null | undefined) => {
  const next = { ...(formData.value.extraConfig || {}) }
  const normalized = value == null ? '' : String(value)
  if (normalized === '') delete next[key]
  else next[key] = normalized
  formData.value.extraConfig = next
}

const extraConfigBool = (key: string) => {
  const raw = (formData.value.extraConfig?.[key] || '').trim().toLowerCase()
  return raw === 'true' || raw === '1' || raw === 'yes'
}

/** Pre-fill vendor defaults for fields the user has not touched. */
const applyExtraFieldDefaults = () => {
  for (const field of plainExtraFields.value) {
    if (!field.default) continue
    if ((formData.value.extraConfig?.[field.key] ?? '') === '') {
      setExtraConfig(field.key, field.default)
    }
  }
}

interface CatalogModelOption {
  value: string
  label: string
  contextWindow: string
  dimension?: number
  reasoning: boolean
  vision: boolean
  entry?: ModelCatalogEntry
}

const catalogEntries = computed<ModelCatalogEntry[]>(() => {
  // The list is already scoped: providers are fetched per model type, so the
  // backend returned exactly the entries that type can use. Filtering again
  // accepts images, so it arrives typed "chat" and every one of them was
  // dropped, leaving the picker empty for every vendor.
  return selectedProvider.value?.models || []
})

const catalogModelOptions = computed<CatalogModelOption[]>(() => {
  const options: CatalogModelOption[] = catalogEntries.value.map(entry => ({
    value: entry.id,
    label: entry.name || entry.id,
    contextWindow: entry.context_window ? formatTokenCount(entry.context_window) : '',
    dimension: entry.dimension || undefined,
    reasoning: !!entry.reasoning || (entry.thinking_levels?.length ?? 0) > 0,
    vision: Array.isArray(entry.input) && entry.input.includes('image'),
    entry,
  }))
  const current = (formData.value.modelName || '').trim()
  if (current && !options.some(o => o.value === current)) {
    options.unshift({ value: current, label: current, contextWindow: '', reasoning: false, vision: false })
  }
  return options
})

const findCatalogEntry = (name: string) => catalogEntries.value.find(m => m.id === name)

/**
 * Where to read about what is configured right now.
 *
 * Prefer the page the selected model's facts were taken from — that is the
 * page listing its context window, thinking levels and price — and fall back
 * to the vendor's own site when the model is not in the catalog. Both come
 * from the catalog, so a new vendor gets the link without a UI change.
 */
const vendorDocLink = computed(() => {
  const provider = selectedProvider.value
  if (!provider) return ''
  const entry = findCatalogEntry((formData.value.modelName || '').trim())
  const url = entry?.source || provider.website || ''
  return /^https?:\/\//i.test(url) ? url : ''
})

/**
 * What applyCatalogEntry filled in for the model currently selected.
 *
 * Switching vendor has to take those values back — a context window from the
 * case this field warns about — but it must not touch a number the operator
 * typed. Remembering what was filled, and only clearing a field that still
 * holds it, separates the two.
 */
const catalogFilled = ref<Partial<ModelFormData>>({})

/** Fill blank capability fields from a catalog entry the user just picked. */
const applyCatalogEntry = (entry: ModelCatalogEntry) => {
  if (activeModelType.value === 'chat' || activeModelType.value === 'vllm') {
    if (!formData.value.contextWindow && entry.context_window) {
      formData.value.contextWindow = entry.context_window
      catalogFilled.value.contextWindow = entry.context_window
    }
    if (!formData.value.maxOutputTokens && entry.max_output_tokens) {
      formData.value.maxOutputTokens = entry.max_output_tokens
      catalogFilled.value.maxOutputTokens = entry.max_output_tokens
    }
  }
  if (activeModelType.value === 'chat' && !formData.value.supportsVision && Array.isArray(entry.input) && entry.input.includes('image')) {
    formData.value.supportsVision = true
    catalogFilled.value.supportsVision = true
  }
  if (activeModelType.value === 'embedding' && !formData.value.dimension && entry.dimension) {
    formData.value.dimension = entry.dimension
    catalogFilled.value.dimension = entry.dimension
  }
}

const handleCatalogModelCreate = (value: string | number | boolean | bigint) => {
  formData.value.modelName = String(value ?? '').trim()
}

const handleCatalogModelChange = (value: unknown) => {
  const name = typeof value === 'string' ? value.trim() : ''
  if (!name) return
  const entry = findCatalogEntry(name)
  if (entry) applyCatalogEntry(entry)
}

const resolved = ref<ResolvedModelCatalog | null>(null)
const resolving = ref(false)
const resolveFailed = ref(false)
/** Human-readable detail of the last failure; '' when the error carried none. */
const resolveError = ref('')
let resolveTimer: ReturnType<typeof setTimeout> | null = null
let resolveRevision = 0

const isChatLike = computed(() => activeModelType.value === 'chat' || activeModelType.value === 'vllm')

const showResolvedPanel = computed(() =>
  isChatLike.value
  && formData.value.source === 'remote'
  && !!formData.value.provider,
)

const resolvedThinkingLevels = computed(() => supportedLevels(resolved.value?.capabilities))

const runResolve = async () => {
  const revision = ++resolveRevision
  const provider = (formData.value.provider || '').trim()
  // Embedding / ReRank / ASR never render the panel, so do not spend a
  // request (and do not leave a stale chat result behind) for them.
  if (!props.visible || !showResolvedPanel.value || !provider) {
    resolved.value = null
    resolving.value = false
    resolveFailed.value = false
    resolveError.value = ''
    return
  }
  resolving.value = true
  try {
    const result = await resolveModelCatalog({
      provider,
      model: formData.value.modelName || '',
      base_url: formData.value.baseUrl || '',
      model_type: activeModelType.value,
      api: formData.value.extraConfig?.api || '',
      thinking_control: formData.value.thinkingControl || '',
      remote_model_name: formData.value.extraConfig?.remote_model_name || '',
      // Vendor fields decide the request too — Azure's api_version picks
      // between the v1 data plane and the dated deployments path — so the
      // preview has to see them or it describes a different request than
      // the one this row will make. Secret fields never travel in a query
      // string; plainExtraFields already excludes them.
      ...Object.fromEntries(
        plainExtraFields.value
          .map(field => [field.key, formData.value.extraConfig?.[field.key] || ''])
          .filter(([, value]) => !!value),
      ),
    })
    if (revision !== resolveRevision) return
    resolved.value = result
    resolveFailed.value = false
    resolveError.value = ''
  } catch (error: any) {
    if (revision !== resolveRevision) return
    resolved.value = null
    resolveFailed.value = true
    const detail = typeof error === 'string'
      ? error
      : (error?.message || error?.error?.message || error?.error)
    resolveError.value = typeof detail === 'string' ? detail.trim() : ''
  } finally {
    if (revision === resolveRevision) resolving.value = false
  }
}

const scheduleResolve = () => {
  if (resolveTimer) clearTimeout(resolveTimer)
  resolveTimer = setTimeout(() => {
    resolveTimer = null
    void runResolve()
  }, 400)
}

const advancedOpen = ref(false)
/** Set when the loaded row carried thinking_control, so the select stays visible after clearing it. */
const legacyThinkingControlLoaded = ref(false)
const showLegacyThinkingControl = computed(() =>
  activeModelType.value === 'chat' && !!(formData.value.thinkingControl || legacyThinkingControlLoaded.value),
)

const specCompatError = computed(() => {
  const text = (formData.value.specCompat || '').trim()
  if (!text) return ''
  try {
    const parsed = JSON.parse(text)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return 'Must be a JSON object'
    }
    return ''
  } catch (error: any) {
    return error?.message || 'invalid JSON'
  }
})

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => {
    if (!saving.value) emit('update:visible', val)
  }
})

const hydratingForm = ref(false)

// Header icon for the SettingDrawer — uses the same TDesign icon name table
// as the model card list, so the drawer's leading badge visually matches the
// card the user just clicked on.
const modelTypeIcon = computed(() => {
  const map: Record<string, string> = {
    chat: 'chat',
    embedding: 'chart-bubble',
    rerank: 'filter-sort',
    vllm: 'image',
    asr: 'sound',
  }
  return map[activeModelType.value] || 'setting'
})

// Credential resource binding for the shared <CredentialResource> component.
// A vendor-declared secret extra field (LKEAP / Volcengine rerank SecretKey)
// is stored as the app_secret credential, so it shows up here in edit mode.
const credentialFields = computed<CredentialFieldDef<ModelCredentialField>[]>(() => {
  const fields: CredentialFieldDef<ModelCredentialField>[] = [
    { key: 'api_key', label: 'API Key (optional)' as string },
  ]
  if (secretExtraField.value) {
    fields.push({ key: 'app_secret', label: extraFieldDisplayLabel(secretExtraField.value) })
  }
  return fields
})

const credentialApi = computed<CredentialResourceApi<ModelCredentialField>>(() => {
  const id = props.modelData?.id ?? ''
  return {
    save: async (patch) => {
      const meta = await putModelCredentials(id, patch)
      return meta.fields
    },
    remove: async (field) => {
      await deleteModelCredentialField(id, field)
    },
  }
})

// Initial credential metadata. ModelSettings.convertToLegacyFormat
// preserves `credentials` from the main ListModels response so the card
// renders the correct "Configured" state on dialog open.
const credentialMeta = computed(() => (props.modelData as any)?.credentials ?? {
  api_key: { configured: false },
  app_secret: { configured: false },
})

// Placeholder hint for the create-mode API key input. Edit mode replaces
// this input entirely with a <CredentialResource> card.
const apiKeyPlaceholder = computed(() => {
  const label = credentialLabel.value
  if (label?.placeholder) return pickLocalized(label.placeholders, currentLocale.value, label.placeholder)
  return 'Enter API Key'
})

const formRef = ref()
const saving = ref(false)
const saveError = ref('')

// Settings itself listens on window for Escape. Capture it while saving so
// the parent cannot unmount this editor before the request finishes.
const handleSaveEscape = (event: KeyboardEvent) => {
  if (props.visible && saving.value && (event.key === 'Escape' || event.code === 'Escape')) {
    event.preventDefault()
    event.stopImmediatePropagation()
  }
}
watch(() => props.visible && saving.value, (locked) => {
  if (locked) window.addEventListener('keydown', handleSaveEscape, true)
  else window.removeEventListener('keydown', handleSaveEscape, true)
}, { flush: 'sync' })
// Toggles the create-mode API key input between masked and plain text. Lets
// the user proofread a freshly pasted secret without losing the password
// affordance for everyday use. Reset every time the drawer closes (see
// reset block in the visible watcher) so we never leak the previous value
// across editor sessions.
const modelChecked = ref(false)
const modelAvailable = ref(false)
const checking = ref(false)
const remoteChecked = ref(false)
const remoteAvailable = ref(false)
const remoteMessage = ref('')
const remoteStale = ref(false)
let connectionRevision = 0
let applyingDetectedDimension = false

// Invalidate pending responses too, even when the user changes a field back.
const invalidateConnectionTest = (showStale = true) => {
  connectionRevision++
  remoteStale.value = showStale && (remoteStale.value || checking.value || remoteChecked.value)
  checking.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

const applyDetectedDimension = (dimension: number) => {
  applyingDetectedDimension = true
  try {
    formData.value.dimension = dimension
  } finally {
    applyingDetectedDimension = false
  }
}
const dimensionChecked = ref(false)
const dimensionSuccess = ref(false)
const dimensionMessage = ref('')

const ollamaModelList = ref<OllamaModelInfo[]>([])
const loadingOllamaModels = ref(false)
const searchKeyword = ref('')
const downloading = ref(false)
const downloadProgress = ref(0)
const currentDownloadModel = ref('')
let downloadInterval: any = null

const ollamaServiceStatus = ref<boolean | null>(null)
const checkingOllamaStatus = ref(false)

const formData = ref<ModelFormData>({
  id: '',
  name: '',
  source: 'remote',
  provider: 'generic',
  modelName: '',
  displayName: '',
  baseUrl: '',
  apiKey: '',
  dimension: undefined,
  supportsDimensionOverride: false,
  interfaceType: 'ollama',
  isDefault: false,
  supportsVision: false,
  contextWindow: undefined,
  maxConcurrency: undefined,
  maxOutputTokens: undefined,
  thinkingControl: '',
  extraConfig: {},
  specCompat: '',
  spec: null,
  customHeaders: [],
  appSecret: '',
})

const rules = computed(() => ({
  modelName: [
    { required: true, message: 'Please enter the model name' },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: 'Model name cannot be empty' }
        }
        if (val.trim().length > 100) {
          return { result: false, message: 'Model name cannot exceed 100 characters' }
        }
        return { result: true }
      },
      trigger: 'blur'
    }
  ],
  baseUrl: [
    {
      required: true,
      message: 'Please enter the Base URL',
      trigger: 'blur'
    },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: 'Base URL cannot be empty' }
        }
        try {
          new URL(val.trim())
          return { result: true }
        } catch {
          return { result: false, message: 'Invalid Base URL, please enter a valid URL' }
        }
      },
      trigger: 'blur'
    }
  ]
}))

watch(
  () => [
    props.visible, formData.value.source, formData.value.provider, formData.value.modelName,
    formData.value.baseUrl, formData.value.extraConfig?.api, formData.value.extraConfig?.remote_model_name,
    formData.value.thinkingControl, activeModelType.value,
  ],
  () => {
    if (!props.visible) return
    scheduleResolve()
  },
)

const getModalDescription = () => {
  const key = `model.editor.description.${activeModelType.value}` as const
  return key || 'Configure model information'
}

const getModelNamePlaceholder = () => {
  if (activeModelType.value === 'vllm') {
    return formData.value.source === 'local'
      ? 'e.g. llava:latest'
      : 'e.g. gpt-4-vision-preview'
  }
  if (activeModelType.value === 'asr') {
    return 'e.g. whisper-1'
  }
  return formData.value.source === 'local'
    ? 'e.g. llama2:latest'
    : 'e.g. gpt-4, claude-3-opus'
}

const getBaseUrlPlaceholder = () => {
  if (activeModelType.value === 'vllm') {
    return 'e.g. http://localhost:11434/v1'
  }
  if (activeModelType.value === 'asr') {
    return 'e.g. https://api.openai.com/v1'
  }
  return 'e.g. https://api.openai.com/v1'
}

const checkOllamaServiceStatus = async () => {
  console.log('Checking Ollama service status...')
  checkingOllamaStatus.value = true
  try {
    const result = await checkOllamaStatus()
    ollamaServiceStatus.value = result.available
    console.log('Ollama service status check complete:', result.available)
  } catch (error) {
    console.error('Failed to check Ollama service status:', error)
    ollamaServiceStatus.value = false
  } finally {
    checkingOllamaStatus.value = false
  }

  if (ollamaServiceStatus.value === false && !isEdit.value && formData.value.source === 'local') {
    formData.value.source = 'remote'
  }
}

const goToOllamaSettings = async () => {
  console.log('Clicked the go-to-Ollama-settings button')
  emit('update:visible', false)

  if (uiStore.showSettingsModal) {
    uiStore.closeSettings()
    await nextTick()
  }

  console.log('Calling uiStore.openSettings')
  uiStore.openSettings('ollama')
  console.log('uiStore.openSettings call complete')
}

const lastOpenedModelId = ref<string | null>(null)

const selectModelType = async (type: EditorModelType) => {
  if (isEdit.value || draftModelType.value === type) return
  draftModelType.value = type

  if (type === 'rerank') {
    formData.value.source = 'remote'
  }
  if (type !== 'embedding') {
    formData.value.dimension = undefined
    formData.value.supportsDimensionOverride = false
    dimensionChecked.value = false
    dimensionSuccess.value = false
    dimensionMessage.value = ''
  }
  if (type !== 'chat') {
    formData.value.supportsVision = false
  }
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''

  await loadProviders()
  const supported = providerOptions.value.some(p => p.value === formData.value.provider)
  if (!supported) {
    formData.value.provider = 'generic'
    formData.value.baseUrl = ''
    formData.value.extraConfig = keepVendorNeutralExtraConfig()
    formData.value.appSecret = ''
  } else {
    handleProviderChange(formData.value.provider || 'generic')
  }
}

watch(() => props.visible, (val) => {
  if (val) {
    checkOllamaServiceStatus()

    loadProviders().then(() => {
      if (props.visible && !isEdit.value) applyExtraFieldDefaults()
    })
    advancedOpen.value = false

    modelChecked.value = false
    modelAvailable.value = false
    remoteChecked.value = false
    remoteAvailable.value = false
    remoteMessage.value = ''
    dimensionChecked.value = false
    dimensionSuccess.value = false
    dimensionMessage.value = ''

    const currentId = props.modelData?.id ?? null
    draftModelType.value = props.modelType

    hydratingForm.value = true
    try {
      if (props.modelData) {
        // edit mode the credential is owned by the <CredentialResource> card,
        // not by this form's apiKey field.
        const loadedExtra: Record<string, string> = {}
        for (const [key, value] of Object.entries(props.modelData.extraConfig || {})) {
          if (RESERVED_EXTRA_CONFIG_KEYS.has(key)) continue
          if (value != null && String(value) !== '') loadedExtra[key] = String(value)
        }
        const loadedSpec = props.modelData.spec || null
        formData.value = {
          ...props.modelData,
          apiKey: '',
          appSecret: '',
          extraConfig: loadedExtra,
          thinkingControl: props.modelData.thinkingControl || '',
          spec: loadedSpec,
          specCompat: props.modelData.specCompat
            ?? (loadedSpec?.compat ? JSON.stringify(loadedSpec.compat, null, 2) : ''),
          customHeaders: Array.isArray(props.modelData.customHeaders)
            ? props.modelData.customHeaders.map(h => ({ key: h.key, value: h.value }))
            : [],
        }
        legacyThinkingControlLoaded.value = !!props.modelData.thinkingControl
      } else if (lastOpenedModelId.value !== null || !formData.value.id) {
        resetForm()
      }

      lastOpenedModelId.value = currentId

      if (activeModelType.value === 'rerank') {
        formData.value.source = 'remote'
      }
    } finally {
      nextTick(() => {
        hydratingForm.value = false
      })
    }
  }
})

watch(
  () => [
    activeModelType.value, props.modelData?.id, formData.value.source,
    formData.value.provider, formData.value.modelName, formData.value.baseUrl,
    formData.value.apiKey, formData.value.appSecret, formData.value.customHeaders,
    formData.value.dimension, formData.value.supportsDimensionOverride,
    formData.value.extraConfig, formData.value.thinkingControl,
  ],
  () => {
    if (!applyingDetectedDimension) invalidateConnectionTest(props.visible && !hydratingForm.value)
  },
  { deep: true, flush: 'sync' },
)

watch(() => props.visible, () => {
  invalidateConnectionTest(false)
  saveError.value = ''
}, { flush: 'sync' })

const resetForm = () => {
  legacyThinkingControlLoaded.value = false
  formData.value = {
    id: generateId(),
    name: '',
    source: 'remote',
    provider: 'generic',
    modelName: '',
    displayName: '',
    baseUrl: '',
    apiKey: '',
    dimension: undefined,
    supportsDimensionOverride: false,
    interfaceType: undefined,
    isDefault: false,
    supportsVision: false,
    contextWindow: undefined,
    maxConcurrency: undefined,
    maxOutputTokens: undefined,
    thinkingControl: '',
    extraConfig: {},
    specCompat: '',
    spec: null,
    customHeaders: [],
    appSecret: '',
  }
  resolved.value = null
  resolveFailed.value = false
  resolveError.value = ''
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

/**
 * Drop a model name the new vendor does not serve, along with whatever the
 * catalog filled in for it.
 *
 * The same id does exist at several vendors — an open-weight model is served
 * by more than one OpenAI-compatible gateway — so switching between them
 * should keep the selection. Anything else is a name from the previous
 * vendor: left in place it is saved verbatim, resolves as an uncatalogued
 * model and fails at the first call.
 */
const resetModelSelectionForVendor = () => {
  const current = (formData.value.modelName || '').trim()
  if (!current || findCatalogEntry(current)) return

  formData.value.modelName = ''
  // Take back only the values applyCatalogEntry put there; a number the
  // operator typed is theirs and survives the switch.
  const filled = catalogFilled.value
  if (filled.contextWindow && formData.value.contextWindow === filled.contextWindow) {
    formData.value.contextWindow = undefined
  }
  if (filled.maxOutputTokens && formData.value.maxOutputTokens === filled.maxOutputTokens) {
    formData.value.maxOutputTokens = undefined
  }
  if (filled.dimension && formData.value.dimension === filled.dimension) {
    formData.value.dimension = undefined
  }
  if (filled.supportsVision && formData.value.supportsVision) {
    formData.value.supportsVision = false
  }
  catalogFilled.value = {}
  modelChecked.value = false
  modelAvailable.value = false
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

const handleProviderChange = (value: string) => {
  const provider = providerOptions.value.find(opt => opt.value === value)
  if (provider?.defaultUrls) {
    const defaultUrl = provider.defaultUrls[activeModelType.value]
    if (defaultUrl) {
      formData.value.baseUrl = defaultUrl
    }
  }
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  if (hydratingForm.value) return
  resetModelSelectionForVendor()
  formData.value.extraConfig = keepVendorNeutralExtraConfig()
  formData.value.appSecret = ''
  applyExtraFieldDefaults()
}


const generateId = () => {
  return `model_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

const addCustomHeader = () => {
  if (!Array.isArray(formData.value.customHeaders)) {
    formData.value.customHeaders = []
  }
  formData.value.customHeaders.push({ key: '', value: '' })
}

const removeCustomHeader = (idx: number) => {
  if (!Array.isArray(formData.value.customHeaders)) return
  formData.value.customHeaders.splice(idx, 1)
}

const filteredOllamaModels = computed(() => {
  if (!searchKeyword.value) return ollamaModelList.value
  return ollamaModelList.value.filter(model =>
    model.name.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

const showDownloadOption = computed(() => {
  if (!searchKeyword.value.trim()) return false
  const exists = ollamaModelList.value.some(model =>
    model.name.toLowerCase() === searchKeyword.value.toLowerCase()
  )
  return !exists
})

const handleModelFilter = (filterWords: string) => {
  searchKeyword.value = filterWords
  return true
}

const loadOllamaModels = async () => {
  if (formData.value.source !== 'local') return

  loadingOllamaModels.value = true
  try {
    const models = await listOllamaModels()
    ollamaModelList.value = models
  } catch (error) {
    console.error('Failed to load model list', error)
    MessagePlugin.error('Failed to load model list')
  } finally {
    loadingOllamaModels.value = false
  }
}

const refreshOllamaModels = async () => {
  ollamaModelList.value = []
  await loadOllamaModels()
  MessagePlugin.success('List refreshed')
}

const handleDropdownVisibleChange = (visible: boolean) => {
  if (!visible) {
    searchKeyword.value = ''
  }
}

const formatModelSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return ''
  const gb = bytes / (1024 * 1024 * 1024)
  return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}

const checkModelStatus = async () => {
  if (!formData.value.modelName || formData.value.source !== 'local') {
    return
  }

  try {
    const result = await checkOllamaModels([formData.value.modelName])
    modelChecked.value = true
    modelAvailable.value = result.models[formData.value.modelName] || false
  } catch (error) {
    console.error('Failed to check model status:', error)
    modelChecked.value = false
    modelAvailable.value = false
  }
}

const checkOllamaDimension = async () => {
  if (checking.value || saving.value) return
  if (!formData.value.modelName || formData.value.source !== 'local' || activeModelType.value !== 'embedding') {
    return
  }

  const revision = ++connectionRevision
  checking.value = true
  dimensionChecked.value = false
  dimensionMessage.value = ''

  try {
    const result = await testEmbeddingModel({
      source: 'local',
      modelName: formData.value.modelName,
      dimension: formData.value.dimension,
      supportsDimensionOverride: formData.value.supportsDimensionOverride ?? false,
    })

    if (revision !== connectionRevision) return
    dimensionChecked.value = true
    dimensionSuccess.value = result.available || false

    if (result.available && result.dimension) {
      applyDetectedDimension(result.dimension)
      dimensionMessage.value = `Detection succeeded. Vector dimension: ${result.dimension}`
      MessagePlugin.success(dimensionMessage.value)
    } else {
      if (result.message) {
        console.debug('Backend dimension message:', result.message)
      }
      dimensionMessage.value = 'Detection failed, please enter the dimension manually'
      MessagePlugin.warning(dimensionMessage.value)
    }
  } catch (error: any) {
    if (revision !== connectionRevision) return
    console.error('Ollama dimension check failed:', error)
    dimensionChecked.value = true
    dimensionSuccess.value = false
    dimensionMessage.value = 'Detection failed, please enter the dimension manually'
    MessagePlugin.error(dimensionMessage.value)
  } finally {
    if (revision === connectionRevision) checking.value = false
  }
}

/** extra_config exactly as it will be persisted: trimmed, empty values dropped. */
const buildExtraConfig = (): Record<string, string> => {
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(formData.value.extraConfig || {})) {
    const trimmed = (value ?? '').toString().trim()
    if (key && trimmed) out[key] = trimmed
  }
  const legacy = (formData.value.thinkingControl || '').trim()
  if (legacy && activeModelType.value === 'chat' && formData.value.source === 'remote') {
    out.thinking_control = legacy
  }
  return out
}

const checkRemoteAPI = async () => {
  if (checking.value || saving.value) return
  if (!formData.value.modelName || !formData.value.baseUrl) {
    MessagePlugin.warning('Please fill in the model identifier and Base URL first')
    return
  }

  const revision = ++connectionRevision
  checking.value = true
  remoteStale.value = false
  remoteChecked.value = false
  remoteMessage.value = ''

  try {
    let result: any

    // Convert the form's key/value array of custom headers into the map the backend
    // expects. Same as ModelSettings.vue does on save: blank rows are dropped, so the test
    // connection and the real production call after saving use exactly the same header set.
    const customHeaders: Record<string, string> = {}
    if (Array.isArray(formData.value.customHeaders)) {
      for (const item of formData.value.customHeaders) {
        const key = (item?.key ?? '').trim()
        const value = (item?.value ?? '').trim()
        if (key && value) customHeaders[key] = value
      }
    }
    const headerPayload = Object.keys(customHeaders).length > 0
      ? { customHeaders }
      : {}

    // In edit mode apiKey is owned by <CredentialResource> and is not in formData. Pass
    // modelId through to the backend so that, when apiKey is empty, it falls back to the
    // stored decrypted value - otherwise "test connection" would fail simply for not
    // carrying an apiKey.
    const idPayload = isEdit.value && props.modelData?.id
      ? { modelId: props.modelData.id as string }
      : {}

    const extraConfig = buildExtraConfig()
    const extraPayload = {
      ...(Object.keys(extraConfig).length > 0 ? { extraConfig } : {}),
      ...(formData.value.appSecret?.trim() ? { appSecret: formData.value.appSecret.trim() } : {}),
    }

    switch (activeModelType.value) {
      case 'chat':
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      case 'embedding':
        result = await testEmbeddingModel({
          source: 'remote',
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          dimension: formData.value.dimension,
          supportsDimensionOverride: formData.value.supportsDimensionOverride ?? false,
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        if (revision !== connectionRevision) return
        if (result.available && result.dimension) applyDetectedDimension(result.dimension)
        break

      case 'rerank':
        result = await checkRerankModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      case 'vllm':
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      case 'asr':
        result = await checkASRModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

      default:
        MessagePlugin.error('Unsupported model type')
        return
    }

    if (revision !== connectionRevision) return
    remoteChecked.value = true
    remoteAvailable.value = result.available || false
    remoteMessage.value = result.available
      ? 'Connection succeeded'
      : result.message || 'Connection failed'
  } catch (error: any) {
    if (revision !== connectionRevision) return
    remoteChecked.value = true
    remoteAvailable.value = false
    remoteMessage.value = error?.message || 'Connection failed, please check the configuration'
  } finally {
    if (revision === connectionRevision) checking.value = false
  }
}

const handleConfirm = async () => {
  if (saving.value) return
  saving.value = true
  if (checking.value) invalidateConnectionTest()
  saveError.value = ''
  try {
    if (!formData.value.modelName || !formData.value.modelName.trim()) {
      MessagePlugin.warning('Please enter the model name')
      return
    }

    if (formData.value.modelName.trim().length > 100) {
      MessagePlugin.warning('Model name cannot exceed 100 characters')
      return
    }

    if (formData.value.source === 'remote') {
      if (!formData.value.baseUrl || !formData.value.baseUrl.trim()) {
        MessagePlugin.warning('Remote API type requires a Base URL')
        return
      }

      try {
        new URL(formData.value.baseUrl.trim())
      } catch {
        MessagePlugin.warning('Invalid Base URL, please enter a valid URL')
        return
      }
    }

    if (formData.value.source === 'remote') {
      for (const field of plainExtraFields.value) {
        if (field.required && !(formData.value.extraConfig?.[field.key] || '').trim()) {
          MessagePlugin.warning(`Please fill in ${extraFieldDisplayLabel(field)}`)
          return
        }
      }
      if (!isEdit.value && secretExtraField.value?.required && !(formData.value.appSecret || '').trim()) {
        MessagePlugin.warning(`Please fill in ${extraFieldDisplayLabel(secretExtraField.value)}`)
        return
      }
      if (specCompatError.value) {
        advancedOpen.value = true
        MessagePlugin.warning(`${'Invalid JSON'}: ${specCompatError.value}`)
        return
      }
    }

    const validation = await formRef.value?.validate()
    if (validation !== undefined && validation !== true) return

    // Credential removal in edit mode is handled inline by the
    // CredentialResource card (it confirms + DELETEs to /credentials), so
    // the main save flow no longer needs to confirm or handle clear flags.

    if (!formData.value.id) {
      formData.value.id = generateId()
    }

    await props.saveModel({
      ...formData.value,
      extraConfig: buildExtraConfig(),
      ...(isEdit.value ? {} : { modelType: activeModelType.value }),
    })
    emit('update:visible', false)
    invalidateConnectionTest(false)
    resetForm()
    lastOpenedModelId.value = null
  } catch (error: any) {
    saveError.value = error?.message || 'Failed to save model'
  } finally {
    saving.value = false
  }
}

watch(() => formData.value.modelName, async (newValue, oldValue) => {
  if (!newValue) return

  if (newValue.startsWith('__download__')) {
    const modelName = newValue.replace('__download__', '')

    formData.value.modelName = ''

    await startDownload(modelName)
    return
  }

  if (activeModelType.value === 'embedding' &&
    formData.value.source === 'local' &&
    newValue !== oldValue &&
    oldValue !== '') {
    MessagePlugin.info('Model selected. Click "Detect Dimension" to fetch the vector dimension automatically.')
  }
})

const startDownload = async (modelName: string) => {
  downloading.value = true
  downloadProgress.value = 0
  currentDownloadModel.value = modelName

  try {
    const result = await downloadOllamaModel(modelName)
    const taskId = result.taskId

    MessagePlugin.success(`Started downloading ${modelName}`)

    downloadInterval = setInterval(async () => {
      try {
        const progress = await getDownloadProgress(taskId)
        downloadProgress.value = progress.progress

        if (progress.status === 'completed') {
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false

          MessagePlugin.success(`${modelName} downloaded successfully`)

          await loadOllamaModels()

          formData.value.modelName = modelName

          downloadProgress.value = 0
          currentDownloadModel.value = ''

        } else if (progress.status === 'failed') {
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false
          MessagePlugin.error(progress.message || `Failed to download ${modelName}`)
          downloadProgress.value = 0
          currentDownloadModel.value = ''
        }
      } catch (error) {
        console.error('Failed to fetch download progress:', error)
      }
    }, 1000)

  } catch (error: any) {
    downloading.value = false
    downloadProgress.value = 0
    currentDownloadModel.value = ''
    console.error('Download start failed:', error)
    MessagePlugin.error('Failed to start download')
  }
}

onUnmounted(() => {
  window.removeEventListener('keydown', handleSaveEscape, true)
  invalidateConnectionTest(false)
  if (resolveTimer) {
    clearTimeout(resolveTimer)
    resolveTimer = null
  }
  resolveRevision++
  if (downloadInterval) {
    clearInterval(downloadInterval)
  }
})

watch(() => formData.value.source, () => {
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''

  searchKeyword.value = ''
  if (downloadInterval) {
    clearInterval(downloadInterval)
    downloadInterval = null
  }
  downloading.value = false
  downloadProgress.value = 0
  currentDownloadModel.value = ''
})

watch(() => formData.value.modelName, () => {
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
})

const handleCancel = () => {
  if (saving.value) return
  resetForm()
  lastOpenedModelId.value = null
  dialogVisible.value = false
}
</script>

<style lang="less" scoped>
.provider-doc-link {
  margin-top: 6px;

  a {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    color: var(--td-text-color-link);
    text-decoration: none;

    &:hover {
      text-decoration: underline;
    }
  }
}

:deep(.t-form) {
  .t-form-item {
    display: none;
  }
}

.form-item {
  // No bottom margin — vertical rhythm is owned by the parent
  // .setting-drawer__section's `gap`. That keeps the spacing inside a section
  // tight and the gap between sections visually distinct.
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;

  // TDesign-style required marker: leading asterisk before the label text,
  // matching the rest of the app's <t-form-item required ...> appearance.
  &.required::before {
    content: '*';
    color: var(--td-error-color);
    margin-right: 4px;
    font-weight: 500;
    line-height: 1;
  }
}

.model-type-options,
.source-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.model-type-option,
.source-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: inherit;
  padding: 6px 12px;
  min-height: 32px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.4;
  cursor: pointer;
  transition: border-color var(--app-motion-fast) ease, color var(--app-motion-fast) ease, background var(--app-motion-fast) ease;

  &__icon {
    font-size: var(--app-text-lg);
    flex-shrink: 0;
  }

  &__label {
    white-space: nowrap;
  }

  &:hover:not(.is-active) {
    border-color: var(--td-brand-color-3);
    color: var(--td-text-color-primary);
  }

  &.is-active {
    border-color: var(--td-brand-color);
    background: var(--td-bg-color-container);
    color: var(--td-brand-color);
    font-weight: 500;
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}



.source-option {
  &.is-disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }
}

:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: var(--app-text-md);
}


:deep(.t-checkbox) {
  font-size: var(--app-text-md);

  .t-checkbox__label {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
  }
}

.api-key-input {
  :deep(.t-input__prefix) {
    color: var(--td-text-color-placeholder);
  }

  :deep(.t-input__suffix) {
    color: var(--td-text-color-placeholder);
  }

}

.api-test-section {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: var(--td-bg-color-container-hover);
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--app-radius-md);

  .test-message {
    font-size: var(--app-text-md);
    line-height: 1.5;
    flex: 1;

    &.success {
      color: var(--td-brand-color-active);
    }

    &.error {
      color: var(--td-error-color);
    }
  }

  :deep(.t-button) {
    min-width: 88px;
    height: 32px;
    font-size: var(--app-text-md);
    border-radius: var(--app-radius-sm);
    flex-shrink: 0;
  }

  .status-icon {
    font-size: var(--app-text-xl);
    flex-shrink: 0;

    &.available {
      color: var(--td-brand-color);
    }

    &.unavailable {
      color: var(--td-error-color);
    }
  }
}

.connection-status {
  font-size: var(--app-text-sm);
  &.success { color: var(--td-brand-color-active); }
  &.error { color: var(--td-error-color); }
}

.connection-hint {
  margin: 0 0 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.5;
}

.connection-result {
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid var(--td-error-color-3);
  border-radius: var(--td-radius-default);
  background: var(--td-error-color-1);
  color: var(--td-error-color);
  font-size: var(--app-text-sm);
  text-align: left;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  &__details {
    max-height: min(160px, 20vh);
    overflow: auto;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    line-height: 1.5;
    user-select: text;
  }
}

// Status icon variant used inside the footer button.
.status-icon {
  font-size: var(--app-text-xl);
  flex-shrink: 0;

  &.available {
    color: var(--td-brand-color);
  }

  &.unavailable {
    color: var(--td-error-color);
  }
}


.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 0;

  .downloaded-icon {
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
    flex-shrink: 0;
  }

  .download-icon {
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
    flex-shrink: 0;
  }

  .model-name {
    flex: 1;
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
  }

  .model-size {
    font-size: var(--app-text-sm);
    color: var(--td-text-color-placeholder);
    margin-left: auto;
  }

  &.download {
    .model-name {
      color: var(--td-brand-color);
      font-weight: 500;
    }
  }
}

.download-suffix {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 4px;

  .spinning {
    animation: wk-spin 1s linear infinite;
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
  }

  .progress-text {
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-brand-color);
  }
}

:deep(.t-select.downloading) {
  .t-input {
    position: relative;
    overflow: hidden;

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: var(--progress, 0%);
      background: linear-gradient(90deg, color-mix(in srgb, var(--td-brand-color) 8%, transparent), color-mix(in srgb, var(--td-brand-color) 15%, transparent));
      transition: width var(--app-motion-slow) ease;
      z-index: 0;
      border-radius: 5px 0 0 5px;
    }

    .t-input__inner,
    input {
      position: relative;
      z-index: 1;
      background: transparent !important;
    }
  }
}

.model-select-row {
  display: flex;
  align-items: center;
  gap: 8px;

  .t-select {
    flex: 1;
  }
}

.refresh-btn {
  flex-shrink: 0;
}

.dimension-control {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.t-input) {
    flex: 1;
  }
}

.dimension-check-btn {
  flex-shrink: 0;
}

.dimension-hint {
  margin: 8px 0 0 0;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-error-color);

  &.success {
    color: var(--td-brand-color);
  }
}

.custom-headers-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.custom-headers-desc {
  margin: 0 0 10px 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
}

.custom-headers-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.custom-header-row {
  display: flex;
  align-items: center;
  gap: 8px;

  .custom-header-key {
    flex: 0 0 38%;
  }

  .custom-header-value {
    flex: 1;
  }

  // Ghost icon button — matches the model-card "more" affordance: invisible
  // until hover/focus, then a subtle background pops in. Avoids painting a
  // permanent red splotch next to every header row.
  .custom-header-remove {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    padding: 0;
    color: var(--td-text-color-placeholder);
    border-radius: var(--app-radius-sm);
    transition: all 0.18s ease;

    &:hover {
      background: var(--td-error-color-light);
      color: var(--td-error-color);
    }
  }
}

.form-desc {
  margin: 4px 0 0 0;
  font-size: var(--app-text-sm);
  line-height: 1.5;
  color: var(--td-text-color-placeholder);

  // Inline with switches/checkboxes — drops the top margin so the label and
  // helper text sit on the same baseline.
  &--inline {
    margin: 0;
  }

  &--recommend {
    color: var(--td-brand-color);
  }

  &--warn {
    color: var(--td-warning-color);
  }

  &--error {
    color: var(--td-error-color);
  }
}

.provider-value {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;

  &__name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.provider-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  border-radius: 4px;
  object-fit: contain;

  &--placeholder {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: var(--app-text-xs);
    font-weight: 600;
    color: var(--td-text-color-secondary);
    background: var(--td-bg-color-secondarycontainer);
  }
}

.catalog-badge {
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  height: 18px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  line-height: 18px;
  white-space: nowrap;

  & + & {
    margin-left: 4px;
  }

  &--accent {
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);
  }
}

.resolved-panel {
  padding: 10px 12px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container-hover);

  &__header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  &__icon {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
  }

  &__title {
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-text-color-secondary);
  }

  &__loading {
    font-size: var(--app-text-md);
    color: var(--td-text-color-placeholder);
    animation: spin 1s linear infinite;
  }

  &__grid {
    display: grid;
    grid-template-columns: max-content minmax(0, 1fr);
    column-gap: 12px;
    row-gap: 4px;
    margin: 0;
    font-size: var(--app-text-sm);

    dt {
      color: var(--td-text-color-placeholder);
      white-space: nowrap;
      line-height: 18px;
    }

    dd {
      margin: 0;
      min-width: 0;
      color: var(--td-text-color-primary);
      line-height: 18px;
      overflow-wrap: anywhere;
    }

    code {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: var(--app-text-sm);
    }
  }
}

.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  margin: 0;
  background: transparent;
  border: none;
  cursor: pointer;
  font-family: inherit;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  user-select: none;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
    border-radius: 4px;
  }

  .toggle-arrow {
    font-size: var(--app-text-base);
    transition: transform 0.15s ease;

    &.open {
      transform: rotate(90deg);
    }
  }
}

.compat-textarea {
  :deep(textarea) {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: var(--app-text-sm);
  }
}

.vision-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ollama-unavailable-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--td-error-color-light);
  border: 1px solid var(--td-error-color-focus);
  border-radius: var(--app-radius-md);
  font-size: var(--app-text-md);

  .tip-icon {
    color: var(--td-error-color);
    font-size: var(--app-text-xl);
    flex-shrink: 0;
    margin-right: 2px;

    &.info {
      color: var(--td-brand-color);
    }
  }

  .tip-text {
    color: var(--td-error-color);
    flex: 1;
    line-height: 1.5;
  }

  &.rerank-tip {
    background: var(--td-success-color-light);
    border: 1px solid var(--td-success-color-focus);
    border-left: 3px solid var(--td-brand-color);

    .tip-text {
      color: var(--td-success-color);
    }
  }

  :deep(.tip-link) {
    color: var(--td-brand-color);
    font-size: var(--app-text-md);
    font-weight: 500;
    padding: 4px 6px 4px 10px !important;
    min-height: auto !important;
    height: auto !important;
    line-height: 1.4 !important;
    text-decoration: none;
    white-space: nowrap;
    display: inline-flex !important;
    align-items: center !important;
    gap: 1px;
    border-radius: var(--app-radius-xs);
    transition: all var(--app-motion-base) ease;

    &:hover {
      background: color-mix(in srgb, var(--td-brand-color) 8%, transparent) !important;
      color: var(--td-brand-color-active) !important;
    }

    &:active {
      background: color-mix(in srgb, var(--td-brand-color) 12%, transparent) !important;
    }

    .t-icon {
      font-size: var(--app-text-base) !important;
      margin: 0 !important;
      line-height: 1 !important;
      display: inline-flex !important;
      align-items: center !important;
    }
  }
}

// Destructive-action checkbox for "Remove this credential". Styled to match
// the pattern used in McpServiceDialog so the two dialogs read identically.
.clear-credential {
  display: inline-flex;
  margin-top: 8px;

  :deep(.t-checkbox__label) {
    color: var(--td-error-color);
    font-size: var(--app-text-md);
  }
}
</style>

<!-- Not scoped: the t-select popup renders under body, so scoped styles cannot reach it -->
<style lang="less">
.catalog-model-select-popup {
  padding: 4px;

  .t-select-option {
    height: auto !important;
    padding: 8px 10px;
    border-radius: var(--app-radius-sm);
    margin: 2px 0;
  }
}

.catalog-model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;

  &__name {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__id {
    font-size: var(--app-text-xs);
    color: var(--td-text-color-placeholder);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__badges {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: auto;
    flex-shrink: 0;

    .catalog-badge {
      display: inline-flex;
      align-items: center;
      padding: 0 6px;
      height: 18px;
      border-radius: 4px;
      background: var(--td-bg-color-secondarycontainer);
      color: var(--td-text-color-secondary);
      font-size: var(--app-text-xs);
      line-height: 18px;
      white-space: nowrap;

      &--accent {
        background: var(--td-brand-color-light);
        color: var(--td-brand-color);
      }
    }
  }
}

.provider-select-popup {
  padding: 4px;

  &.wk-popover .t-popup__content {
    max-height: min(480px, 60vh);
    // Scrolling itself is restored for every skinned select in
    // assets/theme/tdesign-overrides.less; this only makes the bar visible,
    // because macOS overlay scrollbars stay hidden until something moves and
    // a capped list then looks complete.
    scrollbar-color: var(--td-component-border) transparent;
    scrollbar-width: thin;
  }

  + .t-popup .t-tooltip,
  ~ .t-popup .t-tooltip {
    display: none !important;
  }

  .t-select-option {
    height: auto !important;
    padding: 8px 10px;
    border-radius: var(--app-radius-sm);
    margin: 2px 0;
    outline: none;
    transition: background-color var(--app-motion-fast) ease;

    &:focus,
    &:focus-visible {
      outline: none;
    }


  }

  .t-select-option.t-is-selected .provider-name {
    color: var(--td-brand-color);
  }

  .provider-option {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-width: 0;

    .provider-icon {
      width: 18px;
      height: 18px;
      flex-shrink: 0;
      border-radius: 4px;
      object-fit: contain;

      &--placeholder {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        font-size: var(--app-text-xs);
        font-weight: 600;
        color: var(--td-text-color-secondary);
        background: var(--td-bg-color-secondarycontainer);
      }
    }

    &__text {
      display: flex;
      flex-direction: column;
      gap: 2px;
      min-width: 0;
      flex: 1;
    }

    .provider-name {
      font-size: var(--app-text-md);
      font-weight: 500;
      color: var(--td-text-color-primary);
      line-height: 20px;
    }

    .provider-desc {
      font-size: var(--app-text-sm);
      color: var(--td-text-color-placeholder);
      line-height: 18px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }
}
</style>
