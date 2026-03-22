package fallback

import (
	"context"
	"errors"
	"testing"

	"github.com/urmzd/saige/agent/core"
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

// errorProviderSimple returns a fixed error (used in provider tests).
type errorProviderSimple struct {
	err error
}

func (p *errorProviderSimple) ChatStream(_ context.Context, _ []core.Message, _ []core.ToolDef) (<-chan core.Delta, error) {
	return nil, p.err
}

func TestFallbackProvider_FirstSucceeds(t *testing.T) {
	p1 := &mockProvider{response: "from-primary"}
	p2 := &mockProvider{response: "from-backup"}

	fb := New(p1, p2)
	ch, err := fb.ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var text string
	for d := range ch {
		if tc, ok := d.(core.TextContentDelta); ok {
			text += tc.Content
		}
	}
	if text != "from-primary" {
		t.Errorf("got %q, want %q", text, "from-primary")
	}
}

func TestFallbackProvider_FallsBackOnError(t *testing.T) {
	failing := &errorProviderSimple{err: &core.ProviderError{
		Provider: "bad",
		Kind:     core.ErrorKindTransient,
		Err:      errors.New("connection refused"),
	}}
	good := &mockProvider{response: "from-backup"}

	fb := New(failing, good)
	ch, err := fb.ChatStream(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var text string
	for d := range ch {
		if tc, ok := d.(core.TextContentDelta); ok {
			text += tc.Content
		}
	}
	if text != "from-backup" {
		t.Errorf("got %q, want %q", text, "from-backup")
	}
}

func TestFallbackProvider_AllFail(t *testing.T) {
	p1 := &errorProviderSimple{err: &core.ProviderError{Provider: "a", Kind: core.ErrorKindTransient, Err: errors.New("fail-a")}}
	p2 := &errorProviderSimple{err: &core.ProviderError{Provider: "b", Kind: core.ErrorKindTransient, Err: errors.New("fail-b")}}

	fb := New(p1, p2)
	_, err := fb.ChatStream(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var fe *core.FallbackError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FallbackError, got %T", err)
	}
	if len(fe.Errors) != 2 {
		t.Errorf("errors = %d, want 2", len(fe.Errors))
	}
	if !errors.Is(err, core.ErrProviderFailed) {
		t.Error("FallbackError should match ErrProviderFailed")
	}
}

func TestFallbackProvider_StopsOnPermanentWhenConfigured(t *testing.T) {
	perm := &errorProviderSimple{err: &core.ProviderError{Provider: "auth-fail", Kind: core.ErrorKindPermanent, Err: errors.New("unauthorized")}}
	good := &mockProvider{response: "should not reach"}

	fb := &Provider{
		Providers:  []core.Provider{perm, good},
		FallbackOn: core.IsTransient, // only fallback on transient
	}

	_, err := fb.ChatStream(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var fe *core.FallbackError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FallbackError, got %T", err)
	}
	if len(fe.Errors) != 1 {
		t.Errorf("errors = %d, want 1 (should not have tried second provider)", len(fe.Errors))
	}
}

func TestFallbackProvider_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p1 := &errorProviderSimple{err: &core.ProviderError{Provider: "a", Kind: core.ErrorKindTransient, Err: errors.New("fail")}}
	p2 := &mockProvider{response: "should not reach"}

	fb := New(p1, p2)
	_, err := fb.ChatStream(ctx, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var fe *core.FallbackError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FallbackError, got %T", err)
	}
	if len(fe.Errors) != 1 {
		t.Errorf("errors = %d, want 1 (should stop after context cancel)", len(fe.Errors))
	}
}

func TestFallbackProvider_Name(t *testing.T) {
	fb := New()
	if fb.Name() != "fallback" {
		t.Errorf("Name() = %q, want %q", fb.Name(), "fallback")
	}
}
