export type RetrievalSearchSource = 'knowledge' | 'web' | 'mixed'

function collectQueryStrings(value: unknown): string[] {
  if (value == null) return []

  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) return []
    if (trimmed.startsWith('[')) {
      try {
        const parsed = JSON.parse(trimmed)
        if (Array.isArray(parsed)) {
          return parsed.filter((q): q is string => typeof q === 'string' && Boolean(q.trim()))
        }
      } catch {
        // fall through to treat as a single query string
      }
    }
    return [trimmed]
  }

  if (Array.isArray(value)) {
    return value.filter((q): q is string => typeof q === 'string' && Boolean(q.trim()))
  }

  return []
}

export function getQueryText(args: unknown): string {
  if (!args) return ''

  let parsedArgs = args
  if (typeof parsedArgs === 'string') {
    try {
      parsedArgs = JSON.parse(parsedArgs)
    } catch {
      return ''
    }
  }

  if (!parsedArgs || typeof parsedArgs !== 'object') return ''

  const queries: string[] = []
  const record = parsedArgs as Record<string, unknown>

  queries.push(...collectQueryStrings(record.query))
  queries.push(...collectQueryStrings(record.queries))

  return Array.from(new Set(queries)).join(', ')
}

export function getWikiPageText(args: unknown): string {
  if (!args) return ''

  let parsedArgs = args
  if (typeof parsedArgs === 'string') {
    try {
      parsedArgs = JSON.parse(parsedArgs)
    } catch {
      return ''
    }
  }

  if (!parsedArgs || typeof parsedArgs !== 'object') return ''

  const record = parsedArgs as Record<string, unknown>
  const slugs = [
    ...collectQueryStrings(record.slug),
    ...collectQueryStrings(record.slugs),
  ]
  return Array.from(new Set(slugs)).join(', ')
}

export function getRetrievalSearchSource(
  args: unknown,
  toolData?: Record<string, unknown> | null,
): RetrievalSearchSource {
  const fromArgs =
    args && typeof args === 'object'
      ? String((args as Record<string, unknown>).search_source || '')
      : ''
  const fromData = toolData ? String(toolData.search_source || '') : ''
  const source = (fromData || fromArgs).trim()
  if (source === 'web' || source === 'mixed') {
    return source
  }
  return 'knowledge'
}

function getRetrievalStatusKeys(source: RetrievalSearchSource, failed: boolean) {
  if (source === 'web') {
    return failed
      ? { pending: 'Searching the web...', pendingWithQuery: 'Searching the web: "{query}"', done: 'Web search', doneFailed: 'Web search failed' }
      : { pending: 'Searching the web...', pendingWithQuery: 'Searching the web: "{query}"', done: 'Web search', doneFailed: 'Web search failed' }
  }
  if (source === 'mixed') {
    return {
      pending: 'Searching knowledge base and web...',
      pendingWithQuery: 'Searching knowledge base and web: "{query}"',
      done: 'Searched knowledge base and web',
      doneFailed: 'Search failed',
    }
  }
  return {
    pending: 'Searching knowledge base...',
    pendingWithQuery: 'Searching knowledge base: "{query}"',
    done: 'Searching knowledge base',
    doneFailed: 'Knowledge base search failed',
  }
}

export function getKnowledgeSearchSummaryHtml(
  toolData: Record<string, unknown> | null | undefined,
): string {
  if (!toolData) return ''

  const results = toolData.results
  const count = (Array.isArray(results) ? results.length : 0) || Number(toolData.count) || 0
  if (count === 0) {
    // Retrieval found candidates but none cleared the relevance threshold, so the
    // turn answered from the fallback with nothing retrieved in its context. Say
    // that rather than "no matching content": the difference is what tells you to
    // look at the threshold instead of the knowledge base.
    const candidateCount = Number(toolData.candidate_count) || 0
    if (candidateCount > 0) {
      return `Matched ${`<strong>${candidateCount}</strong>`} candidate(s), none relevant enough to use`
    }
    return 'No matching content found'
  }

  const searchSource = getRetrievalSearchSource(null, toolData)
  const webCount = Number(toolData.web_count) || 0
  const docCount = Number(toolData.doc_count) || 0

  if (searchSource === 'web' || (webCount > 0 && docCount === 0)) {
    return `Found ${`<strong>${count}</strong>`} web result(s)`
  }

  const kbCounts = toolData.kb_counts
  const kbCount = kbCounts && typeof kbCounts === 'object' ? Object.keys(kbCounts).length : 0
  if (kbCount > 0) {
    return `Found ${`<strong>${count}</strong>`} result(s) from ${`<strong>${kbCount}</strong>`} file(s)`
  }

  if (searchSource === 'mixed' && docCount > 0 && webCount > 0) {
    return `Found ${`<strong>${count}</strong>`} result(s) (${`<strong>${docCount}</strong>`} documents, ${`<strong>${webCount}</strong>`} web results)`
  }

  return `Found ${`<strong>${count}</strong>`} result(s)`
}

type RagPipelineEvent = {
  tool_name?: string
  pending?: boolean
  success?: boolean
  arguments?: unknown
  tool_data?: Record<string, unknown> | null
}

export function getRagPipelineStepTitle(event: RagPipelineEvent): string {
  const toolName = String(event.tool_name || '')
  const pending = event.pending === true
  const query =
    getQueryText(event.arguments) ||
    getQueryText(event.tool_data)

  if (toolName === 'query_understand') {
    return pending
      ? 'Understanding query...'
      : 'Query understood'
  }

  if (toolName === 'knowledge_search' || toolName === 'search_knowledge') {
    const searchSource = getRetrievalSearchSource(event.arguments, event.tool_data)
    const labels = getRetrievalStatusKeys(searchSource, event.success === false)
    if (pending) {
      return query
        ? labels.pendingWithQuery.replace('{query}', query)
        : labels.pending
    }

    const baseTitle = event.success === false ? labels.doneFailed : labels.done
    return query ? `${baseTitle}: "${query}"` : baseTitle
  }

  if (toolName === 'attachment_parsing') {
    if (pending) return 'Parsing attachments...'
    return event.success === false
      ? 'Attachment parsing failed'
      : 'Attachments parsed'
  }

  if (toolName === 'image_analysis') {
    if (pending) return 'Viewing image content...'
    return event.success === false
      ? 'Image viewing failed'
      : 'Image content viewed'
  }

  return ''
}
