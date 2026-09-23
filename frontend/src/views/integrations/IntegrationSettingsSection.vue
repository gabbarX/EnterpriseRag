<template>
  <div class="integrations-settings">
    <div class="integrations-settings__body" :class="{ 'integrations-settings__body--landing': isLandingSection }">
      <div v-if="tab === 'im'" class="section">
        <div class="section-header">
          <h2>{{ 'IM Integration' }}</h2>
          <p class="section-description">
            {{ 'Connect agent to instant messaging platforms like Slack, Telegram and Mattermost' }}
            <a
              href="https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/README.md"
              target="_blank"
              rel="noopener noreferrer"
              class="doc-link"
            >
              {{ 'Integration Guide' }}
              <t-icon name="link" class="link-icon" />
            </a>
          </p>
        </div>
        <IMChannelPanel v-model:filter-agent-id="filterAgentId" />
      </div>

      <div v-if="tab === 'embed'" class="section">
        <div class="section-header">
          <h2>{{ 'Web Page Embed' }}</h2>
          <p class="section-description">{{ 'Embed this agent on your website so visitors can chat via an in-page window or floating launcher. Knowledge scope follows this agent.' }}</p>
        </div>
        <AgentEmbedChannelPanel v-model:filter-agent-id="filterAgentId" />
      </div>

      <div v-if="tab === 'api'" class="section">
        <div class="section-header">
          <h2>{{ 'API Integration' }}</h2>
          <p class="section-description">{{ 'Integrate via REST API and configure how requests identify end users.' }}</p>
        </div>
        <ApiIntegrationSettings />
      </div>

      <div v-if="tab === 'mcpserver'" class="section">
        <div class="section-header">
          <h2>{{ 'MCP Server' }}</h2>
          <p class="section-description">{{ 'Publish this workspace as an MCP server that Claude Desktop, Cursor, Claude Code and other MCP clients connect to directly. Each endpoint has its own token, knowledge-base scope and tool list.' }}</p>
        </div>
        <McpServerIntegrationSettings />
      </div>

      <ChromeExtensionLanding v-if="tab === 'chrome'" />
      <ClawSkillLanding v-if="tab === 'claw'" />
      <CliIntegrationLanding v-if="tab === 'cli'" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import IMChannelPanel from '@/components/IMChannelPanel.vue'
import AgentEmbedChannelPanel from '@/components/AgentEmbedChannelPanel.vue'
import ApiIntegrationSettings from '@/views/integrations/ApiIntegrationSettings.vue'
import McpServerIntegrationSettings from '@/views/integrations/McpServerIntegrationSettings.vue'
import ChromeExtensionLanding from '@/views/integrations/ChromeExtensionLanding.vue'
import ClawSkillLanding from '@/views/integrations/ClawSkillLanding.vue'
import CliIntegrationLanding from '@/views/integrations/CliIntegrationLanding.vue'
import type { IntegrationTab } from '@/config/integrations'

const filterAgentId = ref('')

const props = defineProps<{
  tab: IntegrationTab
}>()

const route = useRoute()

const isLandingSection = computed(
  () => props.tab === 'chrome' || props.tab === 'claw' || props.tab === 'cli',
)

function applyAgentFilterFromRoute() {
  filterAgentId.value = (route.query.agentId as string) || ''
}

watch(
  () => route.query.agentId,
  applyAgentFilterFromRoute,
  { immediate: true },
)
</script>

<style scoped lang="less">
@import (reference) '@/components/css/settings-section.less';

.integrations-settings {
  display: flex;
  flex-direction: column;
}

.integrations-settings__body {
  min-width: 0;
}

.integrations-settings__body--landing {
  max-width: 760px;
}

.section-header {
  .settings-section-header();
}

.doc-link {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: 6px;
  color: var(--td-brand-color);
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}

.link-icon {
  font-size: var(--app-text-md);
}
</style>
