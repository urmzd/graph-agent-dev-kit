package parentretriever_test

import (
	"context"
	"testing"

	"github.com/urmzd/saige/rag/memstore"
	"github.com/urmzd/saige/rag/parentretriever"
	"github.com/urmzd/saige/rag/types"
)

type mockRetriever struct {
	hits []types.SearchHit
}

func (m *mockRetriever) Retrieve(_ context.Context, _ string, _ *types.SearchOptions) ([]types.SearchHit, error) {
	return m.hits, nil
}

func TestParentRetrieverExpandsSection(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()

	// Create a document with a section that has multiple variants.
	doc := &types.Document{
		UUID:  "doc1",
		Title: "Test Doc",
		Sections: []types.Section{{
			UUID:         "sec1",
			DocumentUUID: "doc1",
			Index:        0,
			Heading:      "Introduction",
			Variants: []types.ContentVariant{
				{UUID: "v1", SectionUUID: "sec1", ContentType: types.ContentText, Text: "First paragraph."},
				{UUID: "v2", SectionUUID: "sec1", ContentType: types.ContentText, Text: "Second paragraph."},
			},
		}},
	}
	if err := store.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}

	// Mock inner retriever returns a hit for just v1.
	inner := &mockRetriever{
		hits: []types.SearchHit{{
			Variant: types.ContentVariant{UUID: "v1", Text: "First paragraph."},
			Score:   0.9,
			Provenance: types.Provenance{
				DocumentUUID:   "doc1",
				SectionUUID:    "sec1",
				SectionHeading: "Introduction",
			},
		}},
	}

	r := parentretriever.New(inner, store)
	hits, err := r.Retrieve(ctx, "test", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}

	// Text should be expanded to include both variants.
	if hits[0].Variant.Text != "First paragraph.\n\nSecond paragraph." {
		t.Errorf("expected expanded text, got %q", hits[0].Variant.Text)
	}
}

func TestParentRetrieverDedupes(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()

	doc := &types.Document{
		UUID: "doc1",
		Sections: []types.Section{{
			UUID:         "sec1",
			DocumentUUID: "doc1",
			Variants: []types.ContentVariant{
				{UUID: "v1", SectionUUID: "sec1", ContentType: types.ContentText, Text: "Text A."},
				{UUID: "v2", SectionUUID: "sec1", ContentType: types.ContentText, Text: "Text B."},
			},
		}},
	}
	if err := store.CreateDocument(ctx, doc); err != nil {
		t.Fatal(err)
	}

	// Inner retriever returns two hits from same section.
	inner := &mockRetriever{
		hits: []types.SearchHit{
			{
				Variant: types.ContentVariant{UUID: "v1", Text: "Text A."},
				Score:   0.9,
				Provenance: types.Provenance{DocumentUUID: "doc1", SectionUUID: "sec1"},
			},
			{
				Variant: types.ContentVariant{UUID: "v2", Text: "Text B."},
				Score:   0.8,
				Provenance: types.Provenance{DocumentUUID: "doc1", SectionUUID: "sec1"},
			},
		},
	}

	r := parentretriever.New(inner, store)
	hits, err := r.Retrieve(ctx, "test", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should be deduped to one hit (highest score).
	if len(hits) != 1 {
		t.Fatalf("expected 1 deduped hit, got %d", len(hits))
	}
	if hits[0].Score != 0.9 {
		t.Errorf("expected highest score 0.9, got %f", hits[0].Score)
	}
}
