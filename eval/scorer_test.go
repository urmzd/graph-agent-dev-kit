package eval

import (
	"math"
	"testing"
)

func TestAggregateEmpty(t *testing.T) {
	agg := Aggregate(nil)
	if len(agg) != 0 {
		t.Errorf("expected empty map, got %v", agg)
	}
}

func TestAggregateSingleMetric(t *testing.T) {
	results := []ObservationResult{
		{Scores: []Score{{Name: "accuracy", Value: 0.8}}},
		{Scores: []Score{{Name: "accuracy", Value: 0.6}}},
		{Scores: []Score{{Name: "accuracy", Value: 1.0}}},
	}

	agg := Aggregate(results)
	want := 0.8
	if math.Abs(agg["accuracy"]-want) > 1e-9 {
		t.Errorf("accuracy: got %f, want %f", agg["accuracy"], want)
	}
}

func TestAggregateMultipleMetrics(t *testing.T) {
	results := []ObservationResult{
		{Scores: []Score{
			{Name: "a", Value: 1.0},
			{Name: "b", Value: 0.5},
		}},
		{Scores: []Score{
			{Name: "a", Value: 0.0},
			{Name: "b", Value: 1.0},
		}},
	}

	agg := Aggregate(results)
	if math.Abs(agg["a"]-0.5) > 1e-9 {
		t.Errorf("a: got %f, want 0.5", agg["a"])
	}
	if math.Abs(agg["b"]-0.75) > 1e-9 {
		t.Errorf("b: got %f, want 0.75", agg["b"])
	}
}

func TestAggregateSkipsEmptyNames(t *testing.T) {
	results := []ObservationResult{
		{Scores: []Score{
			{Name: "", Value: 99.0},
			{Name: "real", Value: 1.0},
		}},
	}

	agg := Aggregate(results)
	if _, ok := agg[""]; ok {
		t.Error("expected empty-name score to be skipped")
	}
	if agg["real"] != 1.0 {
		t.Errorf("real: got %f, want 1.0", agg["real"])
	}
}
