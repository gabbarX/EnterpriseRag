import type { Ref } from 'vue'
import { getChunkByIdOnly } from '@/api/knowledge-base'
import { getEmbedChunkById } from '@/api/embed'
import type { CitationKnowledgeRef } from '@/utils/citationMarkdown'
import { useCitationPopover } from './useCitationPopover'

export { clearCitationChunkCache } from '@/utils/citationChunkCache'
export type { CitationFloatState } from './useCitationPopover'

export type ChatCitationPopoverOptions = {
  getKnowledgeReferences?: () => CitationKnowledgeRef[] | null | undefined
  embedChannelId?: () => string | undefined
  embedToken?: () => string | undefined
  sessionId?: () => string | undefined
}

export function useChatCitationPopover(rootRef: Ref<HTMLElement | null>, options?: ChatCitationPopoverOptions) {
  return useCitationPopover(rootRef, {
    mode: 'chat',
    getKnowledgeReferences: options?.getKnowledgeReferences,
    getCacheScope: () => {
      const channel = options?.embedChannelId?.()
      const token = options?.embedToken?.()
      return channel && token ? `embed:${channel}:${token}` : options?.sessionId?.() || 'default'
    },
    fetchChunk: (chunkId) => {
      const channel = options?.embedChannelId?.()
      const token = options?.embedToken?.()
      return channel && token ? getEmbedChunkById(channel, token, chunkId) : getChunkByIdOnly(chunkId)
    },
    notFoundError: () => 'Content not found',
    loadError: () => 'Failed to load',
  })
}
