package rageval_test

import (
	"math"
	"testing"

	"github.com/urmzd/saige/rag/rageval"
	"github.com/urmzd/saige/rag/ragtypes"
)

func TestContextPrecision(t *testing.T) {
	hits := []ragtypes.SearchHit{
		{Variant: ragtypes.ContentVariant{UUID: "a"}},
		{Variant: ragtypes.ContentVariant{UUID: "b"}},
		{Variant: ragtypes.ContentVariant{UUID: "c"}},
		{Variant: ragtypes.ContentVariant{UUID: "d"}},
	}

	// All relevant at positions 0, 2.
	relevantUUIDs := []string{"a", "c"}
	precision := rageval.ContextPrecision(hits, relevantUUIDs)

	// At position 0: precision@1 = 1/1 = 1.0
	// At position 2: precision@3 = 2/3 ≈ 0.667
	// Average precision = (1.0 + 0.667) / 2 ≈ 0.833
	expected := (1.0 + 2.0/3.0) / 2.0
	if math.Abs(precision-expected) > 0.001 {
		t.Errorf("expected precision %.3f, got %.3f", expected, precision)
	}
}

func TestContextPrecisionPerfect(t *testing.T) {
	hits := []ragtypes.SearchHit{
		{Variant: ragtypes.ContentVariant{UUID: "a"}},
		{Variant: ragtypes.ContentVariant{UUID: "b"}},
	}
	precision := rageval.ContextPrecision(hits, []string{"a", "b"})
	if math.Abs(precision-1.0) > 0.001 {
		t.Errorf("expected perfect precision 1.0, got %.3f", precision)
	}
}

func TestContextRecall(t *testing.T) {
	hits := []ragtypes.SearchHit{
		{Variant: ragtypes.ContentVariant{UUID: "a"}},
		{Variant: ragtypes.ContentVariant{UUID: "b"}},
		{Variant: ragtypes.ContentVariant{UUID: "c"}},
	}

	recall := rageval.ContextRecall(hits, []string{"a", "c", "d"})
	// 2 out of 3 relevant found.
	expected := 2.0 / 3.0
	if math.Abs(recall-expected) > 0.001 {
		t.Errorf("expected recall %.3f, got %.3f", expected, recall)
	}
}

func TestContextRecallPerfect(t *testing.T) {
	hits := []ragtypes.SearchHit{
		{Variant: ragtypes.ContentVariant{UUID: "a"}},
		{Variant: ragtypes.ContentVariant{UUID: "b"}},
	}
	recall := rageval.ContextRecall(hits, []string{"a", "b"})
	if math.Abs(recall-1.0) > 0.001 {
		t.Errorf("expected perfect recall 1.0, got %.3f", recall)
	}
}

func TestContextRecallNoRelevant(t *testing.T) {
	hits := []ragtypes.SearchHit{
		{Variant: ragtypes.ContentVariant{UUID: "a"}},
	}
	recall := rageval.ContextRecall(hits, []string{})
	if recall != 0 {
		t.Errorf("expected 0 recall with no relevant UUIDs, got %.3f", recall)
	}
}

func TestContextPrecisionNoHits(t *testing.T) {
	precision := rageval.ContextPrecision(nil, []string{"a"})
	if precision != 0 {
		t.Errorf("expected 0 precision with no hits, got %.3f", precision)
	}
}
