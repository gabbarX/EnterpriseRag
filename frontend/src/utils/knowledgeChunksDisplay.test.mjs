import assert from 'node:assert/strict'
import test from 'node:test'

import { getKnowledgeChunksSummaryHtml } from './knowledgeChunksDisplay.ts'


test('getKnowledgeChunksSummaryHtml joins chunk range and page', () => {
  const html = getKnowledgeChunksSummaryHtml({
    display_type: 'knowledge_chunks_list',
    fetched_chunks: 20,
    total_chunks: 282,
    page: 3,
    page_size: 20,
  })
  assert.match(html, /Loaded <strong>20<\/strong> \/ <strong>282<\/strong> chunks/)
  assert.match(html, /Page 3, 20 per page/)
})

test('getKnowledgeChunksSummaryHtml reports in-document search results, not a chunk range', () => {
  const none = getKnowledgeChunksSummaryHtml({
    display_type: 'knowledge_chunks_list',
    fetched_chunks: 0,
    total_chunks: 708,
    query: 'sunflower <f>',
    match_count: 0,
  })
  assert.equal(none, 'No matches for "sunflower &lt;f&gt;" in this document')

  const some = getKnowledgeChunksSummaryHtml({
    display_type: 'knowledge_chunks_list',
    fetched_chunks: 9,
    total_chunks: 708,
    query: 'sunflower',
    match_count: 20,
    truncated: true,
  })
  assert.equal(some, '<strong>20+</strong> matches for "sunflower" in this document')
})

test('getKnowledgeChunksSummaryHtml shows the chunk range for offset paging', () => {
  const html = getKnowledgeChunksSummaryHtml({
    display_type: 'knowledge_chunks_list',
    fetched_chunks: 37,
    total_chunks: 708,
    offset: 100,
    page: 2,
    page_size: 100,
  })
  assert.equal(html, 'Loaded <strong>37</strong> / <strong>708</strong> chunks · Chunks 101–137')
})
