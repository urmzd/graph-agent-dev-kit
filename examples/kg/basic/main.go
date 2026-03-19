// Command basic demonstrates building a knowledge graph with kg.
//
// Prerequisites:
//   - SurrealDB running (e.g. surreal start --user root --pass root)
//   - Ollama running with a model pulled (e.g. ollama pull llama3.2)
//
// Usage:
//
//	go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/urmzd/graph-agent-dev-kit/agent/provider/ollama"
	"github.com/urmzd/graph-agent-dev-kit/kg"
	"github.com/urmzd/graph-agent-dev-kit/kg/kgtypes"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 1. Create Ollama client for LLM extraction and embedding.
	// Args: host, chat model, embedding model.
	ollamaClient := ollama.NewClient("http://localhost:11434", "gemma3", "nomic-embed-text")

	// 2. Create the knowledge graph.
	graph, err := kg.NewGraph(ctx,
		kg.WithSurrealDB("ws://localhost:8000", "graph-agent-dev-kit", "example", "root", "root"),
		kg.WithExtractor(kg.NewOllamaExtractor(ollamaClient)),
		kg.WithEmbedder(kg.NewOllamaEmbedder(ollamaClient)),
		kg.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("create graph: %v", err)
	}
	defer graph.Close(ctx)

	// 3. (Optional) Apply an ontology to guide extraction.
	err = graph.ApplyOntology(ctx, &kgtypes.Ontology{
		EntityTypes: []kgtypes.EntityTypeDef{
			{Name: "Person", Description: "A human being"},
			{Name: "Organization", Description: "A company or institution"},
			{Name: "Technology", Description: "A programming language, framework, or tool"},
		},
		RelationTypes: []kgtypes.RelationTypeDef{
			{Name: "works_at", Description: "Employment relationship", SourceType: "Person", TargetType: "Organization"},
			{Name: "uses", Description: "Uses a technology", SourceType: "Person", TargetType: "Technology"},
			{Name: "develops", Description: "Develops or maintains", SourceType: "Organization", TargetType: "Technology"},
		},
	})
	if err != nil {
		log.Fatalf("apply ontology: %v", err)
	}

	// 4. Ingest episodes of text.
	episodes := []kgtypes.EpisodeInput{
		{
			Name:    "team-intro",
			Body:    "Alice is a software engineer at Acme Corp. She primarily uses Go and Python for backend services.",
			Source:  "onboarding doc",
			GroupID: "engineering",
		},
		{
			Name:    "project-update",
			Body:    "Bob joined Acme Corp last month. He works with Alice on the data pipeline team and is learning Go.",
			Source:  "standup notes",
			GroupID: "engineering",
		},
	}

	for _, ep := range episodes {
		result, err := graph.IngestEpisode(ctx, &ep)
		if err != nil {
			log.Printf("ingest %q: %v", ep.Name, err)
			continue
		}
		fmt.Printf("Ingested %q: %d entities, %d relations\n",
			result.Name, len(result.EntityNodes), len(result.EpisodicEdges))
	}

	// 5. Search for facts.
	searchResult, err := graph.SearchFacts(ctx, "Who works at Acme?",
		kgtypes.WithLimit(5),
		kgtypes.WithGroupID("engineering"),
	)
	if err != nil {
		log.Fatalf("search: %v", err)
	}

	fmt.Println("\nSearch results:")
	for _, fact := range kg.FactsToStrings(searchResult.Facts) {
		fmt.Printf("  - %s\n", fact)
	}

	// 6. Explore the graph.
	graphData, err := graph.GetGraph(ctx, 50)
	if err != nil {
		log.Fatalf("get graph: %v", err)
	}
	fmt.Printf("\nGraph: %d nodes, %d edges\n", len(graphData.Nodes), len(graphData.Edges))

	// 7. Get node details with neighbors.
	if len(graphData.Nodes) > 0 {
		detail, err := graph.GetNode(ctx, graphData.Nodes[0].ID, 1)
		if err != nil {
			log.Printf("get node: %v", err)
		} else {
			fmt.Printf("Node %q has %d neighbors\n", detail.Node.Name, len(detail.Neighbors))

			// Build a local subgraph from a node detail.
			sub := kg.Subgraph(detail)
			fmt.Printf("Subgraph: %d nodes, %d edges\n", len(sub.Nodes), len(sub.Edges))
		}
	}
}
