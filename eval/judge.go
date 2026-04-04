package eval

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Generator is the minimal LLM interface for evaluation prompts.
// It is intentionally identical to rag/types.LLM so any provider satisfies both.
type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// JudgeConfig configures a judge scorer.
type JudgeConfig struct {
	Name   string
	Rubric string
}

// JudgeOption configures a judge scorer.
type JudgeOption func(*JudgeConfig)

// WithJudgeName sets the metric name (default: "judge_score").
func WithJudgeName(name string) JudgeOption {
	return func(c *JudgeConfig) { c.Name = name }
}

// WithJudgeRubric sets the evaluation criteria rubric.
func WithJudgeRubric(rubric string) JudgeOption {
	return func(c *JudgeConfig) { c.Rubric = rubric }
}

// NewJudgeScorer creates a [Scorer] that uses an LLM to evaluate output quality.
// It reads Input, Output, and optionally a context annotation from the Observation.
func NewJudgeScorer(gen Generator, opts ...JudgeOption) Scorer {
	cfg := &JudgeConfig{
		Name:   "judge_score",
		Rubric: "Evaluate the response for correctness, completeness, and relevance.",
	}
	for _, o := range opts {
		o(cfg)
	}

	return NewScorerFunc(cfg.Name, func(ctx context.Context, obs Observation) (Score, error) {
		input := string(obs.Input)
		response := string(obs.Output)
		contextText := ""
		if raw, ok := obs.Annotations["context"]; ok {
			contextText = string(raw)
		}

		prompt := renderPrompt(judgeTmpl, map[string]string{
			"Input":    input,
			"Context":  contextText,
			"Response": response,
			"Rubric":   cfg.Rubric,
		})

		result, err := gen.Generate(ctx, prompt)
		if err != nil {
			return Score{}, fmt.Errorf("judge generate: %w", err)
		}

		score, reason := parseJudgeOutput(result)
		return Score{Name: cfg.Name, Value: score, Reason: reason}, nil
	})
}

// NewPairwiseJudgeScorer creates a [Scorer] for comparing two outputs.
// It expects the base output in GroundTruth and the experimental output in Output.
func NewPairwiseJudgeScorer(gen Generator, opts ...JudgeOption) Scorer {
	cfg := &JudgeConfig{
		Name:   "pairwise_judge",
		Rubric: "Compare the two responses for correctness, completeness, and relevance.",
	}
	for _, o := range opts {
		o(cfg)
	}

	return NewScorerFunc(cfg.Name, func(ctx context.Context, obs Observation) (Score, error) {
		input := string(obs.Input)
		responseA := string(obs.GroundTruth)
		responseB := string(obs.Output)

		prompt := renderPrompt(judgePairwiseTmpl, map[string]string{
			"Input":     input,
			"ResponseA": responseA,
			"ResponseB": responseB,
			"Rubric":    cfg.Rubric,
		})

		result, err := gen.Generate(ctx, prompt)
		if err != nil {
			return Score{}, fmt.Errorf("pairwise judge generate: %w", err)
		}

		score, reason := parseJudgeOutput(result)
		return Score{Name: cfg.Name, Value: score, Reason: reason}, nil
	})
}

// parseJudgeOutput extracts SCORE and REASONING from judge LLM output.
func parseJudgeOutput(output string) (float64, string) {
	var score float64
	var reason string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "SCORE:") {
			valStr := strings.TrimSpace(line[len("SCORE:"):])
			if v, err := strconv.ParseFloat(valStr, 64); err == nil {
				score = clamp(v, 0, 1)
			}
		}
		if strings.HasPrefix(upper, "REASONING:") || strings.HasPrefix(upper, "REASON:") {
			idx := strings.Index(line, ":")
			if idx >= 0 {
				reason = strings.TrimSpace(line[idx+1:])
			}
		}
	}
	return score, reason
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
