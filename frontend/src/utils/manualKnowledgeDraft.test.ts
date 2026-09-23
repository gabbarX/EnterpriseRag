import assert from 'node:assert/strict'
import test from 'node:test'
import { buildManualDraft, deriveManualTitle } from './manualKnowledgeDraft'

const labels = { emptyAnswer: '(No content)', sourcesHeading: 'Sources' }

test('a chat answer loses its <kb/> tags and gains a source list', () => {
  const answer = [
    '## Leave policy basics',
    '',
    '> **"Earned leave accrues at 1.5 days a month."** <kb doc="Interview.md" chunk_id="230a27cb-45c8-4ba9-9ba1-babea70bcd95" kb_id="441ee604-a04d-4c9a-a4bd-1c4e198e6226" />',
    '',
    'She then walked through the Pune office as an example <kb doc="Interview.md" chunk_id="ef18491c-34f0-4563-bb4a-062cacbdfd95" kb_id="441ee604-a04d-4c9a-a4bd-1c4e198e6226" />',
  ].join('\n')

  const draft = buildManualDraft(answer, labels)

  assert.ok(!draft.includes('<kb'), draft)
  assert.ok(!draft.includes('chunk_id'), draft)
  assert.ok(draft.includes('> **"Earned leave accrues at 1.5 days a month."**'))
  assert.ok(draft.endsWith('**Sources**\n\n- Interview.md\n'), draft)
})

test('web citations survive as real Markdown links', () => {
  const draft = buildManualDraft('See <web url="https://ziglang.org" title="Zig" /> for details', labels)
  assert.equal(draft, 'See [Zig](https://ziglang.org) for details')
})

test('a tag alone on its line leaves no blank line behind', () => {
  const draft = buildManualDraft('First paragraph\n<kb doc="A.md" chunk_id="1" />\nSecond paragraph', labels)
  assert.equal(draft, 'First paragraph\nSecond paragraph\n\n---\n\n**Sources**\n\n- A.md\n')
})

test('code blocks keep their own spacing while citations are stripped', () => {
  const answer = '```js\nfn( 1 ) ;\n\n\nconst x = 2\n```\n\nNotes <kb doc="A.md" chunk_id="1" />'
  const draft = buildManualDraft(answer, labels)
  assert.ok(draft.startsWith('```js\nfn( 1 ) ;\n\n\nconst x = 2\n```'), draft)
  assert.ok(draft.includes('\n\nNotes\n'), draft)
})

test('an answer without citations is passed through untouched', () => {
  assert.equal(buildManualDraft('A plain text answer', labels), 'A plain text answer')
  assert.equal(buildManualDraft('   ', labels), '(No content)')
})

test('a long question is cut on a clause boundary, with no trailing ellipsis', () => {
  const title = deriveManualTitle(
    'कर्मचारी अवकाश नीति में अर्जित छुट्टी कैसे जुड़ती है, और क्या वह अगले वर्ष तक चलती है?',
    'Conversation excerpt',
  )
  assert.equal(title, 'कर्मचारी अवकाश नीति में अर्जित')
})

test('a short question keeps its wording, minus the question mark', () => {
  assert.equal(deriveManualTitle('Zig पर ध्यान क्यों दें?', 'Conversation excerpt'), 'Zig पर ध्यान क्यों दें')
  assert.equal(deriveManualTitle('  ', 'Conversation excerpt'), 'Conversation excerpt')
})

test('a long question with no boundary is cut at the limit', () => {
  const title = deriveManualTitle('क'.repeat(60), 'Conversation excerpt')
  assert.equal(title, 'क'.repeat(32))
})

test('an English question is cut on a word boundary', () => {
  const title = deriveManualTitle('How does a binary view of technology hurt diversity?', 'Conversation excerpt')
  assert.equal(title, 'How does a binary view of')
})
