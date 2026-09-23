/**
 * Tool Icons Utility
 * Maps tool names and match types to icons for better UI display
 */

const t = (key: string) => key

// Tool name to icon mapping
export const toolIcons: Record<string, string> = {
    multi_kb_search: '🔍',
    search_knowledge: '📚',
    read_document: '🧩',
    list_documents: 'ℹ️',
    get_chunk_detail: '📄',
    list_knowledge_bases: '📂',
    query_knowledge_graph: '🕸️',
    think: '💭',
    todo_write: '📋',
    // Retired names (still present in stored chat history)
    knowledge_search: '📚',
    grep_chunks: '🔎',
    list_knowledge_chunks: '🧩',
    get_document_info: 'ℹ️',
};

// Match type internal keys for icon mapping
const matchTypeIconKeys: Record<string, string> = {
    vector: '🎯',
    keyword: '🔤',
    adjacent: '📌',
    history: '📜',
    parent: '⬆️',
    relation: '🔗',
    graph: '🕸️',
};

// Match type to icon mapping (keys match backend API response)
export const matchTypeIcons: Record<string, string> = {
    'Vector Match': '🎯',
    'Keyword Match': '🔤',
    'Adjacent Chunk Match': '📌',
    'History Match': '📜',
    'Parent Chunk Match': '⬆️',
    'Relation Chunk Match': '🔗',
    'Graph Match': '🕸️',
};

// Get icon for a tool name
export function getToolIcon(toolName: string): string {
    return toolIcons[toolName] || '🛠️';
}

// Get icon for a match type
export function getMatchTypeIcon(matchType: string): string {
    return matchTypeIcons[matchType] || matchTypeIconKeys[matchType] || '📍';
}

// Tool name to i18n key mapping
const toolDisplayNameKeys: Record<string, string> = {
    multi_kb_search: 'Cross-KB Search',
    search_knowledge: 'Search knowledge',
    read_document: 'Read document',
    list_documents: 'List documents',
    get_chunk_detail: 'Get Chunk Detail',
    list_knowledge_bases: 'List Knowledge Bases',
    query_knowledge_graph: 'Query Knowledge Graph',
    think: 'Deep Thinking',
    todo_write: 'Make Plan',
    // Retired names (still present in stored chat history)
    knowledge_search: 'Knowledge Search',
    grep_chunks: 'Text Pattern Search',
    list_knowledge_chunks: 'List Knowledge Chunks',
    get_document_info: 'Get Document Info',
};

// Get tool display name (user-friendly, localized)
export function getToolDisplayName(toolName: string): string {
    const key = toolDisplayNameKeys[toolName];
    if (key) return key;

    // Format MCP tool names: "mcp_service_tool" → "Service Tool"
    if (toolName.startsWith('mcp_')) {
        const parts = toolName.slice(4).split('_');
        return parts.map(p => p.charAt(0).toUpperCase() + p.slice(1)).join(' ');
    }

    return toolName;
}

