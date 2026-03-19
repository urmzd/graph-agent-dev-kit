package google

import (
	"context"

	"google.golang.org/genai"
)

// Embedder implements core.Embedder using the official Google GenAI SDK.
type Embedder struct {
	client *genai.Client
	model  string
}

// NewEmbedder creates a new Google embedder.
func NewEmbedder(ctx context.Context, apiKey, model string) (*Embedder, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}
	return &Embedder{client: client, model: model}, nil
}

// Embed implements core.Embedder.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		resp, err := e.client.Models.EmbedContent(ctx, e.model, genai.Text(text), nil)
		if err != nil {
			return nil, err
		}
		if len(resp.Embeddings) > 0 {
			embeddings[i] = resp.Embeddings[0].Values
		}
	}
	return embeddings, nil
}
