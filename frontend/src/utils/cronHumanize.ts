// cronHumanize maps known cron schedule presets to a human-readable label.
// Falls back to the raw cron expression for unknown patterns.

const CRON_PRESET_MAP: Record<string, string> = {
  '0 */30 * * * *': 'Every 30 min',
  '0 0 * * * *': 'Hourly',
  '0 0 */6 * * *': 'Every 6 hours',
  '0 0 */12 * * *': 'Every 12 hours',
  '0 0 2 * * *': 'Daily',
}

/** Convert a cron expression to a human-readable string. */
export function humanizeCron(cron: string): string {
  const key = CRON_PRESET_MAP[cron]
  if (key) return key
  return cron || '--'
}

/**
 * Format a timestamp as relative time (e.g. "3h ago", "2d ago").
 * Falls back to a date string for timestamps older than 30 days.
 */
export function relativeTime(ts: string | null): string {
  if (!ts) return 'Never synced'
  const then = new Date(ts).getTime()
  if (isNaN(then)) return 'Never synced'
  const now = Date.now()
  const diffMs = now - then

  // Future time or just happened (within 60s)
  if (diffMs < 60000) return 'Just now'

  const minutes = Math.floor(diffMs / 60000)
  if (minutes < 60) return `${minutes}m ago`

  const hours = Math.floor(diffMs / 3600000)
  if (hours < 24) return `${hours}h ago`

  const days = Math.floor(diffMs / 86400000)
  if (days < 30) return `${days}d ago`

  return new Date(ts).toLocaleDateString()
}
