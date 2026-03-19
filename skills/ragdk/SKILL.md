---
name: ragdk
description: Build multi-modal RAG pipelines with graph-enhanced retrieval in Go
version: 0.1.0
author: urmzd
tags: [rag, retrieval, embeddings, knowledge-graph, go]
---

# ragdk

A Go library for multi-modal Retrieval-Augmented Generation with graph-enhanced retrieval.

## What it does

ragdk models documents as hierarchical structures (Document -> Section -> ContentVariant) where each section can have multiple modality representations (text, image, table, audio). It provides a pluggable pipeline for ingesting documents, generating embeddings, performing vector search, and optionally extracting entities into a knowledge graph via kgdk.

## When to use

- Ingesting documents (PDF, text, images) into a searchable vector store
- Building RAG pipelines with multi-modal content support
- Combining vector search with knowledge graph traversal
- Deduplicating documents by content fingerprint

## Usage

### Install

```bash
go get github.com/urmzd/ragdk
```

### Create a pipeline

```go
import (
    "github.com/urmzd/ragdk"
    "github.com/urmzd/ragdk/memstore"
    "github.com/urmzd/ragdk/ragtypes"
)

pipe, err := ragdk.NewPipeline(
    ragdk.WithStore(memstore.New()),
    ragdk.WithContentExtractor(myExtractor),
    ragdk.WithEmbedders(myEmbedderRegistry),
)
```

### Ingest a document

```go
result, err := pipe.Ingest(ctx, &ragtypes.RawDocument{
    SourceURI: "https://example.com/paper.pdf",
    MIMEType:  "application/pdf",
    Data:      pdfBytes,
})
```

### Search

```go
results, err := pipe.Search(ctx, "attention mechanism", ragtypes.WithLimit(5))
for _, v := range results.Variants {
    fmt.Printf("[%.4f] %s\n", v.Score, v.Variant.Text[:80])
}
```

### With kgdk integration

```go
pipe, err := ragdk.NewPipeline(
    ragdk.WithStore(store),
    ragdk.WithContentExtractor(extractor),
    ragdk.WithKGGraph(kgGraph),  // enables entity extraction
)
```

## Key interfaces

| Interface | Purpose |
|-----------|---------|
| `Pipeline` | Orchestrate ingest, search, reconstruct |
| `Store` | Document CRUD + vector search |
| `ContentExtractor` | Raw bytes -> structured Document |
| `Chunker` | Split long sections |
| `EmbedderRegistry` | Dispatch embedding by ContentType |

## Configuration options

| Option | Purpose |
|--------|---------|
| `WithStore(s)` | Set the document store (required) |
| `WithContentExtractor(ext)` | Set the content extractor (required) |
| `WithChunker(ch)` | Set the chunker (optional) |
| `WithEmbedders(reg)` | Set the embedder registry (optional, needed for search) |
| `WithKGGraph(g)` | Enable kgdk entity extraction (optional) |
| `WithDedupBehavior(b)` | Set dedup behavior: DedupSkip (default) or DedupReplace |
| `WithStoreOriginals(true)` | Persist raw document bytes |
