// Package pgstore implements rag/types.Store using PostgreSQL + pgvector.
package pgstore

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/urmzd/saige/rag/types"
)

var _ types.Store = (*Store)(nil)

// Store implements types.Store backed by PostgreSQL with pgvector.
type Store struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewStore creates a new PostgreSQL-backed RAG store.
func NewStore(pool *pgxpool.Pool, logger *slog.Logger) *Store {
	if logger == nil {
		logger = slog.Default()
	}
	return &Store{pool: pool, logger: logger}
}

// Close is a no-op; the pool is externally managed.
func (s *Store) Close(_ context.Context) error {
	return nil
}

// encodeMetadata marshals metadata to JSON for JSONB columns.
func encodeMetadata(meta map[string]string) []byte {
	if meta == nil {
		return nil
	}
	b, _ := json.Marshal(meta)
	return b
}

// decodeMetadata unmarshals JSONB bytes to metadata map.
func decodeMetadata(b []byte) map[string]string {
	if b == nil {
		return nil
	}
	var m map[string]string
	_ = json.Unmarshal(b, &m)
	return m
}

// mergeMetadata merges document and variant metadata.
func mergeMetadata(docMeta, variantMeta map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range docMeta {
		merged[k] = v
	}
	for k, v := range variantMeta {
		merged[k] = v
	}
	return merged
}

// matchFilters checks if metadata matches all filters.
func matchFilters(meta map[string]string, filters []types.MetadataFilter) bool {
	for _, f := range filters {
		val, ok := meta[f.Key]
		switch f.Op {
		case types.FilterEq:
			if !ok || val != f.Value {
				return false
			}
		case types.FilterNeq:
			if ok && val == f.Value {
				return false
			}
		case types.FilterContains:
			if !ok || !strings.Contains(val, f.Value) {
				return false
			}
		}
	}
	return true
}
