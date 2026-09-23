import assert from 'node:assert/strict'
import test from 'node:test'

import {
  getAttachmentParsingSummaryHtml,
  resolveAttachmentParsingCounts,
} from './attachmentParsingDisplay.ts'


test('resolveAttachmentParsingCounts prefers structured tool_data', () => {
  assert.deepEqual(
    resolveAttachmentParsingCounts({
      tool_data: { parsed_count: 2, skipped_count: 1 },
    }),
    { parsed: 2, skipped: 1 },
  )
})

test('resolveAttachmentParsingCounts falls back to the tool output text', () => {
  assert.deepEqual(
    resolveAttachmentParsingCounts({
      output: 'Parsed 3 attachment(s), skipped 1 that were not ready',
    }),
    { parsed: 3, skipped: 1 },
  )
})

test('getAttachmentParsingSummaryHtml renders parsed count', () => {
  const html = getAttachmentParsingSummaryHtml({
    success: true,
    tool_data: { parsed_count: 1, skipped_count: 0 },
  })
  assert.equal(html, 'Parsed <strong>1</strong> attachment(s)')
})

test('getAttachmentParsingSummaryHtml renders skipped count', () => {
  const html = getAttachmentParsingSummaryHtml({
    success: true,
    tool_data: { parsed_count: 2, skipped_count: 1 },
  })
  assert.equal(html, 'Parsed <strong>2</strong> attachment(s), <strong>1</strong> skipped (still processing)')
})
