package openai

import (
	"context"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Embedder implements core.Embedder using the official OpenAI SDK.
type Embedder struct {
	client openai.Client
	model  openai.EmbeddingModel
}

// NewEmbedder creates a new OpenAI embedder.
func NewEmbedder(apiKey, model string, opts ...Option) *Embedder {
	cfg := &config{}
	for _, o := range opts {
		o(cfg)
	}
	clientOpts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if cfg.baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(cfg.baseURL))
	}
	return &Embedder{
		client: openai.NewClient(clientOpts...),
		model:  openai.EmbeddingModel(model),
	}
}

// Embed implements core.Embedder.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	resp, err := e.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfArrayOfStrings: texts,
		},
		Model: e.model,
	})
	if err != nil {
		return nil, classifyOpenAIError(err)
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range resp.Data {
		if int(d.Index) < len(embeddings) {
			f32 := make([]float32, len(d.Embedding))
			for j, v := range d.Embedding {
				f32[j] = float32(v)
			}
			embeddings[d.Index] = f32
		}
	}
	return embeddings, nil
}
