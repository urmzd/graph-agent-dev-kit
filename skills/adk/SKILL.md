---
name: adk
description: Build streaming LLM agent loops in Go with typed deltas, tool execution, context compaction, and sub-agent delegation. Use when building AI agents, integrating LLM providers, or implementing tool-use patterns.
argument-hint: [task]
---

# adk

Build LLM agent loops using `adk`.

## Quick Start

```go
import (
    "github.com/urmzd/adk"
    "github.com/urmzd/adk/ollama"
)

client := ollama.NewClient("http://localhost:11434", "qwen2.5", "nomic-embed-text")
adapter := ollama.NewAdapter(client)

agent := adk.NewAgent(adk.AgentConfig{
    Name:         "assistant",
    SystemPrompt: "You are a helpful assistant.",
    Provider:     adapter,
    Tools:        adk.NewToolRegistry(),
    MaxIter:      10,
})

stream := agent.Invoke(ctx, []adk.Message{
    adk.NewUserMessage("Hello!"),
})

for delta := range stream.Deltas() {
    if d, ok := delta.(adk.TextContentDelta); ok {
        fmt.Print(d.Content)
    }
}
```

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Provider** | Implement `ChatStream` to plug in any LLM backend |
| **Tools** | Register tools via `ToolRegistry`; use `ToolFunc` for inline definitions |
| **Compaction** | Configure via `CompactCfg: &core.CompactConfig{Strategy: core.CompactNone\|Sliding\|Summarize}` |
| **Sub-agents** | Delegate tasks to child agents with their own providers and tools |
| **SSE Bridge** | `WriteSSE(w, flusher, stream)` to stream deltas over HTTP |
| **File Upload** | Attach files via `core.NewFileMessage(uri)` or `core.NewUserMessageWithFiles(text, files...)`; URIs are resolved by `Resolvers` and extracted by `Extractors` in `AgentConfig` |
| **Embeddings** | `core.Embedder` interface; `ollama.NewEmbedder(client)` for Ollama-backed vector embeddings |

## Sending Files

```go
import "github.com/urmzd/adk/core"

// Single file from a URI — media type inferred from extension.
msg := core.NewFileMessage("file:///path/to/image.png")

// Text prompt combined with one or more file attachments.
msg = core.NewUserMessageWithFiles("Describe these images.",
    core.FileContent{URI: "file:///img1.jpg", MediaType: core.MediaJPEG},
    core.FileContent{URI: "https://example.com/chart.png"},
)

stream := agent.Invoke(ctx, []core.Message{msg})
```

## Adding a Tool

```go
tool := &adk.ToolFunc{
    Def: adk.ToolDef{
        Name: "greet", Description: "Greet a person",
        Parameters: adk.ParameterSchema{
            Type: "object", Required: []string{"name"},
            Properties: map[string]adk.PropertyDef{
                "name": {Type: "string", Description: "Person's name"},
            },
        },
    },
    Fn: func(ctx context.Context, args map[string]any) (string, error) {
        return fmt.Sprintf("Hello, %s!", args["name"]), nil
    },
}
registry := adk.NewToolRegistry(tool)
```
