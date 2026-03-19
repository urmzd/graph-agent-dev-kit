// Package hyde implements Hypothetical Document Embeddings (HyDE) query transformation.
package hyde

import (
	"context"
	"fmt"

	"github.com/urmzd/graph-agent-dev-kit/rag/ragtypes"
)

// DefaultPromptTemplate is the default prompt for generating hypothetical documents.
const DefaultPromptTemplate = "Write a short passage that would answer the following question:\n\n%s\n\nPassage:"

// Config holds HyDE transformer parameters.
type Config struct {
	LLM             ragtypes.LLM
	NumHypothetical int
	PromptTemplate  string
}

// Transformer generates hypothetical answer documents via an LLM to improve retrieval recall.
type Transformer struct {
	cfg Config
}

// New creates a HyDE query transformer. NumHypothetical defaults to 3 if <= 0.
func New(cfg Config) *Transformer {
	if cfg.NumHypothetical <= 0 {
		cfg.NumHypothetical = 3
	}
	if cfg.PromptTemplate == "" {
		cfg.PromptTemplate = DefaultPromptTemplate
	}
	return &Transformer{cfg: cfg}
}

// Transform generates hypothetical documents and returns the original query plus all hypotheticals.
func (t *Transformer) Transform(ctx context.Context, query string) ([]string, error) {
	queries := make([]string, 0, 1+t.cfg.NumHypothetical)
	queries = append(queries, query)

	prompt := fmt.Sprintf(t.cfg.PromptTemplate, query)
	for i := 0; i < t.cfg.NumHypothetical; i++ {
		hypothetical, err := t.cfg.LLM.Generate(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("generate hypothetical %d: %w", i, err)
		}
		queries = append(queries, hypothetical)
	}

	return queries, nil
}
