package agent

import (
	"errors"
	"testing"

	"github.com/urmzd/saige/agent/core"
)

func TestProviderError_Is(t *testing.T) {
	err := &core.ProviderError{Provider: "test", Kind: core.ErrorKindTransient, Err: errors.New("timeout")}
	if !errors.Is(err, core.ErrProviderFailed) {
		t.Error("ProviderError should match ErrProviderFailed")
	}
}

func TestProviderError_Unwrap(t *testing.T) {
	inner := errors.New("connection refused")
	err := &core.ProviderError{Provider: "test", Err: inner}
	if !errors.Is(err, inner) {
		t.Error("should unwrap to inner error")
	}
}

func TestProviderError_ErrorString(t *testing.T) {
	err := &core.ProviderError{Provider: "ollama", Model: "llama3", Code: 500, Err: errors.New("server error")}
	s := err.Error()
	if s != "provider ollama (model llama3, status 500): server error" {
		t.Errorf("Error() = %q", s)
	}

	err2 := &core.ProviderError{Provider: "ollama", Model: "llama3", Err: errors.New("timeout")}
	s2 := err2.Error()
	if s2 != "provider ollama (model llama3): timeout" {
		t.Errorf("Error() = %q", s2)
	}
}

func TestFallbackError_Is(t *testing.T) {
	err := &core.FallbackError{Errors: []error{errors.New("a"), errors.New("b")}}
	if !errors.Is(err, core.ErrProviderFailed) {
		t.Error("FallbackError should match ErrProviderFailed")
	}
}

func TestFallbackError_Unwrap(t *testing.T) {
	inner := errors.New("specific")
	err := &core.FallbackError{Errors: []error{inner, errors.New("other")}}
	if !errors.Is(err, inner) {
		t.Error("FallbackError should unwrap to find inner errors")
	}
}

func TestRetryError_Unwrap(t *testing.T) {
	inner := &core.ProviderError{Provider: "test", Kind: core.ErrorKindTransient, Err: errors.New("timeout")}
	err := &core.RetryError{Attempts: 3, Last: inner}
	if !errors.Is(err, core.ErrProviderFailed) {
		t.Error("RetryError should unwrap through ProviderError to match ErrProviderFailed")
	}
}

func TestIsTransient(t *testing.T) {
	transient := &core.ProviderError{Kind: core.ErrorKindTransient, Err: errors.New("timeout")}
	permanent := &core.ProviderError{Kind: core.ErrorKindPermanent, Err: errors.New("unauthorized")}
	plain := errors.New("something")

	if !core.IsTransient(transient) {
		t.Error("expected transient")
	}
	if core.IsTransient(permanent) {
		t.Error("expected not transient")
	}
	if core.IsTransient(plain) {
		t.Error("expected not transient for plain error")
	}
}

func TestIsTransient_FallbackError(t *testing.T) {
	transient := &core.ProviderError{Kind: core.ErrorKindTransient, Err: errors.New("timeout")}
	fe := &core.FallbackError{Errors: []error{transient}}
	if !core.IsTransient(fe) {
		t.Error("FallbackError with transient last error should be transient")
	}

	permanent := &core.ProviderError{Kind: core.ErrorKindPermanent, Err: errors.New("auth")}
	fe2 := &core.FallbackError{Errors: []error{permanent}}
	if core.IsTransient(fe2) {
		t.Error("FallbackError with permanent last error should not be transient")
	}
}

func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		code int
		want core.ErrorKind
	}{
		{200, core.ErrorKindPermanent},
		{400, core.ErrorKindPermanent},
		{401, core.ErrorKindPermanent},
		{403, core.ErrorKindPermanent},
		{404, core.ErrorKindPermanent},
		{408, core.ErrorKindTransient},
		{429, core.ErrorKindTransient},
		{500, core.ErrorKindTransient},
		{502, core.ErrorKindTransient},
		{503, core.ErrorKindTransient},
	}
	for _, tt := range tests {
		got := core.ClassifyHTTPStatus(tt.code)
		if got != tt.want {
			t.Errorf("ClassifyHTTPStatus(%d) = %d, want %d", tt.code, got, tt.want)
		}
	}
}
