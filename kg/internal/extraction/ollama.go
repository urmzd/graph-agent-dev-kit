package extraction

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/urmzd/graph-agent-dev-kit/agent/provider/ollama"
	"github.com/urmzd/graph-agent-dev-kit/kg/kgtypes"
)

// extractionResponse is the JSON schema returned by the LLM.
type extractionResponse struct {
	Entities  []kgtypes.ExtractedEntity  `json:"entities"`
	Relations []kgtypes.ExtractedRelation `json:"relations"`
}

// OllamaExtractor uses an Ollama client to extract entities and relations via structured output.
type OllamaExtractor struct {
	client *ollama.Client
}

// NewOllamaExtractor creates a new OllamaExtractor.
func NewOllamaExtractor(client *ollama.Client) *OllamaExtractor {
	return &OllamaExtractor{client: client}
}

// jsonFormat is the Ollama structured output schema for extraction.
var jsonFormat = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"entities": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":    map[string]any{"type": "string"},
					"type":    map[string]any{"type": "string"},
					"summary": map[string]any{"type": "string"},
				},
				"required": []string{"name", "type", "summary"},
			},
		},
		"relations": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{"type": "string"},
					"target": map[string]any{"type": "string"},
					"type":   map[string]any{"type": "string"},
					"fact":   map[string]any{"type": "string"},
				},
				"required": []string{"source", "target", "type", "fact"},
			},
		},
	},
	"required": []string{"entities", "relations"},
}

// Extract implements kgtypes.Extractor.
func (e *OllamaExtractor) Extract(ctx context.Context, text string) ([]kgtypes.ExtractedEntity, []kgtypes.ExtractedRelation, error) {
	prompt := BuildExtractionPrompt(text, nil)

	raw, err := e.client.GenerateWithModel(ctx, prompt, e.client.Model, jsonFormat, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("ollama extraction: %w", err)
	}

	var resp extractionResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, nil, fmt.Errorf("parse extraction response: %w", err)
	}

	return resp.Entities, resp.Relations, nil
}
