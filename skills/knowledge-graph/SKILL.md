---
name: knowledge-graph
description: Build and query knowledge graphs with SurrealDB — ingest episodes, extract entities/relations via LLM, and search facts by semantic similarity or keyword. Use when working with knowledge graphs, entity extraction, or SurrealDB graph storage.
argument-hint: [query]
---

# kgdk

Build and query knowledge graphs using `kgdk`.

## Quick Start

```go
import (
    kg "github.com/urmzd/kgdk"
    "github.com/urmzd/adk/provider/ollama"
)

// Connect
client := ollama.NewClient("http://localhost:11434", "qwen2.5", "nomic-embed-text")
graph, _ := kg.NewGraph(ctx,
    kg.WithSurrealDB("ws://localhost:8000", "default", "knowledge", "root", "root"),
    kg.WithExtractor(kg.NewOllamaExtractor(client)),
    kg.WithEmbedder(kg.NewOllamaEmbedder(client)),
)
defer graph.Close(ctx)

// Ingest
graph.IngestEpisode(ctx, &kg.EpisodeInput{
    Name: "notes", Body: "Alice presented the roadmap.", Source: "meeting",
})

// Search
facts, _ := graph.SearchFacts(ctx, "roadmap")
```

## Key Operations

| Method | Purpose |
|--------|---------|
| `IngestEpisode` | Extract entities/relations from text and store them |
| `SearchFacts` | Full-text search on relation facts |
| `GetEntity` | Retrieve a single entity by ID |
| `GetNode` | Get a node with its neighborhood (depth N) |
| `GetGraph` | Full graph snapshot for visualization |
| `ApplyOntology` | Constrain entity/relation types |

## Ontology

```go
graph.ApplyOntology(ctx, &kg.Ontology{
    EntityTypes:   []kg.EntityTypeDef{{Name: "Person", Description: "A human"}},
    RelationTypes: []kg.RelationTypeDef{{Name: "works_on", SourceType: "Person", TargetType: "Project"}},
})
```
