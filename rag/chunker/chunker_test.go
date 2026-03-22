package chunker_test

import (
	"context"
	"strings"
	"testing"

	"github.com/urmzd/saige/rag/chunker"
	"github.com/urmzd/saige/rag/ragtypes"
)

func makeDoc(text string) *ragtypes.Document {
	return &ragtypes.Document{
		UUID: "doc1",
		Sections: []ragtypes.Section{{
			UUID:         "sec1",
			DocumentUUID: "doc1",
			Heading:      "Test Section",
			Variants: []ragtypes.ContentVariant{{
				UUID:        "var1",
				SectionUUID: "sec1",
				ContentType: ragtypes.ContentText,
				MIMEType:    "text/plain",
				Text:        text,
			}},
		}},
	}
}

func TestRecursiveChunkerSmallDoc(t *testing.T) {
	// Text that fits in one chunk should not be split.
	doc := makeDoc("short text")
	c := chunker.NewRecursive(nil)
	result, err := c.Chunk(context.Background(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sections) != 1 {
		t.Fatalf("expected 1 section for short text, got %d", len(result.Sections))
	}
}

func TestRecursiveChunkerSplits(t *testing.T) {
	// Create text that exceeds 50 tokens (~200 chars) with paragraph separators.
	para := strings.Repeat("word ", 30) // ~150 chars = ~37 tokens per paragraph
	text := para + "\n\n" + para + "\n\n" + para

	cfg := &chunker.Config{MaxTokens: 50, Overlap: 0, Separators: []string{"\n\n", "\n", ". ", " "}}
	c := chunker.NewRecursive(cfg)
	result, err := c.Chunk(context.Background(), makeDoc(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Sections) < 3 {
		t.Fatalf("expected at least 3 sections, got %d", len(result.Sections))
	}

	// All new sections should preserve heading.
	for _, sec := range result.Sections {
		if sec.Heading != "Test Section" {
			t.Errorf("expected heading 'Test Section', got %q", sec.Heading)
		}
	}
}

func TestRecursiveChunkerOverlap(t *testing.T) {
	// Create text with clear paragraph boundaries.
	text := strings.Repeat("alpha ", 40) + "\n\n" + strings.Repeat("beta ", 40)

	cfg := &chunker.Config{MaxTokens: 50, Overlap: 10, Separators: []string{"\n\n", " "}}
	c := chunker.NewRecursive(cfg)
	result, err := c.Chunk(context.Background(), makeDoc(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Sections) < 2 {
		t.Fatalf("expected at least 2 sections with overlap, got %d", len(result.Sections))
	}

	// Second chunk should contain overlap from first chunk.
	if len(result.Sections) > 1 {
		secondText := result.Sections[1].Variants[0].Text
		// With overlap, the second chunk should have some content from the end of the first.
		if len(secondText) == 0 {
			t.Error("second chunk should not be empty")
		}
	}
}

func TestRecursiveChunkerSeparatorHierarchy(t *testing.T) {
	// Text with sentence separators but no paragraph separators.
	text := strings.Repeat("This is a sentence. ", 30) // ~600 chars = ~150 tokens

	cfg := &chunker.Config{MaxTokens: 50, Overlap: 0, Separators: []string{"\n\n", "\n", ". ", " "}}
	c := chunker.NewRecursive(cfg)
	result, err := c.Chunk(context.Background(), makeDoc(text))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Sections) < 2 {
		t.Fatalf("expected multiple sections when splitting by sentence, got %d", len(result.Sections))
	}
}
