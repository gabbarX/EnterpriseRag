<template>
  <IntegrationLandingLayout
    :title="'sustainability.ai CLI'"
    :subtitle="'Manage knowledge bases and documents, search content, and ask questions from your terminal. Connect scripts and AI tools through the CLI or MCP.'"
  >
    <template #actions>
      <IntegrationExternalCta
        :label="'CLI documentation'"
        :hint="'Installation and complete command reference'"
        @click="openDocs"
      >
        <template #icon><t-icon name="code" /></template>
      </IntegrationExternalCta>
    </template>

    <template #main>
      <div class="landing-group">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ 'Quick start' }}</h4>
          <ol class="landing-steps">
            <li v-for="(step, index) in steps" :key="step.key" class="landing-step">
              <span class="landing-step-num">{{ index + 1 }}</span>
              <div class="landing-step-body">
                <div class="landing-step-title">{{ (INTEGRATIONS_CLI_TITLE_LABELS[step.key] ?? '') }}</div>
                <p class="landing-step-desc">{{ (INTEGRATIONS_CLI_DESC_LABELS[step.key] ?? '') }}</p>
                <div class="landing-step-embed code-toolbar">
                  <pre class="code-toolbar__code">{{ step.command }}</pre>
                  <t-button
                    class="code-toolbar__copy"
                    size="small"
                    variant="text"
                    shape="square"
                    :title="'Copy'"
                    :aria-label="'Copy'"
                    @click="copy(step.command)"
                  >
                    <t-icon name="file-copy" size="16px" />
                  </t-button>
                </div>
              </div>
            </li>
          </ol>
        </section>
      </div>
    </template>

    <template #aside>
      <div class="landing-group">
        <section v-for="example in examples" :key="example.key" class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ (INTEGRATIONS_CLI_TITLE_LABELS[example.key] ?? '') }}</h4>
          <p class="field-desc">{{ (INTEGRATIONS_CLI_DESC_LABELS[example.key] ?? '') }}</p>
          <div class="code-toolbar">
            <pre class="code-toolbar__code">{{ example.command }}</pre>
            <t-button
              class="code-toolbar__copy"
              size="small"
              variant="text"
              shape="square"
              :title="'Copy'"
              :aria-label="'Copy'"
              @click="copy(example.command)"
            >
              <t-icon name="file-copy" size="16px" />
            </t-button>
          </div>
        </section>
      </div>
    </template>
  </IntegrationLandingLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useApiBaseUrlDisplay } from '@/composables/useApiBaseUrlDisplay'
import { copyWithToast } from '@/utils/clipboard'
import IntegrationLandingLayout from './IntegrationLandingLayout.vue'
import IntegrationExternalCta from './IntegrationExternalCta.vue'
import { buildCLIConnectCommand } from './cliIntegration'

const INTEGRATIONS_CLI_TITLE_LABELS: Record<string, string> = {
  install: 'Install the CLI',
  connect: 'Connect to this server',
  verify: 'Verify the connection',
  commands: 'Common commands',
  mcp: 'Connect an MCP client',
}

const INTEGRATIONS_CLI_DESC_LABELS: Record<string, string> = {
  install: 'Build from source with Git and Go 1.26+. This macOS / Linux example updates PATH for the current terminal only. For regular use, place the binary in a directory on PATH.',
  connect: 'Create and activate a profile named enterpriserag, then sign in with your email and password. If that profile already exists, choose another name and update the MCP example to match.',
  verify: 'Check server and authentication status, then list the knowledge bases your account can access.',
  commands: 'Replace KB_ID with a knowledge base ID and adapt the file path, query, and question. Uploaded documents must finish processing before they can be searched.',
  mcp: 'After signing in, add this configuration to an MCP client that supports stdio. If the client cannot find enterpriserag, set command to the absolute path of the binary.',
}

const { apiBaseUrlDisplay } = useApiBaseUrlDisplay()
const steps = computed(() => [
  {
    key: 'install',
    command: 'git clone https://github.com/ORG_PLACEHOLDER/EnterpriseRag.git\ncd EnterpriseRag/cli\ngo build -o enterpriserag .\nexport PATH="$PWD:$PATH"',
  },
  { key: 'connect', command: buildCLIConnectCommand(apiBaseUrlDisplay.value, window.location.origin) },
  { key: 'verify', command: 'enterpriserag doctor\nenterpriserag kb list' },
])
const examples = [
  {
    key: 'commands',
    command: 'enterpriserag doc upload ./document.pdf --kb "KB_ID"\nenterpriserag search chunks "query" --kb "KB_ID"\nenterpriserag chat "question" --kb "KB_ID" --format text\nenterpriserag agent list',
  },
  {
    key: 'mcp',
    command: JSON.stringify({
      mcpServers: {
        enterpriserag: { command: 'enterpriserag', args: ['--profile', 'enterpriserag', 'mcp', 'serve'] },
      },
    }, null, 2),
  },
]

const copy = (command: string) => copyWithToast(command, 'Copied')
const openDocs = () => {
  window.open('https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/cli/README.md', '_blank', 'noopener,noreferrer')
}
</script>
