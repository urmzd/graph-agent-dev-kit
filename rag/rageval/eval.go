// Package rageval provides evaluation metrics for RAG pipelines.
package rageval

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/urmzd/graph-agent-dev-kit/rag/ragtypes"
)

// EvalResult holds computed evaluation metrics.
type EvalResult struct {
	ContextPrecision float64 `json:"context_precision"`
	ContextRecall    float64 `json:"context_recall"`
	Faithfulness     float64 `json:"faithfulness,omitempty"`
	AnswerRelevancy  float64 `json:"answer_relevancy,omitempty"`
}

// EvalCase defines a single evaluation case with ground truth.
type EvalCase struct {
	Query         string   `json:"query"`
	GroundTruth   string   `json:"ground_truth"`
	RelevantUUIDs []string `json:"relevant_uuids"`
	Response      string   `json:"response"`
}

// EvalOptions configures which metrics to compute.
type EvalOptions struct {
	LLM       ragtypes.LLM
	Embedders ragtypes.EmbedderRegistry
}

// ContextPrecision computes precision@k at each relevant UUID's rank, averaged.
// This is the Average Precision metric.
func ContextPrecision(hits []ragtypes.SearchHit, relevantUUIDs []string) float64 {
	if len(relevantUUIDs) == 0 {
		return 0
	}

	relevant := make(map[string]bool, len(relevantUUIDs))
	for _, uuid := range relevantUUIDs {
		relevant[uuid] = true
	}

	sum := 0.0
	found := 0
	for i, hit := range hits {
		if relevant[hit.Variant.UUID] {
			found++
			sum += float64(found) / float64(i+1)
		}
	}

	if found == 0 {
		return 0
	}
	return sum / float64(len(relevantUUIDs))
}

// ContextRecall computes the fraction of relevant UUIDs present in the results.
func ContextRecall(hits []ragtypes.SearchHit, relevantUUIDs []string) float64 {
	if len(relevantUUIDs) == 0 {
		return 0
	}

	hitUUIDs := make(map[string]bool, len(hits))
	for _, hit := range hits {
		hitUUIDs[hit.Variant.UUID] = true
	}

	found := 0
	for _, uuid := range relevantUUIDs {
		if hitUUIDs[uuid] {
			found++
		}
	}

	return float64(found) / float64(len(relevantUUIDs))
}

const faithfulnessPrompt = `Given the following context and response, decompose the response into individual claims, then check if each claim is supported by the context. Return a score between 0.0 and 1.0 representing the fraction of claims that are supported.

Context:
%s

Response:
%s

Return ONLY a decimal number between 0.0 and 1.0:`

// Faithfulness uses an LLM to decompose the response into claims and check each against context.
func Faithfulness(ctx context.Context, response string, contextText string, llm ragtypes.LLM) (float64, error) {
	prompt := fmt.Sprintf(faithfulnessPrompt, contextText, response)
	result, err := llm.Generate(ctx, prompt)
	if err != nil {
		return 0, fmt.Errorf("faithfulness check: %w", err)
	}

	var score float64
	result = strings.TrimSpace(result)
	_, err = fmt.Sscanf(result, "%f", &score)
	if err != nil {
		return 0, fmt.Errorf("parse faithfulness score %q: %w", result, err)
	}

	return math.Max(0, math.Min(1, score)), nil
}

// AnswerRelevancy computes cosine similarity between query and response embeddings.
func AnswerRelevancy(ctx context.Context, query, response string, embedders ragtypes.EmbedderRegistry) (float64, error) {
	variants := []ragtypes.ContentVariant{
		{ContentType: ragtypes.ContentText, Text: query},
		{ContentType: ragtypes.ContentText, Text: response},
	}

	embeddings, err := embedders.Embed(ctx, variants)
	if err != nil {
		return 0, fmt.Errorf("embed for relevancy: %w", err)
	}

	return cosineSimilarity(embeddings[0], embeddings[1]), nil
}

// Evaluate runs all cases through the pipeline and computes all applicable metrics.
func Evaluate(ctx context.Context, cases []EvalCase, pipe ragtypes.Pipeline, opts *EvalOptions) ([]EvalResult, error) {
	results := make([]EvalResult, len(cases))

	for i, tc := range cases {
		sr, err := pipe.Search(ctx, tc.Query, ragtypes.WithLimit(20))
		if err != nil {
			return nil, fmt.Errorf("evaluate case %d: %w", i, err)
		}

		results[i].ContextPrecision = ContextPrecision(sr.Hits, tc.RelevantUUIDs)
		results[i].ContextRecall = ContextRecall(sr.Hits, tc.RelevantUUIDs)

		if opts != nil && opts.LLM != nil && tc.Response != "" {
			var contextParts []string
			for _, hit := range sr.Hits {
				contextParts = append(contextParts, hit.Variant.Text)
			}
			contextText := strings.Join(contextParts, "\n\n")

			faith, err := Faithfulness(ctx, tc.Response, contextText, opts.LLM)
			if err == nil {
				results[i].Faithfulness = faith
			}
		}

		if opts != nil && opts.Embedders != nil && tc.Response != "" {
			rel, err := AnswerRelevancy(ctx, tc.Query, tc.Response, opts.Embedders)
			if err == nil {
				results[i].AnswerRelevancy = rel
			}
		}
	}

	return results, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
