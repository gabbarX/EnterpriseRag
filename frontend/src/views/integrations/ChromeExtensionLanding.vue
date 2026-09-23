<template>
  <IntegrationLandingLayout
    :title="'Knowledge Assistant'"
    :subtitle="'For self-hosted sustainability.ai: ask questions in a sidebar, clip web pages, and save Markdown notes into your knowledge bases while you browse.'"
    variant="chrome"
  >
    <template #tags>
      <span v-for="key in scenarioKeys" :key="key" class="scenario-tag">
        {{ (INTEGRATIONS_CHROME_SCENARIOS_LABELS[key] ?? '') }}
      </span>
    </template>

    <template #actions>
      <IntegrationExternalCta
        variant="chrome"
        :label="'Chrome Web Store'"
        :hint="'Official extension · opens in a new tab'"
        @click="openChromeStore"
      >
        <template #icon>
          <t-icon name="extension" size="18px" />
        </template>
      </IntegrationExternalCta>
    </template>

    <template #main>
      <div class="landing-group">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">
            {{ 'Core capabilities' }}
            <span class="section-head-extra">{{ capabilityKeys.length }}</span>
          </h4>
          <div class="capability-grid">
            <div v-for="key in capabilityKeys" :key="key" class="capability-card">
              <div class="capability-card__icon">
                <t-icon :name="capabilityIcons[key]" />
              </div>
              <h5 class="capability-card__title">{{ (INTEGRATIONS_CHROME_CAPABILITIES_TITLE_LABELS[key] ?? '') }}</h5>
              <p class="capability-card__desc">{{ (INTEGRATIONS_CHROME_CAPABILITIES_DESC_LABELS[key] ?? '') }}</p>
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
                <div class="landing-step-title">{{ (INTEGRATIONS_CHROME_STEPS_TITLE_LABELS[step] ?? '') }}</div>
                <p class="landing-step-desc">{{ (INTEGRATIONS_CHROME_STEPS_DESC_LABELS[step] ?? '') }}</p>
                <t-button
                  v-if="step === 'api'"
                  size="small"
                  variant="outline"
                  class="landing-step-action"
                  @click="openApiSettings"
                >
                  {{ 'Open API Info' }}
                </t-button>
                <div v-if="step === 'connect'" class="landing-step-embed credential-row">
                  <div class="api-key-control landing-api-control">
                    <t-input :model-value="apiBaseUrlDisplay" readonly class="mono-text-input" />
                    <t-button
                      size="small"
                      variant="text"
                      :title="'Copy'"
                      @click="copyApiUrl"
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
      <span class="landing-meta">{{ 'Chrome Web Store · v1.0.0' }}</span>
    </template>
  </IntegrationLandingLayout>
</template>

<script setup lang="ts">
import { copyWithToast } from '@/utils/clipboard'
import { useRouter } from 'vue-router'
import { CHROME_EXTENSION_URL } from '@/config/integrations'
import { useApiBaseUrlDisplay } from '@/composables/useApiBaseUrlDisplay'
import { useUIStore } from '@/stores/ui'
import IntegrationLandingLayout from './IntegrationLandingLayout.vue'
import IntegrationExternalCta from './IntegrationExternalCta.vue'

const INTEGRATIONS_CHROME_SCENARIOS_LABELS: Record<string, string> = {
  research: 'Daily research',
  learning: 'Study notes',
  tech: 'Technical references',
  work: 'Work knowledge',
}

const INTEGRATIONS_CHROME_CAPABILITIES_TITLE_LABELS: Record<string, string> = {
  qa: 'Knowledge-base Q&A',
  clip: 'One-click web capture',
  notes: 'Markdown quick notes',
  shortcuts: 'Keyboard shortcuts',
}

const INTEGRATIONS_CHROME_CAPABILITIES_DESC_LABELS: Record<string, string> = {
  qa: 'Sidebar chat with multi-KB switching and fast/deep/precise answer modes—ask without leaving the page.',
  clip: 'Save page URLs, AI-extract main content, or manually select regions into a target knowledge base.',
  notes: 'Built-in Markdown editor for ideas and notes, saved to your knowledge base in one click.',
  shortcuts: 'Customize shortcuts to ask questions, open the sidebar, and speed up daily workflows.',
}

const INTEGRATIONS_CHROME_STEPS_TITLE_LABELS: Record<string, string> = {
  api: 'Get API credentials',
  port: 'Desktop: fixed port (recommended)',
  install: 'Install the extension',
  connect: 'Connect in the extension',
}

const INTEGRATIONS_CHROME_STEPS_DESC_LABELS: Record<string, string> = {
  api: 'Copy your API Key and base URL from Settings → API Info.',
  port: 'On sustainability.ai Desktop, set a fixed API port (e.g. 37841) in API Info so the URL stays stable across restarts.',
  install: 'Install “Knowledge Assistant” from the Chrome Web Store.',
  connect: 'Open extension settings, choose enterprise/developer mode, and enter the service API URL and API Key. Your current API URL is shown below.',
}

const router = useRouter()
const uiStore = useUIStore()
const { apiBaseUrlDisplay } = useApiBaseUrlDisplay()

const capabilityKeys = ['qa', 'clip', 'notes', 'shortcuts'] as const
const scenarioKeys = ['research', 'learning', 'tech', 'work'] as const
const stepKeys = ['api', 'port', 'install', 'connect'] as const

const capabilityIcons: Record<(typeof capabilityKeys)[number], string> = {
  qa: 'chat-bubble',
  clip: 'file-copy',
  notes: 'edit',
  shortcuts: 'jump',
}

const openChromeStore = () => {
  window.open(CHROME_EXTENSION_URL, '_blank', 'noopener,noreferrer')
}

const openApiSettings = () => {
  router.push({ path: '/platform/settings', query: { section: 'integration-api' } })
  uiStore.openSettings('integration-api')
}

const copyApiUrl = async () => {
  await copyWithToast(apiBaseUrlDisplay.value, 'API URL copied')
}
</script>
