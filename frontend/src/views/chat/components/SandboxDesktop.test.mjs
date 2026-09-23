import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const desktop = readFileSync(new URL('./SandboxDesktop.vue', import.meta.url), 'utf8')

test('panel open looks up a running sandbox and only provisions on an explicit click', () => {
  assert.match(desktop, /connectLookup/)
  assert.match(desktop, /onMounted\(\(\) => \{\s*connectLookup\(\)/)
  assert.match(desktop, /provision:\s*false/)
  assert.match(desktop, /provision:\s*true/)
  assert.match(desktop, /status === 'paused'/)
  assert.match(desktop, /sandbox is paused\. Connecting the desktop resumes it\./)
  assert.match(desktop, /Create and connect/)
})
