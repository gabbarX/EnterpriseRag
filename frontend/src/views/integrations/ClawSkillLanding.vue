<template>
  <IntegrationLandingLayout
    :title="'sustainability.ai Skill'"
    :subtitle="'Import documents and run hybrid retrieval (vector + keyword) via the sustainability.ai REST API—for uploads, URL imports, Markdown entries, and cross-KB search.'"
    variant="claw"
  >
    <template #actions>
      <IntegrationExternalCta
        variant="claw"
        :label="'Open ClawHub'"
        :hint="'Install sustainability.ai Skill · opens in a new tab'"
        @click="openClawHub"
      >
        <template #icon>
          <span class="ext-cta-emoji" role="img" :aria-label="'Claw Skill'">🦞</span>
        </template>
      </IntegrationExternalCta>
    </template>

    <template #main>
      <div class="landing-group">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            {{ 'Skill capabilities' }}
            <span class="section-head-extra">{{ capabilityKeys.length }}</span>
          </h4>
          <div class="capability-grid capability-grid--claw">
            <div v-for="key in capabilityKeys" :key="key" class="capability-card">
              <div class="capability-card__icon">
                <t-icon :name="capabilityIcons[key]" />
              </div>
              <h5 class="capability-card__title">{{ (INTEGRATIONS_CLAW_CAPABILITIES_TITLE_LABELS[key] ?? '') }}</h5>
              <p class="capability-card__desc">{{ (INTEGRATIONS_CLAW_CAPABILITIES_DESC_LABELS[key] ?? '') }}</p>
            </div>
          </div>
        </section>
      </div>
    </template>

    <template #aside>
      <div class="landing-group">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Setup steps' }}</h4>
          <ol class="landing-steps">
            <li v-for="(step, index) in stepKeys" :key="step" class="landing-step">
              <span class="landing-step-num">{{ index + 1 }}</span>
              <div class="landing-step-body">
                <div class="landing-step-title">{{ (INTEGRATIONS_CLAW_STEPS_TITLE_LABELS[step] ?? '') }}</div>
                <p class="landing-step-desc">{{ (INTEGRATIONS_CLAW_STEPS_DESC_LABELS[step] ?? '') }}</p>
                <t-button
                  v-if="step === 'api'"
                  size="small"
                  variant="outline"
                  class="landing-step-action"
                  @click="openApiSettings"
                >
                  {{ 'Open API Info' }}
                </t-button>
                <div v-if="step === 'env'" class="landing-step-embed">
                  <div class="code-toolbar">
                    <pre class="code-toolbar__code">{{ envExample }}</pre>
                    <t-button
                      class="code-toolbar__copy"
                      size="small"
                      variant="text"
                      shape="square"
                      :title="'Copy'"
                      @click="copyEnvExample"
                    >
                      <t-icon name="file-copy" size="16px" />
                    </t-button>
                  </div>
                </div>
                <div v-if="step === 'install'" class="landing-step-embed">
                  <div class="code-toolbar">
                    <pre class="code-toolbar__code">{{ installCommand }}</pre>
                    <t-button
                      class="code-toolbar__copy"
                      size="small"
                      variant="text"
                      shape="square"
                      :title="'Copy'"
                      @click="copyInstallCommand"
                    >
                      <t-icon name="file-copy" size="16px" />
                    </t-button>
                  </div>
                </div>
              </div>
            </li>
          </ol>
        </section>
      </div>
    </template>

    <template #footer>
      <div class="landing-footer-bar" role="note">
        <p class="landing-footer-bar__note">{{ 'Skill hosted on ClawHub ({\'@\'}lyingbug/enterpriserag). See the ClawHub page for full API docs and version history.' }}</p>
        <span class="landing-meta">{{ 'ClawHub · {\'@\'}lyingbug/enterpriserag · MIT-0' }}</span>
      </div>
    </template>
  </IntegrationLandingLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { copyWithToast } from '@/utils/clipboard'
import { useRouter } from 'vue-router'
import { CLAWHUB_SKILL_URL } from '@/config/integrations'
import { useApiBaseUrlDisplay } from '@/composables/useApiBaseUrlDisplay'
import { useUIStore } from '@/stores/ui'
import IntegrationLandingLayout from './IntegrationLandingLayout.vue'
import IntegrationExternalCta from './IntegrationExternalCta.vue'

const INTEGRATIONS_CLAW_CAPABILITIES_TITLE_LABELS: Record<string, string> = {
  upload: 'Upload files',
  url: 'Import URLs',
  manual: 'Write Markdown',
  search: 'Hybrid search',
  browse: 'Browse knowledge',
}

const INTEGRATIONS_CLAW_CAPABILITIES_DESC_LABELS: Record<string, string> = {
  upload: 'Upload PDF, Word, Excel, and more; automatic parsing and vectorization.',
  url: 'Fetch web pages by URL into a knowledge base with parse-status polling.',
  manual: 'Create or edit knowledge entries as Markdown—ideal for meeting notes.',
  search: 'Per-KB hybrid-search and cross-KB knowledge-search combining vector and keyword recall.',
  browse: 'List knowledge bases and entries, view details, and manage imported content.',
}

const INTEGRATIONS_CLAW_STEPS_TITLE_LABELS: Record<string, string> = {
  api: 'Get API credentials',
  env: 'Configure environment',
  install: 'Install the skill',
  verify: 'Verify connection',
}

const INTEGRATIONS_CLAW_STEPS_DESC_LABELS: Record<string, string> = {
  api: 'Copy API Key and base URL from Settings → API Info.',
  env: 'Set ENTERPRISERAG_BASE_URL and ENTERPRISERAG_API_KEY in your shell or ~/.zshrc / ~/.bashrc. The example below uses your current API base URL—replace the API Key with your actual value.',
  install: 'Run the command below in an environment with the OpenClaw CLI installed, or follow the ClawHub page instructions.',
  verify: 'Ask the agent to list knowledge bases or run a search to confirm credentials and connectivity.',
}

const router = useRouter()
const uiStore = useUIStore()
const { apiBaseUrlDisplay } = useApiBaseUrlDisplay()

const capabilityKeys = ['upload', 'url', 'manual', 'search', 'browse'] as const
const stepKeys = ['api', 'env', 'install', 'verify'] as const

const capabilityIcons: Record<(typeof capabilityKeys)[number], string> = {
  upload: 'upload',
  url: 'link',
  manual: 'edit',
  search: 'search',
  browse: 'view-list',
}

const installCommand = 'openclaw skills install @lyingbug/enterpriserag'

const envExample = computed(() => {
  const base = apiBaseUrlDisplay.value || 'https://your-server.com/api/v1'
  return `export ENTERPRISERAG_BASE_URL="${base}"\nexport ENTERPRISERAG_API_KEY="sk-your-api-key"`
})

const openClawHub = () => {
  window.open(CLAWHUB_SKILL_URL, '_blank', 'noopener,noreferrer')
}

const openApiSettings = () => {
  router.push({ path: '/platform/settings', query: { section: 'integration-api' } })
  uiStore.openSettings('integration-api')
}

const copyEnvExample = () => copyWithToast(envExample.value, 'Environment example copied')
const copyInstallCommand = () => copyWithToast(installCommand, 'Install command copied')
</script>
