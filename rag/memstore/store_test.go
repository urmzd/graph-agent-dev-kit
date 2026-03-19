package memstore_test

import (
	"context"
	"testing"

	"github.com/urmzd/graph-agent-dev-kit/rag/memstore"
	"github.com/urmzd/graph-agent-dev-kit/rag/ragtypes"
)

func TestGetVariant(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()

	doc := &ragtypes.Document{
		UUID:      "doc1",
		Title:     "Test",
		SourceURI: "http://example.com",
		Sections: []ragtypes.Section{{
			UUID:         "sec1",
			DocumentUUID: "doc1",
			Index:        0,
			Heading:      "Intro",
			Variants: []ragtypes.ContentVariant{{
				UUID:        "var1",
				SectionUUID: "sec1",
				ContentType: ragtypes.ContentText,
				Text:        "hello world",
			}},
		}},
	}
	if err := store.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}

	v, prov, err := store.GetVariant(ctx, "var1")
	if err != nil {
		t.Fatal(err)
	}
	if v.Text != "hello world" {
		t.Errorf("expected 'hello world', got %q", v.Text)
	}
	if prov.DocumentUUID != "doc1" {
		t.Errorf("expected doc UUID 'doc1', got %q", prov.DocumentUUID)
	}
	if prov.SectionHeading != "Intro" {
		t.Errorf("expected heading 'Intro', got %q", prov.SectionHeading)
	}

	// Not found.
	_, _, err = store.GetVariant(ctx, "nonexistent")
	if err != ragtypes.ErrVariantNotFound {
		t.Errorf("expected ErrVariantNotFound, got %v", err)
	}
}

func TestSearchByEmbedding(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()

	doc := &ragtypes.Document{
		UUID: "doc1",
		Sections: []ragtypes.Section{{
			UUID:         "sec1",
			DocumentUUID: "doc1",
			Variants: []ragtypes.ContentVariant{
				{UUID: "v1", SectionUUID: "sec1", ContentType: ragtypes.ContentText, Text: "cats", Embedding: []float32{1, 0, 0}},
				{UUID: "v2", SectionUUID: "sec1", ContentType: ragtypes.ContentText, Text: "dogs", Embedding: []float32{0, 1, 0}},
				{UUID: "v3", SectionUUID: "sec1", ContentType: ragtypes.ContentImage, Text: "image", Embedding: []float32{1, 0, 0}},
			},
		}},
	}
	if err := store.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}

	// Search without filters.
	hits, err := store.SearchByEmbedding(ctx, []float32{1, 0, 0}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}

	// Filter by content type.
	hits, err = store.SearchByEmbedding(ctx, []float32{1, 0, 0}, &ragtypes.SearchOptions{
		ContentTypes: []ragtypes.ContentType{ragtypes.ContentText},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 text hits, got %d", len(hits))
	}

	// MinScore filter.
	hits, err = store.SearchByEmbedding(ctx, []float32{1, 0, 0}, &ragtypes.SearchOptions{
		MinScore: 0.9,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Only v1 and v3 have cosine similarity 1.0 with query.
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits with min score 0.9, got %d", len(hits))
	}
}

func TestSearchByEmbeddingMetadataFilter(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()

	doc := &ragtypes.Document{
		UUID:     "doc1",
		Metadata: map[string]string{"source": "wiki"},
		Sections: []ragtypes.Section{{
			UUID:         "sec1",
			DocumentUUID: "doc1",
			Variants: []ragtypes.ContentVariant{
				{UUID: "v1", SectionUUID: "sec1", ContentType: ragtypes.ContentText, Text: "a", Embedding: []float32{1, 0}, Metadata: map[string]string{"lang": "en"}},
				{UUID: "v2", SectionUUID: "sec1", ContentType: ragtypes.ContentText, Text: "b", Embedding: []float32{1, 0}, Metadata: map[string]string{"lang": "fr"}},
			},
		}},
	}
	if err := store.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}

	hits, err := store.SearchByEmbedding(ctx, []float32{1, 0}, &ragtypes.SearchOptions{
		MetadataFilters: []ragtypes.MetadataFilter{{Key: "lang", Op: ragtypes.FilterEq, Value: "en"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit with lang=en, got %d", len(hits))
	}
	if hits[0].Variant.UUID != "v1" {
		t.Errorf("expected v1, got %q", hits[0].Variant.UUID)
	}
}
