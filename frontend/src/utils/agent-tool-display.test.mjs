import assert from 'node:assert/strict'
import test from 'node:test'
import { getAgentToolIconName } from './agent-tool-icons.ts'
import {
  getKnowledgeSearchSummaryHtml,
  getQueryText,
  getRagPipelineStepTitle,
  getWikiPageText,
} from './agent-tool-display.ts'


test('getAgentToolIconName maps rag pipeline tools', () => {
  assert.equal(getAgentToolIconName('query_understand'), 'ai-search')
  assert.equal(getAgentToolIconName('knowledge_search'), 'data-search')
})

test('getAgentToolIconName maps the consolidated knowledge tools and their retired names alike', () => {
  assert.equal(getAgentToolIconName('search_knowledge'), 'data-search')
  assert.equal(getAgentToolIconName('search_knowledge', 'web'), 'internet')
  assert.equal(getAgentToolIconName('read_document'), 'file-search')
  assert.equal(getAgentToolIconName('list_documents'), 'file-search')
  assert.equal(getAgentToolIconName('list_knowledge_chunks'), 'file-search')
  assert.equal(getAgentToolIconName('get_document_info'), 'file-search')
})

test('getAgentToolIconName maps sandbox shell tools to the terminal icon', () => {
  assert.equal(getAgentToolIconName('shell_exec'), 'terminal')
})

test('getAgentToolIconName maps skill and sandbox file tools', () => {
  assert.equal(getAgentToolIconName('read_file'), 'file')
  assert.equal(getAgentToolIconName('read_skill'), 'file')
  assert.equal(getAgentToolIconName('list_sandbox_files'), 'folder')
  assert.equal(getAgentToolIconName('read_sandbox_file'), 'file')
  assert.equal(getAgentToolIconName('execute_skill_script'), 'code')
})

test('getAgentToolIconName maps Wiki tools to semantic search and reading icons', () => {
  assert.equal(getAgentToolIconName('wiki_search'), 'search')
  assert.equal(getAgentToolIconName('wiki_read_page'), 'file-search')
  assert.equal(getAgentToolIconName('wiki_read_source_doc'), 'file-search')
})

test('getQueryText joins unique query strings', () => {
  assert.equal(getQueryText({ query: 'foo', queries: ['foo', 'bar'] }), 'foo, bar')
})

test('getQueryText parses JSON-encoded queries string', () => {
  assert.equal(
    getQueryText({
      queries: '["Riverside swim club overview", "Riverside swim training centre", "Riverside swim team"]',
    }),
    'Riverside swim club overview, Riverside swim training centre, Riverside swim team',
  )
})

test('getWikiPageText supports persisted slugs arrays', () => {
  assert.equal(
    getWikiPageText({ slugs: ['entity/knowledge-assistant', 'concept/api-management'] }),
    'entity/knowledge-assistant, concept/api-management',
  )
  assert.equal(getWikiPageText('{"slug":"index"}'), 'index')
})

test('getKnowledgeSearchSummaryHtml includes file count when present', () => {
  const html = getKnowledgeSearchSummaryHtml({
    results: [{}, {}],
    kb_counts: { a: 1, b: 2 },
  })
  assert.match(html, /Found <strong>2<\/strong> result\(s\) from <strong>2<\/strong> file\(s\)/)
})

// Candidates that all fell below the relevance threshold never reached the
// answer, so the row must not read like a plain empty search: the difference
// points at the threshold rather than at the knowledge base.
test('getKnowledgeSearchSummaryHtml distinguishes filtered candidates from an empty search', () => {
  assert.match(
    getKnowledgeSearchSummaryHtml({ count: 0, candidate_count: 10 }),
    /Matched <strong>10<\/strong> candidate\(s\), none relevant enough to use/,
  )
  assert.equal(
    getKnowledgeSearchSummaryHtml({ count: 0, candidate_count: 0 }),
    'No matching content found',
  )
})

test('getRagPipelineStepTitle uses query-aware search labels', () => {
  const title = getRagPipelineStepTitle({
    tool_name: 'knowledge_search',
    pending: true,
    arguments: { query: 'onboarding guide' },
  })
  assert.equal(title, 'Searching knowledge base: "onboarding guide"')
})

test('getRagPipelineStepTitle uses web labels when search_source is web', () => {
  const title = getRagPipelineStepTitle({
    tool_name: 'knowledge_search',
    pending: false,
    success: true,
    arguments: { search_source: 'web' },
  })
  assert.equal(title, 'Web search')
})

test('getRagPipelineStepTitle uses attachment parsing labels', () => {
  assert.equal(
    getRagPipelineStepTitle({ tool_name: 'attachment_parsing', pending: true }),
    'Parsing attachments...',
  )
  assert.equal(
    getRagPipelineStepTitle({ tool_name: 'attachment_parsing', pending: false, success: true }),
    'Attachments parsed',
  )
})
