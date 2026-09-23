import assert from 'node:assert/strict'
import test from 'node:test'
import { rewindSkipMessage } from './rewindNotice'

test('NO_SANDBOX maps to skipNoSandbox', () => {
  assert.equal(rewindSkipMessage('NO_SANDBOX'), 'Conversation rewound; workspace was left unchanged (no sandbox is bound)')
})

test('NO_CHECKPOINT maps to skipNoCheckpoint', () => {
  assert.equal(rewindSkipMessage('NO_CHECKPOINT'), 'Conversation rewound; workspace was left unchanged (no checkpoint to restore)')
})

// A replaced sandbox fail-closes with a 409 rather than truncating, so it
// never arrives here as a skip reason.
test('unknown code falls back to generic skipped copy', () => {
  assert.equal(rewindSkipMessage('WEIRD'), 'Conversation rewound; workspace was left unchanged')
})

test('empty reason returns empty so the UI can skip the workspace notice', () => {
  assert.equal(rewindSkipMessage(''), '')
})
