package pipeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/urmzd/graph-agent-dev-kit/rag/bm25retriever"
	"github.com/urmzd/graph-agent-dev-kit/rag/internal/pipeline"
	"github.com/urmzd/graph-agent-dev-kit/rag/memstore"
	"github.com/urmzd/graph-agent-dev-kit/rag/ragtypes"
	"github.com/urmzd/graph-agent-dev-kit/rag/vectorretriever"
)

type simpleExtractor struct{}

func (e *simpleExtractor) Extract(_ context.Context, raw *ragtypes.RawDocument) (*ragtypes.Document, error) {
	docUUID := "test-doc"
	secUUID := "test-sec"
	varUUID := "test-var"
	return &ragtypes.Document{
		UUID:      docUUID,
		SourceURI: raw.SourceURI,
		Title:     "Test Document",
		Sections: []ragtypes.Section{{
			UUID:         secUUID,
			DocumentUUID: docUUID,
			Index:        0,
			Variants: []ragtypes.ContentVariant{{
				UUID:        varUUID,
				SectionUUID: secUUID,
				ContentType: ragtypes.ContentText,
				MIMEType:    "text/plain",
				Text:        string(raw.Data),
			}},
		}},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

type simpleEmbedder struct{}

func (e *simpleEmbedder) Register(_ ragtypes.ContentType, _ ragtypes.VariantEmbedder) {}
func (e *simpleEmbedder) Embed(_ context.Context, variants []ragtypes.ContentVariant) ([][]float32, error) {
	result := make([][]float32, len(variants))
	for i := range variants {
		// Simple hash-based embedding for testing.
		vec := make([]float32, 4)
		text := variants[i].Text
		for j, ch := range text {
			vec[j%4] += float32(ch)
		}
		result[i] = vec
	}
	return result, nil
}

// trackingIndexer wraps BM25 retriever and tracks calls.
type trackingIndexer struct {
	*bm25retriever.Retriever
	indexCalled  bool
	removeCalled bool
}

func (t *trackingIndexer) Index(ctx context.Context, doc *ragtypes.Document) error {
	t.indexCalled = true
	return t.Retriever.Index(ctx, doc)
}

func (t *trackingIndexer) Remove(ctx context.Context, docUUID string) error {
	t.removeCalled = true
	return t.Retriever.Remove(ctx, docUUID)
}

func TestPipelineIndexerIntegration(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()

	bm25 := bm25retriever.New(store, nil)
	tracker := &trackingIndexer{Retriever: bm25}

	pipe := pipeline.New(pipeline.Config{
		Store:            store,
		ContentExtractor: &simpleExtractor{},
		Retrievers:       []ragtypes.Retriever{tracker},
	})

	// Ingest should call Index.
	result, err := pipe.Ingest(ctx, &ragtypes.RawDocument{
		SourceURI: "test://doc",
		Data:      []byte("the quick brown fox jumps over the lazy dog"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !tracker.indexCalled {
		t.Error("expected Indexer.Index to be called during ingest")
	}

	// Delete should call Remove.
	err = pipe.Delete(ctx, result.DocumentUUID)
	if err != nil {
		t.Fatal(err)
	}
	if !tracker.removeCalled {
		t.Error("expected Indexer.Remove to be called during delete")
	}
}

func TestPipelineHybridSearch(t *testing.T) {
	ctx := context.Background()
	store := memstore.New()
	embedders := &simpleEmbedder{}

	bm25 := bm25retriever.New(store, nil)
	vecRetriever := vectorretriever.New(store, embedders)

	pipe := pipeline.New(pipeline.Config{
		Store:            store,
		ContentExtractor: &simpleExtractor{},
		Embedders:        embedders,
		Retrievers:       []ragtypes.Retriever{vecRetriever, bm25},
	})

	_, err := pipe.Ingest(ctx, &ragtypes.RawDocument{
		SourceURI: "test://doc",
		Data:      []byte("the quick brown fox jumps over the lazy dog"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Search should combine results from both retrievers via RRF.
	sr, err := pipe.Search(ctx, "quick brown fox", ragtypes.WithLimit(5))
	if err != nil {
		t.Fatal(err)
	}

	if len(sr.Hits) == 0 {
		t.Error("expected at least one hit from hybrid search")
	}
}
