package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MigrationOptions configures schema migration.
type MigrationOptions struct {
	KGEmbeddingDim  int // default 768
	RAGEmbeddingDim int // default 768
}

// RunMigrations creates all tables and indexes idempotently.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, opts MigrationOptions) error {
	if opts.KGEmbeddingDim <= 0 {
		opts.KGEmbeddingDim = 768
	}
	if opts.RAGEmbeddingDim <= 0 {
		opts.RAGEmbeddingDim = 768
	}

	rendered := renderTemplate(migrationsTmpl, opts)

	var errs []error
	for _, stmt := range strings.Split(rendered, "---") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			errs = append(errs, fmt.Errorf("migration %q: %w", stmt[:min(len(stmt), 80)], err))
		}
	}
	return errors.Join(errs...)
}
