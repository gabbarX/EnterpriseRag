import type { KnowledgeChunksListData } from '@/types/tool-results'

export function getKnowledgeChunksSummaryHtml(
  toolData: KnowledgeChunksListData | null | undefined,
): string {
  if (!toolData || toolData.fetched_chunks === undefined) {
    return ''
  }

  const query = toolData.query?.trim()
  if (query) {
    const count = Number(toolData.match_count ?? 0)
    const params = {
      query: escapeHtml(query),
      count: `<strong>${count}${toolData.truncated ? '+' : ''}</strong>`,
    }
    return count > 0
      ? `${params.count} matches for "${params.query}" in this document`
      : `No matches for "${params.query}" in this document`
  }

  const parts: string[] = [
    `Loaded ${`<strong>${toolData.fetched_chunks ?? 0}</strong>`} / ${`<strong>${toolData.total_chunks ?? '?'}</strong>`} chunks`,
  ]

  const total = Number(toolData.total_chunks ?? 0)
  const fetched = Number(toolData.fetched_chunks ?? 0)
  // read_document pages by offset and may return fewer than page_size chunks
  // to stay within the output budget, so a page number would be misleading.
  if (toolData.offset !== undefined) {
    if (fetched > 0 && fetched < total) {
      const from = Number(toolData.offset) + 1
      parts.push(
        `Chunks ${from}–${from + fetched - 1}`,
      )
    }
    return parts.join(' · ')
  }

  const pageSize = Number(toolData.page_size ?? 0)
  if (total > pageSize && pageSize > 0) {
    parts.push(
      `Page ${toolData.page ?? 1}, ${pageSize} per page`,
    )
  }

  return parts.join(' · ')
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
