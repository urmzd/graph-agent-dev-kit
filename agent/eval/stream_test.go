package eval

import (
	"testing"
	"time"

	"github.com/urmzd/saige/agent/types"
)

func TestCollectStreamTimingBasic(t *testing.T) {
	ch := make(chan types.Delta, 10)

	// Simulate a stream with text chunks.
	go func() {
		ch <- types.TextStartDelta{}
		ch <- types.TextContentDelta{Content: "Hello"}
		time.Sleep(5 * time.Millisecond)
		ch <- types.TextContentDelta{Content: " world"}
		ch <- types.UsageDelta{PromptTokens: 10, CompletionTokens: 5}
		ch <- types.TextEndDelta{}
		ch <- types.DoneDelta{}
		close(ch)
	}()

	timing, text, deltas := CollectStreamTiming(ch)

	if text != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", text)
	}
	if timing.ChunkCount != 2 {
		t.Errorf("expected 2 chunks, got %d", timing.ChunkCount)
	}
	if timing.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", timing.InputTokens)
	}
	if timing.OutputTokens != 5 {
		t.Errorf("expected 5 output tokens, got %d", timing.OutputTokens)
	}
	if timing.TTFTMs < 0 {
		t.Error("TTFT should be non-negative")
	}
	if timing.TTLTMs < timing.TTFTMs {
		t.Error("TTLT should be >= TTFT")
	}
	if len(deltas) != 6 {
		t.Errorf("expected 6 deltas, got %d", len(deltas))
	}
}

func TestCollectStreamTimingEmpty(t *testing.T) {
	ch := make(chan types.Delta)
	close(ch)

	timing, text, deltas := CollectStreamTiming(ch)

	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
	if timing.ChunkCount != 0 {
		t.Errorf("expected 0 chunks, got %d", timing.ChunkCount)
	}
	if timing.TTFTMs != 0 {
		t.Errorf("expected 0 TTFT, got %d", timing.TTFTMs)
	}
	if len(deltas) != 0 {
		t.Errorf("expected 0 deltas, got %d", len(deltas))
	}
}

func TestCollectStreamTimingMultipleUsage(t *testing.T) {
	ch := make(chan types.Delta, 5)

	go func() {
		ch <- types.UsageDelta{PromptTokens: 10, CompletionTokens: 5}
		ch <- types.UsageDelta{PromptTokens: 20, CompletionTokens: 10}
		close(ch)
	}()

	timing, _, _ := CollectStreamTiming(ch)

	// Usage should accumulate.
	if timing.InputTokens != 30 {
		t.Errorf("expected 30 input tokens, got %d", timing.InputTokens)
	}
	if timing.OutputTokens != 15 {
		t.Errorf("expected 15 output tokens, got %d", timing.OutputTokens)
	}
}
