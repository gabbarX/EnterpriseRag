<template>
  <div class="runtime-queues">
    <header class="section-header rq-header">
      <div class="rq-title-block">
        <h2>{{ 'Task Queue Runtime' }}</h2>
        <p class="section-description">{{ 'Live background-queue load and per-process capacity for each isolated worker pool. Includes task details and safe controls; auto-refreshes every 5s.' }}</p>
      </div>
      <div class="rq-actions">
        <label class="rq-auto-refresh">
          <span class="rq-live-dot" :class="{ 'rq-live-dot--active': autoRefresh }" />
          <span>{{ 'Auto-refresh (every 5s)' }}</span>
          <t-switch
            v-model="autoRefresh"
            size="small"
            :aria-label="'Auto-refresh (every 5s)'"
          />
        </label>
        <button
          type="button"
          class="rq-refresh"
          :disabled="loading"
          :title="'Refresh'"
          :aria-label="'Refresh'"
          @click="reload"
        >
          <t-icon
            :name="loading ? 'loading' : 'refresh'"
            :class="{ 'rq-refresh-spin': loading }"
          />
        </button>
      </div>
    </header>

    <div v-if="loading && !loadedOnce" class="rq-loading" aria-live="polite">
      <div class="rq-loading-metrics">
        <t-skeleton
          v-for="n in 4"
          :key="n"
          animation="gradient"
          :row-col="[{ width: '42%', height: '28px' }, { width: '66%', height: '14px' }]"
        />
      </div>
      <t-skeleton
        animation="gradient"
        :row-col="[
          { width: '100%', height: '42px' },
          { width: '100%', height: '48px' },
          { width: '100%', height: '48px' },
          { width: '100%', height: '48px' },
        ]"
      />
    </div>

    <div v-else-if="error" class="rq-state rq-state--error" role="alert">
      <div class="rq-state-icon"><t-icon name="error-circle" size="24px" /></div>
      <div class="rq-state-copy">
        <strong>{{ 'Failed to load queue stats' }}</strong>
        <span>{{ error }}</span>
      </div>
      <t-button size="small" variant="outline" @click="reload">
        {{ 'Retry' }}
      </t-button>
    </div>

    <div v-else-if="!available && !modelLimiterAvailable" class="rq-state">
      <div class="rq-state-icon"><t-icon name="info-circle" size="24px" /></div>
      <div class="rq-state-copy">
        <strong>{{ 'Task queues unavailable' }}</strong>
        <span>{{ 'This deployment has no Redis / asynq queue (Lite mode) — nothing to display.' }}</span>
      </div>
    </div>

    <template v-else>
      <template v-if="available">
      <section class="rq-overview" :aria-label="'Runtime overview'">
        <div class="rq-overview-title">
          <span class="rq-overview-mark"><t-icon name="chart-line" /></span>
          <span>{{ 'Runtime overview' }}</span>
        </div>
        <div class="rq-overview-metrics">
          <div class="rq-metric rq-metric--active">
            <span class="rq-metric-label">{{ 'Active' }}</span>
            <strong class="rq-metric-value">{{ totalActive }}</strong>
          </div>
          <div class="rq-metric">
            <span class="rq-metric-label">{{ 'Pending' }}</span>
            <strong class="rq-metric-value">{{ totalPending }}</strong>
          </div>
          <div class="rq-metric" :class="{ 'rq-metric--warning': totalRetry > 0 }">
            <span class="rq-metric-label">{{ 'Retrying' }}</span>
            <strong class="rq-metric-value">{{ totalRetry }}</strong>
          </div>
          <div class="rq-metric" :class="{ 'rq-metric--danger': totalArchived > 0 }">
            <span class="rq-metric-label">{{ 'Failed (stopped)' }}</span>
            <strong class="rq-metric-value">{{ totalArchived }}</strong>
          </div>
        </div>
      </section>

      <section class="rq-pools">
        <div class="rq-pools-header">
          <div>
            <h3 class="rq-section-title">{{ 'Worker pools' }}</h3>
            <p>{{ 'Each stage has guaranteed capacity; core and enrichment may also borrow the shared elastic pool.' }}</p>
          </div>
          <span class="rq-pools-note">{{ 'Card value is cluster active/capacity' }}</span>
        </div>
        <div class="rq-pool-grid">
          <div v-for="pool in pools" :key="pool.name" class="rq-pool-card">
            <div class="rq-pool-topline">
              <span class="rq-pool-name">{{ poolLabel(pool.name) }}</span>
              <strong class="rq-pool-value">
                {{ pool.instances > 0 ? `${pool.active}/${pool.cluster_capacity}` : pool.concurrency }}
              </strong>
            </div>
            <p class="rq-pool-desc">
              {{ poolDescription(pool.name) }}
              <span class="rq-pool-meta">
                {{ `Per-process ${pool.concurrency}` }}
                <template v-if="pool.instances > 0">
                  · {{ `${pool.instances} instances` }}
                  · {{ `${poolUtilization(pool)}% utilized` }}
                </template>
                · {{ `${pool.queue_count} queues` }}
              </span>
            </p>
          </div>
        </div>
      </section>

      <section class="rq-details">
        <div class="rq-details-header">
          <div>
            <h3 class="rq-section-title">{{ 'Queue details' }}</h3>
            <p>{{ 'Live load and wait time for every processing lane. “Failed (stopped)” means the task exceeded its retry limit and will not run automatically again.' }}</p>
          </div>
          <span v-if="updatedAt" class="rq-updated-at">
            <t-icon name="time" />
            {{ `Updated at ${updatedAt}` }}
          </span>
        </div>

        <div v-if="totalArchived > 0" class="rq-failed-notice" role="status">
          <span class="rq-failed-notice__icon" aria-hidden="true">
            <t-icon name="error-circle" />
          </span>
          <div class="rq-failed-notice__text">
            <p class="rq-failed-notice__title">
              {{ `${totalArchived} failed tasks need attention` }}
            </p>
            <p class="rq-failed-notice__desc">
              {{ 'Open a red Failed count in the table below to inspect causes, then retry after fixing the root issue.' }}
            </p>
          </div>
        </div>

        <div v-if="queues.length === 0" class="rq-empty">
          <t-icon name="queue" size="28px" />
          <span>{{ 'No queue data available' }}</span>
        </div>

        <div v-else class="data-table-shell rq-table-shell">
          <t-table
            row-key="name"
            :data="queues"
            :columns="columns"
            size="medium"
            hover
          >
            <template #name="{ row }">
              <div class="rq-queue-cell">
                <span class="rq-queue-name">{{ queueLabel(row.name) }}</span>
                <span class="rq-queue-meta">{{ queueMeta(row) }}</span>
              </div>
            </template>
            <template #active="{ row }">
              <t-button
                v-if="row.active > 0"
                variant="text"
                size="small"
                class="rq-task-count rq-task-count--active"
                @click="openRuntimeTasks(row, 'active')"
              >
                {{ row.active }}<t-icon name="chevron-right" />
              </t-button>
              <span v-else class="rq-number">0</span>
            </template>
            <template #pending="{ row }">
              <div class="rq-backlog">
                <t-button
                  v-if="row.pending > 0"
                  variant="text"
                  size="small"
                  class="rq-task-count"
                  @click="openRuntimeTasks(row, 'pending')"
                >{{ row.pending }}<t-icon name="chevron-right" /></t-button>
                <span v-else class="rq-number">0</span>
                <t-button
                  v-if="row.scheduled > 0"
                  variant="text"
                  size="small"
                  class="rq-scheduled-count"
                  @click="openRuntimeTasks(row, 'scheduled')"
                >+{{ row.scheduled }} {{ 'Scheduled' }}</t-button>
              </div>
            </template>
            <template #retry="{ row }">
              <t-button
                v-if="row.retry > 0"
                variant="text"
                theme="warning"
                size="small"
                class="rq-task-count"
                @click="openRuntimeTasks(row, 'retry')"
              >{{ row.retry }}<t-icon name="chevron-right" /></t-button>
              <span v-else class="rq-number">0</span>
            </template>
            <template #archived="{ row }">
              <t-button
                v-if="row.archived > 0"
                variant="text"
                theme="danger"
                size="small"
                class="rq-task-count rq-failed-count"
                :aria-label="`View ${row.archived} ${taskStateLabel('archived')} tasks in ${queueLabel(row.name)}`"
                @click="openRuntimeTasks(row, 'archived')"
              >
                {{ row.archived }}<t-icon name="chevron-right" />
              </t-button>
              <span v-else class="rq-number">0</span>
            </template>
            <template #completed="{ row }">
              <t-button
                v-if="row.completed > 0"
                variant="text"
                size="small"
                class="rq-task-count rq-task-count--completed"
                @click="openRuntimeTasks(row, 'completed')"
              >{{ row.completed }}<t-icon name="chevron-right" /></t-button>
              <span v-else class="rq-number">0</span>
            </template>
            <template #latency_ms="{ row }">
              <span class="rq-latency">{{ formatLatency(row.latency_ms) }}</span>
            </template>
            <template #status="{ row }">
              <span class="rq-status" :class="`rq-status--${queueState(row).tone}`">
                <i />{{ queueState(row).label }}
              </span>
            </template>
          </t-table>
        </div>
      </section>
      </template>

      <section class="rq-details rq-models">
        <div class="rq-details-header">
          <div>
            <h3 class="rq-section-title">{{ 'Model concurrency' }}</h3>
            <p>{{ 'Actual model-service concurrency from background tasks. The section above schedules tasks; this section shows the separate model-service throttle stage.' }}</p>
          </div>
          <span class="rq-pools-note">{{ 'Active is cluster-wide · waiting is local' }}</span>
        </div>
        <div v-if="!modelLimiterAvailable" class="rq-empty">
          <t-icon name="info-circle" size="28px" />
          <span>{{ 'Background model concurrency governance is disabled. Configure the default model concurrency limit in Global settings.' }}</span>
        </div>
        <div v-else-if="models.length === 0" class="rq-empty">
          <t-icon name="server" size="28px" />
          <span>{{ 'No model activity yet. A model appears after its first background call.' }}</span>
        </div>
        <div v-else class="data-table-shell rq-table-shell">
          <t-table row-key="model_id" :data="models" :columns="modelColumns" size="medium" hover>
            <template #model_id="{ row }">
              <div class="rq-queue-cell">
                <span class="rq-queue-name">{{ row.name || row.model_id }}</span>
                <span class="rq-queue-meta">{{ row.name ? row.model_id : 'Background tasks only; interactive chat is excluded' }}</span>
              </div>
            </template>
            <template #active="{ row }"><span class="rq-number" :class="{ 'rq-number--active': row.active > 0 }">{{ row.active }}</span></template>
            <template #waiting="{ row }"><span class="rq-number" :class="{ 'rq-number--warning': row.waiting > 0 }">{{ row.waiting }}</span></template>
            <template #usage="{ row }">
              <div class="rq-model-usage">
                <t-progress :percentage="modelUsage(row)" size="small" :label="false" />
                <span>{{ row.active }} / {{ row.limit }}</span>
              </div>
            </template>
            <template #status="{ row }">
              <span class="rq-status" :class="`rq-status--${modelState(row).tone}`"><i />{{ modelState(row).label }}</span>
            </template>
          </t-table>
        </div>
      </section>

      <p class="rq-footnote">{{ 'Card values show cluster active/live capacity; per-process configuration requires restart. The shared elastic pool only consumes core and enrichment queues.' }}</p>
    </template>

    <SettingDrawer
      v-model:visible="taskDrawerVisible"
      class="rq-failed-drawer"
      :title="`Task details · ${taskQueueLabel}`"
      :description="'Inspect every task state. Scheduled and retry tasks show the next run first; other states show the newest tasks first.'"
      icon="queue"
      width="720px"
      :min-width="520"
      :max-width="1040"
      storage-key="setting-drawer:width:runtime-tasks"
      hide-footer
    >
      <section class="setting-drawer__section">
        <div
          class="rq-task-state-filter"
          role="tablist"
          :aria-label="'Filter by task state'"
        >
          <button
            v-for="state in taskStates"
            :key="state"
            type="button"
            role="tab"
            class="rq-task-state-option"
            :class="{ 'is-active': taskState === state }"
            :aria-selected="taskState === state"
            @click="selectTaskState(state)"
          >
            <span class="rq-task-state-option__label">{{ taskStateLabel(state) }}</span>
            <span
              class="rq-task-state-option__count"
              :class="{ 'has-value': taskStateCount(taskQueue, state) > 0 }"
            >
              {{ taskStateCount(taskQueue, state) }}
            </span>
          </button>
        </div>
        <p class="rq-failed-guide-desc">{{ taskStateGuide }}</p>
      </section>

      <section class="setting-drawer__section">
        <div class="rq-failed-section-head">
          <h4 class="setting-drawer__section-title">
            {{ `${taskStateLabel(taskState)} tasks` }}
          </h4>
          <div class="rq-failed-section-actions">
            <t-popconfirm
              v-if="taskState === 'archived' && tasks.length > 0"
              theme="danger"
              :content="`Remove all ${taskStateCount(taskQueue, 'archived')} failed records from this queue at once. This only clears failed tasks in the queue — it does not affect running or pending tasks, nor does it roll back the business status of documents that already failed. Clear them?`"
              @confirm="purgeArchivedTasks"
            >
              <t-button
                variant="text"
                size="small"
                theme="danger"
                :loading="purging"
                :disabled="Boolean(taskActionID)"
              >
                <template #icon><t-icon name="clear" /></template>
                {{ 'Clear all failed tasks' }}
              </t-button>
            </t-popconfirm>
            <t-button
              variant="text"
              size="small"
              :loading="tasksLoading && !tasksLoadingMore"
              @click="reloadRuntimeTasks"
            >
              <template #icon><t-icon name="refresh" /></template>
              {{ 'Refresh' }}
            </t-button>
          </div>
        </div>

        <div v-if="tasksLoading && tasks.length === 0" class="rq-failed-loading">
          <t-loading size="small" />
          <span>{{ 'Loading...' }}</span>
        </div>
        <div v-else-if="tasksError" class="rq-failed-error-state">
          <span>{{ tasksError }}</span>
          <t-button size="small" variant="outline" @click="reloadRuntimeTasks">
            {{ 'Retry' }}
          </t-button>
        </div>
        <t-empty
          v-else-if="tasks.length === 0"
          :description="`This queue has no ${taskStateLabel(taskState)} tasks`"
        />
        <div v-else class="rq-failed-list-panel">
          <article
            v-for="task in tasks"
            :key="task.id"
            class="rq-failed-row"
          >
            <div class="rq-failed-row-content">
              <div class="rq-failed-row-summary">
                <span class="rq-failed-row-type">{{ runtimeTaskTypeLabel(task.type) }}</span>
                <span class="rq-failed-row-sep" aria-hidden="true">·</span>
                <span class="rq-task-state-pill" :class="`rq-task-state-pill--${task.state}`">
                  {{ taskStateLabel(task.state) }}
                </span>
                <span class="rq-failed-row-sep" aria-hidden="true">·</span>
                <span class="rq-failed-row-stat">
                  {{ `Attempt ${task.retried + 1}/${task.max_retry + 1}` }}
                </span>
              </div>
              <dl v-if="runtimeTaskMeta(task).length > 0" class="rq-failed-row-refs">
                <div v-for="ref in runtimeTaskMeta(task)" :key="ref.key" class="rq-failed-ref">
                  <dt>{{ ref.label }}</dt>
                  <dd :title="ref.value">{{ ref.value }}</dd>
                </div>
              </dl>
              <p v-else class="rq-failed-row-unknown">
                {{ 'No related object identified' }}
              </p>
              <p v-if="task.last_error" class="rq-failed-row-error">
                {{ task.last_error }}
              </p>
            </div>

            <div v-if="task.allowed_actions.length > 0" class="rq-failed-row-actions">
              <t-popconfirm
                v-if="task.allowed_actions.includes('cancel')"
                theme="danger"
                :content="'This uses the business cancellation flow and also stops related tasks for the same document. Cancel it?'"
                @confirm="runTaskAction(task, 'cancel')"
              >
                <t-button
                  shape="square"
                  variant="text"
                  size="small"
                  theme="danger"
                  class="rq-failed-icon-btn"
                  :title="'Cancel task'"
                  :aria-label="'Cancel task'"
                  :loading="taskActionID === task.id && taskAction === 'cancel'"
                  :disabled="Boolean(taskActionID)"
                ><t-icon name="close-circle" /></t-button>
              </t-popconfirm>
              <t-popconfirm
                v-if="task.allowed_actions.includes('run_now')"
                theme="warning"
                :content="'This task will move to pending immediately without resetting its retry count. Continue?'"
                @confirm="runTaskAction(task, 'run_now')"
              >
                <t-button
                  shape="square"
                  variant="text"
                  size="small"
                  class="rq-failed-icon-btn"
                  :title="'Run now'"
                  :aria-label="'Run now'"
                  :loading="taskActionID === task.id && taskAction === 'run_now'"
                  :disabled="Boolean(taskActionID)"
                >
                  <t-icon name="refresh" />
                </t-button>
              </t-popconfirm>
              <t-popconfirm
                v-if="task.allowed_actions.includes('delete')"
                theme="danger"
                :content="'This only removes the record from this queue. It will not run or complete the original task, and historical logs remain. Clear it?'"
                @confirm="runTaskAction(task, 'delete')"
              >
                <t-button
                  shape="square"
                  variant="text"
                  size="small"
                  theme="danger"
                  class="rq-failed-icon-btn"
                  :title="'Clear record'"
                  :aria-label="'Clear record'"
                  :loading="taskActionID === task.id && taskAction === 'delete'"
                  :disabled="Boolean(taskActionID)"
                >
                  <t-icon name="delete" />
                </t-button>
              </t-popconfirm>
            </div>
          </article>

          <div ref="tasksSentinelRef" class="rq-failed-load-sentinel" aria-hidden="true" />

          <div class="rq-failed-list-footer">
            <span class="rq-failed-list-status">
              <template v-if="tasksLoadingMore">
                {{ 'Loading more…' }}
              </template>
              <template v-else-if="!tasksHasMore">
                {{ `All ${tasks.length} loaded` }}
              </template>
              <template v-else>
                {{ `${tasks.length} loaded — scroll or tap to load more` }}
              </template>
            </span>
            <t-button
              v-if="tasksHasMore"
              variant="outline"
              block
              :loading="tasksLoadingMore"
              @click="loadMoreRuntimeTasks"
            >
              {{ 'Load more' }}
            </t-button>
          </div>
        </div>
      </section>
    </SettingDrawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import { mergeRuntimeTaskPage } from './runtimeTaskPagination'
import {
  getRuntimeTasks,
  getRuntimeQueues,
  mutateRuntimeTask,
  purgeArchivedRuntimeTasks,
  type ModelRuntimeStat,
  type QueueStat,
  type RuntimeTask,
  type RuntimeTaskAction,
  type RuntimeTaskState,
  type RuntimeWorkerPool,
} from '@/api/system'

const RUNTIME_QUEUE_NAME_LABELS: Record<string, string> = {
  default: 'Document parsing',
  chat_attachment: 'Chat attachments',
  postprocess: 'Post-process',
  summary: 'Summaries',
  sync: 'Source sync',
  low: 'Maintenance & batch',
  multimodal: 'Multimodal',
  graph: 'Graph extraction',
  question: 'Questions',
  wiki: 'Wiki pipeline',
}

const RUNTIME_QUEUE_DESCRIPTION_LABELS: Record<string, string> = {
  default: 'Document parse, manual reparse',
  chat_attachment: 'Session-scoped chat upload parsing',
  postprocess: 'Parse finalization, enrichment fan-out',
  summary: 'Document & table summaries',
  sync: 'Manual & scheduled sync',
  low: 'FAQ import, batch reparse, cleanup',
  multimodal: 'Image OCR, vision captions',
  graph: 'Chunk-level graph extraction',
  question: 'Chunk-level question generation',
  wiki: 'Content generation, index finalize',
}

const RUNTIME_TASK_TYPE_LABELS: Record<string, string> = {
  documentProcess: 'Document parsing',
  manualProcess: 'Manual reprocessing',
  temporaryDocumentProcess: 'Chat attachment parsing',
  postProcess: 'Document post-processing',
  summary: 'Summary generation',
  tableSummary: 'Table summary generation',
  question: 'Question generation',
  multimodal: 'Image multimodal processing',
  graph: 'Knowledge graph extraction',
  sync: 'Data-source sync',
  faqImport: 'FAQ import',
  batchReparse: 'Batch reparse',
  batchDelete: 'Batch delete',
  move: 'Document move',
  indexDelete: 'Index deletion',
  kbClone: 'Knowledge-base clone',
  kbDelete: 'Knowledge-base deletion',
  wikiIngest: 'Wiki content generation',
  wikiFinalize: 'Wiki finalization',
}

const RUNTIME_POOL_LABELS: Record<string, string> = {
  core: 'Core parsing',
  postprocess: 'Post-process orchestration',
  enrichment: 'Enrichment',
  maintenance: 'Maintenance & sync',
  shared: 'Shared elastic',
  wiki: 'Wiki pool',
}

const RUNTIME_POOL_DESCRIPTION_LABELS: Record<string, string> = {
  core: 'Guaranteed document and manual parsing capacity',
  postprocess: 'Parse finalization and enrichment fan-out',
  enrichment: 'Summaries, images, graph, and question generation',
  maintenance: 'Source sync, batch work, and deletion cleanup',
  shared: 'Borrowed by core or enrichment according to backlog',
  wiki: 'Wiki content generation and global finalization',
}

const SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_GUIDES_LABELS: Record<string, string> = {
  active: 'Inspect the worker, start time, deadline, and orphan status. Cancel is offered only when the business state can be updated safely.',
  pending: 'Pending tasks have not been claimed. Cancellation updates business state instead of only removing Redis data.',
  scheduled: 'Scheduled tasks can run early; document pipeline tasks can also use safe business cancellation.',
  retry: 'Use the last error, attempt count, and next run time to decide whether to run now or cancel.',
  archived: 'Fix the root cause before running again. Clearing a record does not complete the original business task.',
  completed: 'Only recently completed tasks with result retention are shown. No management actions are available.',
}

const SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_STATES_LABELS: Record<string, string> = {
  active: 'Active',
  pending: 'Pending',
  scheduled: 'Scheduled',
  retry: 'Retrying',
  archived: 'Failed',
  completed: 'Completed',
}

const SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_ACTION_SUCCESS_LABELS: Record<string, string> = {
  cancel: 'Task cancelled',
  run_now: 'Task moved to pending',
  delete: 'Failure record cleared',
}

const SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_ACTION_ERROR_LABELS: Record<string, string> = {
  cancel: 'Failed to cancel task',
  run_now: 'Failed to run task',
  delete: 'Failed to clear record',
}


const POLL_INTERVAL_MS = 5000

const queues = ref<QueueStat[]>([])
const pools = ref<RuntimeWorkerPool[]>([])
const models = ref<ModelRuntimeStat[]>([])
const modelLimiterAvailable = ref(false)
const available = ref(true)
const loading = ref(false)
const loadedOnce = ref(false)
const error = ref('')
const autoRefresh = ref(true)
const updatedAt = ref('')
const taskDrawerVisible = ref(false)
const taskQueue = ref<QueueStat | null>(null)
const taskState = ref<RuntimeTaskState>('archived')
const tasks = ref<RuntimeTask[]>([])
const tasksLoading = ref(false)
const tasksLoadingMore = ref(false)
const tasksError = ref('')
const tasksCursor = ref('')
const tasksHasMore = ref(false)
const tasksSentinelRef = ref<HTMLElement | null>(null)
const taskActionID = ref('')
const taskAction = ref<RuntimeTaskAction | ''>('')
const purging = ref(false)

const TASK_PAGE_SIZE = 20
const taskStates: RuntimeTaskState[] = ['active', 'pending', 'scheduled', 'retry', 'archived', 'completed']
const runtimeTaskTypeKeys: Record<string, string> = {
  'document:process': 'documentProcess',
  'manual:process': 'manualProcess',
	'temporary_document:process': 'temporaryDocumentProcess',
  'knowledge:post_process': 'postProcess',
  'summary:generation': 'summary',
  'datatable:summary': 'tableSummary',
  'question:generation': 'question',
  'image:multimodal': 'multimodal',
  'chunk:extract': 'graph',
  'datasource:sync': 'sync',
  'faq:import': 'faqImport',
  'knowledge:list_reparse': 'batchReparse',
  'knowledge:list_delete': 'batchDelete',
  'knowledge:move': 'move',
  'index:delete': 'indexDelete',
  'kb:clone': 'kbClone',
  'kb:delete': 'kbDelete',
  'wiki:ingest': 'wikiIngest',
  'wiki:finalize': 'wikiFinalize',
}

let pollTimer: ReturnType<typeof setInterval> | null = null
let tasksScrollObserver: IntersectionObserver | null = null
let tasksRequestID = 0

const columns = computed(() => [
  { colKey: 'name', title: 'Queue', minWidth: 188 },
  { colKey: 'active', title: 'Active', width: 74, align: 'center' as const },
  { colKey: 'pending', title: 'Pending', width: 84, align: 'center' as const },
  { colKey: 'retry', title: 'Retry', width: 68, align: 'center' as const },
  { colKey: 'archived', title: 'Failed', width: 96, align: 'center' as const },
  { colKey: 'completed', title: 'Completed', width: 84, align: 'center' as const },
  { colKey: 'latency_ms', title: 'Oldest wait', width: 104, align: 'center' as const },
  { colKey: 'status', title: 'Status', width: 96 },
])
const modelColumns = computed(() => [
  { colKey: 'model_id', title: 'Model ID', minWidth: 240 },
  { colKey: 'active', title: 'In flight', width: 86, align: 'center' as const },
  { colKey: 'waiting', title: 'Throttled', width: 86, align: 'center' as const },
  { colKey: 'usage', title: 'Concurrency', width: 190 },
  { colKey: 'status', title: 'Status', width: 96 },
])

function modelUsage(row: ModelRuntimeStat): number {
  return row.limit > 0 ? Math.min(100, Math.round(row.active / row.limit * 100)) : 0
}

function modelState(row: ModelRuntimeStat): { label: string; tone: string } {
  if (row.waiting > 0) return { label: 'Throttling', tone: 'attention' }
  if (row.active >= row.limit) return { label: 'At limit', tone: 'waiting' }
  if (row.active > 0) return { label: 'Working', tone: 'working' }
  return { label: 'Idle', tone: 'idle' }
}

const totalActive = computed(() => queues.value.reduce((s, q) => s + q.active, 0))
const totalPending = computed(() => queues.value.reduce((s, q) => s + q.pending, 0))
const totalRetry = computed(() => queues.value.reduce((s, q) => s + q.retry, 0))
const totalArchived = computed(() => queues.value.reduce((s, q) => s + q.archived, 0))
const taskQueueLabel = computed(() => taskQueue.value ? queueLabel(taskQueue.value.name) : '')
const taskStateGuide = computed(() => (SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_GUIDES_LABELS[taskState.value] ?? ''))

// Falls back to the raw queue name so a queue added on the backend still
// renders before a label is declared for it.
function queueLabel(name: string): string {
  return RUNTIME_QUEUE_NAME_LABELS[name] ?? name
}

function queueDescription(name: string): string {
  return RUNTIME_QUEUE_DESCRIPTION_LABELS[name] ?? name
}

function queueMeta(row: QueueStat): string {
  const scope = queueDescription(row.name)
  if (poolQueueCount(row.pool) > 1) {
    return `${scope} · ${`Weight ${row.weight}`}`
  }
  return scope
}

function runtimeTaskTypeLabel(type: string): string {
  const key = runtimeTaskTypeKeys[type]
  if (!key) return type
  return RUNTIME_TASK_TYPE_LABELS[key] ?? type
}

interface RuntimeTaskMeta {
  key: string
  label: string
  value: string
}

function taskStateLabel(state: RuntimeTaskState): string {
  return (SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_STATES_LABELS[state] ?? '')
}

function taskStateCount(row: QueueStat | null, state: RuntimeTaskState): number {
  if (!row) return 0
  return row[state] ?? 0
}

function formatTaskTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

function runtimeTaskMeta(task: RuntimeTask): RuntimeTaskMeta[] {
  const refs: RuntimeTaskMeta[] = []
  if (task.knowledge_base_id) {
    refs.push({
      key: 'kb',
      label: 'Knowledge base ID',
      value: task.knowledge_base_id,
    })
  }
  if (task.knowledge_id) {
    refs.push({
      key: 'knowledge',
      label: 'Document ID',
      value: task.knowledge_id,
    })
  }
  if (task.task_id) {
    refs.push({
      key: 'task',
      label: 'Business task ID',
      value: task.task_id,
    })
  }
  if (task.source_id) refs.push({ key: 'source', label: 'Source ID', value: task.source_id })
  if (task.target_id) refs.push({ key: 'target', label: 'Target ID', value: task.target_id })
  if (task.source_kb_id) refs.push({ key: 'source-kb', label: 'Source knowledge base', value: task.source_kb_id })
  if (task.target_kb_id) refs.push({ key: 'target-kb', label: 'Target knowledge base', value: task.target_kb_id })
  if (task.data_source_id) refs.push({ key: 'datasource', label: 'Data source ID', value: task.data_source_id })
  if (task.sync_log_id) refs.push({ key: 'sync-log', label: 'Sync log ID', value: task.sync_log_id })
  if (task.knowledge_count) {
    refs.push({ key: 'knowledge-count', label: 'Documents', value: String(task.knowledge_count) })
  }
  if (task.tenant_id) {
    refs.push({
      key: 'tenant',
      label: 'Workspace ID',
      value: String(task.tenant_id),
    })
  }
  if (task.enqueued_at) refs.push({ key: 'enqueued', label: 'Enqueued', value: formatTaskTime(task.enqueued_at) })
  if (task.started_at) refs.push({ key: 'started', label: 'Started', value: formatTaskTime(task.started_at) })
  if (task.next_process_at) refs.push({ key: 'next', label: 'Next run', value: formatTaskTime(task.next_process_at) })
  if (task.last_failed_at) refs.push({ key: 'failed', label: 'Last failed', value: formatTaskTime(task.last_failed_at) })
  if (task.completed_at) refs.push({ key: 'completed', label: 'Completed', value: formatTaskTime(task.completed_at) })
  if (task.deadline) refs.push({ key: 'deadline', label: 'Deadline', value: formatTaskTime(task.deadline) })
  if (task.worker) refs.push({ key: 'worker', label: 'Worker', value: task.worker })
  if (task.is_orphaned) refs.push({ key: 'orphaned', label: 'Runtime health', value: 'Worker heartbeat lost; waiting for recovery' })
  return refs
}

function poolLabel(pool: string): string {
  return RUNTIME_POOL_LABELS[pool] ?? pool
}

function poolDescription(pool: string): string {
  return RUNTIME_POOL_DESCRIPTION_LABELS[pool] ?? pool
}

function poolQueueCount(pool: string): number {
  return pools.value.find((item) => item.name === pool)?.queue_count ?? 0
}

function poolUtilization(pool: RuntimeWorkerPool): number {
  return Math.round(Math.max(0, Math.min(1, pool.utilization || 0)) * 100)
}

function formatLatency(ms: number): string {
  if (!ms || ms <= 0) return '—'
  if (ms < 1000) return `${ms} ms`
  const s = ms / 1000
  if (s < 60) return `${s.toFixed(1)} s`
  const m = Math.floor(s / 60)
  const rem = Math.round(s % 60)
  return `${m}m ${rem}s`
}

function queueState(row: QueueStat): { label: string; tone: string } {
  if (row.paused) {
    return { label: 'Paused', tone: 'paused' }
  }
  if (row.archived > 0) {
    return { label: 'Action needed', tone: 'danger' }
  }
  if (row.retry > 0) {
    return { label: 'Retrying', tone: 'attention' }
  }
  if (row.active > 0) {
    return { label: 'Working', tone: 'working' }
  }
  if (row.pending > 0 || row.scheduled > 0) {
    return { label: 'Waiting', tone: 'waiting' }
  }
  return { label: 'Idle', tone: 'idle' }
}

async function fetchRuntimeTasks(reset: boolean) {
  const queue = taskQueue.value?.name
  if (!queue) return
  if (!reset && (tasksLoadingMore.value || !tasksHasMore.value)) return

  const requestedState = taskState.value
  const requestID = ++tasksRequestID
  const cursor = reset ? '' : tasksCursor.value
  if (reset) {
    tasksCursor.value = ''
    tasksHasMore.value = false
    tasks.value = []
    tasksLoading.value = true
  } else {
    tasksLoadingMore.value = true
  }
  tasksError.value = ''
  try {
    const response = await getRuntimeTasks(queue, requestedState, cursor, TASK_PAGE_SIZE)
    if (requestID !== tasksRequestID || taskQueue.value?.name !== queue || taskState.value !== requestedState) return
    if (!response.available) {
      tasksError.value = 'Task details are unavailable in this deployment'
      return
    }
    tasks.value = reset ? response.tasks : mergeRuntimeTaskPage(tasks.value, response.tasks)
    tasksCursor.value = response.next_cursor || ''
    tasksHasMore.value = response.has_more && Boolean(response.next_cursor)
  } catch (err: any) {
    if (requestID !== tasksRequestID) return
    if (!reset && err?.code === 'runtime_task_cursor_expired') {
      tasksLoadingMore.value = false
      await fetchRuntimeTasks(true)
      return
    }
    tasksError.value = err?.message || 'Failed to load task details'
  } finally {
    if (requestID !== tasksRequestID) return
    if (reset) {
      tasksLoading.value = false
    } else {
      tasksLoadingMore.value = false
    }
    await nextTick()
    attachTasksScrollObserver()
  }
}

function detachTasksScrollObserver() {
  tasksScrollObserver?.disconnect()
  tasksScrollObserver = null
}

function attachTasksScrollObserver() {
  detachTasksScrollObserver()
  const sentinel = tasksSentinelRef.value
  if (!sentinel || !taskDrawerVisible.value || !tasksHasMore.value) return
  const root = sentinel.closest('.t-drawer__body') as HTMLElement | null
  if (!root) return
  tasksScrollObserver = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) {
        void loadMoreRuntimeTasks()
      }
    },
    { root, rootMargin: '96px 0px', threshold: 0 },
  )
  tasksScrollObserver.observe(sentinel)
}

function openRuntimeTasks(row: QueueStat, state: RuntimeTaskState) {
  taskQueue.value = row
  taskState.value = state
  taskDrawerVisible.value = true
  void fetchRuntimeTasks(true)
}

function selectTaskState(state: RuntimeTaskState) {
  if (taskState.value === state) return
  taskState.value = state
  void fetchRuntimeTasks(true)
}

function reloadRuntimeTasks() {
  return fetchRuntimeTasks(true)
}

function loadMoreRuntimeTasks() {
  if (tasksLoading.value || tasksLoadingMore.value || !tasksHasMore.value) return
  return fetchRuntimeTasks(false)
}

async function runTaskAction(task: RuntimeTask, action: RuntimeTaskAction) {
  const queue = taskQueue.value?.name
  if (!queue) return
  taskActionID.value = task.id
  taskAction.value = action
  try {
    await mutateRuntimeTask(queue, task.id, action)
    MessagePlugin.success((SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_ACTION_SUCCESS_LABELS[action] ?? ''))
    await Promise.all([reloadRuntimeTasks(), load(false)])
    taskQueue.value = queues.value.find((item) => item.name === queue) ?? taskQueue.value
  } catch (err: any) {
    MessagePlugin.error(err?.message || (SYSTEM_GLOBAL_SETTINGS_RUNTIME_TASKS_ACTION_ERROR_LABELS[action] ?? ''))
  } finally {
    taskActionID.value = ''
    taskAction.value = ''
  }
}

async function purgeArchivedTasks() {
  const queue = taskQueue.value?.name
  if (!queue || purging.value) return
  purging.value = true
  try {
    const { deleted } = await purgeArchivedRuntimeTasks(queue)
    MessagePlugin.success(`Cleared ${deleted} failed tasks`)
    await Promise.all([reloadRuntimeTasks(), load(false)])
    taskQueue.value = queues.value.find((item) => item.name === queue) ?? taskQueue.value
  } catch (err: any) {
    MessagePlugin.error(err?.message || 'Failed to clear failed tasks')
  } finally {
    purging.value = false
  }
}

async function load(showSpinner: boolean) {
  if (showSpinner) loading.value = true
  try {
    const resp = await getRuntimeQueues()
    available.value = resp.available
    pools.value = resp.pools || []
    queues.value = resp.queues || []
    if (taskQueue.value) {
      taskQueue.value = queues.value.find((item) => item.name === taskQueue.value?.name) ?? taskQueue.value
    }
    models.value = resp.models || []
    modelLimiterAvailable.value = Boolean(resp.model_limiter_available)
    updatedAt.value = new Date((resp.timestamp || Date.now() / 1000) * 1000)
      .toLocaleTimeString('en-US', { hour12: false })
    error.value = ''
    loadedOnce.value = true
  } catch (err: any) {
    error.value = err?.message || 'Failed to load queue stats'
  } finally {
    if (showSpinner) loading.value = false
  }
}

function reload() {
  load(true)
}

function startPolling() {
  stopPolling()
  if (!autoRefresh.value) return
  pollTimer = setInterval(() => {
    // Silent background refresh — no spinner so the table doesn't flash.
    if (!loading.value) load(false)
  }, POLL_INTERVAL_MS)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

watch(autoRefresh, (on) => {
  if (on) startPolling()
  else stopPolling()
})

watch(taskDrawerVisible, async (open) => {
  if (!open) {
    detachTasksScrollObserver()
    return
  }
  await nextTick()
  attachTasksScrollObserver()
}, { flush: 'post' })

watch(tasksHasMore, async () => {
  if (!taskDrawerVisible.value) return
  await nextTick()
  attachTasksScrollObserver()
})

onMounted(() => {
  load(true)
  startPolling()
})

onUnmounted(() => {
  stopPolling()
  detachTasksScrollObserver()
})
</script>

<style lang="less" scoped>
.runtime-queues {
  color: var(--td-text-color-primary);
}

.rq-models {
  margin-top: 40px;
  padding-top: 32px;
}

.rq-model-usage {
  display: grid;
  grid-template-columns: minmax(72px, 1fr) auto;
  align-items: center;
  gap: 10px;
  color: var(--td-text-color-secondary);
  font-variant-numeric: tabular-nums;
  font-size: var(--app-text-sm);
}

.rq-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;

  h2 {
    margin: 0 0 8px;
    color: var(--td-text-color-primary);
    font-size: var(--app-text-3xl);
    font-weight: 600;
    line-height: 1.3;
    letter-spacing: -0.01em;
  }

  .section-description {
    max-width: 560px;
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-base);
    line-height: 1.6;
    text-wrap: pretty;
  }
}

.rq-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.rq-refresh {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  transition: color var(--app-motion-base) cubic-bezier(0.16, 1, 0.3, 1), background var(--app-motion-base) cubic-bezier(0.16, 1, 0.3, 1);

  :deep(.t-icon) {
    font-size: var(--app-text-sm);
  }

  &:hover:not(:disabled) {
    color: var(--td-brand-color);
    background: var(--td-bg-color-secondarycontainer);
  }

  &:active:not(:disabled) {
    background: var(--td-bg-color-secondarycontainer);
  }

  &:disabled {
    cursor: default;
    opacity: 0.7;
  }
}

.rq-refresh-spin {
  animation: wk-spin 0.8s linear infinite;
}

.rq-auto-refresh {
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  white-space: nowrap;
  cursor: pointer;
}

.rq-live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--td-text-color-placeholder);
  transition: background-color var(--app-motion-base) ease, box-shadow var(--app-motion-base) ease;

  &--active {
    background: var(--td-success-color);
    box-shadow: 0 0 0 3px var(--td-success-color-1);
  }
}

.rq-loading {
  display: grid;
  gap: 22px;
  padding-top: 4px;
}

.rq-loading-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background: var(--td-component-stroke);

  :deep(.t-skeleton) {
    padding: 18px;
    background: var(--td-bg-color-container);
  }
}

.rq-state {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 14px;
  min-height: 112px;
  padding: 20px 22px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-secondarycontainer);
}

.rq-state-icon {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  border-radius: var(--app-radius-md);
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-container);
}

.rq-state-copy {
  display: flex;
  flex-direction: column;
  gap: 5px;

  strong {
    font-size: var(--app-text-base);
    font-weight: 600;
  }

  span {
    max-width: 560px;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-base);
    line-height: 1.55;
  }
}

.rq-state--error .rq-state-icon {
  color: var(--td-error-color);
  background: var(--td-error-color-1);
}

.rq-overview {
  display: flex;
  min-height: 64px;
  align-items: center;
  gap: 28px;
  margin-bottom: 30px;
  padding: 13px 16px;
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-secondarycontainer);
}

.rq-overview-title {
  display: inline-flex;
  min-width: 112px;
  align-items: center;
  gap: 9px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  font-weight: 600;
  white-space: nowrap;
}

.rq-overview-mark {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  border-radius: var(--app-radius-sm);
  color: var(--td-brand-color);
  background: var(--td-bg-color-container);
  font-size: var(--app-text-lg);
}

.rq-overview-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(64px, 1fr));
  align-items: stretch;
  gap: 24px;
  flex: 1;
}

.rq-metric {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
}

.rq-metric-value {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-3xl);
  font-weight: 600;
  line-height: 1.1;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.rq-metric-label {
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.35;
  white-space: nowrap;
}

.rq-metric--warning .rq-metric-value {
  color: var(--td-warning-color);
}

.rq-metric--danger .rq-metric-value {
  color: var(--td-error-color);
}

.rq-pools {
  margin-bottom: 30px;
}

.rq-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 6px;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-lg);
  font-weight: 600;
  line-height: 1.35;
  user-select: none;

  &::before {
    content: '';
    flex-shrink: 0;
    width: 3px;
    height: 15px;
    border-radius: 2px;
    background: var(--td-brand-color);
  }
}

.rq-pools-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 14px;

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    line-height: 1.55;
  }
}

.rq-pools-note {
  flex-shrink: 0;
  margin-top: 2px;
  padding: 4px 10px;
  border-radius: var(--app-radius-pill);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.4;
  white-space: nowrap;
  background: var(--td-bg-color-secondarycontainer);
}

.rq-pool-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.rq-pool-card {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 8px;
  padding: 16px 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-container);
}

.rq-pool-topline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.rq-pool-name {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  font-weight: 500;
  line-height: 1.35;
}

.rq-pool-value {
  color: var(--td-brand-color);
  font-size: 22px;
  font-weight: 600;
  line-height: 1;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.rq-pool-desc {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.55;
  text-wrap: pretty;
}

.rq-pool-meta {
  display: block;
  margin-top: 4px;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.4;
}

.rq-details {
  margin-top: 2px;
}

.rq-details-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 14px;

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-md);
    line-height: 1.55;
  }
}

.rq-updated-at {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  font-variant-numeric: tabular-nums;
}

.rq-failed-notice {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 14px;
  padding: 12px 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-container);
}

.rq-failed-notice__icon {
  display: grid;
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  place-items: center;
  border-radius: var(--app-radius-md);
  color: var(--td-error-color);
  font-size: var(--app-text-xl);
  background: color-mix(in srgb, var(--td-error-color) 10%, transparent);
}

.rq-failed-notice__text {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.rq-failed-notice__title {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
  line-height: 1.45;
  font-variant-numeric: tabular-nums;
}

.rq-failed-notice__desc {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.55;
}

.rq-failed-guide-desc {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.6;
}

.rq-task-state-filter {
  display: flex;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.rq-task-state-option {
  position: relative;
  display: inline-flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 0;
  padding: 10px 6px;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: var(--app-text-md);
  line-height: 1.2;
  white-space: nowrap;
  transition: color var(--app-motion-fast) ease;

  &:hover:not(.is-active) {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: -2px;
  }

  &.is-active {
    color: var(--td-brand-color);
    font-weight: 600;

    &::after {
      content: '';
      position: absolute;
      left: 8px;
      right: 8px;
      bottom: -1px;
      height: 2px;
      border-radius: 2px 2px 0 0;
      background: var(--td-brand-color);
    }
  }
}

.rq-task-state-option__label {
  overflow: hidden;
  text-overflow: ellipsis;
}

.rq-task-state-option__count {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-xs);
  font-weight: 500;
  line-height: 1;
  font-variant-numeric: tabular-nums;

  &.has-value {
    color: var(--td-text-color-secondary);
  }

  .rq-task-state-option.is-active & {
    color: var(--td-brand-color);
    font-weight: 600;
  }
}

.rq-failed-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 4px;

  .setting-drawer__section-title {
    margin-bottom: 0;
  }
}

.rq-failed-section-actions {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.rq-failed-loading {
  display: flex;
  min-height: 180px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
}

.rq-failed-error-state {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.55;
  background: var(--td-bg-color-container);
}

.rq-empty {
  display: flex;
  min-height: 180px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 10px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-md);
}

.rq-queue-cell {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.rq-queue-name {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: var(--app-text-base);
  font-weight: 500;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rq-queue-meta {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.45;
}

.rq-number,
.rq-latency {
  color: var(--td-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

.rq-number--active {
  color: var(--td-brand-color);
  font-weight: 600;
}

.rq-number--warning {
  color: var(--td-warning-color);
  font-weight: 600;
}

.rq-number--danger {
  color: var(--td-error-color);
  font-weight: 600;
}

.rq-task-count {
  min-width: 0;
  height: 28px;
  padding: 0 2px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;

  :deep(.t-button__text) {
    display: inline-flex;
    align-items: center;
    gap: 2px;
  }
}

.rq-task-count--active {
  color: var(--td-brand-color);
}

.rq-task-count--completed {
  color: var(--td-success-color);
}

.rq-scheduled-count {
  min-width: 0;
  height: 18px;
  padding: 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-xs);
}

.rq-backlog {
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 1px;

  small {
    color: var(--td-text-color-placeholder);
    font-size: var(--app-text-xs);
    white-space: nowrap;
  }
}

.rq-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  white-space: nowrap;

  i {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--td-text-color-placeholder);
  }

  &--working i {
    background: var(--td-brand-color);
  }

  &--attention i,
  &--paused i {
    background: var(--td-warning-color);
  }

  &--attention,
  &--paused {
    color: var(--td-warning-color);
  }

  &--danger {
    color: var(--td-error-color);
  }

  &--danger i {
    background: var(--td-error-color);
  }
}

.data-table-shell.rq-table-shell {
  overflow-x: auto;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background-color: var(--td-bg-color-container);

  &:deep(thead th) {
    height: 40px;
    color: var(--td-text-color-secondary);
    font-size: var(--app-text-sm);
    font-weight: 500;
    letter-spacing: 0.01em;
    white-space: nowrap;
    background-color: var(--td-bg-color-secondarycontainer) !important;
  }

  &:deep(.t-table td) {
    height: 56px;
    padding-top: 10px;
    padding-bottom: 10px;
    font-size: var(--app-text-base);
    font-variant-numeric: tabular-nums;
  }

  /* Metric columns: center short numbers under multi-char headers. */
  &:deep(.t-table th.t-align-center),
  &:deep(.t-table td.t-align-center) {
    text-align: center;
  }

  &:deep(td.t-align-center .rq-task-count) {
    margin-inline: auto;
  }

  &:deep(.t-table__body tr:last-child td) {
    border-bottom: 0;
  }
}

.rq-footnote {
  margin: 12px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.55;
}

.rq-failed-list-panel {
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  background: var(--td-bg-color-container);
}

.rq-failed-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 12px 12px 16px;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-of-type {
    border-bottom: 0;
  }
}

.rq-failed-row-content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 4px;
}

.rq-failed-row-summary {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.45;
}

.rq-failed-row-type {
  color: var(--td-text-color-primary);
  font-size: var(--app-text-md);
  font-weight: 600;
}

.rq-task-state-pill {
  padding: 1px 6px;
  border-radius: var(--app-radius-pill);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  background: var(--td-bg-color-secondarycontainer);

  &--active {
    color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }

  &--retry,
  &--scheduled {
    color: var(--td-warning-color);
    background: var(--td-warning-color-1);
  }

  &--archived {
    color: var(--td-error-color);
    background: var(--td-error-color-1);
  }

  &--completed {
    color: var(--td-success-color);
    background: var(--td-success-color-1);
  }
}

.rq-failed-row-sep {
  color: var(--td-text-color-placeholder);
}

.rq-failed-row-stat,
.rq-failed-row-summary time {
  font-variant-numeric: tabular-nums;
}

.rq-failed-row-refs {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 2px 0 0;
}

.rq-failed-ref {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 8px;
  align-items: baseline;
  margin: 0;

  dt {
    margin: 0;
    color: var(--td-text-color-placeholder);
    font-size: var(--app-text-sm);
    line-height: 1.45;
    white-space: nowrap;
  }

  dd {
    margin: 0;
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: var(--app-text-xs);
    line-height: 1.45;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.rq-failed-row-unknown {
  margin: 2px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.45;
}

.rq-failed-row-error {
  margin: 8px 0 0;
  padding: 8px 10px;
  border-radius: var(--app-radius-sm);
  color: var(--td-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: var(--app-text-xs);
  line-height: 1.55;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  background: var(--td-bg-color-secondarycontainer);
}

.rq-failed-row-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 0;
  margin-top: -2px;
}

.rq-failed-icon-btn {
  width: 28px;
  height: 28px;
}

.rq-failed-load-sentinel {
  height: 1px;
}

.rq-failed-list-footer {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 16px 14px;
  border-top: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
}

.rq-failed-list-status {
  color: var(--td-text-color-placeholder);
  font-size: var(--app-text-sm);
  line-height: 1.5;
  text-align: center;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 860px) {
  .rq-header,
  .rq-details-header,
  .rq-pools-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .rq-loading-metrics {
    grid-template-columns: repeat(2, 1fr);
  }

  .rq-actions {
    width: 100%;
    justify-content: space-between;
  }

  .rq-overview {
    align-items: flex-start;
    flex-wrap: wrap;
  }

  .rq-overview-metrics {
    width: 100%;
    grid-template-columns: repeat(4, minmax(64px, 1fr));
  }

  .rq-pool-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 620px) {
  .rq-task-state-filter {
    overflow-x: auto;
    scrollbar-width: none;

    &::-webkit-scrollbar {
      display: none;
    }
  }

  .rq-task-state-option {
    flex: 0 0 auto;
    min-width: 72px;
    padding: 10px 10px;
  }

  .rq-loading-metrics {
    grid-template-columns: 1fr;
  }

  .rq-overview-metrics {
    width: 100%;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }

  .rq-pool-grid {
    grid-template-columns: 1fr;
  }

  .rq-state {
    grid-template-columns: auto minmax(0, 1fr);

    .t-button {
      grid-column: 2;
      justify-self: start;
    }
  }

  .rq-failed-row {
    padding: 12px;
  }

  .rq-failed-row-summary {
    gap: 4px;
  }
}
</style>
