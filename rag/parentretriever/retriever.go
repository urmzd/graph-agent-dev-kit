// Package parentretriever wraps any Retriever to expand hits with full parent section text.
package parentretriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/urmzd/saige/rag/types"
)

// Retriever wraps an inner retriever and expands each hit with the full parent section text.
type Retriever struct {
	inner types.Retriever
	store types.Store
}

// New creates a parent-context retriever wrapping the given inner retriever.
func New(inner types.Retriever, store types.Store) *Retriever {
	return &Retriever{inner: inner, store: store}
}

// Retrieve calls the inner retriever, deduplicates by section UUID (keeping the highest score
// per section), then replaces each hit's text with the concatenation of all variant texts
// in the parent section.
func (r *Retriever) Retrieve(ctx context.Context, query string, opts *types.SearchOptions) ([]types.SearchHit, error) {
	hits, err := r.inner.Retrieve(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("inner retrieve: %w", err)
	}

	if len(hits) == 0 {
		return hits, nil
	}

	// Dedupe by section UUID, keeping highest score.
	bestBySection := make(map[string]types.SearchHit)
	for _, hit := range hits {
		secUUID := hit.Provenance.SectionUUID
		if existing, ok := bestBySection[secUUID]; !ok || hit.Score > existing.Score {
			bestBySection[secUUID] = hit
		}
	}

	// Expand each hit with full section text.
	result := make([]types.SearchHit, 0, len(bestBySection))
	for _, hit := range bestBySection {
		docUUID := hit.Provenance.DocumentUUID
		sections, err := r.store.GetSections(ctx, docUUID)
		if err != nil {
			result = append(result, hit)
			continue
		}

		// Find the parent section.
		for _, sec := range sections {
			if sec.UUID == hit.Provenance.SectionUUID {
				var texts []string
				for _, v := range sec.Variants {
					if v.Text != "" {
						texts = append(texts, v.Text)
					}
				}
				if len(texts) > 0 {
					hit.Variant.Text = strings.Join(texts, "\n\n")
				}
				break
			}
		}

		result = append(result, hit)
	}

	// Sort by score descending.
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Score > result[j-1].Score; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	return result, nil
}
