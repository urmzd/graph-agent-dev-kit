package types

import (
	"context"
	"time"
)

// Metrics collects operational telemetry from the agent loop.
type Metrics interface {
	RecordTokenUsage(ctx context.Context, operationName, provider string, input, output int)
	RecordToolCall(ctx context.Context, toolName string, duration time.Duration, err error)
	RecordProviderCall(ctx context.Context, operationName, provider string, duration time.Duration, err error)
	RecordAgentInvocation(ctx context.Context, agentID string, duration time.Duration)
}

// NoopMetrics is a no-op implementation of Metrics.
type NoopMetrics struct{}

func (NoopMetrics) RecordTokenUsage(context.Context, string, string, int, int)              {}
func (NoopMetrics) RecordToolCall(context.Context, string, time.Duration, error)             {}
func (NoopMetrics) RecordProviderCall(context.Context, string, string, time.Duration, error) {}
func (NoopMetrics) RecordAgentInvocation(context.Context, string, time.Duration)             {}
