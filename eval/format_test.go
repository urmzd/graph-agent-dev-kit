package eval

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

func TestWriteReadExperiment(t *testing.T) {
	dir := t.TempDir()

	original := &ExperimentResult{
		Name:      "round-trip",
		CreatedAt: time.Now().Truncate(time.Second),
		BaseResults: []ObservationResult{
			{
				Observation: Observation{ID: "o1", Output: json.RawMessage(`"base"`)},
				Scores:      []Score{{Name: "accuracy", Value: 0.9}},
			},
		},
		ExpResults: []ObservationResult{
			{
				Observation: Observation{ID: "o1", Output: json.RawMessage(`"exp"`)},
				Scores:      []Score{{Name: "accuracy", Value: 0.95}},
			},
		},
		BaseAggregate: map[string]float64{"accuracy": 0.9},
		ExpAggregate:  map[string]float64{"accuracy": 0.95},
		Deltas:        map[string]float64{"accuracy": 0.05},
	}

	if err := WriteExperiment(dir, original); err != nil {
		t.Fatal(err)
	}

	// Verify files exist.
	for _, path := range []string{
		dir + "/result.json",
		dir + "/inputs/000.json",
		dir + "/outputs/base/000.json",
		dir + "/outputs/exp/000.json",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s to exist", path)
		}
	}

	// Read back.
	loaded, err := ReadExperiment(dir)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Name != original.Name {
		t.Errorf("name: got %q, want %q", loaded.Name, original.Name)
	}
	if len(loaded.BaseResults) != 1 {
		t.Fatalf("expected 1 base result, got %d", len(loaded.BaseResults))
	}
	if math.Abs(loaded.Deltas["accuracy"]-0.05) > 0.001 {
		t.Errorf("delta: got %f, want 0.05", loaded.Deltas["accuracy"])
	}
}

func TestWriteReadSuiteResult(t *testing.T) {
	path := t.TempDir() + "/suite.json"

	original := &SuiteResult{
		Name:      "suite-test",
		CreatedAt: time.Now().Truncate(time.Second),
		Results: []ObservationResult{
			{
				Observation: Observation{ID: "s1"},
				Scores:      []Score{{Name: "f1", Value: 0.88}},
			},
		},
		Aggregate: map[string]float64{"f1": 0.88},
	}

	if err := WriteSuiteResult(path, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := ReadSuiteResult(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Name != original.Name {
		t.Errorf("name: got %q, want %q", loaded.Name, original.Name)
	}
	if math.Abs(loaded.Aggregate["f1"]-0.88) > 0.001 {
		t.Errorf("aggregate f1: got %f, want 0.88", loaded.Aggregate["f1"])
	}
}
