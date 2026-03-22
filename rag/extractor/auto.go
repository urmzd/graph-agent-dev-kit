package extractor

import (
	"context"
	"strings"

	"github.com/urmzd/saige/rag/types"
)

// Auto dispatches to the appropriate extractor based on MIME type.
type Auto struct {
	extractors map[string]types.ContentExtractor
	fallback   types.ContentExtractor
}

// NewAuto creates an Auto extractor with built-in support for text/plain, text/html, and application/pdf.
func NewAuto() *Auto {
	return &Auto{
		extractors: map[string]types.ContentExtractor{
			"text/plain":      &PlainText{},
			"text/html":       &HTML{},
			"application/pdf": &PDF{},
		},
		fallback: &PlainText{},
	}
}

// Register adds a custom extractor for a MIME type.
func (a *Auto) Register(mimeType string, extractor types.ContentExtractor) {
	a.extractors[mimeType] = extractor
}

// Extract dispatches to the appropriate extractor based on the document's MIME type.
func (a *Auto) Extract(ctx context.Context, raw *types.RawDocument) (*types.Document, error) {
	mime := normalizeMIME(raw.MIMEType)

	if ext, ok := a.extractors[mime]; ok {
		return ext.Extract(ctx, raw)
	}

	// Try prefix matching (e.g., "text/markdown" → "text/plain").
	if strings.HasPrefix(mime, "text/") {
		return a.fallback.Extract(ctx, raw)
	}

	return nil, types.ErrUnsupportedMIMEType
}

func normalizeMIME(mime string) string {
	// Strip parameters (e.g., "text/html; charset=utf-8" → "text/html").
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = mime[:idx]
	}
	return strings.TrimSpace(strings.ToLower(mime))
}
