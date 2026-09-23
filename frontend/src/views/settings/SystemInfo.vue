<template>
  <div class="system-info">
    <div class="section-header">
      <h2>{{ 'System Information' }}</h2>
      <p class="section-description">{{ 'View system version information and user account configuration' }}</p>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="loading-inline">
      <t-loading size="small" />
      <span>{{ 'Loading information...' }}</span>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">{{ 'Retry' }}</t-button>
        </template>
      </t-alert>
    </div>

    <!-- Content -->
    <div v-else class="settings-group">
      <!-- System version -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'App Version' }}</label>
          <p class="desc">{{ 'Version of the application service (enterpriserag-app)' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">
              {{ systemInfo?.version || 'Unknown' }}
              <t-tag
                v-if="systemInfo?.edition"
                :theme="systemInfo.edition === 'lite' ? 'primary' : 'default'"
                variant="light"
                size="small"
                style="margin-left: 8px;"
              >{{ systemInfo.edition === 'lite' ? 'Lite' : 'Standard' }}</t-tag>
              <span v-if="systemInfo?.commit_id" class="commit-info">
                ({{ systemInfo.commit_id }})
              </span>
          </span>
        </div>
      </div>

      <!-- Frontend version -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'UI Version' }}</label>
          <p class="desc">{{ 'Build version of the UI (enterpriserag-ui)' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">
            {{ frontendVersion }}
            <t-tag
              v-if="systemInfo?.version && systemInfo.version !== 'unknown' && frontendVersion !== 'unknown' && systemInfo.version !== frontendVersion"
              theme="warning"
              variant="light"
              size="small"
              style="margin-left: 8px;"
            >{{ 'Mismatch with app version' }}</t-tag>
            <span v-if="frontendCommit && frontendCommit !== 'unknown'" class="commit-info">
              ({{ frontendCommit }})
            </span>
          </span>
        </div>
      </div>

      <!-- Build time -->
      <div v-if="systemInfo?.build_time" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Build Time' }}</label>
          <p class="desc">{{ 'Time when the system was built' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ systemInfo.build_time }}</span>
        </div>
      </div>

      <!-- Go version -->
      <div v-if="systemInfo?.go_version" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Go Version' }}</label>
          <p class="desc">{{ 'Go language version used by the backend' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ systemInfo.go_version }}</span>
        </div>
      </div>

      <!-- Service started at -->
      <div v-if="systemInfo?.started_at" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Service started at' }}</label>
          <p class="desc">{{ 'When the current backend process last started' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ formatStartedAt(systemInfo.started_at) }}</span>
        </div>
      </div>

      <!-- Service uptime -->
      <div v-if="displayUptimeSeconds != null" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Uptime' }}</label>
          <p class="desc">{{ 'How long the service has been running since this start' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ formatUptime(displayUptimeSeconds) }}</span>
        </div>
      </div>

      <!-- DB Version -->
      <div v-if="systemInfo?.db_version || systemInfo?.db_migration_error" class="setting-row">
        <div class="setting-info">
          <label>{{ 'Database Version' }}</label>
          <p class="desc">{{ 'Current database migration version' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">
            {{ systemInfo?.db_version || 'Unknown' }}
            <t-tag
              v-if="systemInfo?.db_migration_error"
              theme="danger"
              variant="light"
              size="small"
              style="margin-left: 8px;"
            >{{ 'Migration failed' }}</t-tag>
          </span>
        </div>
      </div>

      <!-- DB migration error: full-width banner under the row -->
      <div v-if="systemInfo?.db_migration_error" class="setting-row migration-error-row">
        <t-alert theme="error" :title="'Database migration failed'" style="width: 100%;">
          <template #default>
            <p class="migration-error-desc">{{ 'The startup database migration did not complete successfully. Some tables or indexes may be missing, which can break Wiki ingest, the knowledge graph, and other features. Check the troubleshooting guide below first; if the issue persists, report it via the link.' }}</p>
            <pre class="migration-error-detail">{{ systemInfo.db_migration_error }}</pre>
            <div class="migration-error-actions">
              <t-link
                theme="primary"
                :href="troubleshootingDocsURL"
                target="_blank"
                rel="noopener noreferrer"
              >{{ 'View troubleshooting guide' }}</t-link>
              <span class="migration-error-actions-sep">·</span>
              <t-link
                theme="primary"
                :href="reportIssueURL"
                target="_blank"
                rel="noopener noreferrer"
              >{{ 'Can\'t fix it? Report an issue' }}</t-link>
            </div>
          </template>
        </t-alert>
      </div>

      <!-- Keyword Index Engine -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Keyword Index Engine' }}</label>
          <p class="desc">{{ 'Currently used keyword index engine' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ systemInfo?.keyword_index_engine || 'Unknown' }}</span>
        </div>
      </div>

      <!-- Vector Store Engine -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Vector Store Engine' }}</label>
          <p class="desc">{{ 'Currently used vector store engine' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ systemInfo?.vector_store_engine || 'Unknown' }}</span>
        </div>
      </div>

      <!-- Graph Database Engine -->
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ 'Graph Database Engine' }}</label>
          <p class="desc">{{ 'Currently used graph database engine' }}</p>
        </div>
        <div class="setting-control">
          <span class="info-value">{{ systemInfo?.graph_database_engine || 'Unknown' }}</span>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getSystemInfo, type SystemInfo } from '@/api/system'


// Reactive state
const systemInfo = ref<SystemInfo | null>(null)
const loading = ref(true)
const error = ref('')
const frontendVersion = __FRONTEND_VERSION__
const frontendCommit = __FRONTEND_COMMIT__

let uptimeTicker: ReturnType<typeof setInterval> | null = null
const uptimeTick = ref(0)

const displayUptimeSeconds = computed(() => {
  void uptimeTick.value
  const info = systemInfo.value
  if (info?.started_at) {
    const boot = new Date(info.started_at).getTime()
    if (!Number.isNaN(boot)) {
      return Math.floor((Date.now() - boot) / 1000)
    }
  }
  if (info?.uptime_seconds != null) return info.uptime_seconds
  return null
})

function formatStartedAt(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('en-US')
}

function formatUptime(totalSeconds: number): string {
  const sec = Math.max(0, Math.floor(totalSeconds))
  const days = Math.floor(sec / 86400)
  const hours = Math.floor((sec % 86400) / 3600)
  const minutes = Math.floor((sec % 3600) / 60)
  const seconds = sec % 60
  const parts: string[] = []
  if (days > 0) parts.push(`${days}d`)
  if (hours > 0 || days > 0) parts.push(`${hours}h`)
  if (minutes > 0 || hours > 0 || days > 0) parts.push(`${minutes}m`)
  if (parts.length === 0) return `${seconds}s`
  if (seconds > 0 && days === 0) parts.push(`${seconds}s`)
  return parts.join(' ')
}

const troubleshootingDocsURL =
  'https://github.com/ORG_PLACEHOLDER/EnterpriseRag/blob/main/README.md'

// Pre-fills a new issue with the current migration error so users don't have to
// paste it manually. Body is intentionally minimal — the bug template will fill
// in the rest. Encode aggressively to survive newlines / quotes.
const reportIssueURL = computed(() => {
  const base = 'https://github.com/ORG_PLACEHOLDER/EnterpriseRag/issues/new'
  const params = new URLSearchParams({
    template: 'bug_report.yml',
    title: '[Bug]: Database migration failed at startup',
    labels: 'bug',
  })
  const errMsg = systemInfo.value?.db_migration_error
  if (errMsg) {
    const body = [
      '### Environment',
      `- sustainability.ai version: ${systemInfo.value?.version || 'unknown'}`,
      `- Commit: ${systemInfo.value?.commit_id || 'unknown'}`,
      `- Frontend version: ${frontendVersion} (${frontendCommit})`,
      `- DB version reported: ${systemInfo.value?.db_version || 'unknown'}`,
      '',
      '### Migration error',
      '```',
      errMsg,
      '```',
    ].join('\n')
    params.set('body', body)
  }
  return `${base}?${params.toString()}`
})

// Methods
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''
    
    const systemResponse = await getSystemInfo()
    
    if (systemResponse.data) {
      systemInfo.value = systemResponse.data
    } else {
      error.value = 'Failed to fetch system information'
    }
  } catch (err: any) {
    error.value = err?.message || 'Network error, please try again later'
  } finally {
    loading.value = false
  }
}

// Lifecycle
onMounted(() => {
  loadInfo()
  uptimeTicker = setInterval(() => {
    uptimeTick.value++
  }, 30_000)
})

onUnmounted(() => {
  if (uptimeTicker) {
    clearInterval(uptimeTicker)
    uptimeTicker = null
  }
})
</script>

<style lang="less" scoped>
@import (reference) '@/components/css/settings-section.less';

.system-info {
  width: 100%;
}

.section-header {
  .settings-section-header();
}

.loading-inline {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 40px 0;
  justify-content: center;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-base);
}

.error-inline {
  padding: 20px 0;
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  .setting-row();
}

.setting-info {
  .setting-info();
}

.migration-error-row {
  display: block;
  padding: 0 0 20px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.migration-error-desc {
  margin: 0 0 8px 0;
  font-size: var(--app-text-md);
  line-height: 1.5;
  color: var(--td-text-color-primary);
}

.migration-error-detail {
  margin: 0 0 12px 0;
  padding: 8px 12px;
  background: var(--td-bg-color-container-hover);
  border-radius: var(--app-radius-xs);
  font-size: var(--app-text-sm);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow: auto;
  color: var(--td-text-color-secondary);
}

.migration-error-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: var(--app-text-md);

  .migration-error-actions-sep {
    color: var(--td-text-color-placeholder);
  }
}

.setting-control {
  .setting-control();

  .info-value {
    font-size: var(--app-text-base);
    color: var(--td-text-color-primary);
    text-align: right;
    word-break: break-word;

    .commit-info {
      color: var(--td-text-color-placeholder);
      font-size: var(--app-text-sm);
      margin-left: 6px;
    }
  }
}
</style>
