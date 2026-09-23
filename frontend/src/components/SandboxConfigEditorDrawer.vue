<template>
  <SettingDrawer
    class="sandbox-config-drawer"
    :visible="visible"
    :title="record ? 'Edit sandbox' : 'Add sandbox'"
    :description="stepDescription"
    icon="code"
    width="680px"
    :min-width="560"
    :max-width="920"
    storage-key="setting-drawer:width:sandbox-config-v2"
    :confirm-loading="saving || checking || templatesLoading"
    :confirm-disabled="primaryDisabled"
    :confirm-text="primaryText"
    @confirm="handlePrimaryAction"
    @cancel="close"
    @update:visible="(v: boolean) => emit('update:visible', v)"
  >
    <template #footer-left>
      <t-button v-if="wizardStep > 0" variant="outline" @click="previousStep">
        {{ 'Back' }}
      </t-button>
      <t-popconfirm
        v-if="canDeepCheck"
        :content="'Full verification runs a throwaway script. Remote backends also create and destroy a real sandbox and may consume a small amount of sandbox time. Continue?'"
        @confirm="runCheck(true)"
      >
        <t-button variant="outline" :loading="checking">
          {{ lastCheckWasDeep ? 'Verify again' : 'Full verification' }}
        </t-button>
      </t-popconfirm>
    </template>

    <template #header-extra>
      <nav class="sandbox-steps" :aria-label="'Sandbox setup progress'">
        <component
          :is="canJumpTo(index) ? 'button' : 'div'"
          v-for="(item, index) in wizardSteps"
          :key="item.key"
          :type="canJumpTo(index) ? 'button' : undefined"
          :class="['sandbox-step', {
            'is-active': wizardStep === index,
            'is-done': wizardStep > index,
            'is-clickable': canJumpTo(index),
          }]"
          :aria-current="wizardStep === index ? 'step' : undefined"
          @click="goToStep(index)"
        >
          <span class="sandbox-step__marker">
            <t-icon v-if="wizardStep > index" name="check" />
            <template v-else>{{ index + 1 }}</template>
          </span>
          <span class="sandbox-step__title">{{ item.title }}</span>
          <span v-if="index < wizardSteps.length - 1" class="sandbox-step__line" aria-hidden="true" />
        </component>
      </nav>
    </template>

    <!--
      Identity-change refusals must sit at the top: the form is long and the
      admin otherwise saves, sees nothing, and assumes the click did nothing.
    -->
    <div v-if="conflict" ref="conflictAlertRef" class="blocked blocked-top">
      <t-alert v-if="conflict.code === 'sandboxes_still_live'" theme="warning"
        :message="`This config still owns ${conflict.inventory?.sandbox_count ?? 0} running or paused sandbox(es), so identity fields cannot be changed yet.`">
        <template #description>
          <p v-if="affectedSessionCount">{{ `${affectedSessionCount} session(s) affected.` }}</p>
          <p v-if="conflict.inventory?.agent_names?.length">
            {{ `Agents using this config: ${conflict.inventory.agent_names.join(', ')}` }}
          </p>
          <p>{{ 'End or delete those sessions (deleting a session destroys its sandbox), or create a second config and point the agents at it.' }}</p>
        </template>
      </t-alert>
      <t-alert v-else-if="conflict.code === 'skill_snapshot_blocks_template'" theme="warning"
        :message="'This sandbox already has skills. The skill environment is bound to the current snapshot, so the runtime template cannot be changed or rebuilt. Create a new sandbox and install skills from the new template.'" />
      <t-alert v-else theme="warning" :message="'The backend cannot be reached to verify whether sandboxes remain, so the stored credentials will not be overwritten.'">
        <template #description>
          <p>{{ 'Restore connectivity first; if the backend is gone for good, create a second config and point the agents at it.' }}</p>
        </template>
      </t-alert>
    </div>

    <t-form label-align="top" class="sandbox-editor-form">
      <section v-if="currentStepKey === 'connection'" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Basic information' }}</h4>
        <t-form-item :label="'Sandbox type'">
          <t-select :value="backend" :placeholder="'Select a sandbox type'"
            class="backend-select" :popup-props="{ overlayClassName: 'sandbox-backend-popup' }"
            :disabled="retargetFrozen"
            @change="(v: any) => selectBackend(String(v))">
            <t-option v-for="opt in backendOptions" :key="opt" :value="opt" :label="backendLabel(opt)">
              <span class="backend-choice">
                <SandboxBackendBadge :type="opt" size="sm" />
                <span class="backend-choice__text">
                  <span class="backend-choice__name">{{ backendLabel(opt) }}</span>
                  <span class="backend-choice__desc">
                    {{ (SETTINGS_SANDBOX_BACKEND_DESCRIPTIONS_LABELS[opt] ?? '') }}
                  </span>
                </span>
              </span>
            </t-option>
          </t-select>
          <p v-if="backend" class="section-help section-help--field">
            {{ (SETTINGS_SANDBOX_BACKEND_DESCRIPTIONS_LABELS[backend] ?? '') }}
          </p>
        </t-form-item>
        <t-alert v-if="backend === 'docker' && !dockerBackendEnabled" theme="warning" class="compact-alert"
          :message="'Docker sandbox is not enabled on this deployment'">
          <template #description>
            <p>{{ 'A local docker.sock is equivalent to root on the host. For a single-machine private install, a system admin can enable it under Settings → System settings → Network security.' }}</p>
          </template>
        </t-alert>
        <t-form-item :label="'Config name'" :status="nameError ? 'error' : undefined"
          :tips="nameError || undefined">
          <t-input v-model="name" :placeholder="'e.g. Production E2B'" />
        </t-form-item>
        <t-form-item :label="'Description'">
          <t-input v-model="description" :placeholder="'Optional, helps tell several configs of the same type apart'" />
        </t-form-item>
      </section>

      <section v-if="currentStepKey === 'connection' && isRemoteBackend" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Cluster connection' }}</h4>
        <t-alert v-if="hasSkillSnapshot" theme="info" class="identity-hint compact-alert"
          :message="'This sandbox already has skills. Connection, credentials, and DNS would retarget the skill snapshot, and DNS only applies after a template rebuild. Create a new sandbox instead.'" />
        <t-alert v-else-if="hasInFlightSkill" theme="info" class="identity-hint compact-alert"
          :message="'A skill is still installing or being removed. Connection, credentials, and DNS cannot change until that finishes.'" />
        <t-alert v-else-if="record" theme="info" class="identity-hint compact-alert"
          :message="'While this config owns sandboxes, these cannot be changed: backend type, API endpoint, API key, sandbox domain, proxy endpoint.'" />

        <template v-if="backend === 'cube'">
          <t-form-item :label="requiredLabel('apiUrl')" :status="fieldStatus('api_url')" :tips="fieldTip('api_url')">
            <t-input v-model="cube.api_url" placeholder="http://cube.example.com:33000"
              :disabled="retargetFrozen" @input="onConnectionInput('api_url')" />
          </t-form-item>
          <div class="form-grid form-grid--two">
            <t-form-item :label="requiredLabel('proxyUrl')" :status="fieldStatus('proxy_url')"
              :tips="fieldTip('proxy_url')">
              <t-input v-model="cube.proxy_url" placeholder="http://cube.example.com:80"
                :disabled="retargetFrozen" @input="onConnectionInput('proxy_url')" />
            </t-form-item>
            <t-form-item :label="requiredLabel('sandboxDomain')" :status="fieldStatus('sandbox_domain')"
              :tips="fieldTip('sandbox_domain')">
              <t-input v-model="cube.sandbox_domain" placeholder="cube.app"
                :disabled="retargetFrozen" @input="onConnectionInput('sandbox_domain')" />
            </t-form-item>
          </div>
          <t-form-item :label="'API key'">
            <t-input v-model="cube.api_key" type="password" :placeholder="secretInputPlaceholder('cube')"
              :disabled="retargetFrozen" @input="invalidateConnection" />
            <div class="field-hints">
              <p class="section-help">
                {{ storedSecrets.cube
                  ? 'Key configured (not shown again); enter a new value to rotate'
                  : 'Optional — leave empty for an unauthenticated self-hosted CubeSandbox' }}
              </p>
              <a class="inline-guide-link" :href="clusterGuideUrl" target="_blank" rel="noopener noreferrer">
                <t-icon name="link" />
                {{ 'How to enable auth on a self-hosted cluster' }}
              </a>
            </div>
          </t-form-item>
          <t-form-item :label="'DNS servers'"
            :help="'Optional. Nameserver IPs written into the sustainability.ai standard template. Leave empty to use the cluster default. If UDP/53 to public resolvers is blocked, use reachable addresses from the Cube host\'s /etc/resolv.conf, excluding 10/8, 172.16/12, and 192.168/16. Existing standard templates take effect only after Rebuild on the template card.'">
            <t-tag-input v-model="cube.dns_servers" :placeholder="'e.g. 8.8.8.8, press Enter to add'"
              :disabled="retargetFrozen" clearable @change="invalidateConnection" />
          </t-form-item>
        </template>

        <template v-else-if="backend === 'e2b'">
          <t-form-item :label="requiredLabel('apiKey')" :status="fieldStatus('api_key')"
            :tips="fieldTip('api_key')">
            <t-input v-model="e2b.api_key" type="password" :placeholder="secretInputPlaceholder('e2b')"
              :disabled="retargetFrozen" @input="onConnectionInput('api_key')" />
            <div class="field-hints">
              <p class="section-help">
                {{ storedSecrets.e2b
                  ? 'Key configured (not shown again); enter a new value to rotate'
                  : 'Create one on the API Keys page of the E2B dashboard; it usually starts with e2b_.' }}
              </p>
              <a class="inline-guide-link" :href="e2bApiKeysUrl" target="_blank" rel="noopener noreferrer">
                <t-icon name="link" />
                {{ 'Get an API key from the E2B dashboard' }}
              </a>
            </div>
          </t-form-item>
          <div class="form-grid form-grid--two">
            <t-form-item :label="'API endpoint'" :help="'Optional — the SDK default is used when empty'">
              <t-input v-model="e2b.api_url" placeholder="https://api.e2b.app"
                :disabled="retargetFrozen" @input="invalidateConnection" />
            </t-form-item>
            <t-form-item :label="'Sandbox domain'" :help="'Optional — the SDK default is used when empty'">
              <t-input v-model="e2b.sandbox_domain" placeholder="e2b.app"
                :disabled="retargetFrozen" @input="invalidateConnection" />
            </t-form-item>
          </div>
          <t-form-item :label="'Proxy endpoint'" :help="'Data-plane gateway of a self-hosted E2B-compatible cluster. Leave empty to reach sandboxes through the sandbox domain, as E2B Cloud expects.'">
            <t-input v-model="e2b.proxy_url" placeholder="http://sandbox-gateway.example.com"
              :disabled="retargetFrozen" @input="invalidateConnection" />
          </t-form-item>
        </template>
        <div class="private-endpoint-row">
          <div>
            <p class="private-endpoint-row__title">{{ 'Allow private cluster endpoints' }}</p>
            <p class="section-help">{{ 'Use for self-hosted private control planes. Link-local and cloud metadata addresses remain blocked.' }}</p>
          </div>
          <t-switch
            v-model="allowPrivateEndpoints"
            :disabled="retargetFrozen"
            @change="invalidateConnection"
          />
        </div>
      </section>

      <section v-if="currentStepKey === 'connection' && !isRemoteBackend" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Runtime environment' }}</h4>
        <div class="enterpriserag-template-card is-active">
          <SandboxBackendBadge type="docker" />
          <div class="enterpriserag-template-card__content">
            <div class="enterpriserag-template-card__title-row">
              <span class="enterpriserag-template-card__title">{{ 'sustainability.ai standard image' }}</span>
              <t-tag theme="primary" variant="light" size="small">{{ 'Recommended' }}</t-tag>
            </div>
            <p>{{ 'Each session gets its own long-lived container. Scripts, shell commands and files all share it until the session ends or the idle timeout reclaims it.' }}</p>
          </div>
        </div>
        <t-form-item :label="requiredLabel('dockerImage')" :status="fieldStatus('image')"
          :tips="fieldTip('image')">
          <t-input v-model="docker.image" :placeholder="defaultDockerImage"
            :disabled="retargetFrozen" @input="onFieldInput('image')" />
          <p v-if="retargetFrozen" class="section-help section-help--field">
            {{ hasSkillSnapshot
              ? 'This sandbox already has skills. The skill environment is bound to the current snapshot, so the runtime template cannot be changed or rebuilt. Create a new sandbox and install skills from the new template.'
              : 'A skill is still installing or being removed. The runtime template cannot be changed or rebuilt until that finishes.' }}
          </p>
        </t-form-item>
        <t-form-item :label="'Docker daemon endpoint'" :help="'Empty follows the local docker CLI (DOCKER_HOST or the current docker context), so you do not have to type /var/run/docker.sock. For a remote daemon use tcp://host:2376, fill in the TLS certificate directory, and turn on \u0022allow private endpoints\u0022 for RFC1918 addresses.'">
          <t-input v-model="docker.host" placeholder="unix:///var/run/docker.sock"
            :disabled="retargetFrozen" @input="onFieldInput('host')" />
        </t-form-item>
        <t-form-item :label="'TLS certificate directory'"
          :help="'Directory on the sustainability.ai host holding ca.pem, cert.pem and key.pem. Required for a remote daemon; certificates are mounted by the deployment, never stored here.'">
          <t-input v-model="docker.tls_cert_path" placeholder="/etc/enterpriserag/docker-certs"
            :disabled="retargetFrozen" @input="onFieldInput('tls_cert_path')" />
        </t-form-item>
        <t-alert theme="warning" class="compact-alert" :message="'Empty or unix:// uses the Docker daemon on the sustainability.ai host, which is equivalent to root on that machine. Use this only for a private single-node install. Prefer Cube or E2B when multiple workspaces share a host. Remote tcp:// endpoints require a TLS certificate directory.'" />
        <div class="private-endpoint-row">
          <div>
            <p class="private-endpoint-row__title">{{ 'Allow private cluster endpoints' }}</p>
            <p class="section-help">{{ 'Use for self-hosted private control planes. Link-local and cloud metadata addresses remain blocked.' }}</p>
          </div>
          <t-switch
            v-model="allowPrivateEndpoints"
            :disabled="retargetFrozen"
            @change="invalidateConnection"
          />
        </div>
      </section>

      <section v-if="currentStepKey === 'template'" class="setting-drawer__section">
        <div class="section-title-row">
          <h4 class="setting-drawer__section-title">{{ 'Runtime template' }}</h4>
          <t-button variant="text" size="small" :loading="templatesLoading" @click="loadTemplates()">
            <template #icon><t-icon name="refresh" /></template>
            {{ 'Refresh templates' }}
          </t-button>
        </div>
        <t-alert v-if="hasSkillSnapshot" theme="info" class="compact-alert"
          :message="'This sandbox already has skills. The skill environment is bound to the current snapshot, so the runtime template cannot be changed or rebuilt. Create a new sandbox and install skills from the new template.'" />
        <t-alert v-else-if="hasInFlightSkill" theme="info" class="compact-alert"
          :message="'A skill is still installing or being removed. The runtime template cannot be changed or rebuilt until that finishes.'" />
        <div v-if="templatesLoading && !templatesLoaded" class="template-loading">
          <t-loading size="small" />
          <span>{{ 'Loading templates from the cluster...' }}</span>
        </div>
        <div v-else class="template-list" role="radiogroup" :aria-label="'Runtime template'">
          <div v-if="canCreateStandard" class="template-row template-row--offer">
            <div class="template-row__main">
              <div class="template-row__head">
                <span class="template-row__title">{{ 'sustainability.ai standard template' }}</span>
                <t-tag theme="primary" variant="outline" size="small">
                  {{ 'Recommended' }}
                </t-tag>
                <span class="template-row__spacer" />
                <t-button theme="primary" variant="outline" size="small" :loading="templatesLoading"
                  @click="createStandardTemplate">
                  {{ 'Create' }}
                </t-button>
              </div>
              <p class="template-row__hint">{{ 'Built with the current connection settings, including DNS. After changing those settings, rebuild from the card.' }}</p>
            </div>
          </div>
          <div v-if="canCreateDesktop" class="template-row template-row--offer">
            <div class="template-row__main">
              <div class="template-row__head">
                <span class="template-row__title">{{ 'sustainability.ai desktop template' }}</span>
                <t-tag theme="warning" variant="outline" size="small">
                  {{ 'Desktop' }}
                </t-tag>
                <span class="template-row__spacer" />
                <t-button theme="primary" variant="outline" size="small" :loading="templatesLoading"
                  @click="createDesktopTemplate">
                  {{ 'Create' }}
                </t-button>
              </div>
              <p class="template-row__hint">{{ 'Builds an XFCE graphical desktop from the official desktop image. It is much larger than the CLI template; create it only when you need a GUI.' }}</p>
            </div>
          </div>
          <div
            v-for="item in templates"
            :key="item.id"
            class="template-row"
            :class="{
              'is-active': currentTemplateId === item.id,
              'is-pending': isTemplatePending(item),
              'is-disabled': !isTemplateSelectable(item) || (retargetFrozen && currentTemplateId !== item.id),
            }"
            role="radio"
            :aria-checked="currentTemplateId === item.id"
            :aria-disabled="!isTemplateSelectable(item) || (retargetFrozen && currentTemplateId !== item.id)"
            :tabindex="isTemplateSelectable(item) && !(retargetFrozen && currentTemplateId !== item.id) ? 0 : -1"
            @click="onTemplateCardClick(item)"
            @keydown.enter.prevent="onTemplateCardClick(item)"
            @keydown.space.prevent="onTemplateCardClick(item)"
          >
            <span class="template-row__marker" aria-hidden="true" />
            <div class="template-row__main">
              <div class="template-row__head">
                <span class="template-row__title" :title="templateDisplayName(item)">
                  {{ templateDisplayName(item) }}
                </span>
                <t-tag v-if="item.standard" theme="primary" variant="outline" size="small">
                  {{ 'Recommended' }}
                </t-tag>
                <t-tag v-else-if="item.desktop" theme="warning" variant="outline" size="small">
                  {{ 'Desktop' }}
                </t-tag>
                <span class="template-row__spacer" />
                <t-tag :theme="templateStatusTheme(item)" variant="outline" size="small">
                  {{ templateStatusLabel(item) }}
                </t-tag>
                <span v-if="canRebuildTemplate(item)" class="template-row__rebuild" @click.stop>
                  <t-popconfirm
                    theme="warning"
                    :content="item.desktop
                      ? 'Rebuild the sustainability.ai desktop template with the current settings, including DNS. The previous spawnable desktop template is not deleted until the replacement is ready. The CLI template is left untouched.'
                      : 'Rebuild the sustainability.ai standard template with the current settings, including DNS. The previous spawnable template is not deleted until the replacement is ready.'"
                    @confirm="replaceFirstPartyTemplate(item)"
                  >
                    <t-button variant="text" size="small" :loading="templatesLoading">
                      {{ 'Rebuild' }}
                    </t-button>
                  </t-popconfirm>
                </span>
              </div>
              <dl v-if="templateFieldRows(item).length" class="template-row__fields">
                <div v-for="field in templateFieldRows(item)" :key="field.key" class="template-row__field">
                  <dt>{{ field.label }}</dt>
                  <dd :class="{ 'is-mono': field.mono }" :title="field.value">{{ field.value }}</dd>
                </div>
              </dl>
              <p v-if="isTemplateUntagged(item)" class="template-row__hint template-row__hint--error">
                {{ 'The builds finished but none carries the default tag, so sandbox creation cannot resolve this template. Delete it in E2B and refresh; sustainability.ai will rebuild it.' }}
              </p>
              <p v-else-if="templateFailureReason(item)" class="template-row__hint template-row__hint--error">
                {{ templateFailureReason(item) }}
              </p>
              <p v-else-if="isTemplatePending(item) && (item.standard || item.desktop)" class="template-row__hint">
                {{ 'This template is being built. The list will refresh.' }}
              </p>
            </div>
          </div>
          <div
            v-if="templatesLoaded && !templates.length && !canCreateStandard && !templatesError"
            class="env-empty"
          >
            {{ 'No templates were returned by this cluster.' }}
          </div>
        </div>
        <t-alert v-if="templatesError" theme="warning" class="compact-alert" :message="templatesError" />
        <a class="inline-guide-link" :href="clusterGuideUrl" target="_blank" rel="noopener noreferrer">
          <t-icon name="link" />
          {{ 'Sandbox cluster setup and template guide' }}
        </a>
      </section>

      <section v-if="currentStepKey === 'runtime'" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Execution settings' }}</h4>
        <div class="runtime-fields">
          <!--
            Each number keeps its own label and its own sentence — the reason
            these were one-per-row originally was that three unlabelled numbers
            side by side shared a single footnote. Moving the sentence into the
            field's own tips keeps that fixed while halving the height.
          -->
          <div class="form-grid form-grid--two">
            <template v-if="isRemoteBackend">
              <t-form-item :label="'HTTP timeout (s)'"
                :tips="'How long a management call to the backend may take before the endpoint counts as unreachable. Empty means 30 seconds.'">
                <t-input-number v-if="backend === 'cube'" v-model="cube.http_timeout_sec" :min="0"
                  theme="column" placeholder="30" />
                <t-input-number v-else v-model="e2b.http_timeout_sec" :min="0" theme="column"
                  placeholder="30" />
              </t-form-item>
              <t-form-item :label="'Sandbox TTL (s)'"
                :tips="'How long until the sandbox is paused'">
                <t-input-number v-if="backend === 'cube'" v-model="cube.cube_sandbox_ttl_seconds"
                  :min="0" theme="column" placeholder="1800" />
                <t-input-number v-else v-model="e2b.e2b_sandbox_ttl_seconds" :min="0"
                  theme="column" placeholder="300" />
              </t-form-item>
            </template>
            <!--
              Docker has no provider-side timeout at all: an abandoned container
              keeps its memory and CPU share on the daemon host until sustainability.ai
              reclaims it, so the idle TTL and the resource caps are the only
              things bounding what one workspace can hold.
            -->
            <template v-if="backend === 'docker'">
              <t-form-item :label="'Idle reclaim (seconds)'"
                :tips="'The Docker daemon has no idle timeout of its own. A container that runs no command for this long is reclaimed by sustainability.ai and rebuilt when the session continues. Empty means 1800 seconds.'">
                <t-input-number v-model="docker.idle_ttl_seconds" :min="0" theme="column"
                  placeholder="1800" />
              </t-form-item>
              <t-form-item :label="'CPU cores'"
                :tips="'CPU cores available to one sandbox; 0 uses the built-in default.'">
                <t-input-number v-model="docker.cpu_limit" :min="0" :step="0.5" theme="column"
                  placeholder="2" />
              </t-form-item>
              <t-form-item :label="'Memory limit (MB)'"
                :tips="'Memory limit in MB for one sandbox; 0 uses the built-in default.'">
                <t-input-number v-model="docker.memory_limit_mb" :min="0" theme="column"
                  placeholder="2048" />
              </t-form-item>
              <t-form-item :label="'Process limit'"
                :tips="'Maximum processes one sandbox can create; 0 uses the built-in default.'">
                <t-input-number v-model="docker.pids_limit" :min="0" theme="column"
                  placeholder="512" />
              </t-form-item>
            </template>
            <t-form-item :label="'Execution timeout (s)'"
              :tips="'Longest a single skill script may run before it is killed. Empty means 60 seconds.'">
              <t-input-number v-model="defaultTimeoutSec" :min="0" theme="column" placeholder="60" />
            </t-form-item>
            <t-form-item :label="'Terminal / desktop idle disconnect (s)'"
              :tips="'After the terminal or desktop is open, close it if there is no interaction for this long so the sandbox can pause on its TTL. Terminal counts keyboard and PTY output; desktop counts mouse and keyboard. Empty means 900 seconds; minimum 60 seconds, maximum 24 hours.'">
              <t-input-number v-model="terminalIdleDisconnectSec" :min="0" :max="86400"
                theme="column" placeholder="900" />
            </t-form-item>
          </div>
        </div>
      </section>

      <section v-if="currentStepKey === 'runtime'" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'Network policy' }}</h4>
        <p class="section-help section-help--under-title">
          {{ 'Controls outbound networking for every sandbox using this configuration. Changes affect only newly created sandboxes; existing sandboxes keep their policy until reclaimed.' }}
        </p>

        <template v-if="backend !== 'docker'">
          <t-form-item :label="'Default egress'"
            :tips="'Evaluation order: allow, deny, then the default. Allow rules take precedence over deny rules.'">
            <t-radio-group v-model="denyEgressByDefault">
              <t-radio :value="false">{{ 'Allow public network (default)' }}</t-radio>
              <t-radio :value="true">{{ 'Deny by default' }}</t-radio>
            </t-radio-group>
          </t-form-item>
        </template>

        <template v-if="backend === 'docker'">
          <t-form-item :label="'Network mode'"
            :tips="'Defaults to bridge, which skills need to install packages. Choose none for no egress at all. Docker filters by network only; per-domain rules are not possible here.'">
            <t-select v-model="docker.network_mode"
              :placeholder="'bridge (egress allowed)'" clearable>
              <t-option value="bridge" :label="'bridge (egress allowed)'" />
              <t-option value="none" :label="'none (no egress)'" />
            </t-select>
          </t-form-item>
        </template>

        <template v-else>
          <div class="net-list">
            <div class="section-title-row">
              <span class="net-list__title">{{ 'Allowed destinations' }}</span>
              <t-button variant="text" size="small" @click="allowOutRows.push('')">
                <template #icon><t-icon name="add" /></template>
                {{ 'Add destination' }}
              </t-button>
            </div>
            <div v-for="(_, index) in allowOutRows" :key="`allow-${index}`" class="net-row">
              <t-input v-model="allowOutRows[index]"
                :placeholder="'Domain / IP / CIDR, for example *.example.com'" />
              <t-button variant="text" shape="square" size="small"
                :aria-label="'Delete'" @click="allowOutRows.splice(index, 1)">
                <t-icon name="close" />
              </t-button>
            </div>
            <p class="section-help">{{ 'Supports IPv4, CIDR, domains, and single-label wildcards such as *.example.com (which do not match the root domain).' }}</p>
            <t-alert v-if="domainAllowNeedsDenyAll" theme="warning" class="compact-alert"
              :message="'When allowed destinations contain a domain, also choose “Deny by default” or add 0.0.0.0/0 to denied destinations; otherwise the allowlist is ineffective.'" />
          </div>

          <div class="net-list">
            <div class="section-title-row">
              <span class="net-list__title">{{ 'Denied destinations' }}</span>
              <t-button variant="text" size="small" @click="denyOutRows.push('')">
                <template #icon><t-icon name="add" /></template>
                {{ 'Add destination' }}
              </t-button>
            </div>
            <div v-for="(_, index) in denyOutRows" :key="`deny-${index}`" class="net-row">
              <t-input v-model="denyOutRows[index]"
                :placeholder="'IP / CIDR only, for example 169.254.169.254/32'" />
              <t-button variant="text" shape="square" size="small"
                :aria-label="'Delete'" @click="denyOutRows.splice(index, 1)">
                <t-icon name="close" />
              </t-button>
            </div>
            <p class="section-help">{{ 'Deny rules match destination IP only, so domains are not supported.' }}</p>
          </div>
        </template>

        <div v-if="backend === 'cube'" class="net-list">
          <div class="section-title-row">
            <span class="net-list__title">{{ 'HTTP access rules (L7)' }}</span>
            <t-button variant="text" size="small" @click="addCubeRule()">
              <template #icon><t-icon name="add" /></template>
              {{ 'Add rule' }}
            </t-button>
          </div>
          <p class="section-help">{{ 'Every rule requires host or sni; the network layer derives allowed targets only from those fields. Fields are AND-ed and methods are OR-ed. Applies only to HTTP 80 / HTTPS 443. Rules are first-match-wins from top to bottom.' }}</p>
          <div v-for="(rule, index) in cubeRules" :key="rule.key"
            class="net-rule net-rule--collapsible" :class="{ 'is-open': rule.expanded }">
            <div class="net-rule__bar">
              <button
                type="button"
                class="net-rule__toggle"
                :aria-expanded="rule.expanded"
                :aria-label="rule.expanded
                  ? 'Collapse rule'
                  : 'Expand rule'"
                @click="rule.expanded = !rule.expanded"
              >
                <t-icon :name="rule.expanded ? 'chevron-down' : 'chevron-right'" size="14px" />
                <span class="net-rule__name" :class="{ 'is-empty': !rule.name.trim() }">
                  {{ rule.name.trim() || 'Untitled rule' }}
                </span>
              </button>
              <div class="net-rule__actions">
                <button type="button" class="net-rule__move"
                  :disabled="index === 0"
                  :aria-label="'Move rule up'"
                  @click="moveCubeRule(index, -1)">
                  <t-icon name="chevron-up" size="14px" />
                </button>
                <button type="button" class="net-rule__move"
                  :disabled="index === cubeRules.length - 1"
                  :aria-label="'Move rule down'"
                  @click="moveCubeRule(index, 1)">
                  <t-icon name="chevron-down" size="14px" />
                </button>
                <button type="button" class="net-rule__remove"
                  :aria-label="'Delete'" @click="cubeRules.splice(index, 1)">
                  <t-icon name="close" size="14px" />
                </button>
              </div>
            </div>
            <div v-if="rule.expanded" class="net-rule__body">
              <div class="form-grid form-grid--two">
                <t-form-item :label="'Rule name'">
                  <t-input v-model="rule.name" placeholder="allow-payment-api" />
                </t-form-item>
                <t-form-item :label="'Scheme'">
                  <t-select v-model="rule.scheme" clearable>
                    <t-option value="https" label="https" />
                    <t-option value="http" label="http" />
                  </t-select>
                </t-form-item>
                <t-form-item :label="'SNI'">
                  <t-input v-model="rule.sni" placeholder="api.example.com" />
                </t-form-item>
                <t-form-item :label="'Host'">
                  <t-input v-model="rule.host" placeholder="api.example.com" />
                </t-form-item>
                <t-form-item :label="'HTTP methods'">
                  <t-input v-model="rule.methodsText" placeholder="POST, GET" />
                </t-form-item>
                <t-form-item :label="'Path'">
                  <t-input v-model="rule.path" placeholder="/v1/*" />
                </t-form-item>
                <t-form-item :label="'Action'">
                  <t-select v-model="rule.deny">
                    <t-option :value="false" :label="'Allow'" />
                    <t-option :value="true" :label="'Deny'" />
                  </t-select>
                </t-form-item>
                <t-form-item :label="'Audit level'">
                  <t-select v-model="rule.audit" clearable>
                    <t-option value="metadata" label="metadata" />
                    <t-option value="full" label="full" />
                    <t-option value="none" label="none" />
                  </t-select>
                </t-form-item>
              </div>
              <div v-if="!rule.deny" class="net-inject">
                <span class="net-list__title">{{ 'Inject headers' }}</span>
                <div v-for="(inject, injectIndex) in rule.inject" :key="`inject-${injectIndex}`"
                  class="net-row net-row--triple">
                  <t-input v-model="inject.header" :placeholder="'Header name'" />
                  <t-input v-model="inject.secret" type="password"
                    :placeholder="isStoredNetworkSecretRecoverable(
                      inject,
                      inject.originalRuleName,
                      inject.originalHeader,
                      rule.name,
                      inject.header,
                    )
                      ? 'Configured — leave empty to keep it'
                      : 'Header value'" />
                  <t-input v-model="inject.format" placeholder="Bearer ${SECRET}" />
                  <t-button variant="text" shape="square" size="small"
                    :aria-label="'Delete'" @click="rule.inject.splice(injectIndex, 1)">
                    <t-icon name="close" />
                  </t-button>
                </div>
                <t-button variant="text" size="small"
                  @click="rule.inject.push({ header: '', secret: '', format: '' })">
                  <template #icon><t-icon name="add" /></template>
                  {{ 'Add header' }}
                </t-button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="backend === 'e2b'" class="net-list">
          <div class="section-title-row">
            <span class="net-list__title">{{ 'Host request transforms' }}</span>
            <t-button variant="text" size="small" @click="addE2BHostRule()">
              <template #icon><t-icon name="add" /></template>
              {{ 'Add rule' }}
            </t-button>
          </div>
          <p class="section-help">{{ 'Inject headers by host. A rule does not authorize egress; its host must also appear in allowed destinations.' }}</p>
          <div v-for="(rule, index) in e2bHostRules" :key="`e2b-rule-${index}`"
            class="net-rule net-rule--collapsible" :class="{ 'is-open': rule.expanded }">
            <div class="net-rule__bar">
              <button
                type="button"
                class="net-rule__toggle"
                :aria-expanded="rule.expanded"
                :aria-label="rule.expanded
                  ? 'Collapse rule'
                  : 'Expand rule'"
                @click="rule.expanded = !rule.expanded"
              >
                <t-icon :name="rule.expanded ? 'chevron-down' : 'chevron-right'" size="14px" />
                <span class="net-rule__name" :class="{ 'is-empty': !rule.host.trim() }">
                  {{ rule.host.trim() || 'Untitled rule' }}
                </span>
              </button>
              <button type="button" class="net-rule__remove"
                :aria-label="'Delete'" @click="e2bHostRules.splice(index, 1)">
                <t-icon name="close" size="14px" />
              </button>
            </div>
            <div v-if="rule.expanded" class="net-rule__body">
              <t-form-item :label="'Host'">
                <t-input v-model="rule.host" placeholder="api.example.com" />
              </t-form-item>
              <div v-for="(header, headerIndex) in rule.headers" :key="`header-${headerIndex}`"
                class="net-row net-row--double">
                <t-input v-model="header.name" :placeholder="'Header name'" />
                <t-input v-model="header.value" type="password"
                  :placeholder="isStoredNetworkSecretRecoverable(
                    header,
                    header.originalHost,
                    header.originalName,
                    rule.host,
                    header.name,
                  )
                    ? 'Configured — leave empty to keep it'
                    : 'Header value'" />
                <t-button variant="text" shape="square" size="small"
                  :aria-label="'Delete'" @click="rule.headers.splice(headerIndex, 1)">
                  <t-icon name="close" />
                </t-button>
              </div>
              <t-button variant="text" size="small"
                @click="rule.headers.push({ name: '', value: '' })">
                <template #icon><t-icon name="add" /></template>
                {{ 'Add header' }}
              </t-button>
            </div>
          </div>
        </div>
      </section>

      <section v-if="currentStepKey === 'runtime'" class="setting-drawer__section">
        <div class="section-title-row">
          <h4 class="setting-drawer__section-title">{{ 'Environment variables' }}</h4>
          <t-button variant="text" size="small" @click="envRows.push({ key: '', value: '' })">
            <template #icon><t-icon name="add" /></template>
            {{ 'Add variable' }}
          </t-button>
        </div>
        <p class="section-help section-help--under-title">{{ 'Injected into every sandbox created from this config. Values are encrypted at rest but visible to scripts inside the sandbox.\nInjected only when a sandbox is created; to inject on each run, add them on the Sandbox secrets page.' }}</p>
        <div v-if="envRows.length" class="env-rows">
          <div v-for="(row, index) in envRows" :key="index" class="env-row">
            <t-input v-model="row.key" :placeholder="'Name'" class="env-key" />
            <t-input v-model="row.value" type="password"
              :placeholder="row.stored ? 'Configured — leave empty to keep it' : 'Value'"
              class="env-value" />
            <t-button variant="text" shape="square" size="small" :aria-label="'Delete'"
              @click="envRows.splice(index, 1)">
              <t-icon name="close" />
            </t-button>
          </div>
        </div>
        <div v-else class="env-empty">{{ 'No additional environment variables.' }}</div>
      </section>

      <section v-if="currentStepKey === 'runtime'" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ 'How the new skill image takes effect' }}</h4>
        <p class="section-help section-help--under-title">{{ 'Installing or removing a skill builds a new image. This controls whether already-open sessions switch to it.' }}</p>
        <t-radio-group v-model="skillRollout" class="skill-rollout-group">
          <t-radio value="next_turn">{{ 'Rebuild open sessions on the next chat turn' }}</t-radio>
          <t-radio value="new_session">{{ 'Keep open sessions on their current sandbox; only new sessions use the new image' }}</t-radio>
        </t-radio-group>
      </section>
    </t-form>

    <div
      v-if="checkResult && showCheckResult"
      ref="checkResultRef"
      class="check-result"
    >
      <p :class="['check-result__title', checkResult.ok ? 'is-success' : 'is-error']">
        <t-icon :name="checkResult.ok ? 'check-circle-filled' : 'close-circle-filled'" />
        {{ checkResult.ok ? 'All checks passed' : 'Some checks failed' }}
      </p>
      <p class="check-result__subtitle">{{ checkScopeHint }}</p>
      <ul class="check-list">
        <li v-for="item in reportedChecks" :key="item.name" class="check-item">
          <t-icon :name="item.ok === true ? 'check-circle-filled'
            : item.ok === false ? 'close-circle-filled' : 'minus-circle'"
            :class="item.ok === true ? 'ok' : item.ok === false ? 'err' : 'skip'" />
          <span class="check-name">{{ checkLabel(item.name) }}</span>
          <span v-if="item.latency_ms" class="check-latency">{{ item.latency_ms }} ms</span>
          <span v-if="checkDetail(item)" class="check-message">{{ checkDetail(item) }}</span>
        </li>
      </ul>
      <p v-if="pendingCheckNames.length" class="check-result__hint">
        {{ `${pendingCheckNames.join(', ')} can only be confirmed by a full verification, which creates a throwaway sandbox, runs one script and destroys it.` }}
      </p>
    </div>

  </SettingDrawer>
</template>

<script setup lang="ts">
import { computed, nextTick, onUnmounted, reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import SandboxBackendBadge from '@/components/settings/SandboxBackendBadge.vue'
import { useDeploymentCapabilitiesStore } from '@/stores/deploymentCapabilities'
import {
  checkSandboxConfig,
  createSandboxConfig,
  parseSandboxConflict,
  updateSandboxConfigById,
  querySandboxTemplates,
  listConfigSkills,
  type SandboxCheckItem,
  type SandboxCheckResult,
  type SandboxConfig,
  type SandboxConfigRecord,
  type SandboxConflict,
  type SandboxCubeConfig,
  type SandboxE2BConfig,
  type SandboxDockerConfig,
  type SandboxNetworkPolicy,
  type SandboxTemplate,
  isNamedSandboxBackend,
  NAMED_SANDBOX_BACKEND_TYPES,
} from '@/api/system'

const SETTINGS_SANDBOX_BACKEND_DESCRIPTIONS_LABELS: Record<string, string> = {
  cube: 'Self-hosted MicroVM cluster for private or on-premises deployments',
  e2b: 'Managed MicroVM service or an E2B-compatible deployment',
  docker: 'Keep a long-lived container per session on this sustainability.ai host; scripts and files stay in that container',
}

const SETTINGS_SANDBOX_STEP_DESCRIPTIONS_LABELS: Record<string, string> = {
  connection: 'Configure the backend and verify the connection before loading templates.',
  template: 'Choose a ready template returned by the connected cluster.',
  runtime: 'Configure execution settings, environment variables, and when skill image updates apply, then save.',
  skills: 'Install skills into this config\'s sandbox image. Once the config is saved you can come back any time.',
}

const SETTINGS_SANDBOX_BACKENDS_LABELS: Record<string, string> = {
  disabled: 'Disabled',
  local: 'Local process',
  docker: 'Docker',
  cube: 'CubeSandbox',
  e2b: 'E2B',
}

const SETTINGS_SANDBOX_LABELS: Record<string, string> = {
  title: 'Sandbox Config',
  description: 'Configure isolated runtimes for agent skill scripts. Each agent picks one workspace config.',
  pageHintTitle: 'What is a sandbox?',
  pageHint: 'A sandbox is the isolated environment where agent skill scripts run. A workspace can have several configs (Docker, E2B, CubeSandbox); each agent picks one. Skills are installed on the Skill Management page into the selected config\'s image. Scripts stay disabled until a config is selected.',
  editorDescription: 'Configure a workspace runtime. Docker, CubeSandbox, and E2B use the same management flow.',
  stepConnection: 'Connect',
  stepTemplate: 'Template',
  stepRuntime: 'Runtime',
  stepSkills: 'Skills',
  stepSkillsLocked: 'Skills are installed into this config\'s image. Save the config first, then come back to this step.',
  setupProgress: 'Sandbox setup progress',
  back: 'Back',
  connectAndContinue: 'Connect and continue',
  loading: 'Loading sandbox configuration...',
  loadFailed: 'Failed to load sandbox configuration',
  backend: 'Sandbox',
  backendType: 'Sandbox type',
  backendTypePlaceholder: 'Select a sandbox type',
  scriptPolicyLabel: 'Allow skill scripts to run in sandboxes',
  scriptPolicyDesc: 'When off, agents in this workspace can only read skill content. Remote sandboxes already running are released once their sessions end.',
  dockerDisabledAlert: 'Docker sandbox is not enabled on this deployment',
  dockerDisabledHint: 'A local docker.sock is equivalent to root on the host. For a single-machine private install, a system admin can enable it under Settings → System settings → Network security.',
  dockerDisabledCard: 'Docker sandbox is disabled on this deployment; this config will not create containers',
  dockerHostRisk: 'Empty or unix:// uses the Docker daemon on the sustainability.ai host, which is equivalent to root on that machine. Use this only for a private single-node install. Prefer Cube or E2B when multiple workspaces share a host. Remote tcp:// endpoints require a TLS certificate directory.',
  addConfig: 'Add sandbox',
  viewClusterGuide: 'Cluster setup guide',
  configName: 'Config name',
  configNamePlaceholder: 'e.g. Production E2B',
  configNameRequired: 'Please enter a config name',
  configDescription: 'Description',
  configDescriptionPlaceholder: 'Optional, helps tell several configs of the same type apart',
  createTitle: 'Add sandbox',
  editTitle: 'Edit sandbox',
  sectionBasic: 'Basic information',
  sectionConnection: 'Cluster connection',
  sectionRuntimeEnvironment: 'Runtime environment',
  sectionTemplate: 'Runtime template',
  sectionRuntime: 'Execution settings',
  sectionNetwork: 'Network policy',
  sectionEnvironment: 'Environment variables',
  networkHint: 'Controls outbound networking for every sandbox using this configuration. Changes affect only newly created sandboxes; existing sandboxes keep their policy until reclaimed.',
  egressDefault: 'Default egress',
  egressAllowAll: 'Allow public network (default)',
  egressDenyAll: 'Deny by default',
  egressPrecedence: 'Evaluation order: allow, deny, then the default. Allow rules take precedence over deny rules.',
  allowOut: 'Allowed destinations',
  allowOutPlaceholder: 'Domain / IP / CIDR, for example *.example.com',
  allowOutHelp: 'Supports IPv4, CIDR, domains, and single-label wildcards such as *.example.com (which do not match the root domain).',
  denyOut: 'Denied destinations',
  denyOutPlaceholder: 'IP / CIDR only, for example 169.254.169.254/32',
  denyOutHelp: 'Deny rules match destination IP only, so domains are not supported.',
  domainAllowNeedsDenyAll: 'When allowed destinations contain a domain, also choose “Deny by default” or add 0.0.0.0/0 to denied destinations; otherwise the allowlist is ineffective.',
  cubeL7Rules: 'HTTP access rules (L7)',
  cubeL7RulesHelp: 'Every rule requires host or sni; the network layer derives allowed targets only from those fields. Fields are AND-ed and methods are OR-ed. Applies only to HTTP 80 / HTTPS 443. Rules are first-match-wins from top to bottom.',
  e2bHostRules: 'Host request transforms',
  e2bHostRulesHelp: 'Inject headers by host. A rule does not authorize egress; its host must also appear in allowed destinations.',
  ruleUntitled: 'Untitled rule',
  expandRule: 'Expand rule',
  collapseRule: 'Collapse rule',
  moveRuleUp: 'Move rule up',
  moveRuleDown: 'Move rule down',
  ruleName: 'Rule name',
  ruleScheme: 'Scheme',
  ruleSni: 'SNI',
  ruleHost: 'Host',
  ruleMethods: 'HTTP methods',
  rulePath: 'Path',
  ruleAction: 'Action',
  ruleAllow: 'Allow',
  ruleDeny: 'Deny',
  ruleAudit: 'Audit level',
  ruleInject: 'Inject headers',
  headerName: 'Header name',
  headerValue: 'Header value',
  addTarget: 'Add destination',
  addRule: 'Add rule',
  addHeader: 'Add header',
  removeRule: 'Remove rule',
  noConfigs: 'No sandbox yet. Agents without a workspace configuration will not run skill scripts.',
  identityFieldHint: 'While this config owns sandboxes, these cannot be changed: backend type, API endpoint, API key, sandbox domain, proxy endpoint.',
  connectionLockedBySkills: 'This sandbox already has skills. Connection, credentials, and DNS would retarget the skill snapshot, and DNS only applies after a template rebuild. Create a new sandbox instead.',
  connectionLockedByInFlight: 'A skill is still installing or being removed. Connection, credentials, and DNS cannot change until that finishes.',
  viewSandboxes: 'Running instances',
  inventoryTitle: 'Running instances',
  inventoryDrawerDesc: 'Sandboxes this config still holds, plus the sessions and agents affected.',
  inventoryFailed: 'Failed to load sandbox usage',
  sandboxCount: 'Sandboxes',
  sandboxCountUnknown: 'Unknown (backend unreachable)',
  inventoryUnverifiableHint: 'The backend is unreachable, so the sandbox count is unknown — that is not the same as none.',
  inventorySessions: 'Sessions',
  inventorySessionKind: 'Conversation',
  inventoryEmpty: 'No occupying sessions',
  inventoryAgentsTitle: 'Linked agents',
  inventoryUntitledSession: 'Untitled session',
  sandboxesStillLive: 'This config still owns {count} running or paused sandbox(es), so identity fields cannot be changed yet.',
  blockedHint: 'End or delete those sessions (deleting a session destroys its sandbox), or create a second config and point the agents at it.',
  unverifiableBlocked: 'The backend cannot be reached to verify whether sandboxes remain, so the stored credentials will not be overwritten.',
  unverifiableSaveHint: 'Restore connectivity first; if the backend is gone for good, create a second config and point the agents at it.',
  affectedSessions: '{count} session(s) affected.',
  affectedAgents: 'Agents using this config: {names}',
  confirmDelete: 'Delete sandbox "{name}"?',
  confirmDeleteWithAgents: 'Delete sandbox "{name}"? {agents}They will fail on their next skill execution.',
  deleteFailed: 'Failed to delete',
  deleted: 'Deleted',
  forceDeleteTitle: 'Sandbox usage cannot be verified',
  forceDeleteConfirm: 'The backend cannot be reached to verify whether sandboxes remain. If it is gone for good you can force-delete this config; if it is only temporarily unreachable, forcing it leaves any remaining sandboxes with nobody to reclaim them. Force delete anyway?',
  forceDelete: 'Force delete',
  disableScripts: 'Disable sandbox execution',
  enableScripts: 'Enable sandbox execution',
  disableScriptsConfirm: 'All agents in this workspace — will no longer run skill scripts in a sandbox. They can still read skill content. Existing remote sandboxes are not destroyed automatically; end or delete the related sessions to release them. Continue?',
  scriptsDisabled: 'Sandbox execution disabled for this workspace',
  scriptsEnabled: 'Sandbox execution restored for this workspace',
  policySaveFailed: 'Failed to update sandbox execution policy',
  legacyConfig: 'Deprecated',
  namedBackendHint: 'Workspace configuration is the only runtime source. Agents without one cannot execute skill scripts.',
  enterpriseragTemplateTitle: 'sustainability.ai standard template',
  enterpriseragDockerImage: 'sustainability.ai standard image',
  enterpriseragDockerImageHint: 'Each session gets its own long-lived container. Scripts, shell commands and files all share it until the session ends or the idle timeout reclaims it.',
  enterpriseragTemplateOverview: 'sustainability.ai provides the standard runtime. Templates are discovered after connecting and the standard one is created when missing.',
  enterpriseragTemplateDescription: 'Includes the Python, Node.js, CLI tools, workspace path, and non-root execution user expected by sustainability.ai skills.',
  recommendedTag: 'Recommended',
  cardTemplateConfigured: 'Template configured',
  cardCredentialMissing: 'API key missing',
  cardTimeout: 'Timeout {sec}s',
  cardTtl: 'Sandbox TTL {sec}s',
  cardVolumeMounted: 'Volume mounted',
  cardEnvVars: '{count} env vars',
  cardPrivateEndpoints: 'Private endpoints allowed',
  templateNotConfigured: 'Template not configured',
  imageNotConfigured: 'Image not configured',
  templateApplied: 'Applied',
  refreshTemplates: 'Refresh templates',
  templateSelectHelp: 'Templates are loaded from this cluster. The saved configuration stores the ID automatically.',
  templateSelectPlaceholder: 'Connect to the cluster to load templates',
  templateLoadHint: 'Enter the cluster connection and refresh. Missing sustainability.ai CLI templates are built from the official Hub image. Desktop templates are larger; create them from the row below. After changing DNS or the image, rebuild from that card.',
  templateLoadFailed: 'Failed to load templates',
  standardTemplateProvisioning: 'The sustainability.ai standard template is being created. Refresh shortly to see its status.',
  standardTemplateReplaced: 'The previous standard template was deleted and a rebuild has started. Wait until it is ready.',
  templateNotReady: 'The selected template is not ready. Refresh and wait for the build to finish.',
  connectionPassed: 'Connection verified. Templates below are loaded from this cluster.',
  connectionPassedTitle: 'Cluster connected',
  templateStepHint: 'This step lists cluster templates and builds the official CLI image if it is missing. Desktop (XFCE) templates are heavier and are created only when you click Create. After changing DNS or the image, rebuild that card. You can continue after a template is ready.',
  loadingTemplates: 'Loading templates from the cluster...',
  templateBuildingHint: 'This template is being built. The list will refresh.',
  templateUntaggedHint: 'The builds finished but none carries the default tag, so sandbox creation cannot resolve this template. Delete it in E2B and refresh; sustainability.ai will rebuild it.',
  templateFailedReason: 'Build failed: {reason}',
  noTemplates: 'No templates were returned by this cluster.',
  enterpriseragStandardTemplate: 'sustainability.ai standard template',
  createStandardTemplate: 'Create',
  createStandardTemplateHint: 'Built with the current connection settings, including DNS. After changing those settings, rebuild from the card.',
  enterpriseRagDesktopTemplate: 'sustainability.ai desktop template',
  createDesktopTemplate: 'Create',
  createDesktopTemplateHint: 'Builds an XFCE graphical desktop from the official desktop image. It is much larger than the CLI template; create it only when you need a GUI.',
  replaceStandardTemplate: 'Rebuild',
  replaceStandardTemplateConfirm: 'Rebuild the sustainability.ai standard template with the current settings, including DNS. The previous spawnable template is not deleted until the replacement is ready.',
  desktopTemplateProvisioning: 'The sustainability.ai desktop template is being created. Refresh shortly to see its status.',
  desktopTemplateReplaced: 'The previous desktop template was deleted and a rebuild has started. Wait until it is ready. The CLI template is left untouched.',
  desktopTemplateTag: 'Desktop',
  replaceDesktopTemplateConfirm: 'Rebuild the sustainability.ai desktop template with the current settings, including DNS. The previous spawnable desktop template is not deleted until the replacement is ready. The CLI template is left untouched.',
  templateLockedBySkills: 'This sandbox already has skills. The skill environment is bound to the current snapshot, so the runtime template cannot be changed or rebuilt. Create a new sandbox and install skills from the new template.',
  templateLockedByInFlight: 'A skill is still installing or being removed. The runtime template cannot be changed or rebuilt until that finishes.',
  templateUnnamed: 'Unnamed template',
  templateFieldImage: 'Image',
  templateFieldVersion: 'Version',
  templateFieldId: 'ID',
  templateFieldCreated: 'Created',
  templateFieldInstance: 'Instance type',
  templateFieldNetwork: 'Network',
  templateFieldInternet: 'Internet access',
  templateInternetOn: 'On',
  templateInternetOff: 'Off',
  templateReadyHint: 'Template “{name}” is ready and selected.',
  templateProvisioningHint: 'A template is still being built. The status refreshes automatically.',
  allowPrivateEndpoints: 'Allow private cluster endpoints',
  allowPrivateEndpointsHint: 'Use for self-hosted private control planes. Link-local and cloud metadata addresses remain blocked.',
  howToBuildTemplate: 'Sandbox cluster setup and template guide',
  noEnvVars: 'No additional environment variables.',
  fieldRequired: 'Required',
  cubeApiKeyOptional: 'Optional — leave empty for an unauthenticated self-hosted CubeSandbox',
  cubeApiKeyWhere: 'How to enable auth on a self-hosted cluster',
  cubeDnsServers: 'DNS servers',
  cubeDnsServersHelp: 'Optional. Nameserver IPs written into the sustainability.ai standard template. Leave empty to use the cluster default. If UDP/53 to public resolvers is blocked, use reachable addresses from the Cube host\'s /etc/resolv.conf, excluding 10/8, 172.16/12, and 192.168/16. Existing standard templates take effect only after Rebuild on the template card.',
  cubeDnsServersPlaceholder: 'e.g. 8.8.8.8, press Enter to add',
  e2bApiKeyHelp: 'Create one on the API Keys page of the E2B dashboard; it usually starts with e2b_.',
  e2bApiKeyWhere: 'Get an API key from the E2B dashboard',
  apiKeyPlaceholder: 'Paste the API key',
  secretConfigured: 'Key configured (not shown again); enter a new value to rotate',
  secretKeepHint: 'Configured — leave empty to keep it',
  e2bApiUrlOptional: 'Optional — the SDK default is used when empty',
  e2bDomainOptional: 'Optional — the SDK default is used when empty',
  e2bProxyUrlOptional: 'Data-plane gateway of a self-hosted E2B-compatible cluster. Leave empty to reach sandboxes through the sandbox domain, as E2B Cloud expects.',
  apiUrl: 'API endpoint',
  proxyUrl: 'Proxy endpoint',
  sandboxDomain: 'Sandbox domain',
  apiKey: 'API key',
  templateId: 'Template ID',
  httpTimeout: 'HTTP timeout (s)',
  httpTimeoutHelp: 'How long a management call to the backend may take before the endpoint counts as unreachable. Empty means 30 seconds.',
  sandboxTtl: 'Sandbox TTL (s)',
  sandboxTtlHelp: 'How long until the sandbox is paused',
  dockerImage: 'Docker image',
  dockerHost: 'Docker daemon endpoint',
  dockerHostHelp: 'Empty follows the local docker CLI (DOCKER_HOST or the current docker context), so you do not have to type /var/run/docker.sock. For a remote daemon use tcp://host:2376, fill in the TLS certificate directory, and turn on "allow private endpoints" for RFC1918 addresses.',
  dockerTlsCertPath: 'TLS certificate directory',
  dockerTlsCertPathHelp: 'Directory on the sustainability.ai host holding ca.pem, cert.pem and key.pem. Required for a remote daemon; certificates are mounted by the deployment, never stored here.',
  dockerIdleTtl: 'Idle reclaim (seconds)',
  dockerIdleTtlHelp: 'The Docker daemon has no idle timeout of its own. A container that runs no command for this long is reclaimed by sustainability.ai and rebuilt when the session continues. Empty means 1800 seconds.',
  dockerCpuLimit: 'CPU cores',
  dockerCpuLimitHelp: 'CPU cores available to one sandbox; 0 uses the built-in default.',
  dockerMemoryLimit: 'Memory limit (MB)',
  dockerMemoryLimitHelp: 'Memory limit in MB for one sandbox; 0 uses the built-in default.',
  dockerPidsLimit: 'Process limit',
  dockerPidsLimitHelp: 'Maximum processes one sandbox can create; 0 uses the built-in default.',
  dockerNetworkMode: 'Network mode',
  dockerNetworkModeHelp: 'Defaults to bridge, which skills need to install packages. Choose none for no egress at all. Docker filters by network only; per-domain rules are not possible here.',
  dockerNetworkBridge: 'bridge (egress allowed)',
  dockerNetworkNone: 'none (no egress)',
  defaultTimeout: 'Execution timeout (s)',
  defaultTimeoutHelp: 'Longest a single skill script may run before it is killed. Empty means 60 seconds.',
  terminalIdleDisconnect: 'Terminal / desktop idle disconnect (s)',
  terminalIdleDisconnectHelp: 'After the terminal or desktop is open, close it if there is no interaction for this long so the sandbox can pause on its TTL. Terminal counts keyboard and PTY output; desktop counts mouse and keyboard. Empty means 900 seconds; minimum 60 seconds, maximum 24 hours.',
  envVars: 'Environment variables',
  envKey: 'Name',
  envValue: 'Value',
  envVarsHint: 'Injected into every sandbox created from this config. Values are encrypted at rest but visible to scripts inside the sandbox.\nInjected only when a sandbox is created; to inject on each run, add them on the Sandbox secrets page.',
  addRow: 'Add variable',
  removeRow: 'Remove',
  save: 'Save',
  saveAndContinue: 'Save and continue',
  saved: 'Saved. Applies to new sessions.',
  saveFailed: 'Failed to save',
  testConnection: 'Test connection',
  deepCheck: 'Full verification',
  recheck: 'Verify again',
  deepCheckIntro: 'Full verification creates a throwaway sandbox, runs one script, checks outbound access, and destroys it.',
  deepCheckConfirm: 'Full verification runs a throwaway script. Remote backends also create and destroy a real sandbox and may consume a small amount of sandbox time. Continue?',
  checkPassed: 'All checks passed',
  checkFailed: 'Some checks failed',
  checkScopeConnection: 'Only the control plane was verified: the endpoint responds and the credential is valid. Whether a script actually runs is still unverified.',
  checkScopeFull: 'Endpoint, credential, template, in-sandbox execution and outbound network were all verified for real.',
  checkScopePolicyRestricted: 'Outbound access is restricted by policy, so egress was not probed. Endpoint, credential, template and in-sandbox execution were verified.',
  checkPendingHint: '{names} can only be confirmed by a full verification, which creates a throwaway sandbox, runs one script and destroys it.',
  manageSkills: 'Manage skills',
  cardSkillsNone: 'No skills installed',
  cardSkillsMore: '+{count}',
  skillInstallerModel: 'Installer model',
  skillInstallerModelHint: 'Used by the built-in installer agent. Choose a chat model that supports tool calls.',
  skillInstallerModelRequired: 'Select an installer model first',
  skillInstallerModelSaveFailed: 'Failed to save the installer model',
  skillRollout: 'How the new skill image takes effect',
  skillRolloutHint: 'Installing or removing a skill builds a new image. This controls whether already-open sessions switch to it.',
  skillRolloutNextTurn: 'Rebuild open sessions on the next chat turn',
  skillRolloutNewSession: 'Keep open sessions on their current sandbox; only new sessions use the new image',
  skillRolloutSaveFailed: 'Failed to save the rollout setting',
  skillInstallGroup: 'Install a new skill',
  skillInstalledGroup: 'Installed',
  skillUploadClick: 'Click to upload a zip bundle',
  skillUploadDrag: 'or drop a file here',
  skillUploadHint: 'Install writes the skill onto the current image and takes a new snapshot. This can take several minutes. The current chat turn is not interrupted; open sessions rebuild their sandbox on the next turn, which clears the session workspace scratch.',
  skillUploadHintNewSession: 'Install writes the skill onto the current image and takes a new snapshot. This can take several minutes. Already-open sessions keep their current sandbox until they end; only newly started sessions pick up this install.',
  skillSourceSection: 'Install from a source',
  skillSourceSectionHint: 'Paste a ClawHub, GitHub or SkillHub link, or {\'@\'}owner/slug. The bundle cannot exceed {size} MB.',
  skillUploadSection: 'Upload a local bundle',
  skillUploadSectionHint: 'Drop a zip that contains SKILL.md below, or click to pick a file. The bundle cannot exceed {size} MB.',
  skillSourcePlaceholder: 'ClawHub: {\'@\'}owner/slug. GitHub/SkillHub: paste the full URL',
  skillSourceInstall: 'Install',
  skillInstallOr: 'or',
  skillSourceFailed: 'Failed to install the skill from the registry',
  skillUploadFailed: 'Failed to upload the skill',
  skillBundleTooLarge: 'The skill bundle cannot exceed {size} MB.',
  skillBundleTooManyFiles: 'The skill directory cannot hold more than {count} files.',
  skillBundleTooManyZipEntries: 'The archive cannot have more than {count} zip entries.',
  skillUploading: 'Uploading {percent}%',
  skillUploadAccepted: 'Skill install started',
  skillStatusInstalling: 'Installing',
  skillStatusReady: 'Ready',
  skillStatusFailed: 'Failed',
  skillStatusRemoving: 'Removing',
  skillStatusLabel: 'Status',
  skillDisableHint: 'Disable = the skill is invisible to the agent, files stay in the image. Changes take effect on the session\'s next execution.',
  skillDeleteHint: 'Delete removes the skill directory from the image and takes a new snapshot. The current chat turn is not interrupted; open sessions rebuild their sandbox on the next turn, which clears the session workspace scratch.',
  skillDeleteHintNewSession: 'Delete removes the skill directory from the image and takes a new snapshot. Already-open sessions keep their current sandbox until they end; only newly started sessions lose this skill.',
  skillRemoveInProgress: 'Uninstalling',
  skillRemoveWaiting: 'Uninstall from the image has started. Waiting for progress…',
  skillRemoveDone: 'Uninstalled “{name}” from this sandbox. It remains in the catalog, so you can install it again later.',
  imageInfoTitle: 'Current image',
  imageInfoSnapshot: 'Snapshot ID',
  imageInfoGeneration: 'Version',
  imageInfoBuiltAt: 'Built at',
  imageInfoBaseTemplate: 'Base template',
  imageInfoRuntimeTemplate: 'Runtime template',
  imageInfoUsingBase: 'No skills are installed yet. Sessions boot from the selected runtime template.',
  imageInfoEmpty: 'Using the base template',
  imageInfoUnset: 'Not set',
  skillTranscript: 'View install run',
  skillTranscriptLive: 'Install log',
  skillTranscriptLiveHint: 'Installing — click to watch',
  skillTranscriptTitle: 'Install run',
  skillTranscriptHide: 'Hide install run',
  skillTranscriptEmpty: 'This install left no transcript.',
  skillTranscriptWaiting: 'Install has started. Waiting for the process log…',
  installCommandRunning: 'Command running',
  installCommandWaiting: 'Waiting for command output. Elapsed time continues to update.',
  skillFiles: 'View files',
  skillFilesTitle: 'Files',
  skillFilesEmpty: 'This skill has no files to browse yet.',
  skillFilesLoadFailed: 'Failed to load skill files',
  skillFilesFileLoadFailed: 'Could not read this file',
  skillFilesBinary: 'This file is binary and cannot be previewed.',
  skillFilesTruncated: 'The file is large; only the beginning is shown.',
  skillFilesSelectHint: 'Select a file on the left to view it',
  skillFilesPreview: 'Preview',
  skillFilesSource: 'Source',
  skillLoadFailed: 'Failed to load skills',
  skillToggleFailed: 'Failed to update the skill',
  skillDeleteAccepted: 'Skill removal started',
  skillRetry: 'Reinstall',
  skillRetryHint: 'Retry with the stored bundle; no re-upload needed',
  skillRetryAccepted: 'Reinstall started',
  skillRetryFailed: 'Failed to start the reinstall',
  skillStop: 'Stop install',
  skillStopHint: 'Abort this install, then retry or uninstall',
  skillStopAccepted: 'Stopped',
  skillStopFailed: 'Failed to stop',
  skillEmpty: 'No skills installed yet. Paste a registry URL, or upload a zip.',
  skillVersion: 'Version',
  skillVersionEmpty: 'Not specified',
  skillError: 'Error',
  skillEnabled: 'Skill enabled',
  skillDisabled: 'Skill disabled',
}

type SandboxStepKey = 'connection' | 'template' | 'runtime'

const props = defineProps<{
  visible: boolean
  record: SandboxConfigRecord | null
  presetType?: string
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'saved'): void
}>()

const deploymentCapabilities = useDeploymentCapabilitiesStore()
const dockerBackendEnabled = computed(() =>
  deploymentCapabilities.isSupported('settings.sandbox.docker'),
)

// The backend echoes stored secrets as this placeholder. It never leaves the
// form as visible text: inputs stay empty and say "configured", and the
// placeholder is only re-attached on submit so the stored value survives.
const secretPlaceholder = '***'
const isMaskedSecret = (value?: string) => value === secretPlaceholder

// Mirrors DefaultDockerImage on the server, including why it tracks main
// instead of latest: the latest tag still carries an image whose /workspace
// the sandbox account cannot write.
const defaultDockerImage = 'ORG_PLACEHOLDER/enterpriserag-sandbox:main'

const clusterGuideUrl = 'https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/README.md'
const e2bApiKeysUrl = 'https://e2b.dev/dashboard?tab=keys'

const backendOptions = computed(() => {
  const types = [...NAMED_SANDBOX_BACKEND_TYPES]
  if (dockerBackendEnabled.value || backend.value === 'docker') return types
  return types.filter((type) => type !== 'docker')
})

const saving = ref(false)
const checking = ref(false)
const checkResult = ref<SandboxCheckResult | null>(null)
// Distinguishes "run the deep probes" from "run them again" on the panel button.
const lastCheckWasDeep = ref(false)
const conflict = ref<SandboxConflict | null>(null)
const conflictAlertRef = ref<HTMLElement | null>(null)
const checkResultRef = ref<HTMLElement | null>(null)
const nameError = ref('')

const name = ref('')
const description = ref('')
const backend = ref('')
// undefined rather than 0 so the input renders empty and shows its placeholder,
// matching the HTTP timeout / TTL fields. A literal 0 would read as a real value.
const defaultTimeoutSec = ref<number | undefined>(undefined)
const terminalIdleDisconnectSec = ref<number | undefined>(undefined)
const allowPrivateEndpoints = ref(false)
const cube = reactive<SandboxCubeConfig>({})
const e2b = reactive<SandboxE2BConfig>({})
const docker = reactive<SandboxDockerConfig>({})
// Tracks which secrets the tenant already has stored, so an empty input can
// mean "keep the saved key" instead of "no key configured".
const storedSecrets = reactive({ cube: false, e2b: false })
const envRows = ref<{ key: string; value: string; stored?: boolean }[]>([])
const skillRollout = ref<'next_turn' | 'new_session'>('next_turn')
// Both defaults are the zero value on the server too: egress allowed, inbound
// requiring the per-sandbox credential. Inbound is not editable in this form.
const denyEgressByDefault = ref(false)
const allowOutRows = ref<string[]>([])
const denyOutRows = ref<string[]>([])

type CubeRuleForm = {
  key: string
  name: string
  scheme?: string
  sni?: string
  host?: string
  methodsText: string
  path?: string
  deny: boolean
  audit?: string
  expanded: boolean
  inject: {
    header: string
    secret: string
    format: string
    stored?: boolean
    originalRuleName?: string
    originalHeader?: string
  }[]
}
type E2BRuleForm = {
  host: string
  expanded: boolean
  headers: {
    name: string
    value: string
    stored?: boolean
    originalHost?: string
    originalName?: string
  }[]
}
const cubeRules = ref<CubeRuleForm[]>([])
const e2bHostRules = ref<E2BRuleForm[]>([])
let cubeRuleKeySeq = 0

function newCubeRuleKey(): string {
  cubeRuleKeySeq += 1
  return `cube-rule-${cubeRuleKeySeq}`
}

function isStoredNetworkSecretRecoverable(
  row: { stored?: boolean },
  originalParentIdentity: string | undefined,
  originalChildIdentity: string | undefined,
  currentParentIdentity: string,
  currentChildIdentity: string,
): boolean {
  return row.stored === true
    && originalParentIdentity === currentParentIdentity.trim()
    && originalChildIdentity === currentChildIdentity.trim()
}

function addCubeRule() {
  for (const rule of cubeRules.value) rule.expanded = false
  cubeRules.value.push({
    key: newCubeRuleKey(),
    name: '', scheme: 'https', sni: '', host: '',
    methodsText: '', path: '', deny: false, audit: '', expanded: true, inject: [],
  })
}

function moveCubeRule(index: number, delta: number) {
  const next = index + delta
  if (next < 0 || next >= cubeRules.value.length) return
  const [row] = cubeRules.value.splice(index, 1)
  cubeRules.value.splice(next, 0, row)
}

function addE2BHostRule() {
  for (const rule of e2bHostRules.value) rule.expanded = false
  e2bHostRules.value.push({ host: '', expanded: true, headers: [] })
}

// Both providers refuse a domain allow-list without a deny-all fallback, and
// for a good reason: destinations never resolved through the sandbox's DNS
// stay reachable, so the list would be decorative. Warn here rather than
// letting the save round-trip fail.
const domainAllowNeedsDenyAll = computed(() => {
  if (denyEgressByDefault.value) return false
  if (denyOutRows.value.some((row) => denyOutRowCoversAllIPv4(row))) return false
  return allowOutRows.value.some((row) => {
    const value = row.trim()
    if (!value) return false
    return !/^[0-9./]+$/.test(value)
  })
})

// Mirrors types.DenyOutCoversAllIPv4: net.ParseCIDR collapses any IPv4 /0
// onto 0.0.0.0/0, so 1.2.3.4/0 is a real deny-all, not a false warning.
function denyOutRowCoversAllIPv4(row: string): boolean {
  const value = row.trim()
  if (value === '0.0.0.0/0') return true
  return /^\d{1,3}(?:\.\d{1,3}){3}\/0$/.test(value)
}
const inFlightFromSkills = ref(false)
const templates = ref<SandboxTemplate[]>([])
const templatesLoading = ref(false)
const templatesLoaded = ref(false)
const templatesError = ref('')
const wizardStep = ref(0)
let templatePollTimer: ReturnType<typeof setTimeout> | undefined

// Remote backends additionally expose a template catalog and control-plane
// settings. Cube, E2B and Docker still share the same save/check API.
const isRemoteBackend = computed(() => backend.value === 'cube' || backend.value === 'e2b')
const hasImageCatalog = computed(() => isRemoteBackend.value || backend.value === 'docker')
const currentTemplateId = computed(() => (
  backend.value === 'cube' ? cube.template_id : backend.value === 'e2b' ? e2b.template_id : ''
)?.trim() || '')
const selectedTemplate = computed(() => templates.value.find((item) => item.id === currentTemplateId.value))
const clusterStandardTemplate = computed(() => templates.value.find((item) => item.standard && item.id))
const clusterDesktopTemplate = computed(() => templates.value.find((item) => item.desktop && item.id))
const canCreateStandard = computed(() => (
  isRemoteBackend.value && templatesLoaded.value && !clusterStandardTemplate.value && !retargetFrozen.value
))
const canCreateDesktop = computed(() => (
  isRemoteBackend.value && templatesLoaded.value && !clusterDesktopTemplate.value && !retargetFrozen.value
))
const wizardSteps = computed<Array<{ key: SandboxStepKey; title: string }>>(() => {
  const steps: Array<{ key: SandboxStepKey; title: string }> = [
    { key: 'connection', title: 'Connect' },
  ]
  if (isRemoteBackend.value) {
    steps.push({ key: 'template', title: 'Template' })
  }
  steps.push({ key: 'runtime', title: 'Runtime' })
  return steps
})
const currentStepKey = computed<SandboxStepKey>(() => wizardSteps.value[wizardStep.value]?.key || 'connection')
const stepDescription = computed(() => (SETTINGS_SANDBOX_STEP_DESCRIPTIONS_LABELS[currentStepKey.value] ?? ''))
const primaryText = computed(() => {
  if (currentStepKey.value === 'connection') return 'Connect and continue'
  if (currentStepKey.value === 'template') return 'Next'
  return 'Save'
})
// Deep check needs the fields that actually get probed. Docker collects
// the image on the connection step; Cube/E2B still have an empty template_id
// there, so the action waits until the template step.
const canDeepCheck = computed(() => {
  if (backend.value === 'docker' && !dockerBackendEnabled.value) return false
  if (currentStepKey.value === 'template') return true
  return currentStepKey.value === 'connection' && !isRemoteBackend.value
})
const showCheckResult = computed(() => canDeepCheck.value)
const primaryDisabled = computed(() => (
  (backend.value === 'docker' && !dockerBackendEnabled.value)
  || (
    currentStepKey.value === 'template'
    && (!selectedTemplate.value || !isTemplateSelectable(selectedTemplate.value))
  )
))

// savedRecord is the config this drawer is editing, including one it just
// created: a second press of save must update that config rather than create
// another one if the admin walks back through the wizard before closing.
const savedRecord = ref<SandboxConfigRecord | null>(null)
const effectiveRecord = computed(() => savedRecord.value || props.record)
const hasSkillSnapshot = computed(() => (
  Boolean(effectiveRecord.value?.config?.skill_image?.snapshot_id?.trim())
))
const hasInFlightSkill = computed(() => inFlightFromSkills.value)
const retargetFrozen = computed(() => hasSkillSnapshot.value || hasInFlightSkill.value)

// Jumping is what separates editing from creating. A config that does not exist
// yet has to be built in order — its connection has to check out before there
// are templates to choose from. Once it exists, every step is just a page of
// its settings, so all of them open directly; steps already visited stay
// clickable during creation so the rail works as a way back.
function canJumpTo(index: number): boolean {
  if (index === wizardStep.value) return false
  return Boolean(effectiveRecord.value) || index < wizardStep.value
}

function goToStep(index: number) {
  if (!canJumpTo(index)) return
  wizardStep.value = index
  if (currentStepKey.value !== 'template') {
    stopTemplatePolling()
    return
  }
  // Landing on the template step without having passed through the connection
  // step still has to ask the cluster what it offers; a step already loaded
  // just resumes its poll, as walking back through it always has.
  if (templatesLoaded.value) scheduleTemplatePolling()
  else void loadTemplates()
}
const hasPendingTemplates = computed(() => templates.value.some(isTemplatePending))

const SANDBOX_CHECK_LABELS: Record<string, string> = {
  client_build: 'Client construction',
  api_url_reachable: 'Endpoint reachability',
  credential_valid: 'Credential validity',
  template_exists: 'Template existence',
  sandbox_exec: 'In-sandbox execution',
  egress_available: 'Outbound network',
}

const SANDBOX_SKIP_REASON_LABELS: Record<string, string> = {
  needs_deep_check: 'Needs full verification',
  control_plane_unreachable: 'Skipped: control plane unreachable',
  sandbox_not_created: 'Skipped: sandbox was not created',
  sandbox_exec_failed: 'Skipped: in-sandbox execution failed',
  egress_restricted_by_policy: 'Restricted by network policy (this config denies egress by default)',
}

const backendLabel = (value: string) => (SETTINGS_SANDBOX_BACKENDS_LABELS[value] ?? '')
const checkLabel = (probe: string) => SANDBOX_CHECK_LABELS[probe] ?? probe

// Probes deferred to the opt-in deep check are expected, not a problem, so they
// are pulled out of the result list and explained once.
const PENDING_SKIP_REASON = 'needs_deep_check'

const reportedChecks = computed(() => (checkResult.value?.checks || []).filter(
  (item) => item.ok !== null || item.reason !== PENDING_SKIP_REASON,
))

const pendingCheckNames = computed(() => (checkResult.value?.checks || [])
  .filter((item) => item.ok === null && item.reason === PENDING_SKIP_REASON)
  .map((item) => checkLabel(item.name)))

const EGRESS_RESTRICTED_REASON = 'egress_restricted_by_policy'
const egressRestrictedByPolicy = computed(() => (checkResult.value?.checks || []).some(
  (item) => item.reason === EGRESS_RESTRICTED_REASON,
))

// works" after a connection-only probe, or after egress was skipped because
// the config denies outbound access by policy.
const checkScopeHint = computed(() => {
  if (pendingCheckNames.value.length) return 'Only the control plane was verified: the endpoint responds and the credential is valid. Whether a script actually runs is still unverified.'
  if (egressRestrictedByPolicy.value) return 'Outbound access is restricted by policy, so egress was not probed. Endpoint, credential, template and in-sandbox execution were verified.'
  return 'Endpoint, credential, template, in-sandbox execution and outbound network were all verified for real.'
})

function checkDetail(item: SandboxCheckItem): string {
  if (item.message) return item.message
  if (!item.reason) return ''
  return SANDBOX_SKIP_REASON_LABELS[item.reason] ?? item.reason
}
const secretInputPlaceholder = (target: 'cube' | 'e2b') => (
  storedSecrets[target] ? 'Configured — leave empty to keep it' : 'Paste the API key'
)

// Mirrors sandbox.MissingRequiredFields on the server. Duplicated on purpose:
// the server stays the authority, this only spares the admin a round-trip and
// points at the offending input instead of showing one combined message.
const REQUIRED_FIELDS: Record<string, string[]> = {
  cube: ['api_url', 'proxy_url', 'sandbox_domain', 'template_id'],
  e2b: ['api_key', 'template_id'],
  docker: ['image'],
}

const fieldErrors = ref<Record<string, string>>({})

const fieldStatus = (field: string) => (fieldErrors.value[field] ? 'error' : undefined)
const fieldTip = (field: string) => fieldErrors.value[field] || undefined

const requiredLabel = (labelKey: string) => `${(SETTINGS_SANDBOX_LABELS[labelKey] ?? '')} *`

// Clearing on input rather than re-validating keeps the error from flickering
// back while the admin is still halfway through typing a URL.
function onFieldInput(field: string) {
  delete fieldErrors.value[field]
  invalidateCheck()
}

function onConnectionInput(field: string) {
  delete fieldErrors.value[field]
  invalidateConnection()
}

// Snapshot of the active backend block as it will be submitted, so a secret the
// admin left blank on purpose still counts as filled in.
function submittedBackendValues(): Record<string, unknown> {
  if (backend.value === 'cube') return withStoredSecret({ ...cube }, storedSecrets.cube)
  if (backend.value === 'e2b') return withStoredSecret({ ...e2b }, storedSecrets.e2b)
  return { ...docker }
}

function validateRequiredFields(includeTemplate = true): boolean {
  const required = (REQUIRED_FIELDS[backend.value] || []).filter((field) => includeTemplate || field !== 'template_id')
  const values = submittedBackendValues()
  const errors: Record<string, string> = {}
  for (const field of required) {
    const value = values[field]
    if (typeof value !== 'string' || value.trim() === '') {
      errors[field] = 'Required'
    }
  }
  fieldErrors.value = errors
  return Object.keys(errors).length === 0
}

const affectedSessionCount = computed(() => conflict.value?.inventory?.session_ids?.length || 0)

function defaultBackendType(): string {
  const fromRecord = props.record?.config?.sandbox_type || props.presetType || ''
  if (isNamedSandboxBackend(fromRecord)) return fromRecord
  return 'cube'
}

function reset() {
  stopTemplatePolling()
  const cfg: SandboxConfig = props.record?.config || {}
  name.value = props.record?.name || ''
  description.value = props.record?.description || ''
  backend.value = isNamedSandboxBackend(cfg.sandbox_type || '')
    ? cfg.sandbox_type!
    : defaultBackendType()
  defaultTimeoutSec.value = cfg.default_timeout_sec || undefined
  terminalIdleDisconnectSec.value = cfg.terminal_idle_disconnect_sec || undefined
  allowPrivateEndpoints.value = cfg.allow_private_endpoints === true
  // Replace rather than merge: a reused reactive object would otherwise carry
  // the previously edited config's fields into the next one opened.
  Object.keys(cube).forEach((key) => delete (cube as Record<string, unknown>)[key])
  Object.keys(e2b).forEach((key) => delete (e2b as Record<string, unknown>)[key])
  Object.keys(docker).forEach((key) => delete (docker as Record<string, unknown>)[key])
  Object.assign(cube, cfg.cube || {})
  Object.assign(e2b, cfg.e2b || {})
  Object.assign(docker, cfg.docker || {})
  if (!Array.isArray(cube.dns_servers)) cube.dns_servers = []
  if (backend.value === 'docker' && !docker.image) {
    docker.image = defaultDockerImage
  }
  storedSecrets.cube = isMaskedSecret(cube.api_key)
  storedSecrets.e2b = isMaskedSecret(e2b.api_key)
  if (storedSecrets.cube) cube.api_key = ''
  if (storedSecrets.e2b) e2b.api_key = ''
  envRows.value = Object.entries(cfg.env_vars || {}).map(([key, value]) => (
    isMaskedSecret(value) ? { key, value: '', stored: true } : { key, value }
  ))
  skillRollout.value = cfg.skill_rollout === 'new_session' ? 'new_session' : 'next_turn'
  const net = cfg.network || {}
  denyEgressByDefault.value = net.deny_egress_by_default === true
  allowOutRows.value = [...(net.allow_out || [])]
  denyOutRows.value = [...(net.deny_out || [])]
  cubeRules.value = (net.cube_rules || []).map((rule) => ({
    key: newCubeRuleKey(),
    name: rule.name || '',
    scheme: rule.scheme || '',
    sni: rule.sni || '',
    host: rule.host || '',
    methodsText: (rule.methods || []).join(', '),
    path: rule.path || '',
    deny: rule.deny === true,
    audit: rule.audit || '',
    expanded: false,
    inject: (rule.inject || []).map((inject) => ({
      header: inject.header || '',
      // A stored secret arrives masked; keep the input empty and say
      // "configured", exactly like the env var rows do.
      secret: isMaskedSecret(inject.secret) ? '' : (inject.secret || ''),
      format: inject.format || '',
      stored: isMaskedSecret(inject.secret),
      originalRuleName: rule.name?.trim() || '',
      originalHeader: inject.header?.trim() || '',
    })),
  }))
  e2bHostRules.value = (net.e2b_host_rules || []).map((rule) => ({
    host: rule.host || '',
    expanded: false,
    headers: Object.entries(rule.headers || {}).map(([name, value]) => ({
      name,
      value: isMaskedSecret(value) ? '' : value,
      stored: isMaskedSecret(value),
      originalHost: rule.host?.trim() || '',
      originalName: name.trim(),
    })),
  }))
  checkResult.value = null
  conflict.value = null
  nameError.value = ''
  fieldErrors.value = {}
  templates.value = []
  templatesLoaded.value = false
  templatesError.value = ''
  savedRecord.value = null
  wizardStep.value = 0
}

function selectBackend(value: string) {
  if (backend.value === value) return
  backend.value = value
  if (value === 'docker' && !docker.image) {
    docker.image = defaultDockerImage
  }
  onBackendChange()
}

async function refreshInFlightSkill() {
  const id = props.record?.id
  if (!id) {
    inFlightFromSkills.value = false
    return
  }
  try {
    const res = await listConfigSkills(id)
    inFlightFromSkills.value = (res?.data || []).some(
      (skill) => skill.status === 'installing' || skill.status === 'removing',
    )
  } catch {
    inFlightFromSkills.value = false
  }
}

watch(() => props.visible, (open) => {
  if (open) {
    void deploymentCapabilities.ensureLoaded()
    reset()
    void refreshInFlightSkill()
  } else {
    stopTemplatePolling()
    inFlightFromSkills.value = false
  }
})

function connectionReady(): boolean {
  if (!isRemoteBackend.value) return true
  const required = backend.value === 'cube'
    ? ['api_url', 'proxy_url', 'sandbox_domain']
    : ['api_key']
  const values = submittedBackendValues()
  const errors: Record<string, string> = {}
  for (const field of required) {
    if (typeof values[field] !== 'string' || String(values[field]).trim() === '') {
      errors[field] = 'Required'
    }
  }
  fieldErrors.value = { ...fieldErrors.value, ...errors }
  return Object.keys(errors).length === 0
}

function selectTemplate(value: string | number) {
  if (backend.value === 'cube') cube.template_id = String(value)
  else e2b.template_id = String(value)
  onFieldInput('template_id')
}

function clearTemplateSelection() {
  if (backend.value === 'cube') cube.template_id = ''
  if (backend.value === 'e2b') e2b.template_id = ''
  delete fieldErrors.value.template_id
}

function onTemplateCardClick(item: SandboxTemplate) {
  if (retargetFrozen.value && item.id !== currentTemplateId.value) return
  if (!isTemplateSelectable(item)) return
  selectTemplate(item.id)
}

function canRebuildTemplate(item: SandboxTemplate): boolean {
  return Boolean((item.standard || item.desktop) && item.id) && !isTemplatePending(item) && !retargetFrozen.value
}

function templateDisplayName(item: SandboxTemplate): string {
  if (item.standard) return 'sustainability.ai standard template'
  const name = item.name?.trim() || ''
  const id = item.id?.trim() || ''
  if (!name || name === id) return 'Unnamed template'
  return name
}

function formatTemplateTime(value?: string): string {
  const raw = value?.trim()
  if (!raw) return ''
  const ms = Date.parse(raw)
  if (Number.isNaN(ms)) return raw
  return new Date(ms).toLocaleString()
}

function templateFieldRows(item: SandboxTemplate): Array<{ key: string; label: string; value: string; mono?: boolean }> {
  const rows: Array<{ key: string; label: string; value: string; mono?: boolean }> = []
  const image = item.image?.trim()
  if (image) {
    rows.push({ key: 'image', label: 'Image', value: image, mono: true })
  }
  const version = item.version?.trim()
  if (version) {
    rows.push({ key: 'version', label: 'Version', value: version })
  }
  const id = item.id?.trim()
  if (id) {
    rows.push({ key: 'id', label: 'ID', value: id, mono: true })
  }
  const created = formatTemplateTime(item.created_at)
  if (created) {
    rows.push({ key: 'created', label: 'Created', value: created })
  }
  const instanceType = item.instance_type?.trim()
  if (instanceType) {
    rows.push({ key: 'instance', label: 'Instance type', value: instanceType })
  }
  const networkType = item.network_type?.trim()
  if (networkType) {
    rows.push({ key: 'network', label: 'Network', value: networkType })
  }
  if (item.allow_internet_access === true) {
    rows.push({
      key: 'internet',
      label: 'Internet access',
      value: 'On',
    })
  } else if (item.allow_internet_access === false) {
    rows.push({
      key: 'internet',
      label: 'Internet access',
      value: 'Off',
    })
  }
  return rows
}

function isTemplateSelectable(item: SandboxTemplate): boolean {
  const status = item.status?.trim().toLowerCase()
  if (!status) return true
  return ['ready', 'available', 'complete', 'completed', 'success', 'succeeded'].includes(status)
}

function isTemplatePending(item: SandboxTemplate): boolean {
  const status = item.status?.trim().toLowerCase()
  return ['building', 'waiting', 'pending', 'queued', 'processing', 'running'].includes(status || '')
}

// The backend reports this when a template's builds finished without the tag
// sandbox creation resolves, so it can never be spawned as it stands.
function isTemplateUntagged(item: SandboxTemplate): boolean {
  return item.status?.trim().toLowerCase() === 'untagged'
}

function isTemplateFailed(item: SandboxTemplate): boolean {
  const status = item.status?.trim().toLowerCase()
  return ['failed', 'failure', 'error', 'cancelled', 'canceled'].includes(status || '')
}

// A red "failed" badge on its own leaves no way to tell a registry credential
// problem from a node that ran out of disk, so the provider's own message is
// shown verbatim when it sends one.
function templateFailureReason(item: SandboxTemplate): string {
  if (!isTemplateFailed(item)) return ''
  const reason = item.error?.trim()
  return reason ? `Build failed: ${reason}` : ''
}

function templateStatusTheme(item: SandboxTemplate): 'success' | 'warning' | 'danger' | 'default' {
  if (isTemplateSelectable(item)) return 'success'
  if (isTemplateUntagged(item)) return 'danger'
  if (isTemplatePending(item)) return 'warning'
  if (isTemplateFailed(item)) return 'danger'
  return 'default'
}

function templateStatusLabel(item: SandboxTemplate): string {
  if (isTemplateSelectable(item)) return 'Ready'
  if (isTemplateUntagged(item)) return 'No default tag'
  if (isTemplatePending(item)) return 'Building'
  if (isTemplateFailed(item)) return 'Failed'
  return 'Unknown'
}

function stopTemplatePolling() {
  if (templatePollTimer) clearTimeout(templatePollTimer)
  templatePollTimer = undefined
}

function scheduleTemplatePolling() {
  stopTemplatePolling()
  if (!props.visible || currentStepKey.value !== 'template' || !hasPendingTemplates.value) return
  templatePollTimer = setTimeout(() => {
    void loadTemplates({ silent: true })
  }, 3000)
}

async function loadTemplates(opts: {
  ensureStandard?: boolean
  ensureDesktop?: boolean
  silent?: boolean
  replaceStandard?: boolean
  replaceDesktop?: boolean
} = {}): Promise<boolean> {
  if (!hasImageCatalog.value) return true
  if (!connectionReady()) return false
  if (!opts.silent) templatesLoading.value = true
  templatesError.value = ''
  try {
    // Cube/E2B: listing ensures the published CLI image when it is missing.
    // Desktop is opt-in: XFCE images are much heavier, so they are created
    // only when the admin clicks the desktop offer row.
    const ensureFirstParty = isRemoteBackend.value
    const res = await querySandboxTemplates({
      config: collectPayload(),
      config_id: effectiveRecord.value?.id,
      ensure_standard: opts.ensureStandard || (ensureFirstParty && !opts.replaceStandard && !opts.replaceDesktop && !opts.ensureDesktop),
      ensure_desktop: Boolean(opts.ensureDesktop),
      replace_standard: opts.replaceStandard,
      replace_desktop: opts.replaceDesktop,
    })
    templates.value = res.data?.templates || []
    templatesLoaded.value = true
    const standardID = res.data?.standard_template_id
    const desktopID = res.data?.desktop_template_id
    const current = templates.value.find((item) => item.id === currentTemplateId.value)
    if (opts.replaceDesktop && desktopID) {
      selectTemplate(desktopID)
    } else if (opts.replaceStandard && standardID) {
      selectTemplate(standardID)
    } else if (opts.ensureDesktop && desktopID) {
      const next = templates.value.find((item) => item.id === desktopID)
      if (next && (isTemplateSelectable(next) || isTemplatePending(next))) {
        selectTemplate(desktopID)
      }
    } else if (opts.ensureStandard && standardID) {
      const next = templates.value.find((item) => item.id === standardID)
      if (next && (isTemplateSelectable(next) || isTemplatePending(next))) {
        selectTemplate(standardID)
      }
    } else if (
      currentTemplateId.value
      && (!current || (!isTemplateSelectable(current) && !isTemplatePending(current)))
      && !retargetFrozen.value
    ) {
      clearTemplateSelection()
    }
    if (!currentTemplateId.value && !retargetFrozen.value) {
      const firstPartyReady = templates.value.filter((item) => (
        (item.standard || item.desktop) && isTemplateSelectable(item)
      ))
      if (firstPartyReady.length === 1) selectTemplate(firstPartyReady[0].id)
    }
    if (res.data?.provisioned && !opts.silent && (
      opts.replaceDesktop || opts.replaceStandard || opts.ensureStandard || opts.ensureDesktop
    )) {
      MessagePlugin.info(provisionedTemplateMessage(opts))
    }
    scheduleTemplatePolling()
    return true
  } catch (e: any) {
    templatesError.value = e?.message || 'Failed to load templates'
    return false
  } finally {
    if (!opts.silent) templatesLoading.value = false
  }
}

function provisionedTemplateMessage(opts: {
  ensureStandard?: boolean
  ensureDesktop?: boolean
  replaceStandard?: boolean
  replaceDesktop?: boolean
}): string {
  if (opts.replaceDesktop) return 'The previous desktop template was deleted and a rebuild has started. Wait until it is ready. The CLI template is left untouched.'
  if (opts.ensureDesktop) return 'The sustainability.ai desktop template is being created. Refresh shortly to see its status.'
  if (opts.replaceStandard) return 'The previous standard template was deleted and a rebuild has started. Wait until it is ready.'
  return 'The sustainability.ai standard template is being created. Refresh shortly to see its status.'
}

function createStandardTemplate() {
  return loadTemplates({ ensureStandard: true })
}

function createDesktopTemplate() {
  return loadTemplates({ ensureDesktop: true })
}

function replaceFirstPartyTemplate(item: SandboxTemplate) {
  if (retargetFrozen.value) return
  if (item.desktop) return loadTemplates({ replaceDesktop: true })
  return loadTemplates({ replaceStandard: true })
}

// Re-attaches the redaction placeholder to a secret the admin left untouched:
// the server reads it as "preserve the stored value".
function withStoredSecret<T extends { api_key?: string }>(block: T, stored: boolean): T {
  if (stored && !block.api_key?.trim()) block.api_key = secretPlaceholder
  return block
}

function collectedDesktopEnabled(): boolean | undefined {
  if (selectedTemplate.value) {
    // Persist false when the CLI card is selected; omitempty on the
    // server would otherwise keep a stale true from a previous desktop save.
    return Boolean(selectedTemplate.value.desktop)
  }
  // Skill snapshots replace template_id with a UUID that is not in the
  // catalog. Keep the stored bit. If the catalog loaded and this ID is
  // simply unmatched, do not keep a stale true that could disagree with
  // template_id.
  if (templatesLoaded.value && !retargetFrozen.value) {
    return false
  }
  return effectiveRecord.value?.config?.desktop_enabled || undefined
}

function collectPayload(): SandboxConfig {
  const envVars: Record<string, string> = {}
  for (const row of envRows.value) {
    const key = row.key.trim()
    if (!key) continue
    envVars[key] = row.stored && row.value === '' ? secretPlaceholder : row.value
  }
  const payload: SandboxConfig = {
    sandbox_type: backend.value,
    default_timeout_sec: defaultTimeoutSec.value || undefined,
    terminal_idle_disconnect_sec: terminalIdleDisconnectSec.value || undefined,
    allow_private_endpoints: allowPrivateEndpoints.value || undefined,
    desktop_enabled: collectedDesktopEnabled(),
    env_vars: envVars,
    skill_rollout: skillRollout.value,
    network: collectNetworkPolicy(),
  }
  // Send only the selected backend's block so an unused one cannot fail
  // validation (e.g. a stale private URL left in the other tab).
  if (backend.value === 'cube') payload.cube = withStoredSecret({ ...cube }, storedSecrets.cube)
  if (backend.value === 'e2b') payload.e2b = withStoredSecret({ ...e2b }, storedSecrets.e2b)
  if (backend.value === 'docker') payload.docker = { ...docker }
  return payload
}

// Sends the policy as its own block. Empty lists are dropped so a config the
// admin never touched serializes to the same thing as a fresh default.
function collectNetworkPolicy(): SandboxNetworkPolicy {
  const policy: SandboxNetworkPolicy = {}
  // Docker can only honour network_mode on the docker block.
  if (backend.value === 'docker') {
    return policy
  }
  if (denyEgressByDefault.value) policy.deny_egress_by_default = true
  // Inbound stays the zero value: require the per-sandbox credential.

  if (backend.value !== 'docker') {
    const allowOut = allowOutRows.value.map((row) => row.trim()).filter(Boolean)
    const denyOut = denyOutRows.value.map((row) => row.trim()).filter(Boolean)
    if (allowOut.length) policy.allow_out = allowOut
    if (denyOut.length) policy.deny_out = denyOut
  }

  if (backend.value === 'cube' && cubeRules.value.length) {
    policy.cube_rules = cubeRules.value.map((rule) => ({
      name: rule.name.trim(),
      scheme: rule.scheme || undefined,
      sni: rule.sni?.trim() || undefined,
      host: rule.host?.trim() || undefined,
      methods: rule.methodsText
        .split(',')
        .map((method) => method.trim().toUpperCase())
        .filter(Boolean),
      path: rule.path?.trim() || undefined,
      deny: rule.deny || undefined,
      audit: rule.audit || undefined,
      inject: rule.deny
        ? undefined
        : rule.inject
          .filter((inject) => inject.header.trim())
          .map((inject) => ({
            header: inject.header.trim(),
            // Re-attach the placeholder only while the server-side lookup key
            // remains the identity under which this credential was loaded.
            secret: isStoredNetworkSecretRecoverable(
              inject,
              inject.originalRuleName,
              inject.originalHeader,
              rule.name,
              inject.header,
            )
              && inject.secret === ''
              ? secretPlaceholder
              : inject.secret,
            format: inject.format?.trim() || undefined,
          })),
    }))
  }
  if (backend.value === 'e2b' && e2bHostRules.value.length) {
    policy.e2b_host_rules = e2bHostRules.value.map((rule) => {
      const headers: Record<string, string> = {}
      for (const header of rule.headers) {
        const name = header.name.trim()
        if (!name) continue
        headers[name] = isStoredNetworkSecretRecoverable(
          header,
          header.originalHost,
          header.originalName,
          rule.host,
          header.name,
        )
          && header.value === ''
          ? secretPlaceholder
          : header.value
      }
      return { host: rule.host.trim(), headers }
    })
  }
  return policy
}

function close() {
  stopTemplatePolling()
  emit('update:visible', false)
}

function validateName(): boolean {
  if (!name.value.trim()) {
    nameError.value = 'Please enter a config name'
    return false
  }
  nameError.value = ''
  return true
}

async function handlePrimaryAction() {
  if (backend.value === 'docker' && !dockerBackendEnabled.value) return
  if (currentStepKey.value === 'connection') {
    if (!validateName() || !validateRequiredFields(false)) return
    if (!(await runCheck(false))) return
    if (isRemoteBackend.value) {
      invalidateCheck()
      wizardStep.value += 1
      await loadTemplates()
      return
    }
    // Docker's template is the image typed on this step. Kick a background
    // pull so the first session does not block on a cold registry fetch.
    if (backend.value === 'docker') {
      void loadTemplates({ ensureStandard: true })
    }
    invalidateCheck()
    wizardStep.value += 1
    return
  }
  if (currentStepKey.value === 'template') {
    if (!selectedTemplate.value || !isTemplateSelectable(selectedTemplate.value)) {
      fieldErrors.value.template_id = 'The selected template is not ready. Refresh and wait for the build to finish.'
      return
    }
    stopTemplatePolling()
    wizardStep.value += 1
    return
  }
  await save()
}

function previousStep() {
  if (wizardStep.value <= 0) return
  wizardStep.value -= 1
  if (currentStepKey.value === 'template') scheduleTemplatePolling()
  else stopTemplatePolling()
}

async function save() {
  const trimmed = name.value.trim()
  if (!validateName()) return
  if (!validateRequiredFields()) return
  if (isRemoteBackend.value && selectedTemplate.value && !isTemplateSelectable(selectedTemplate.value)) {
    fieldErrors.value.template_id = 'The selected template is not ready. Refresh and wait for the build to finish.'
    return
  }
  saving.value = true
  conflict.value = null
  try {
    const payload = { name: trimmed, description: description.value, config: collectPayload() }
    const existing = effectiveRecord.value
    const res = existing
      ? await updateSandboxConfigById(existing.id, payload)
      : await createSandboxConfig(payload)
    MessagePlugin.success('Saved successfully')
    // The list behind the drawer refreshes either way, so closing here is only
    // about whether the wizard has anything left to offer.
    emit('saved')
    const saved = res?.data
    if (saved) savedRecord.value = saved
    close()
  } catch (e: any) {
    const refusal = parseSandboxConflict(e)
    if (refusal) {
      // Keep the drawer open with the form intact: the admin has to act
      // elsewhere first, and retyping everything afterwards would be cruel.
      conflict.value = refusal
      await nextTick()
      conflictAlertRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      return
    }
    MessagePlugin.error(e?.message || 'Failed to save')
  } finally {
    saving.value = false
  }
}

async function runCheck(deep: boolean): Promise<boolean> {
  // The probe builds a real client, so an incomplete config would come back as
  // a generic client_build failure instead of naming the empty field.
  if (!validateRequiredFields(deep)) return false
  checking.value = true
  checkResult.value = null
  try {
    // config_id lets the backend resolve masked secrets against the stored row,
    // so an edited form can be probed without retyping the API key.
    const res = await checkSandboxConfig({
      config: collectPayload(),
      config_id: effectiveRecord.value?.id,
      deep,
    })
    checkResult.value = res?.data || null
    lastCheckWasDeep.value = deep
    if (checkResult.value) {
      await nextTick()
      checkResultRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
    return checkResult.value?.ok === true
  } catch (e: any) {
    MessagePlugin.error(e?.message || 'Some checks failed')
    return false
  } finally {
    checking.value = false
  }
}

// A result that no longer matches the form is worse than none.
function invalidateCheck() {
  checkResult.value = null
}

function invalidateConnection() {
  stopTemplatePolling()
  invalidateCheck()
  templates.value = []
  templatesLoaded.value = false
  templatesError.value = ''
  clearTemplateSelection()
}

// The two backends require different fields, so carrying errors across a switch
// would flag inputs the admin can no longer even see.
function onBackendChange() {
  fieldErrors.value = {}
  invalidateConnection()
}

onUnmounted(stopTemplatePolling)
</script>

<style lang="less" scoped>
/*
  A step rail rather than segmented buttons: the connector line carries the
  "these run in order" meaning that boxed segments only implied, and the flat
  background keeps the drawer's first visual weight on the form itself.
*/
.sandbox-steps {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
}

.sandbox-step {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: var(--td-text-color-placeholder);
  transition: color var(--app-motion-fast) ease;

  /* Only the steps that draw a connector need to absorb the leftover width. */
  &:not(:last-child) {
    flex: 1;
  }

  &.is-active {
    color: var(--td-brand-color);
  }

  &.is-done {
    color: var(--td-text-color-secondary);
  }

  /*
    Reachable steps render as <button>, so the browser's own control styling has
    to be undone to keep the rail looking identical either way. Only the cursor
    and hover state give the affordance away.
  */
  &.is-clickable {
    padding: 0;
    font: inherit;
    text-align: left;
    background: none;
    border: 0;
    cursor: pointer;

    &:hover:not(.is-active) {
      color: var(--td-brand-color);
    }

    &:focus-visible {
      outline: 2px solid var(--td-brand-color);
      outline-offset: 2px;
      border-radius: var(--app-radius-xs);
    }
  }
}

.sandbox-step__marker {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  border: 1px solid currentColor;
  border-radius: 50%;
  font-size: var(--app-text-xs);
  font-weight: 600;
  line-height: 1;

  .is-active & {
    background: var(--td-brand-color);
    border-color: var(--td-brand-color);
    color: #fff;
  }

  .is-done & {
    background: color-mix(in srgb, var(--td-brand-color) 12%, transparent);
    border-color: color-mix(in srgb, var(--td-brand-color) 35%, transparent);
    color: var(--td-brand-color);
  }
}

.sandbox-step__title {
  overflow: hidden;
  font-size: var(--app-text-md);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sandbox-step__line {
  flex: 1;
  min-width: 16px;
  height: 1px;
  margin: 0 4px;
  background: var(--td-component-stroke);

  .is-done & {
    background: color-mix(in srgb, var(--td-brand-color) 35%, transparent);
  }
}

.sandbox-editor-form {
  min-width: 0;
  max-width: 100%;
  overflow-x: hidden;

  :deep(.setting-drawer__section) {
    min-width: 0;
  }

  :deep(.t-form__item) {
    margin-bottom: 0;
  }

  /*
    Form items here stack a control with its own hint text and doc link, so the
    controls area has to be a block instead of TDesign's default single row.
  */
  :deep(.t-form__controls-content) {
    display: block;
  }

  :deep(.t-form__label) {
    padding-bottom: 6px;
    font-size: var(--app-text-md);
    font-weight: 500;
    line-height: 1.4;
  }

  :deep(.t-input),
  :deep(.t-input-number),
  :deep(.t-select) {
    width: 100%;
    font-size: var(--app-text-md);
  }
}

.identity-hint {
  margin: 0;
}

.compact-alert {
  padding: 9px 11px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 7px;
  background: var(--td-bg-color-secondarycontainer);

  :deep(.t-alert__icon) {
    font-size: var(--app-text-lg);
  }

  :deep(.t-alert__message) {
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);
    line-height: 1.5;
  }
}

.private-endpoint-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-sm);
  background: var(--td-bg-color-secondarycontainer);
}

.private-endpoint-row__title {
  margin: 0 0 3px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.form-grid {
  display: grid;
  gap: 12px;

  &--two {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.backend-choice {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 2px 0;
}

.backend-choice__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.backend-choice__name {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 500;
  line-height: 1.4;
}

.backend-choice__desc {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.45;
  white-space: normal;
}

.section-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;

  .setting-drawer__section-title {
    margin-bottom: 0;
  }
}

.enterpriserag-template-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);

  &.is-active {
    border-color: color-mix(in srgb, var(--td-brand-color) 45%, var(--td-component-stroke));
    background: color-mix(in srgb, var(--td-brand-color) 5%, var(--td-bg-color-container));
  }
}

.enterpriserag-template-card__content {
  flex: 1;
  min-width: 0;

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);
    line-height: 1.5;
  }
}

.enterpriserag-template-card__title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.enterpriserag-template-card__title {
  font-size: var(--app-text-md);
  font-weight: 600;
}

.template-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  min-height: 88px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
}

.template-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  overflow-x: hidden;
}

.template-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  padding: 10px 12px;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  text-align: left;
  cursor: pointer;
  transition: border-color var(--app-motion-fast) ease, box-shadow var(--app-motion-fast) ease;

  &:hover:not(.is-disabled):not(.template-row--offer) {
    border-color: var(--td-brand-color-3);
  }

  &.is-active {
    border-color: var(--td-brand-color);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }

  &.is-disabled {
    cursor: default;
  }
}

.template-row--offer {
  cursor: default;
  border-style: dashed;
}

.template-row__marker {
  flex-shrink: 0;
  width: 14px;
  height: 14px;
  margin-top: 3px;
  box-sizing: border-box;
  border: 1.5px solid var(--td-border-level-2-color);
  border-radius: 50%;
  background: var(--td-bg-color-container);

  .is-active & {
    border-color: var(--td-brand-color);
    box-shadow: inset 0 0 0 3.5px var(--td-brand-color);
  }

  .is-disabled & {
    opacity: 0.45;
  }
}

.template-row__main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  max-width: 100%;
}

.template-row__head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;

  :deep(.t-tag) {
    flex-shrink: 0;
  }
}

.template-row__title {
  flex: 0 1 auto;
  min-width: 0;
  overflow: hidden;
  font-size: var(--app-text-md);
  font-weight: 600;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.template-row__spacer {
  flex: 1 1 8px;
  min-width: 8px;
}

.template-row__rebuild {
  flex-shrink: 0;

  :deep(.t-button) {
    color: var(--td-text-color-secondary);
    padding-left: 4px;
    padding-right: 4px;

    &:hover {
      color: var(--td-text-color-primary);
    }
  }
}

.template-row__fields {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 4px 16px;
  margin: 0;
  min-width: 0;
}

.template-row__field {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: baseline;
  gap: 8px;
  min-width: 0;

  dt {
    color: var(--td-text-color-placeholder);
    font-size: var(--app-text-xs);
    line-height: 18px;
    white-space: nowrap;
  }

  dd {
    margin: 0;
    min-width: 0;
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);
    line-height: 18px;
    text-overflow: ellipsis;
    white-space: nowrap;

    &.is-mono {
      font-family: var(--td-font-family-mono);
      font-size: var(--app-text-xs);
    }
  }
}

.template-row__hint {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.5;

  &--error {
    color: var(--td-error-color);
  }
}

.field-hints {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  margin-top: 6px;

  .inline-guide-link {
    margin-top: 0;
  }
}

.inline-guide-link {
  display: inline-flex;
  align-items: center;
  align-self: flex-start;
  gap: 5px;
  margin-top: -4px;
  color: var(--td-brand-color);
  font-size: var(--app-text-sm);
  text-decoration: none;

  &:hover {
    color: var(--td-brand-color-hover);
  }
}

.runtime-fields :deep(.t-form__item) {
  margin-bottom: 0;
}

.runtime-fields :deep(.t-input-number) {
  width: 100%;
}

.net-list {
  margin-top: 16px;
}

.net-list__title {
  font-size: var(--app-text-md);
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.net-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 32px;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.net-row--double {
  grid-template-columns: minmax(140px, 0.7fr) minmax(160px, 1.3fr) 32px;
}

.net-row--triple {
  grid-template-columns: minmax(120px, 0.6fr) minmax(140px, 1fr) minmax(120px, 0.8fr) 32px;
}

.net-rule {
  border: 1px solid var(--td-component-border);
  border-radius: var(--app-radius-sm);
  padding: 12px;
  margin-bottom: 12px;
}

.net-rule--collapsible {
  padding: 0;
  margin-bottom: 2px;
  border: 0;
  border-radius: var(--app-radius-xs);
}

.net-rule--collapsible.is-open {
  margin-bottom: 6px;
  border: 1px solid var(--td-component-border);
}

.net-rule__bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  min-height: 26px;
  padding: 0 2px;
}

.net-rule__actions {
  display: flex;
  align-items: center;
}

.net-rule__toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  height: 26px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  border-radius: var(--app-radius-xs);
  text-align: left;
}

.net-rule__toggle:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.net-rule__name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--app-text-sm);
  line-height: 18px;
  color: var(--td-text-color-primary);
}

.net-rule__name.is-empty {
  color: var(--td-text-color-placeholder);
}

.net-rule__move,
.net-rule__remove {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  border-radius: var(--app-radius-xs);
}

.net-rule__move:hover:not(:disabled) {
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container-hover);
}

.net-rule__move:disabled {
  opacity: 0.35;
  cursor: default;
}

.net-rule__remove:hover {
  color: var(--td-error-color);
  background: var(--td-bg-color-container-hover);
}

.net-rule__body {
  padding: 8px 10px 10px;
  border-top: 1px solid var(--td-component-border);
}

.net-inject {
  margin-top: 8px;
}

.env-rows {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.env-row {
  display: grid;
  grid-template-columns: minmax(140px, 0.7fr) minmax(200px, 1.3fr) 32px;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.env-empty {
  padding: 18px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  text-align: center;
}

.skill-rollout-group {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}

.section-help {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.5;

  &--under-title {
    margin-top: 5px;
    max-width: 540px;
    white-space: pre-line;
  }

  /* Sits under an input inside the same form item. */
  &--field {
    margin-top: 6px;
  }
}

.check-result {
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.check-result__title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
  line-height: 1.45;

  &.is-success {
    color: var(--td-success-color);
  }

  &.is-error {
    color: var(--td-error-color);
  }
}

.check-result__subtitle,
.check-result__hint {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.55;
}

.check-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.check-item {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  padding: 2px 0;
  font-size: var(--app-text-sm);
}

.check-item .ok {
  color: var(--td-success-color);
}

.check-item .err {
  color: var(--td-error-color);
}

.check-item .skip {
  color: var(--td-text-color-placeholder);
}

.check-name {
  min-width: 140px;
}

.check-latency,
.check-message {
  color: var(--td-text-color-secondary);
}

.footer-check-ok {
  color: var(--td-success-color);
}

.footer-check-error {
  color: var(--td-error-color);
}

.blocked {
  margin-top: 16px;
}

.blocked-top {
  margin-top: 0;
  margin-bottom: 16px;
}

.blocked p {
  margin: 4px 0 0;
}

@media (max-width: 720px) {
  .form-grid--two,
  .enterpriserag-template-card {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .sandbox-step__title {
    font-size: var(--app-text-sm);
  }

  .template-row__fields {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>

<!--
  The select popup is attached to body, outside the scoped style boundary, so
  the two-line backend options need their row height relaxed globally. The
  overlay class keeps it from touching any other select in the app.
-->
<style lang="less">
.sandbox-backend-popup {
  .t-select-option {
    height: auto;
    min-height: 44px;
    padding: 6px 8px;
    line-height: 1.4;
  }
}
</style>
