package kg

import (
	"github.com/urmzd/saige/agent/provider/ollama"
	"github.com/urmzd/saige/kg/internal/extraction"
	"github.com/urmzd/saige/kg/kgtypes"
)

// NewOllamaExtractor creates an Extractor backed by an Ollama client.
func NewOllamaExtractor(client *ollama.Client) kgtypes.Extractor {
	return extraction.NewOllamaExtractor(client)
}

// NewOllamaEmbedder creates an Embedder backed by an Ollama client.
// This delegates to adk's ollama.NewEmbedder which implements
// the batch Embed(ctx, []string) API.
func NewOllamaEmbedder(client *ollama.Client) kgtypes.Embedder {
	return ollama.NewEmbedder(client)
}
