import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const manager = readFileSync(new URL('./FAQEntryManager.vue', import.meta.url), 'utf8')
const batchBar = readFileSync(new URL('./FAQBatchBar.vue', import.meta.url), 'utf8')

test('FAQ bulk actions are wired through component events rather than a global event bus', () => {
  assert.match(manager, /import FAQBatchBar from '\.\/FAQBatchBar\.vue'/)
  assert.match(manager, /<FAQBatchBar[\s\S]*?@batch-tag="openBatchTagDialog"/)
  assert.match(manager, /@enable="handleBatchStatusChange\(true\)"/)
  assert.match(manager, /@disable="handleBatchStatusChange\(false\)"/)
  assert.match(manager, /@delete="handleBatchDelete"/)
  assert.doesNotMatch(manager, /faqMenuAction/)
  assert.doesNotMatch(manager, /faqSelectionChanged/)
})

test('the bulk action bar offers only the actions the selection and permissions allow', () => {
  assert.match(batchBar, /v-if="count > 0 && \(canEdit \|\| canManage\)"/)
  assert.match(batchBar, /v-if="canEdit && disabledCount > 0"/)
  assert.match(batchBar, /v-if="canEdit && enabledCount > 0"/)
  assert.match(batchBar, /<t-popconfirm v-if="canManage"/)
  assert.match(manager, /const canSelectEntries = computed\(\(\) => canEdit\.value \|\| canManage\.value\)/)
  assert.match(manager, /if \(!canSelectEntries\.value \|\| batchActionLoading\.value\) return/)
})

test('destructive actions need confirmation and every bulk request guards against double submit', () => {
  assert.match(batchBar, /Delete the selected \$\{count\} FAQ entries\?/)
  assert.match(batchBar, /@confirm="emit\('delete'\)"/)
  assert.match(batchBar, /const actionLoading = computed/)
  assert.match(manager, /const batchDeleteLoading = ref\(false\)/)
  assert.match(manager, /const batchTagLoading = ref\(false\)/)
  assert.match(manager, /const batchStatusAction = ref<'enable' \| 'disable' \| null>\(null\)/)
  assert.match(manager, /finally \{\s*batchDeleteLoading\.value = false\s*\}/)
  assert.match(manager, /finally \{\s*batchTagLoading\.value = false\s*\}/)
  assert.match(manager, /finally \{\s*batchStatusAction\.value = null\s*\}/)
})

test('the batch bar labels every bulk FAQ action', () => {
  for (const label of ['Enable selected', 'Disable selected', 'Delete selected']) {
    assert.ok(batchBar.includes(`'${label}'`), `the batch bar must offer ${label}`)
  }
  assert.match(batchBar, /Delete the selected \$\{count\} FAQ entries\?/)
  assert.ok(batchBar.includes("'Confirm Delete'"))
})
