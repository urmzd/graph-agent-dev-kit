package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/urmzd/saige/rag/types"
)

// CreateSection inserts a new section for a document.
func (s *Store) CreateSection(ctx context.Context, section *types.Section) error {
	docID, err := s.documentID(ctx, section.DocumentUUID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, sectionCreateSQL,
		section.UUID, docID, section.Index, section.Heading,
	)
	if err != nil {
		return fmt.Errorf("create section: %w", err)
	}
	return nil
}

// GetSections retrieves all sections (with variants) for a document.
func (s *Store) GetSections(ctx context.Context, documentUUID string) ([]types.Section, error) {
	rows, err := s.pool.Query(ctx, sectionGetSQL, documentUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sectionMap := make(map[string]*types.Section)
	var sectionOrder []string

	for rows.Next() {
		var (
			sUUID, sHeading            string
			sIdx                       int
			vUUID, vContentType, vMIME *string
			vData                      []byte
			vText                      *string
			vEmbedding                 *pgvector.Vector
			vMeta                      []byte
		)
		if err := rows.Scan(
			&sUUID, &sIdx, &sHeading,
			&vUUID, &vContentType, &vMIME, &vData, &vText, &vEmbedding, &vMeta,
		); err != nil {
			return nil, err
		}

		sec, ok := sectionMap[sUUID]
		if !ok {
			sec = &types.Section{
				UUID:         sUUID,
				DocumentUUID: documentUUID,
				Index:        sIdx,
				Heading:      sHeading,
			}
			sectionMap[sUUID] = sec
			sectionOrder = append(sectionOrder, sUUID)
		}

		if vUUID != nil {
			variant := types.ContentVariant{
				UUID:        *vUUID,
				SectionUUID: sUUID,
				ContentType: types.ContentType(derefStr(vContentType)),
				MIMEType:    derefStr(vMIME),
				Data:        vData,
				Text:        derefStr(vText),
				Metadata:    decodeMetadata(vMeta),
			}
			if vEmbedding != nil {
				variant.Embedding = vEmbedding.Slice()
			}
			sec.Variants = append(sec.Variants, variant)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	sections := make([]types.Section, 0, len(sectionOrder))
	for _, uuid := range sectionOrder {
		sections = append(sections, *sectionMap[uuid])
	}
	return sections, nil
}

// sectionID resolves a section UUID to its internal BIGSERIAL id.
func (s *Store) sectionID(ctx context.Context, sectionUUID string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, sectionIDSQL, sectionUUID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, types.ErrDocumentNotFound
		}
		return 0, err
	}
	return id, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
