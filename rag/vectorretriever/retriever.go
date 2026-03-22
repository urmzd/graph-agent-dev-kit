// Package vectorretriever implements a Retriever that embeds queries and searches a Store.
package vectorretriever

import (
	"context"
	"fmt"

	"github.com/urmzd/saige/rag/ragtypes"
)

// VectorRetriever embeds a query and delegates to store.SearchByEmbedding.
type VectorRetriever struct {
	embedders ragtypes.EmbedderRegistry
	store     ragtypes.Store
}

// New creates a VectorRetriever with the given store and embedder registry.
func New(store ragtypes.Store, embedders ragtypes.EmbedderRegistry) *VectorRetriever {
	return &VectorRetriever{store: store, embedders: embedders}
}

// Retrieve embeds the query as a text variant and searches the store.
func (r *VectorRetriever) Retrieve(ctx context.Context, query string, opts *ragtypes.SearchOptions) ([]ragtypes.SearchHit, error) {
	queryVariant := ragtypes.ContentVariant{
		ContentType: ragtypes.ContentText,
		Text:        query,
	}
	embeddings, err := r.embedders.Embed(ctx, []ragtypes.ContentVariant{queryVariant})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	hits, err := r.store.SearchByEmbedding(ctx, embeddings[0], opts)
	if err != nil {
		return nil, fmt.Errorf("search by embedding: %w", err)
	}
	return hits, nil
}
