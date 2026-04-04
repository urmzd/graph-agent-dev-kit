package eval

import (
	"context"
	"encoding/json"
	"testing"
)

type mockGenerator struct {
	response string
}

func (m *mockGenerator) Generate(_ context.Context, _ string) (string, error) {
	return m.response, nil
}

func TestJudgeScorerParsesOutput(t *testing.T) {
	gen := &mockGenerator{response: "REASONING: Good response\nSCORE: 0.85"}

	scorer := NewJudgeScorer(gen)
	obs := Observation{
		ID:     "j1",
		Input:  json.RawMessage(`"What is Go?"`),
		Output: json.RawMessage(`"Go is a programming language."`),
	}

	score, err := scorer.Score(context.Background(), obs)
	if err != nil {
		t.Fatal(err)
	}

	if score.Name != "judge_score" {
		t.Errorf("expected name judge_score, got %s", score.Name)
	}
	assertClose(t, "value", score.Value, 0.85, 0.001)
	if score.Reason != "Good response" {
		t.Errorf("expected reason 'Good response', got %q", score.Reason)
	}
}

func TestJudgeScorerCustomName(t *testing.T) {
	gen := &mockGenerator{response: "REASONING: Fine\nSCORE: 0.5"}

	scorer := NewJudgeScorer(gen, WithJudgeName("custom_judge"))
	obs := Observation{
		ID:     "j2",
		Output: json.RawMessage(`"answer"`),
	}

	score, err := scorer.Score(context.Background(), obs)
	if err != nil {
		t.Fatal(err)
	}

	if score.Name != "custom_judge" {
		t.Errorf("expected name custom_judge, got %s", score.Name)
	}
}

func TestJudgeScorerClampsScore(t *testing.T) {
	gen := &mockGenerator{response: "REASONING: Over\nSCORE: 1.5"}

	scorer := NewJudgeScorer(gen)
	obs := Observation{ID: "j3", Output: json.RawMessage(`"x"`)}

	score, err := scorer.Score(context.Background(), obs)
	if err != nil {
		t.Fatal(err)
	}
	assertClose(t, "clamped", score.Value, 1.0, 0.001)
}

func TestPairwiseJudgeScorer(t *testing.T) {
	gen := &mockGenerator{response: "REASONING: B is better\nSCORE: 0.8"}

	scorer := NewPairwiseJudgeScorer(gen)
	obs := Observation{
		ID:          "pw1",
		Input:       json.RawMessage(`"query"`),
		GroundTruth: json.RawMessage(`"response A"`),
		Output:      json.RawMessage(`"response B"`),
	}

	score, err := scorer.Score(context.Background(), obs)
	if err != nil {
		t.Fatal(err)
	}

	if score.Name != "pairwise_judge" {
		t.Errorf("expected pairwise_judge, got %s", score.Name)
	}
	assertClose(t, "value", score.Value, 0.8, 0.001)
}

func TestParseJudgeOutputMissingScore(t *testing.T) {
	score, reason := parseJudgeOutput("REASONING: no score line here")
	if score != 0.0 {
		t.Errorf("expected 0.0 for missing score, got %f", score)
	}
	if reason != "no score line here" {
		t.Errorf("expected reason text, got %q", reason)
	}
}
