package eval

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRunSingleObservation(t *testing.T) {
	obs := []Observation{
		{
			ID:     "test-1",
			Output: json.RawMessage(`"hello world"`),
		},
	}

	constant := NewScorerFunc("always_one", func(_ context.Context, _ Observation) (Score, error) {
		return Score{Name: "always_one", Value: 1.0}, nil
	})

	result, err := Run(context.Background(), "test-suite", obs, []Scorer{constant})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if len(result.Results[0].Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(result.Results[0].Scores))
	}
	if result.Results[0].Scores[0].Value != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Results[0].Scores[0].Value)
	}
	if result.Aggregate["always_one"] != 1.0 {
		t.Errorf("expected aggregate 1.0, got %f", result.Aggregate["always_one"])
	}
}

func TestRunMultipleObservations(t *testing.T) {
	obs := []Observation{
		{ID: "a", Output: json.RawMessage(`"foo"`)},
		{ID: "b", Output: json.RawMessage(`"bar"`)},
	}

	counter := 0.0
	scorer := NewScorerFunc("incremental", func(_ context.Context, _ Observation) (Score, error) {
		counter += 0.5
		return Score{Name: "incremental", Value: counter}, nil
	})

	result, err := Run(context.Background(), "multi", obs, []Scorer{scorer}, WithConcurrency(1))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	// With concurrency=1, scores should be 0.5 and 1.0, mean = 0.75
	if result.Aggregate["incremental"] != 0.75 {
		t.Errorf("expected aggregate 0.75, got %f", result.Aggregate["incremental"])
	}
}

func TestRunSkipsEmptyScores(t *testing.T) {
	obs := []Observation{{ID: "x"}}

	// Returns empty name → should be skipped.
	noop := NewScorerFunc("noop", func(_ context.Context, _ Observation) (Score, error) {
		return Score{}, nil
	})

	result, err := Run(context.Background(), "skip", obs, []Scorer{noop})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results[0].Scores) != 0 {
		t.Errorf("expected 0 scores, got %d", len(result.Results[0].Scores))
	}
}

func TestPopulate(t *testing.T) {
	obs := []Observation{
		{ID: "p1", Input: json.RawMessage(`"input1"`)},
		{ID: "p2", Input: json.RawMessage(`"input2"`)},
	}

	subject := Subject(func(_ context.Context, o *Observation) error {
		o.Output = json.RawMessage(`"processed"`)
		o.Timing.TotalMs = 42
		return nil
	})

	if err := Populate(context.Background(), obs, subject); err != nil {
		t.Fatal(err)
	}

	for _, o := range obs {
		if string(o.Output) != `"processed"` {
			t.Errorf("expected processed output, got %s", o.Output)
		}
		if o.Timing.TotalMs != 42 {
			t.Errorf("expected 42ms, got %d", o.Timing.TotalMs)
		}
	}
}
