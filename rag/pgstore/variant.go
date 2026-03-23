package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/urmzd/saige/rag/types"
)

// CreateVariant inserts a new content variant for a section.
func (s *Store) CreateVariant(ctx context.Context, variant *types.ContentVariant) error {
	secID, err := s.sectionID(ctx, variant.SectionUUID)
	if err != nil {
		return err
	}

	var emb *pgvector.Vector
	if variant.Embedding != nil {
		v := pgvector.NewVector(variant.Embedding)
		emb = &v
	}

	_, err = s.pool.Exec(ctx, variantCreateSQL,
		variant.UUID, secID, string(variant.ContentType), variant.MIMEType,
		variant.Data, variant.Text, emb, encodeMetadata(variant.Metadata),
	)
	if err != nil {
		return fmt.Errorf("create variant: %w", err)
	}
	return nil
}

// UpdateVariantEmbedding updates the embedding for an existing variant.
func (s *Store) UpdateVariantEmbedding(ctx context.Context, variantUUID string, embedding []float32) error {
	emb := pgvector.NewVector(embedding)
	tag, err := s.pool.Exec(ctx, variantUpdateEmbeddingSQL, emb, variantUUID)
	if err != nil {
		return fmt.Errorf("update variant embedding: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return types.ErrDocumentNotFound
	}
	return nil
}

// GetVariant retrieves a variant with its provenance information.
func (s *Store) GetVariant(ctx context.Context, variantUUID string) (*types.ContentVariant, *types.Provenance, error) {
	var (
		v    types.ContentVariant
		prov types.Provenance
		ct   string
		emb  *pgvector.Vector
		vMeta []byte
	)
	err := s.pool.QueryRow(ctx, variantGetSQL, variantUUID).Scan(
		&v.UUID, &ct, &v.MIMEType, &v.Data, &v.Text, &emb, &vMeta,
		&prov.SectionUUID, &prov.SectionHeading, &prov.SectionIndex,
		&prov.DocumentUUID, &prov.DocumentTitle, &prov.SourceURI,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, types.ErrVariantNotFound
		}
		return nil, nil, err
	}

	v.ContentType = types.ContentType(ct)
	v.SectionUUID = prov.SectionUUID
	if emb != nil {
		v.Embedding = emb.Slice()
	}
	v.Metadata = decodeMetadata(vMeta)

	return &v, &prov, nil
}
