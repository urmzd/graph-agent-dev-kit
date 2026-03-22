package embedderregistry_test

import (
	"context"
	"testing"

	"github.com/urmzd/saige/rag/embedderregistry"
	"github.com/urmzd/saige/rag/types"
)

type stubEmbedder struct {
	dim int
}

func (s *stubEmbedder) Embed(_ context.Context, variants []types.ContentVariant) ([][]float32, error) {
	result := make([][]float32, len(variants))
	for i := range variants {
		result[i] = make([]float32, s.dim)
		result[i][0] = float32(s.dim) // marker to identify which embedder was used
	}
	return result, nil
}

func TestRegistryDispatch(t *testing.T) {
	r := embedderregistry.New()
	r.Register(types.ContentText, &stubEmbedder{dim: 3})
	r.Register(types.ContentImage, &stubEmbedder{dim: 5})

	variants := []types.ContentVariant{
		{UUID: "v1", ContentType: types.ContentText, Text: "hello"},
		{UUID: "v2", ContentType: types.ContentImage, Text: "img"},
		{UUID: "v3", ContentType: types.ContentText, Text: "world"},
	}

	embeddings, err := r.Embed(context.Background(), variants)
	if err != nil {
		t.Fatal(err)
	}

	if len(embeddings) != 3 {
		t.Fatalf("expected 3 embeddings, got %d", len(embeddings))
	}

	// v1 (text) should use dim=3 embedder.
	if len(embeddings[0]) != 3 {
		t.Errorf("v1: expected dim 3, got %d", len(embeddings[0]))
	}
	// v2 (image) should use dim=5 embedder.
	if len(embeddings[1]) != 5 {
		t.Errorf("v2: expected dim 5, got %d", len(embeddings[1]))
	}
	// v3 (text) should use dim=3 embedder.
	if len(embeddings[2]) != 3 {
		t.Errorf("v3: expected dim 3, got %d", len(embeddings[2]))
	}
}

func TestRegistryFallback(t *testing.T) {
	r := embedderregistry.New(embedderregistry.WithFallback(&stubEmbedder{dim: 7}))

	variants := []types.ContentVariant{
		{UUID: "v1", ContentType: types.ContentAudio, Text: "audio data"},
	}

	embeddings, err := r.Embed(context.Background(), variants)
	if err != nil {
		t.Fatal(err)
	}

	if len(embeddings[0]) != 7 {
		t.Errorf("expected fallback dim 7, got %d", len(embeddings[0]))
	}
}

func TestRegistryNoEmbedder(t *testing.T) {
	r := embedderregistry.New() // no registered embedders, no fallback

	variants := []types.ContentVariant{
		{UUID: "v1", ContentType: types.ContentText, Text: "hello"},
	}

	_, err := r.Embed(context.Background(), variants)
	if err == nil {
		t.Fatal("expected error for unregistered content type without fallback")
	}
}

func TestRegistryEmpty(t *testing.T) {
	r := embedderregistry.New()
	embeddings, err := r.Embed(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if embeddings != nil {
		t.Errorf("expected nil for empty input, got %v", embeddings)
	}
}

func TestNewTextOnly(t *testing.T) {
	r := embedderregistry.NewTextOnly(&stubEmbedder{dim: 4})

	variants := []types.ContentVariant{
		{UUID: "v1", ContentType: types.ContentText, Text: "text"},
		{UUID: "v2", ContentType: types.ContentImage, Text: "image"},
	}

	embeddings, err := r.Embed(context.Background(), variants)
	if err != nil {
		t.Fatal(err)
	}

	// Both should use the fallback embedder (dim=4).
	for i, emb := range embeddings {
		if len(emb) != 4 {
			t.Errorf("variant %d: expected dim 4, got %d", i, len(emb))
		}
	}
}
