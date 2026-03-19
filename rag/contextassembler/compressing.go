// Package contextassembler provides context assembly strategies for RAG pipelines.
package contextassembler

import (
	"context"
	"fmt"

	"github.com/urmzd/graph-agent-dev-kit/rag/ragtypes"
)

// CompressingAssembler uses an LLM to extract query-relevant sentences from each hit
// before assembling context with citations.
type CompressingAssembler struct {
	LLM       ragtypes.LLM
	MaxTokens int
}

// NewCompressing creates a compressing context assembler.
func NewCompressing(llm ragtypes.LLM, maxTokens int) *CompressingAssembler {
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &CompressingAssembler{LLM: llm, MaxTokens: maxTokens}
}

const compressionPrompt = `Extract only the sentences from the following text that are relevant to the query. Return only the relevant sentences, nothing else. If nothing is relevant, return "N/A".

Query: %s

Text: %s

Relevant sentences:`

// Assemble compresses each hit's text via the LLM and builds context with citations.
func (a *CompressingAssembler) Assemble(ctx context.Context, query string, hits []ragtypes.SearchHit) (*ragtypes.AssembledContext, error) {
	var blocks []ragtypes.ContextBlock
	var parts []string
	tokenCount := 0

	for i, hit := range hits {
		prompt := fmt.Sprintf(compressionPrompt, query, hit.Variant.Text)
		compressed, err := a.LLM.Generate(ctx, prompt)
		if err != nil {
			return nil, fmt.Errorf("compress hit %d: %w", i, err)
		}

		if compressed == "" || compressed == "N/A" {
			continue
		}

		tokens := len(compressed) / 4
		if a.MaxTokens > 0 && tokenCount+tokens > a.MaxTokens {
			break
		}
		tokenCount += tokens

		citation := fmt.Sprintf("[%d]", len(blocks)+1)
		blocks = append(blocks, ragtypes.ContextBlock{
			Text:       compressed,
			Citation:   citation,
			Provenance: hit.Provenance, // Original provenance preserved.
		})

		source := hit.Provenance.SourceURI
		if source == "" {
			source = hit.Provenance.DocumentTitle
		}
		parts = append(parts, fmt.Sprintf("%s %s (Source: %s)", citation, compressed, source))
	}

	promptText := fmt.Sprintf("Context for query %q:\n\n%s", query, joinStrings(parts, "\n\n"))

	return &ragtypes.AssembledContext{
		Prompt:     promptText,
		Blocks:     blocks,
		TokenCount: tokenCount,
	}, nil
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += sep + p
	}
	return result
}
