package memstore

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/urmzd/saige/rag/types"
)

func makeEmbedding(dim int) []float32 {
	emb := make([]float32, dim)
	for i := range emb {
		emb[i] = rand.Float32()
	}
	return emb
}

func BenchmarkCreateDocument(b *testing.B) {
	s := New()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := &types.Document{
			UUID:      fmt.Sprintf("doc-%d", i),
			SourceURI: fmt.Sprintf("bench://doc-%d", i),
			Sections: []types.Section{{
				UUID:         fmt.Sprintf("sec-%d", i),
				DocumentUUID: fmt.Sprintf("doc-%d", i),
				Variants: []types.ContentVariant{{
					UUID:      fmt.Sprintf("var-%d", i),
					Text:      "benchmark text content",
					Embedding: makeEmbedding(768),
				}},
			}},
		}
		s.CreateDocument(ctx, doc)
	}
}

func BenchmarkSearchByEmbedding(b *testing.B) {
	s := New()
	ctx := context.Background()

	for i := range 100 {
		doc := &types.Document{
			UUID:      fmt.Sprintf("doc-%d", i),
			SourceURI: fmt.Sprintf("bench://doc-%d", i),
			Sections: []types.Section{{
				UUID:         fmt.Sprintf("sec-%d", i),
				DocumentUUID: fmt.Sprintf("doc-%d", i),
				Variants: []types.ContentVariant{{
					UUID:      fmt.Sprintf("var-%d", i),
					Text:      "benchmark text content for search",
					Embedding: makeEmbedding(768),
				}},
			}},
		}
		s.CreateDocument(ctx, doc)
	}

	query := makeEmbedding(768)
	opts := &types.SearchOptions{Limit: 10}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.SearchByEmbedding(ctx, query, opts)
	}
}
