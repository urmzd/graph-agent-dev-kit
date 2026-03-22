// Package bm25retriever implements a BM25 lexical retriever with an in-memory inverted index.
package bm25retriever

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/urmzd/saige/rag/ragtypes"
)

// Config holds BM25 parameters.
type Config struct {
	K1 float64
	B  float64
}

// DefaultConfig returns the standard BM25 parameters.
func DefaultConfig() *Config {
	return &Config{K1: 1.2, B: 0.75}
}

type posting struct {
	variantUUID string
	termFreq    float64
}

// Retriever implements both ragtypes.Retriever and ragtypes.Indexer using BM25 scoring.
type Retriever struct {
	mu       sync.RWMutex
	store    ragtypes.Store
	cfg      Config
	index    map[string][]posting // term -> postings
	docLen   map[string]float64   // variantUUID -> document length (token count)
	avgDL    float64
	docCount int
	// Track which variants belong to which document for Remove.
	docVariants map[string][]string // documentUUID -> []variantUUID
}

// New creates a BM25 retriever. If cfg is nil, defaults are used.
func New(store ragtypes.Store, cfg *Config) *Retriever {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Retriever{
		store:       store,
		cfg:         *cfg,
		index:       make(map[string][]posting),
		docLen:      make(map[string]float64),
		docVariants: make(map[string][]string),
	}
}

// tokenize splits text into lowercase tokens.
func tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// Index indexes all text variants in a document.
func (r *Retriever) Index(_ context.Context, doc *ragtypes.Document) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, sec := range doc.Sections {
		for _, v := range sec.Variants {
			if v.ContentType != ragtypes.ContentText || v.Text == "" {
				continue
			}
			tokens := tokenize(v.Text)
			dl := float64(len(tokens))

			// Count term frequencies.
			tf := make(map[string]float64)
			for _, t := range tokens {
				tf[t]++
			}

			for term, freq := range tf {
				r.index[term] = append(r.index[term], posting{
					variantUUID: v.UUID,
					termFreq:    freq,
				})
			}

			r.docLen[v.UUID] = dl
			r.docVariants[doc.UUID] = append(r.docVariants[doc.UUID], v.UUID)
			r.docCount++
		}
	}

	r.recomputeAvgDL()
	return nil
}

// Remove removes all postings for variants belonging to the given document.
func (r *Retriever) Remove(_ context.Context, documentUUID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	variants, ok := r.docVariants[documentUUID]
	if !ok {
		return nil
	}

	variantSet := make(map[string]bool, len(variants))
	for _, v := range variants {
		variantSet[v] = true
	}

	// Remove postings.
	for term, postings := range r.index {
		filtered := postings[:0]
		for _, p := range postings {
			if !variantSet[p.variantUUID] {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			delete(r.index, term)
		} else {
			r.index[term] = filtered
		}
	}

	// Remove doc lengths and variant tracking.
	for _, v := range variants {
		delete(r.docLen, v)
		r.docCount--
	}
	delete(r.docVariants, documentUUID)

	r.recomputeAvgDL()
	return nil
}

func (r *Retriever) recomputeAvgDL() {
	if r.docCount == 0 {
		r.avgDL = 0
		return
	}
	total := 0.0
	for _, dl := range r.docLen {
		total += dl
	}
	r.avgDL = total / float64(r.docCount)
}

// Retrieve computes BM25 scores for each variant matching the query terms.
func (r *Retriever) Retrieve(ctx context.Context, query string, opts *ragtypes.SearchOptions) ([]ragtypes.SearchHit, error) {
	r.mu.RLock()
	queryTokens := tokenize(query)
	if len(queryTokens) == 0 || r.docCount == 0 {
		r.mu.RUnlock()
		return nil, nil
	}

	scores := make(map[string]float64)
	N := float64(r.docCount)
	k1 := r.cfg.K1
	b := r.cfg.B

	for _, term := range queryTokens {
		postings, ok := r.index[term]
		if !ok {
			continue
		}
		df := float64(len(postings))
		idf := math.Log(1 + (N-df+0.5)/(df+0.5))

		for _, p := range postings {
			dl := r.docLen[p.variantUUID]
			tf := p.termFreq
			score := idf * (tf * (k1 + 1)) / (tf + k1*(1-b+b*dl/r.avgDL))
			scores[p.variantUUID] += score
		}
	}
	r.mu.RUnlock()

	// Build search hits via store lookups.
	limit := 10
	if opts != nil && opts.Limit > 0 {
		limit = opts.Limit
	}

	type scoredVariant struct {
		uuid  string
		score float64
	}
	ranked := make([]scoredVariant, 0, len(scores))
	for uuid, score := range scores {
		if opts != nil && opts.MinScore > 0 && score < opts.MinScore {
			continue
		}
		ranked = append(ranked, scoredVariant{uuid, score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	hits := make([]ragtypes.SearchHit, 0, len(ranked))
	for _, sv := range ranked {
		variant, prov, err := r.store.GetVariant(ctx, sv.uuid)
		if err != nil {
			continue
		}

		// Apply content type filters.
		if opts != nil && len(opts.ContentTypes) > 0 {
			match := false
			for _, ct := range opts.ContentTypes {
				if variant.ContentType == ct {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// Apply metadata filters.
		if opts != nil && len(opts.MetadataFilters) > 0 {
			if !matchFilters(variant.Metadata, opts.MetadataFilters) {
				continue
			}
		}

		hits = append(hits, ragtypes.SearchHit{
			Variant:    *variant,
			Score:      sv.score,
			Provenance: *prov,
		})
	}

	return hits, nil
}

func matchFilters(meta map[string]string, filters []ragtypes.MetadataFilter) bool {
	for _, f := range filters {
		val, ok := meta[f.Key]
		switch f.Op {
		case ragtypes.FilterEq:
			if !ok || val != f.Value {
				return false
			}
		case ragtypes.FilterNeq:
			if ok && val == f.Value {
				return false
			}
		case ragtypes.FilterContains:
			if !ok || !strings.Contains(val, f.Value) {
				return false
			}
		}
	}
	return true
}
