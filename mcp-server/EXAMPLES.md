# EnterpriseRag MCP Server Usage Examples

This document provides detailed usage examples for the EnterpriseRag MCP Server.

## Basic Usage

### 1. Start the Server

```bash
# Recommended way - use the main entry point
python main.py

# Check the environment configuration
python main.py --check-only

# Enable verbose logging
python main.py --verbose
```

### 2. Environment Configuration Example

```bash
# Set environment variables
export ENTERPRISERAG_BASE_URL="http://localhost:8080/api/v1"
export ENTERPRISERAG_API_KEY="your_api_key_here"

# Or set them in a .env file
echo "ENTERPRISERAG_BASE_URL=http://localhost:8080/api/v1" > .env
echo "ENTERPRISERAG_API_KEY=your_api_key_here" >> .env
```

## MCP Tool Usage Examples

The following are usage examples for the various MCP tools:

### Space Management

#### Create a Space
```json
{
  "tool": "create_tenant",
  "arguments": {
    "name": "My Company",
    "description": "Company knowledge management system",
    "business": "technology",
    "retriever_engines": {
      "engines": [
        {"retriever_type": "keywords", "retriever_engine_type": "postgres"},
        {"retriever_type": "vector", "retriever_engine_type": "postgres"}
      ]
    }
  }
}
```

#### List All Spaces
```json
{
  "tool": "list_tenants",
  "arguments": {}
}
```

### Knowledge Base Management

#### Create a Knowledge Base
```json
{
  "tool": "create_knowledge_base",
  "arguments": {
    "name": "Product Documentation Library",
    "description": "Product-related documents and materials",
    "embedding_model_id": "text-embedding-ada-002",
    "summary_model_id": "gpt-3.5-turbo"
  }
}
```

#### List Knowledge Bases
```json
{
  "tool": "list_knowledge_bases",
  "arguments": {}
}
```

#### Get Knowledge Base Details
```json
{
  "tool": "get_knowledge_base",
  "arguments": {
    "kb_id": "kb_123456"
  }
}
```

#### Hybrid Search
```json
{
  "tool": "hybrid_search",
  "arguments": {
    "kb_id": "kb_123456",
    "query": "How do I use the API",
    "vector_threshold": 0.7,
    "keyword_threshold": 0.5,
    "match_count": 10
  }
}
```

### Knowledge Management

#### Create Knowledge from a URL
```json
{
  "tool": "create_knowledge_from_url",
  "arguments": {
    "kb_id": "kb_123456",
    "url": "https://docs.example.com/api-guide",
    "enable_multimodel": true
  }
}
```

#### Create Knowledge from Text
```json
{
  "tool": "create_knowledge_from_text",
  "arguments": {
    "kb_id": "my-knowledge-base",
    "title": "Summary of the Attention Mechanism",
    "content": "# Attention Mechanism\n\nThe attention mechanism allows a model to assign weights dynamically while processing a sequence...",
    "status": "publish"
  }
}
```

#### Update Manually Created Knowledge
```json
{
  "tool": "update_knowledge_from_text",
  "arguments": {
    "knowledge_id": "know_789012",
    "title": "Summary of the Attention Mechanism (Revised)",
    "content": "# Attention Mechanism\n\nThis is the revised Markdown content...",
    "status": "publish"
  }
}
```

Omitting `title` or passing an empty string retains the original title; setting `status` to `draft` saves the content only, without re-indexing it.

#### List Knowledge
```json
{
  "tool": "list_knowledge",
  "arguments": {
    "kb_id": "kb_123456",
    "page": 1,
    "page_size": 20
  }
}
```

#### Get Knowledge Details
```json
{
  "tool": "get_knowledge",
  "arguments": {
    "knowledge_id": "know_789012"
  }
}
```

### Model Management

#### Create a Model
```json
{
  "tool": "create_model",
  "arguments": {
    "name": "GPT-4 Chat Model",
    "type": "KnowledgeQA",
    "source": "openai",
    "description": "OpenAI GPT-4 model used for knowledge-based question answering",
    "base_url": "https://api.openai.com/v1",
    "api_key": "sk-...",
    "is_default": true
  }
}
```

#### List Models
```json
{
  "tool": "list_models",
  "arguments": {}
}
```

### Session Management

#### Create a Chat Session
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "kb_123456",
    "max_rounds": 10,
    "enable_rewrite": true,
    "fallback_response": "Sorry, I am unable to answer this question.",
    "summary_model_id": "gpt-3.5-turbo"
  }
}
```

#### Get Session Details
```json
{
  "tool": "get_session",
  "arguments": {
    "session_id": "sess_345678"
  }
}
```

#### List Sessions
```json
{
  "tool": "list_sessions",
  "arguments": {
    "page": 1,
    "page_size": 10
  }
}
```

### Chat Features

#### Send a Chat Message
```json
{
  "tool": "chat",
  "arguments": {
    "session_id": "sess_345678",
    "query": "Please give an overview of the product's main features"
  }
}
```

### Chunk Management

#### List Knowledge Chunks
```json
{
  "tool": "list_chunks",
  "arguments": {
    "knowledge_id": "know_789012",
    "page": 1,
    "page_size": 50
  }
}
```

#### Delete a Knowledge Chunk
```json
{
  "tool": "delete_chunk",
  "arguments": {
    "knowledge_id": "know_789012",
    "chunk_id": "chunk_456789"
  }
}
```

## Complete Workflow Example

### Scenario: Building a Complete Knowledge-Based Question Answering System

```bash
# 1. Start the server
python main.py --verbose

# 2. Perform the following steps in your MCP client:
```

#### Step 1: Create a Space
```json
{
  "tool": "create_tenant",
  "arguments": {
    "name": "Technical Documentation Centre",
    "description": "Knowledge management for company technical documentation",
    "business": "technology"
  }
}
```

#### Step 2: Create a Knowledge Base
```json
{
  "tool": "create_knowledge_base",
  "arguments": {
    "name": "API Documentation Library",
    "description": "All API-related documentation"
  }
}
```

#### Step 3: Add Knowledge Content
```json
{
  "tool": "create_knowledge_from_url",
  "arguments": {
    "kb_id": "the knowledge base ID returned earlier",
    "url": "https://docs.company.com/api",
    "enable_multimodel": true
  }
}
```

#### Step 4: Create a Chat Session
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "knowledge base ID",
    "max_rounds": 5,
    "enable_rewrite": true
  }
}
```

#### Step 5: Start a Conversation
```json
{
  "tool": "chat",
  "arguments": {
    "session_id": "session ID",
    "query": "How do I use the user authentication API?"
  }
}
```

## Error Handling Examples

### Common Errors and Solutions

#### 1. Connection Error
```json
{
  "error": "Connection refused",
  "solution": "Check whether ENTERPRISERAG_BASE_URL is correct and confirm that the service is running"
}
```

#### 2. Authentication Error
```json
{
  "error": "Unauthorized",
  "solution": "Check whether ENTERPRISERAG_API_KEY is set correctly"
}
```

#### 3. Resource Not Found
```json
{
  "error": "Knowledge base not found",
  "solution": "Confirm that the knowledge base ID is correct, or create the knowledge base first"
}
```

## Advanced Configuration Examples

### Custom Retrieval Configuration
```json
{
  "tool": "hybrid_search",
  "arguments": {
    "kb_id": "kb_123456",
    "query": "search query",
    "vector_threshold": 0.8,
    "keyword_threshold": 0.6,
    "match_count": 15
  }
}
```

### Custom Session Strategy
```json
{
  "tool": "create_session",
  "arguments": {
    "kb_id": "kb_123456",
    "max_rounds": 20,
    "enable_rewrite": true,
    "fallback_response": "Based on the available knowledge, I am unable to answer your question accurately. Please try rephrasing it or contact technical support."
  }
}
```

## Performance Optimisation Tips

1. **Batch operations**: Process knowledge creation and updates in batches wherever possible
2. **Caching strategy**: Set search thresholds sensibly to balance accuracy against performance
3. **Session management**: Clean up sessions you no longer need promptly to save resources
4. **Log monitoring**: Use the `--verbose` option to monitor performance metrics

## Integration Examples

### Integration with Claude Desktop
Add the following to the Claude Desktop configuration file:
```json
{
  "mcpServers": {
    "enterpriserag": {
      "command": "python",
      "args": ["path/to/main.py"],
      "env": {
        "ENTERPRISERAG_BASE_URL": "http://localhost:8080/api/v1",
        "ENTERPRISERAG_API_KEY": "your_api_key"
      }
    }
  }
}
```

Project repository: https://github.com/ORG_PLACEHOLDER/EnterpriseRag/tree/main/mcp-server

### Integration with Other MCP Clients
Refer to each client's documentation to configure the server startup command and environment variables.

## Troubleshooting

If you run into problems:
1. Run `python main.py --check-only` to check the environment
2. Use `python main.py --verbose` to view detailed logs
3. Check whether the EnterpriseRag service is running normally
4. Verify network connectivity and firewall settings
