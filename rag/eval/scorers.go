package eval

import (
	"context"
	"encoding/json"

	topeval "github.com/urmzd/saige/eval"
	"github.com/urmzd/saige/rag/types"
)

// Annotation keys used by RAG subjects.
const (
	AnnotationHits          = "rag.hits"           // []types.SearchHit
	AnnotationRelevantUUIDs = "rag.relevant_uuids" // []string
	AnnotationContextText   = "rag.context_text"   // string
)

// ContextPrecisionScorer wraps [ContextPrecision] as a [topeval.Scorer].
func ContextPrecisionScorer() topeval.Scorer {
	return topeval.NewScorerFunc("context_precision", func(_ context.Context, obs topeval.Observation) (topeval.Score, error) {
		hits, uuids, err := extractRAGAnnotations(obs)
		if err != nil {
			return topeval.Score{}, err
		}
		if hits == nil {
			return topeval.Score{}, nil
		}
		return topeval.Score{Name: "context_precision", Value: ContextPrecision(hits, uuids)}, nil
	})
}

// ContextRecallScorer wraps [ContextRecall] as a [topeval.Scorer].
func ContextRecallScorer() topeval.Scorer {
	return topeval.NewScorerFunc("context_recall", func(_ context.Context, obs topeval.Observation) (topeval.Score, error) {
		hits, uuids, err := extractRAGAnnotations(obs)
		if err != nil {
			return topeval.Score{}, err
		}
		if hits == nil {
			return topeval.Score{}, nil
		}
		return topeval.Score{Name: "context_recall", Value: ContextRecall(hits, uuids)}, nil
	})
}

// NDCGScorer wraps [NDCG] as a [topeval.Scorer].
func NDCGScorer(k int) topeval.Scorer {
	return topeval.NewScorerFunc("ndcg", func(_ context.Context, obs topeval.Observation) (topeval.Score, error) {
		hits, uuids, err := extractRAGAnnotations(obs)
		if err != nil {
			return topeval.Score{}, err
		}
		if hits == nil {
			return topeval.Score{}, nil
		}
		return topeval.Score{Name: "ndcg", Value: NDCG(hits, uuids, k)}, nil
	})
}

// MRRScorer wraps [MRR] as a [topeval.Scorer].
func MRRScorer() topeval.Scorer {
	return topeval.NewScorerFunc("mrr", func(_ context.Context, obs topeval.Observation) (topeval.Score, error) {
		hits, uuids, err := extractRAGAnnotations(obs)
		if err != nil {
			return topeval.Score{}, err
		}
		if hits == nil {
			return topeval.Score{}, nil
		}
		return topeval.Score{Name: "mrr", Value: MRR(hits, uuids)}, nil
	})
}

// HitRateScorer wraps [HitRate] as a [topeval.Scorer].
func HitRateScorer(k int) topeval.Scorer {
	return topeval.NewScorerFunc("hit_rate", func(_ context.Context, obs topeval.Observation) (topeval.Score, error) {
		hits, uuids, err := extractRAGAnnotations(obs)
		if err != nil {
			return topeval.Score{}, err
		}
		if hits == nil {
			return topeval.Score{}, nil
		}
		return topeval.Score{Name: "hit_rate", Value: HitRate(hits, uuids, k)}, nil
	})
}

// FaithfulnessScorer wraps [Faithfulness] as a [topeval.Scorer].
// It reads the response from Output and context from the AnnotationContextText annotation.
func FaithfulnessScorer(llm types.LLM) topeval.Scorer {
	return topeval.NewScorerFunc("faithfulness", func(ctx context.Context, obs topeval.Observation) (topeval.Score, error) {
		response, contextText, err := extractResponseAndContext(obs)
		if err != nil || response == "" {
			return topeval.Score{}, err
		}
		score, _, err := Faithfulness(ctx, response, contextText, llm)
		if err != nil {
			return topeval.Score{}, err
		}
		return topeval.Score{Name: "faithfulness", Value: score}, nil
	})
}

// AnswerRelevancyScorer wraps [AnswerRelevancy] as a [topeval.Scorer].
func AnswerRelevancyScorer(llm types.LLM, embedders types.EmbedderRegistry, sampleCount int) topeval.Scorer {
	return topeval.NewScorerFunc("answer_relevancy", func(ctx context.Context, obs topeval.Observation) (topeval.Score, error) {
		var query, response string
		if obs.Input != nil {
			if err := json.Unmarshal(obs.Input, &query); err != nil {
				return topeval.Score{}, err
			}
		}
		if obs.Output != nil {
			if err := json.Unmarshal(obs.Output, &response); err != nil {
				return topeval.Score{}, err
			}
		}
		if response == "" {
			return topeval.Score{}, nil
		}
		score, err := AnswerRelevancy(ctx, query, response, llm, embedders, sampleCount)
		if err != nil {
			return topeval.Score{}, err
		}
		return topeval.Score{Name: "answer_relevancy", Value: score}, nil
	})
}

// AnswerCorrectnessScorer wraps [AnswerCorrectness] as a [topeval.Scorer].
func AnswerCorrectnessScorer(llm types.LLM) topeval.Scorer {
	return topeval.NewScorerFunc("answer_correctness", func(ctx context.Context, obs topeval.Observation) (topeval.Score, error) {
		var response, groundTruth string
		if obs.Output != nil {
			if err := json.Unmarshal(obs.Output, &response); err != nil {
				return topeval.Score{}, err
			}
		}
		if obs.GroundTruth != nil {
			if err := json.Unmarshal(obs.GroundTruth, &groundTruth); err != nil {
				return topeval.Score{}, err
			}
		}
		if response == "" {
			return topeval.Score{}, nil
		}
		score, err := AnswerCorrectness(ctx, response, groundTruth, llm)
		if err != nil {
			return topeval.Score{}, err
		}
		return topeval.Score{Name: "answer_correctness", Value: score}, nil
	})
}

func extractRAGAnnotations(obs topeval.Observation) ([]types.SearchHit, []string, error) {
	var hits []types.SearchHit
	if raw, ok := obs.Annotations[AnnotationHits]; ok {
		if err := json.Unmarshal(raw, &hits); err != nil {
			return nil, nil, err
		}
	}
	var uuids []string
	if raw, ok := obs.Annotations[AnnotationRelevantUUIDs]; ok {
		if err := json.Unmarshal(raw, &uuids); err != nil {
			return nil, nil, err
		}
	}
	return hits, uuids, nil
}

func extractResponseAndContext(obs topeval.Observation) (string, string, error) {
	var response string
	if obs.Output != nil {
		if err := json.Unmarshal(obs.Output, &response); err != nil {
			return "", "", err
		}
	}
	var contextText string
	if raw, ok := obs.Annotations[AnnotationContextText]; ok {
		if err := json.Unmarshal(raw, &contextText); err != nil {
			return "", "", err
		}
	}
	return response, contextText, nil
}
