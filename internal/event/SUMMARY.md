# EnterpriseRag Event System Summary

## Overview

A complete event emission and listening mechanism has been built for the EnterpriseRag project, supporting event handling at every step of the user query processing pipeline.

## Core Features

### ✅ Implemented Features

1. **Event bus (EventBus)**
   - `Emit(ctx, event)` — emit an event
   - `On(eventType, handler)` — register an event listener
   - `Off(eventType)` — remove an event listener
   - `EmitAndWait(ctx, event)` — emit an event and wait for all handlers to finish
   - Both synchronous and asynchronous modes

2. **Event types**
   - Query processing events (received, validated, pre-processed, rewritten)
   - Retrieval events (start, vector retrieval, keyword retrieval, entity retrieval, complete)
   - Rerank events (start, complete)
   - Merge events (start, complete)
   - Chat generation events (start, complete, streaming output)
   - Error events

3. **Event data structures**
   - `QueryData` — query data
   - `RetrievalData` — retrieval data
   - `RerankData` — rerank data
   - `MergeData` — merge data
   - `ChatData` — chat data
   - `ErrorData` — error data

4. **Middleware support**
   - `WithLogging()` — logging middleware
   - `WithTiming()` — timing middleware
   - `WithRecovery()` — error recovery middleware
   - `Chain()` — middleware composition

5. **Global event bus**
   - A singleton global event bus
   - Global convenience functions (`On`, `Emit`, `EmitAndWait` and so on)

6. **Examples and tests**
   - Complete unit tests
   - Performance benchmarks
   - Complete usage examples
   - Real-world scenario demos

## File Structure

```
internal/event/
├── event.go                    # Core event bus implementation
├── event_data.go              # Event data structure definitions
├── middleware.go              # Middleware implementation
├── global.go                  # Global event bus
├── integration_example.go     # Integration examples (monitoring and analytics handlers)
├── example_test.go            # Tests and examples
├── demo/
│   └── main.go               # Complete RAG pipeline demo
├── README.md                 # Detailed documentation
├── usage_example.md          # Usage examples document
└── SUMMARY.md                # This document
```

## Performance Figures

- **Event emission performance**: ~9 nanoseconds per call (benchmark)
- **Concurrency safety**: guaranteed by `sync.RWMutex`
- **Memory overhead**: extremely low; only references to handler functions are stored

## Use Cases

### 1. Monitoring and Metrics Collection

```go
bus.On(event.EventRetrievalComplete, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.RetrievalData)
    // Send to Prometheus or another monitoring system
    metricsCollector.RecordRetrievalDuration(data.Duration)
    return nil
})
```

### 2. Logging

```go
bus.On(event.EventQueryRewritten, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.QueryData)
    logger.Infof(ctx, "Query rewritten: %s -> %s", 
        data.OriginalQuery, data.RewrittenQuery)
    return nil
})
```

### 3. User Behaviour Analytics

```go
bus.On(event.EventQueryReceived, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.QueryData)
    // Send to the analytics platform
    analytics.TrackQuery(data.UserID, data.OriginalQuery)
    return nil
})
```

### 4. Error Tracking

```go
bus.On(event.EventError, func(ctx context.Context, e event.Event) error {
    data := e.Data.(event.ErrorData)
    // Send to the error tracking system
    sentry.CaptureException(data.Error)
    return nil
})
```

## How to Integrate

### Step 1: Initialise the event system

At application startup (for example in `main.go` or `container.go`):

```go
import "github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/event"

func Initialize() {
    // Get the global event bus
    bus := event.GetGlobalEventBus()
    
    // Set up monitoring and analytics
    event.NewMonitoringHandler(bus)
    event.NewAnalyticsHandler(bus)
}
```

### Step 2: Emit events at each processing stage

Add event emission to the individual plugins of the query processing pipeline:

```go
// In search.go
event.Emit(ctx, event.NewEvent(event.EventRetrievalStart, event.RetrievalData{
    Query:           chatManage.ProcessedQuery,
    KnowledgeBaseID: chatManage.KnowledgeBaseID,
    TopK:            chatManage.EmbeddingTopK,
}).WithSessionID(chatManage.SessionID))

// In rerank.go
event.Emit(ctx, event.NewEvent(event.EventRerankComplete, event.RerankData{
    Query:       chatManage.ProcessedQuery,
    InputCount:  len(chatManage.SearchResult),
    OutputCount: len(rerankResults),
    Duration:    time.Since(startTime).Milliseconds(),
}).WithSessionID(chatManage.SessionID))
```

### Step 3: Register custom event handlers

Register your own handlers as needed:

```go
event.On(event.EventQueryRewritten, func(ctx context.Context, e event.Event) error {
    // Custom handling logic
    return nil
})
```

## Advantages

1. **Low coupling**: emitters and listeners are fully decoupled, which makes the system easier to maintain and extend
2. **High performance**: extremely low overhead (~9 nanoseconds per call)
3. **Flexibility**: supports synchronous and asynchronous modes, and single or multiple listeners
4. **Extensibility**: new event types and handlers are easy to add
5. **Type safety**: predefined event data structures
6. **Middleware support**: cross-cutting concerns (logging, timing, error handling and so on) are easy to add
7. **Test friendly**: event behaviour is easy to assert in tests

## Test Results

✅ All unit tests pass
✅ Performance tests pass (~9 nanoseconds per call)
✅ Asynchronous handling tests pass
✅ Multi-handler tests pass
✅ Full pipeline demo runs successfully

## Suggested Next Steps

### Optional enhancements

1. **Event persistence**: store key events in a database or message queue
2. **Event replay**: support replaying events for debugging or analysis
3. **Event filtering**: support more elaborate event filtering and routing
4. **Priority queue**: support prioritised event handling
5. **Distributed events**: support cross-service events through a message queue

### Integration suggestions

1. **Monitoring integration**: integrate Prometheus for metrics collection
2. **Logging integration**: unified structured logging
3. **Tracing integration**: integrate with the existing tracing system
4. **Alerting integration**: event-driven alerting

## Sample Output

Running `go run ./internal/event/demo/main.go` prints the events of a complete RAG pipeline:

```
Step 1: Query Received
[MONITOR] Query received - Session: session-xxx, Query: What is RAG technology?
[ANALYTICS] Query tracked - User: user-123, Session: session-xxx

Step 2: Query Rewriting
[MONITOR] Query rewrite started
[MONITOR] Query rewritten - Original: What is RAG technology?, Rewritten: retrieval-augmented generation technology...
[CUSTOM] Query Transformation: ...

Step 3: Vector Retrieval
[MONITOR] Retrieval started - Type: vector, TopK: 20
[MONITOR] Retrieval completed - Results: 18, Duration: 301ms
[CUSTOM] Retrieval Efficiency: Rate: 90.00%

Step 4: Result Reranking
[MONITOR] Rerank started - Input: 18
[MONITOR] Rerank completed - Output: 5, Duration: 201ms
[CUSTOM] Rerank Statistics: Reduction: 72.22%

Step 5: Chat Completion
[MONITOR] Chat generation started
[MONITOR] Chat generation completed - Tokens: 256, Duration: 801ms
[ANALYTICS] Chat metrics - Model: gpt-4, Tokens: 256
```

## Summary

The event system is fully implemented and verified by tests, and can be integrated into the EnterpriseRag project right away for monitoring, logging, analytics and debugging of every stage of the query processing pipeline. The design is simple, the performance is excellent, and it is easy to use and extend.
