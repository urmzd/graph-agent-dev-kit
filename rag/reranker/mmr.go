// Package reranker provides reranking strategies for search results.
package reranker

import (
	"context"
	"math"

	"github.com/urmzd/saige/rag/types"
)

// MMRConfig holds MMR reranker parameters.
type MMRConfig struct {
	Lambda float64 // Higher = more relevance vs diversity. Default 0.7.
}

// MMRReranker implements maximal marginal relevance reranking for diversity.
type MMRReranker struct {
	lambda float64
}

// NewMMR creates an MMR diversity reranker. If lambda is 0, defaults to 0.7.
func NewMMR(lambda float64) *MMRReranker {
	if lambda == 0 {
		lambda = 0.7
	}
	return &MMRReranker{lambda: lambda}
}

// Rerank selects hits greedily to maximize relevance while minimizing redundancy.
func (r *MMRReranker) Rerank(_ context.Context, _ string, hits []types.SearchHit) ([]types.SearchHit, error) {
	if len(hits) <= 1 {
		return hits, nil
	}

	selected := make([]types.SearchHit, 0, len(hits))
	remaining := make([]int, len(hits))
	for i := range remaining {
		remaining[i] = i
	}

	for len(remaining) > 0 {
		bestIdx := -1
		bestMMR := math.Inf(-1)

		for _, ri := range remaining {
			relevance := hits[ri].Score

			maxSim := 0.0
			for _, s := range selected {
				sim := cosineSimilarity(hits[ri].Variant.Embedding, s.Variant.Embedding)
				if sim > maxSim {
					maxSim = sim
				}
			}

			mmr := r.lambda*relevance - (1-r.lambda)*maxSim
			if mmr > bestMMR {
				bestMMR = mmr
				bestIdx = ri
			}
		}

		hits[bestIdx].Score = bestMMR
		selected = append(selected, hits[bestIdx])

		// Remove bestIdx from remaining.
		newRemaining := remaining[:0]
		for _, ri := range remaining {
			if ri != bestIdx {
				newRemaining = append(newRemaining, ri)
			}
		}
		remaining = newRemaining
	}

	return selected, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
