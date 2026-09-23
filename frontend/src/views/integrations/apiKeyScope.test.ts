import assert from 'node:assert/strict'
import test from 'node:test'

import { normalizeAPIKeyKnowledgeBaseIDs } from './apiKeyScope.ts'

/**
 * A fully authorised key has a null/undefined scope; it must normalise to an empty array so the
 * list render does not blank out when it reads `length`. The empty array means "every knowledge base".
 */
test('normalizes missing API key knowledge base scope to an empty array', () => {
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs(null), [])
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs(undefined), [])
})

/**
 * A scoped key's knowledge base IDs are copied in full into a NEW array, so the edit form cannot
 * write back through it and corrupt the list data. Order is preserved.
 */
test('copies configured API key knowledge base scope', () => {
  const ids = ['kb-1', 'kb-2']
  const normalized = normalizeAPIKeyKnowledgeBaseIDs(ids)

  assert.deepEqual(normalized, ids)
  assert.notEqual(normalized, ids)
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs([]), [])
})
