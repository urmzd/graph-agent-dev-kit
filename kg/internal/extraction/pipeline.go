package extraction

import (
	"context"
	"fmt"

	"github.com/urmzd/graph-agent-dev-kit/kg/kgtypes"
)

// Pipeline orchestrates text → extract → embed → upsert.
type Pipeline struct {
	Extractor kgtypes.Extractor
	Embedder  kgtypes.Embedder
}

// NewPipeline creates a new extraction pipeline.
func NewPipeline(ext kgtypes.Extractor, emb kgtypes.Embedder) *Pipeline {
	return &Pipeline{Extractor: ext, Embedder: emb}
}

// EntityWithEmbedding is an extracted entity with its embedding.
type EntityWithEmbedding struct {
	Entity    kgtypes.ExtractedEntity
	Embedding []float32
}

// Process extracts entities and relations, then generates embeddings.
func (p *Pipeline) Process(ctx context.Context, text string) ([]EntityWithEmbedding, []kgtypes.ExtractedRelation, error) {
	entities, relations, err := p.Extractor.Extract(ctx, text)
	if err != nil {
		return nil, nil, fmt.Errorf("extract: %w", err)
	}

	results := make([]EntityWithEmbedding, len(entities))
	if p.Embedder != nil && len(entities) > 0 {
		texts := make([]string, len(entities))
		for i, e := range entities {
			texts[i] = fmt.Sprintf("%s %s", e.Name, e.Summary)
		}
		embeddings, err := p.Embedder.Embed(ctx, texts)
		if err == nil && len(embeddings) == len(entities) {
			for i, e := range entities {
				results[i] = EntityWithEmbedding{Entity: e, Embedding: embeddings[i]}
			}
			return results, relations, nil
		}
	}
	for i, e := range entities {
		results[i] = EntityWithEmbedding{Entity: e}
	}

	return results, relations, nil
}
