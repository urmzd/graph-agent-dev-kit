package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/urmzd/saige/rag/types"
)

// CreateDocument inserts a new document record.
func (s *Store) CreateDocument(ctx context.Context, doc *types.Document) error {
	_, err := s.pool.Exec(ctx, documentCreateSQL,
		doc.UUID, doc.SourceURI, doc.Fingerprint, doc.Title,
		encodeMetadata(doc.Metadata), doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	return nil
}

// GetDocument retrieves a document with all its sections and variants.
func (s *Store) GetDocument(ctx context.Context, uuid string) (*types.Document, error) {
	var doc types.Document
	var metaBytes []byte
	err := s.pool.QueryRow(ctx, documentGetSQL, uuid).Scan(
		&doc.UUID, &doc.SourceURI, &doc.Fingerprint, &doc.Title,
		&metaBytes, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, types.ErrDocumentNotFound
		}
		return nil, err
	}
	doc.Metadata = decodeMetadata(metaBytes)

	sections, err := s.GetSections(ctx, uuid)
	if err != nil {
		return nil, err
	}
	doc.Sections = sections

	return &doc, nil
}

// FindByFingerprint finds a document by content fingerprint.
func (s *Store) FindByFingerprint(ctx context.Context, fingerprint string) (*types.Document, error) {
	var docUUID string
	err := s.pool.QueryRow(ctx, documentFindFingerprintSQL, fingerprint).Scan(&docUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, types.ErrDocumentNotFound
		}
		return nil, err
	}
	return s.GetDocument(ctx, docUUID)
}

// DeleteDocument removes a document and all its sections/variants via CASCADE.
func (s *Store) DeleteDocument(ctx context.Context, uuid string) error {
	_, err := s.pool.Exec(ctx, documentDeleteSQL, uuid)
	return err
}

// StoreOriginal stores the original raw bytes for a document.
func (s *Store) StoreOriginal(ctx context.Context, documentUUID string, data []byte) error {
	docID, err := s.documentID(ctx, documentUUID)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, originalUpsertSQL, docID, data)
	return err
}

// GetOriginal retrieves the original raw bytes for a document.
func (s *Store) GetOriginal(ctx context.Context, documentUUID string) ([]byte, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, originalGetSQL, documentUUID).Scan(&data)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, types.ErrDocumentNotFound
		}
		return nil, err
	}
	return data, nil
}

// documentID resolves a document UUID to its internal BIGSERIAL id.
func (s *Store) documentID(ctx context.Context, docUUID string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, documentIDSQL, docUUID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, types.ErrDocumentNotFound
		}
		return 0, err
	}
	return id, nil
}
