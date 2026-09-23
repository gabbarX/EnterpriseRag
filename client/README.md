# EnterpriseRag HTTP Client

This package provides a client library for interacting with EnterpriseRag services, supporting all HTTP-based interface calls, making it easier for other modules to integrate with EnterpriseRag services without having to write HTTP request code directly.

## Main Features

The client includes the following main functional modules:

1. **Session Management**: Create, retrieve, update, and delete sessions
2. **Knowledge Base Management**: Create, retrieve, update, and delete knowledge bases
3. **Knowledge Management**: Add, retrieve, and delete knowledge content
4. **Workspace Management**: CRUD operations for workspaces
5. **Knowledge Q&A**: Supports regular Q&A and streaming Q&A
6. **Agent Q&A**: Agent-based intelligent Q&A, including the thinking process, tool calls, and reflection
7. **Chunk Management**: Query, update, and delete knowledge chunks
8. **Message Management**: Retrieve and delete session messages
9. **Model Management**: Create, retrieve, update, and delete models
10. **Evaluation Function**: Start evaluation tasks and get evaluation results
11. **Sandbox skills**: Install a skill onto a sandbox config (zip upload, or ClawHub / SkillHub / GitHub source) and configure the environment variables it needs
12. **Long-term memory**: The caller's cross-session memories (settings, items, confirm/reject, topics, document affinity, export, consolidate)
13. **Auth**: Login, refresh tokens, and switch the active workspace (`SwitchTenant` records the last-active-tenant preference)

## Usage

### Creating Client Instance

```go
import (
    "context"
    "github.com/ORG_PLACEHOLDER/EnterpriseRag/client"
    "time"
)

// Create client instance
apiClient := client.NewClient(
    "http://api.example.com", 
    client.WithToken("your-auth-token"),
    client.WithTimeout(30*time.Second),
)
```

### Workspace Configuration

You can set a default workspace with `WithTenantID`; the client will automatically send the `X-Tenant-ID` header:

```go
tenantID := uint64(10000)
apiClient := client.NewClient(
    "http://api.example.com",
    client.WithToken("your-auth-token"),
    client.WithTenantID(tenantID),
)
```

If a single request needs a different workspace, set `TenantID` in the request context. The value can be a `uint64`, `*uint64`, or a numeric string, and it will take precedence over the client default:

```go
ctx := context.WithValue(context.Background(), "TenantID", uint64(10000))
// Pass ctx into any client method to switch to workspace 10000 for that request
```

### Example: Create Knowledge Base and Upload File

```go
// Create knowledge base
kb := &client.KnowledgeBase{
    Name:        "Test Knowledge Base",
    Description: "This is a test knowledge base",
    ChunkingConfig: client.ChunkingConfig{
        ChunkSize:    500,
        ChunkOverlap: 50,
        Separators:   []string{"\n\n", "\n", ". ", "? ", "! "},
    },
    ImageProcessingConfig: client.ImageProcessingConfig{
        ModelID: "image_model_id",
    },
    EmbeddingModelID: "embedding_model_id",
    SummaryModelID:   "summary_model_id",
}

kb, err := apiClient.CreateKnowledgeBase(context.Background(), kb)
if err != nil {
    // Handle error
}

// Upload knowledge file with metadata
metadata := map[string]string{
    "source": "local",
    "type":   "document",
}
knowledge, err := apiClient.CreateKnowledgeFromFile(context.Background(), kb.ID, "path/to/file.pdf", metadata)
if err != nil {
    // Handle error
}

// Download original files from one knowledge base as a ZIP (max 200 IDs, 512 MiB)
err = apiClient.DownloadKnowledgeFiles(context.Background(), kb.ID, []string{knowledge.ID}, "knowledge-files.zip")
if err != nil {
    // Handle error
}
```

### Example: Create Session and Chat

```go
// Create session
sessionRequest := &client.CreateSessionRequest{
    KnowledgeBaseID: knowledgeBaseID,
    SessionStrategy: &client.SessionStrategy{
        MaxRounds:        10,
        EnableRewrite:    true,
        FallbackStrategy: "fixed_answer",
        FallbackResponse: "Sorry, I cannot answer this question",
        EmbeddingTopK:    5,
        KeywordThreshold: 0.5,
        VectorThreshold:  0.7,
        RerankModelID:    "rerank_model_id",
        RerankTopK:       3,
        RerankThreshold:  0.8,
        SummaryModelID:   "summary_model_id",
    },
}

session, err := apiClient.CreateSession(context.Background(), sessionRequest)
if err != nil {
    // Handle error
}

// Regular Q&A
answer, err := apiClient.KnowledgeQA(context.Background(), session.ID, &client.KnowledgeQARequest{
    Query: "What is artificial intelligence?",
})
if err != nil {
    // Handle error
}

// Streaming Q&A
err = apiClient.KnowledgeQAStream(context.Background(), session.ID, &client.KnowledgeQARequest{
    Query:            "What is machine learning?",
    KnowledgeBaseIDs: []string{knowledgeBaseID}, // Optional: restrict to specific knowledge bases
    WebSearchEnabled: false,                      // Optional: enable web search
}, func(response *client.StreamResponse) error {
    // Handle each response chunk
    fmt.Print(response.Content)
    return nil
})
if err != nil {
    // Handle error
}
```

### Example: Agent Q&A

Agent Q&A offers a more capable conversational experience, with tool calls, a visible thinking process, and self-reflection.

```go
// Create an Agent session
agentSession := apiClient.NewAgentSession(session.ID)

// Ask the Agent, handling every event type
err := agentSession.Ask(context.Background(), "Search for material on machine learning and summarise the key points",
    func(resp *client.AgentStreamResponse) error {
        switch resp.ResponseType {
        case client.AgentResponseTypeThinking:
            // The Agent is thinking
            if resp.Done {
                fmt.Printf("💭 Thinking: %s\n", resp.Content)
            }
        
        case client.AgentResponseTypeToolCall:
            // The Agent is calling a tool
            if resp.Data != nil {
                toolName := resp.Data["tool_name"]
                fmt.Printf("🔧 Tool call: %v\n", toolName)
            }
        
        case client.AgentResponseTypeToolResult:
            // Result of the tool run
            fmt.Printf("✓ Tool result: %s\n", resp.Content)
        
        case client.AgentResponseTypeReferences:
            // Knowledge references
            if resp.KnowledgeReferences != nil {
                fmt.Printf("📚 Found %d related knowledge entries\n", len(resp.KnowledgeReferences))
                for _, ref := range resp.KnowledgeReferences {
                    fmt.Printf("  - [%.3f] %s\n", ref.Score, ref.KnowledgeTitle)
                }
            }
        
        case client.AgentResponseTypeAnswer:
            // Final answer (streamed)
            fmt.Print(resp.Content)
            if resp.Done {
                fmt.Println() // Newline once the answer is complete
            }
        
        case client.AgentResponseTypeReflection:
            // The Agent's self-reflection
            if resp.Done {
                fmt.Printf("🤔 Reflection: %s\n", resp.Content)
            }
        
        case client.AgentResponseTypeError:
            // Error message
            fmt.Printf("❌ Error: %s\n", resp.Content)
        }
        return nil
    })

if err != nil {
    // Handle error
}

// Simplified version: only the final answer matters
var finalAnswer string
err = agentSession.Ask(context.Background(), "What is deep learning?",
    func(resp *client.AgentStreamResponse) error {
        if resp.ResponseType == client.AgentResponseTypeAnswer {
            finalAnswer += resp.Content
        }
        return nil
    })
```

### Agent event types

| Event type | Description | When it fires |
|---------|------|---------|
| `AgentResponseTypeThinking` | Agent thinking process | While the Agent analyses the question and plans its steps |
| `AgentResponseTypeToolCall` | Tool call | When the Agent decides to use a tool |
| `AgentResponseTypeToolResult` | Tool result | Once the tool has finished running |
| `AgentResponseTypeReferences` | Knowledge references | When related knowledge is retrieved |
| `AgentResponseTypeAnswer` | Final answer | While the Agent generates its reply (streamed) |
| `AgentResponseTypeArtifactsPending` | Generated files uploading | After the answer ends, before the files finish being written to object storage |
| `AgentResponseTypeReflection` | Self-reflection | When the Agent evaluates its own answer |
| `AgentResponseTypeError` | Error | When an error occurs |

### Example: Managing Models

```go
// Create model
modelRequest := &client.CreateModelRequest{
    Name:        "Test Model",
    Type:        client.ModelTypeChat,
    Source:      client.ModelSourceInternal,
    Description: "This is a test model",
    Parameters: client.ModelParameters{
        "temperature": 0.7,
        "top_p":       0.9,
    },
    IsDefault: true,
}
model, err := apiClient.CreateModel(context.Background(), modelRequest)
if err != nil {
    // Handle error
}

// List all models
models, err := apiClient.ListModels(context.Background())
if err != nil {
    // Handle error
}
```

### Example: Managing Knowledge Chunks

```go
// List knowledge chunks
chunks, total, err := apiClient.ListKnowledgeChunks(context.Background(), knowledgeID, 1, 10)
if err != nil {
    // Handle error
}

// Update chunk
updateRequest := &client.UpdateChunkRequest{
    Content:   "Updated chunk content",
    IsEnabled: true,
}
updatedChunk, err := apiClient.UpdateChunk(context.Background(), knowledgeID, chunkID, updateRequest)
if err != nil {
    // Handle error
}
```

### Example: Re-parsing Knowledge

```go
// Re-parse a knowledge entry (drops the existing content and parses it again)
// Use this when:
// 1. The original parse failed and needs to be retried
// 2. The parsing configuration changed (chunking strategy, multimodal settings, and so on)
// 3. The knowledge content was updated and the parsed output must be refreshed

knowledge, err := apiClient.ReparseKnowledge(context.Background(), knowledgeID)
if err != nil {
    // Handle error
}

// The knowledge entry moves to the "pending" state and is re-parsed asynchronously
fmt.Printf("Knowledge ID: %s\n", knowledge.ID)
fmt.Printf("Parse Status: %s\n", knowledge.ParseStatus)      // "pending"
fmt.Printf("Enable Status: %s\n", knowledge.EnableStatus)    // "disabled"

// You can poll for the parse status
for {
    time.Sleep(5 * time.Second)
    knowledge, err := apiClient.GetKnowledge(context.Background(), knowledgeID)
    if err != nil {
        // Handle error
    }
    
    if knowledge.ParseStatus == "completed" {
        fmt.Println("Knowledge re-parsing completed!")
        break
    } else if knowledge.ParseStatus == "failed" {
        fmt.Printf("Knowledge re-parsing failed: %s\n", knowledge.ErrorMessage)
        break
    }
}
```

### Example: Cancelling a Parse

```go
// Cancel a parse that is still running (useful when resources are tight or the wrong file was uploaded)
// - Knowledge already in the completed / failed state cannot be cancelled
// - Chunks and index entries already written are kept; call ReparseKnowledge later to parse again

knowledge, err := apiClient.CancelKnowledgeParse(context.Background(), knowledgeID)
if err != nil {
    // Handle error
}
fmt.Printf("Parse Status: %s\n", knowledge.ParseStatus) // "cancelled"
```

### Example: Inspecting the Document Parsing Trace (Span Tree)

```go
// Fetch the span tree of the document parsing pipeline (root → stage → subspan)
// - Pass 0 as the attempt to get the most recent parse attempt
// - Always returns the 5 standard stages: docreader / chunking / embedding / multimodal / postprocess
trace, err := apiClient.GetKnowledgeProcessingSpans(context.Background(), knowledgeID, 0)
if err != nil {
    // Handle error
}
fmt.Printf("ParseStatus=%s CurrentStage=%s\n", trace.ParseStatus, trace.CurrentStage)
for _, stage := range trace.Trace.Children {
    fmt.Printf("- %s: %s (%dms)\n", stage.Name, stage.Status, stage.DurationMs)
}
```

### Example: Getting Session Messages

```go
// Get recent messages
messages, err := apiClient.GetRecentMessages(context.Background(), sessionID, 10)
if err != nil {
    // Handle error
}

// Get messages before a specific time
beforeTime := time.Now().Add(-24 * time.Hour)
olderMessages, err := apiClient.GetMessagesBefore(context.Background(), sessionID, beforeTime, 10)
if err != nil {
    // Handle error
}
```

### Example: Install a sandbox skill from a registry

`source` must be explicit: `@owner/slug` for ClawHub, a full `https://clawhub.ai/skills-sh/owner/repo/slug` (or `skills-sh:owner/repo/slug`) for federated skills.sh listings, or a full GitHub / SkillHub URL. Bare `owner/slug` is rejected.

```go
skillID, err := apiClient.InstallSandboxSkillFromSource(
    context.Background(), sandboxConfigID, "@owner/slug")
if err != nil {
    // Handle error
}
_ = skillID // follow /sandbox-configs/{id}/skills/{skillID}/install-events
```

### Example: Stop a stuck install

After a process restart the row may sit at `installing` with nothing running, which hides retry and uninstall. Stop rewrites the row immediately (and cancels the in-process goroutine if one is still alive). Then retry or uninstall as usual.

```go
skill, err := apiClient.StopSandboxSkill(context.Background(), sandboxConfigID, skillID)
if err != nil {
    // Handle error
}
_ = skill
```

### Example: Retry a failed install

Installs usually fail for reasons the bundle cannot fix — an unreachable
sandbox, a package index that timed out. The server still holds the archive,
so the retry needs nothing from you.

```go
skillID, err := apiClient.ReinstallSandboxSkill(context.Background(), sandboxConfigID, skillID)
if err != nil {
    // Handle error
}
```

### Example: Browse files of an installed skill

```go
files, err := apiClient.ListSandboxSkillFiles(context.Background(), sandboxConfigID, skillID)
if err != nil {
    // Handle error
}
content, err := apiClient.GetSandboxSkillFile(context.Background(), sandboxConfigID, skillID, "SKILL.md")
if err != nil {
    // Handle error
}
_ = files
_ = content
```

### Example: Configure a skill's environment variables

A skill declares the environment variables it needs when it is installed. Values live in two layers: a workspace value an admin sets for everybody, and a per-identity value that overrides it for the caller alone. No endpoint reads a stored value back; they only report whether one is set.

An API key and a web login are different identities: a personal value entered through the web UI does not apply to runs driven by an API key. Prefer workspace values for integrations.

```go
// Workspace-wide, applies to everybody; requires Admin or above
skill, err := apiClient.SetSandboxSkillEnvValues(
    context.Background(), sandboxConfigID, skillID,
    map[string]string{"TAVILY_API_KEY": "tvly-xxxxx"})
if err != nil {
    // Handle error
}

// Applies to the calling identity alone
err = apiClient.SetMySkillEnvVar(
    context.Background(), skillID, "TAVILY_API_KEY", "tvly-yyyyy")
if err != nil {
    // Handle error
}

// See what is still unset. Clearing a value is a delete, not a write of ""
groups, err := apiClient.ListMyEnvVars(context.Background())
if err != nil {
    // Handle error
}
_ = skill
_ = groups
```

## Complete Example

Please refer to the `ExampleUsage` function in the `example.go` file, which demonstrates the complete usage flow of the client.
