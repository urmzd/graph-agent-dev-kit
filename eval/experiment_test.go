package eval

import (
	"context"
	"encoding/json"
	"math"
	"testing"
)

func TestRunExperiment(t *testing.T) {
	inputs := []Observation{
		{ID: "e1", Input: json.RawMessage(`"query1"`)},
		{ID: "e2", Input: json.RawMessage(`"query2"`)},
	}

	base := Subject(func(_ context.Context, obs *Observation) error {
		obs.Output = json.RawMessage(`"base response"`)
		obs.Timing.TotalMs = 100
		return nil
	})

	exp := Subject(func(_ context.Context, obs *Observation) error {
		obs.Output = json.RawMessage(`"experimental response"`)
		obs.Timing.TotalMs = 50
		return nil
	})

	latencyScorer := NewScorerFunc("latency_ms", func(_ context.Context, obs Observation) (Score, error) {
		return Score{Name: "latency_ms", Value: float64(obs.Timing.TotalMs)}, nil
	})

	result, err := RunExperiment(context.Background(), inputs, base, exp, []Scorer{latencyScorer},
		WithExperimentName("test-experiment"))
	if err != nil {
		t.Fatal(err)
	}

	if result.Name != "test-experiment" {
		t.Errorf("expected name test-experiment, got %s", result.Name)
	}
	if len(result.BaseResults) != 2 {
		t.Fatalf("expected 2 base results, got %d", len(result.BaseResults))
	}
	if len(result.ExpResults) != 2 {
		t.Fatalf("expected 2 exp results, got %d", len(result.ExpResults))
	}

	// Base latency should be 100, exp should be 50, delta = -50
	if math.Abs(result.BaseAggregate["latency_ms"]-100) > 0.001 {
		t.Errorf("base latency: got %f, want 100", result.BaseAggregate["latency_ms"])
	}
	if math.Abs(result.ExpAggregate["latency_ms"]-50) > 0.001 {
		t.Errorf("exp latency: got %f, want 50", result.ExpAggregate["latency_ms"])
	}
	if math.Abs(result.Deltas["latency_ms"]-(-50)) > 0.001 {
		t.Errorf("delta: got %f, want -50", result.Deltas["latency_ms"])
	}
}
