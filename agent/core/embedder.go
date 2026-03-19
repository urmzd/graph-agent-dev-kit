package core

import "context"

// Embedder generates vector embeddings from text.
// Batch-first API: single embed = Embed(ctx, []string{text}).
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
