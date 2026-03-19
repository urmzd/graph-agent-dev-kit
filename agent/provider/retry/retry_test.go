package retry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/urmzd/graph-agent-dev-kit/agent/core"
)

// mockProvider returns a fixed text response.
type mockProvider struct {
	response string
}

func (m *mockProvider) ChatStream(_ context.Context, _ []core.Message, _ []core.ToolDef) (<-chan core.Delta, error) {
	ch := make(chan core.Delta, 3)
	ch <- core.TextStartDelta{}
	ch <- core.TextContentDelta{Content: m.response}
	ch <- core.TextEndDelta{}
	close(ch)
	return ch, nil
}

// errorProviderSimple returns a fixed error.
type errorProviderSimple struct {
	err error
}

func (p *errorProviderSimple) ChatStream(_ context.Context, _ []core.Message, _ []core.ToolDef) (<-chan core.Delta, error) {
	return nil, p.err
}

func TestRetryProvider_SucceedsFirstTry(t *testing.T) {
	inner := &mockProvider{response: "ok"}
	rp := New(inner, DefaultConfig())

	ch, err := rp.ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var text string
	for d := range ch {
		if tc, ok := d.(core.TextContentDelta); ok {
			text += tc.Content
		}
	}
	if text != "ok" {
		t.Errorf("got %q, want %q", text, "ok")
	}
}

func TestRetryProvider_RetriesOnTransient(t *testing.T) {
	var calls atomic.Int32
	inner := &countingProvider{
		calls:     &calls,
		failUntil: 2,
		err: &core.ProviderError{
			Provider: "flaky",
			Kind:     core.ErrorKindTransient,
			Err:      errors.New("timeout"),
		},
		response: "recovered",
	}

	cfg := Config{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Millisecond, // fast for tests
		MaxDelay:    5 * time.Millisecond,
		Multiplier:  2.0,
	}
	rp := New(inner, cfg)

	ch, err := rp.ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var text string
	for d := range ch {
		if tc, ok := d.(core.TextContentDelta); ok {
			text += tc.Content
		}
	}
	if text != "recovered" {
		t.Errorf("got %q, want %q", text, "recovered")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestRetryProvider_StopsOnPermanent(t *testing.T) {
	inner := &errorProviderSimple{err: &core.ProviderError{
		Provider: "auth",
		Kind:     core.ErrorKindPermanent,
		Err:      errors.New("unauthorized"),
	}}

	cfg := Config{
		MaxAttempts: 5,
		BaseDelay:   1 * time.Millisecond,
	}
	rp := New(inner, cfg)

	_, err := rp.ChatStream(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	// Should not have retried -- permanent error
	var pe *core.ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ProviderError, got %T", err)
	}
}

func TestRetryProvider_ExhaustsAttempts(t *testing.T) {
	inner := &errorProviderSimple{err: &core.ProviderError{
		Provider: "down",
		Kind:     core.ErrorKindTransient,
		Err:      errors.New("server error"),
	}}

	cfg := Config{
		MaxAttempts: 2,
		BaseDelay:   1 * time.Millisecond,
	}
	rp := New(inner, cfg)

	_, err := rp.ChatStream(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var re *core.RetryError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RetryError, got %T", err)
	}
	if re.Attempts != 2 {
		t.Errorf("attempts = %d, want 2", re.Attempts)
	}
	if !errors.Is(re.Last, core.ErrProviderFailed) {
		t.Error("last error should match ErrProviderFailed")
	}
}

func TestRetryProvider_ContextCancelledDuringBackoff(t *testing.T) {
	inner := &errorProviderSimple{err: &core.ProviderError{
		Provider: "slow",
		Kind:     core.ErrorKindTransient,
		Err:      errors.New("timeout"),
	}}

	ctx, cancel := context.WithCancel(context.Background())

	cfg := Config{
		MaxAttempts: 10,
		BaseDelay:   1 * time.Second, // long delay
	}
	rp := New(inner, cfg)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := rp.ChatStream(ctx, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRetryProvider_Name(t *testing.T) {
	inner := &mockProvider{response: "ok"}
	rp := New(inner, DefaultConfig())
	if rp.Name() != "retry(unknown)" {
		t.Errorf("Name() = %q, want %q", rp.Name(), "retry(unknown)")
	}
}

func TestRetryProvider_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 500*time.Millisecond {
		t.Errorf("BaseDelay = %v, want 500ms", cfg.BaseDelay)
	}
}

// countingProvider fails for the first N calls, then succeeds.
type countingProvider struct {
	calls     *atomic.Int32
	failUntil int32
	err       error
	response  string
}

func (p *countingProvider) ChatStream(_ context.Context, _ []core.Message, _ []core.ToolDef) (<-chan core.Delta, error) {
	n := p.calls.Add(1)
	if n <= p.failUntil {
		return nil, p.err
	}
	ch := make(chan core.Delta, 3)
	ch <- core.TextStartDelta{}
	ch <- core.TextContentDelta{Content: p.response}
	ch <- core.TextEndDelta{}
	close(ch)
	return ch, nil
}
