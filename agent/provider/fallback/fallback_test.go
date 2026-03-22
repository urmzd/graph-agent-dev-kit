package fallback

import (
	"context"
	"errors"
	"testing"

	"github.com/urmzd/saige/agent/types"
)

// mockProvider returns a fixed text response.
type mockProvider struct {
	response string
}

func (m *mockProvider) ChatStream(_ context.Context, _ []types.Message, _ []types.ToolDef) (<-chan types.Delta, error) {
	ch := make(chan types.Delta, 3)
	ch <- types.TextStartDelta{}
	ch <- types.TextContentDelta{Content: m.response}
	ch <- types.TextEndDelta{}
	close(ch)
	return ch, nil
}

// errorProviderSimple returns a fixed error (used in provider tests).
type errorProviderSimple struct {
	err error
}

func (p *errorProviderSimple) ChatStream(_ context.Context, _ []types.Message, _ []types.ToolDef) (<-chan types.Delta, error) {
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
		if tc, ok := d.(types.TextContentDelta); ok {
			text += tc.Content
		}
	}
	if text != "from-primary" {
		t.Errorf("got %q, want %q", text, "from-primary")
	}
}

func TestFallbackProvider_FallsBackOnError(t *testing.T) {
	failing := &errorProviderSimple{err: &types.ProviderError{
		Provider: "bad",
		Kind:     types.ErrorKindTransient,
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
		if tc, ok := d.(types.TextContentDelta); ok {
			text += tc.Content
		}
	}
	if text != "from-backup" {
		t.Errorf("got %q, want %q", text, "from-backup")
	}
}

func TestFallbackProvider_AllFail(t *testing.T) {
	p1 := &errorProviderSimple{err: &types.ProviderError{Provider: "a", Kind: types.ErrorKindTransient, Err: errors.New("fail-a")}}
	p2 := &errorProviderSimple{err: &types.ProviderError{Provider: "b", Kind: types.ErrorKindTransient, Err: errors.New("fail-b")}}

	fb := New(p1, p2)
	_, err := fb.ChatStream(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var fe *types.FallbackError
	if !errors.As(err, &fe) {
		t.Fatalf("expected *FallbackError, got %T", err)
	}
	if len(fe.Errors) != 2 {
		t.Errorf("errors = %d, want 2", len(fe.Errors))
	}
	if !errors.Is(err, types.ErrProviderFailed) {
		t.Error("FallbackError should match ErrProviderFailed")
	}
}

func TestFallbackProvider_StopsOnPermanentWhenConfigured(t *testing.T) {
	perm := &errorProviderSimple{err: &types.ProviderError{Provider: "auth-fail", Kind: types.ErrorKindPermanent, Err: errors.New("unauthorized")}}
	good := &mockProvider{response: "should not reach"}

	fb := &Provider{
		Providers:  []types.Provider{perm, good},
		FallbackOn: types.IsTransient, // only fallback on transient
	}

	_, err := fb.ChatStream(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var fe *types.FallbackError
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

	p1 := &errorProviderSimple{err: &types.ProviderError{Provider: "a", Kind: types.ErrorKindTransient, Err: errors.New("fail")}}
	p2 := &mockProvider{response: "should not reach"}

	fb := New(p1, p2)
	_, err := fb.ChatStream(ctx, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	var fe *types.FallbackError
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
