package ollama

import "context"

// OllamaEmbedder implements core.Embedder using the Ollama API.
type OllamaEmbedder struct {
	Client *Client
}

// NewEmbedder creates a new OllamaEmbedder.
func NewEmbedder(client *Client) *OllamaEmbedder {
	return &OllamaEmbedder{Client: client}
}

// Embed implements core.Embedder.
func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.Client.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = vec
	}
	return results, nil
}
