/**
 * Normalizes the knowledge base scope returned by the API into an array the
 * front-end forms can use safely.
 *
 * @param ids Knowledge base IDs of the API key; a fully authorized key may come
 * back from the server as null.
 * @returns A fresh array of knowledge base IDs; null or undefined yields an
 * empty array, meaning all knowledge bases.
 */
export function normalizeAPIKeyKnowledgeBaseIDs(
  ids: readonly string[] | null | undefined,
): string[] {
  return ids ? [...ids] : []
}
