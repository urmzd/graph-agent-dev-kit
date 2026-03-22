package reranker_test

import (
	"context"
	"testing"

	"github.com/urmzd/saige/rag/types"
	"github.com/urmzd/saige/rag/reranker"
)

func TestMMRRerankerDiversity(t *testing.T) {
	// Create hits with synthetic embeddings.
	// Hit A and B are very similar (same embedding), Hit C is different.
	hits := []types.SearchHit{
		{
			Variant:    types.ContentVariant{UUID: "a", Text: "similar 1", Embedding: []float32{1, 0, 0, 0}},
			Score:      1.0,
			Provenance: types.Provenance{DocumentUUID: "d1"},
		},
		{
			Variant:    types.ContentVariant{UUID: "b", Text: "similar 2", Embedding: []float32{1, 0, 0, 0}},
			Score:      0.9,
			Provenance: types.Provenance{DocumentUUID: "d1"},
		},
		{
			Variant:    types.ContentVariant{UUID: "c", Text: "different", Embedding: []float32{0, 1, 0, 0}},
			Score:      0.8,
			Provenance: types.Provenance{DocumentUUID: "d2"},
		},
	}

	r := reranker.NewMMR(0.5) // Balance relevance and diversity.
	result, err := r.Rerank(context.Background(), "query", hits)
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	// First should be "a" (highest relevance).
	if result[0].Variant.UUID != "a" {
		t.Errorf("expected 'a' first, got %q", result[0].Variant.UUID)
	}

	// Second should be "c" (diverse from "a"), not "b" (similar to "a").
	if result[1].Variant.UUID != "c" {
		t.Errorf("expected 'c' second for diversity, got %q", result[1].Variant.UUID)
	}
}

func TestMMRSingleHit(t *testing.T) {
	hits := []types.SearchHit{
		{Variant: types.ContentVariant{UUID: "a"}, Score: 1.0},
	}

	r := reranker.NewMMR(0.7)
	result, err := r.Rerank(context.Background(), "q", hits)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
}
