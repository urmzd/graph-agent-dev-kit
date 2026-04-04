// Package eval provides a universal evaluation framework for SAIGE subsystems.
//
// The framework is built on three abstractions:
//   - [Observation] — a universal eval case carrying typed I/O as JSON
//   - [Scorer] — an interface for computing a named metric from an Observation
//   - [Subject] — a function that populates an Observation's output and annotations
//
// Subsystem-specific scorers live in sub-packages (ragscore, agentscore, kgscore)
// and operate on well-known annotation keys set by their respective subjects.
package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Observation is the universal eval case. Input, Output, and GroundTruth use
// json.RawMessage so the same structure works for RAG queries, agent
// conversations, KG episodes, or end-to-end flows.
type Observation struct {
	ID          string                     `json:"id"`
	Turn        int                        `json:"turn"`
	Input       json.RawMessage            `json:"input"`
	Output      json.RawMessage            `json:"output"`
	GroundTruth json.RawMessage            `json:"ground_truth,omitempty"`
	Annotations map[string]json.RawMessage `json:"annotations,omitempty"`
	Timing      ObservationTiming          `json:"timing"`
}

// ObservationTiming captures latency and token usage for a single observation.
type ObservationTiming struct {
	TotalMs      int64   `json:"total_ms"`
	TTFTMs       int64   `json:"ttft_ms,omitempty"`
	TTLTMs       int64   `json:"ttlt_ms,omitempty"`
	MedianITL    float64 `json:"median_itl_ms,omitempty"`
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
}

// Score is a single named metric value.
type Score struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Reason string  `json:"reason,omitempty"`
}

// ObservationResult pairs an observation with its scores.
type ObservationResult struct {
	Observation Observation `json:"observation"`
	Scores      []Score     `json:"scores"`
}

// SuiteResult is the complete output of an evaluation run.
type SuiteResult struct {
	Name      string              `json:"name"`
	CreatedAt time.Time           `json:"created_at"`
	Results   []ObservationResult `json:"results"`
	Aggregate map[string]float64  `json:"aggregate"`
}

// Run executes an evaluation suite: for each observation, it runs all scorers
// and collects results. Observations should have Output already populated
// (typically by a [Subject]).
func Run(ctx context.Context, name string, observations []Observation, scorers []Scorer, opts ...Option) (*SuiteResult, error) {
	cfg := &Config{
		Concurrency: 1,
		Logger:      slog.Default(),
	}
	for _, o := range opts {
		o(cfg)
	}

	results := make([]ObservationResult, len(observations))

	sem := make(chan struct{}, cfg.Concurrency)
	var mu sync.Mutex
	var firstErr error

	var wg sync.WaitGroup
	for i, obs := range observations {
		wg.Add(1)
		go func(idx int, obs Observation) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			scores, err := scoreObservation(ctx, obs, scorers)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("observation %q: %w", obs.ID, err)
				}
				mu.Unlock()
				cfg.Logger.Error("scorer failed", "observation", obs.ID, "error", err)
				return
			}

			results[idx] = ObservationResult{
				Observation: obs,
				Scores:      scores,
			}
		}(i, obs)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return &SuiteResult{
		Name:      name,
		CreatedAt: time.Now(),
		Results:   results,
		Aggregate: Aggregate(results),
	}, nil
}

// scoreObservation runs all scorers against a single observation.
func scoreObservation(ctx context.Context, obs Observation, scorers []Scorer) ([]Score, error) {
	var scores []Score
	for _, s := range scorers {
		score, err := s.Score(ctx, obs)
		if err != nil {
			return nil, fmt.Errorf("scorer %q: %w", s.Name(), err)
		}
		if score.Name != "" {
			scores = append(scores, score)
		}
	}
	return scores, nil
}

// Populate runs a [Subject] against each observation, populating Output,
// Annotations, and Timing fields in place.
func Populate(ctx context.Context, observations []Observation, subject Subject) error {
	for i := range observations {
		if err := subject(ctx, &observations[i]); err != nil {
			return fmt.Errorf("observation %q: %w", observations[i].ID, err)
		}
	}
	return nil
}
