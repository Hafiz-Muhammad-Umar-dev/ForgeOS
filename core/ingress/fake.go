package ingress

import (
	"context"
	"sync/atomic"
)

// Compile-time check.
var _ IntentIngress = (*FakeIngress)(nil)

// FakeIngress is an in-memory IntentIngress implementation for testing.
// It records all submissions and returns configurable results.
type FakeIngress struct {
	// SubmitFunc overrides the SubmitIntent behavior.
	SubmitFunc func(ctx context.Context, payload IntentPayload) (IntentResult, error)

	// ResultValue is returned by the default SubmitIntent implementation.
	ResultValue IntentResult

	// SubmitCount tracks the number of SubmitIntent calls.
	SubmitCount atomic.Int64

	// ReceivedPayloads records every payload received.
	ReceivedPayloads []IntentPayload
}

// NewFakeIngress creates a FakeIngress with a default success result.
func NewFakeIngress() *FakeIngress {
	return &FakeIngress{
		ResultValue: IntentResult{
			IntentID: "intent-fake-1",
			Status:   "accepted",
			TraceID:  "trace-fake-1",
		},
	}
}

// SubmitIntent records the call and returns the configured result.
func (f *FakeIngress) SubmitIntent(_ context.Context, payload IntentPayload) (IntentResult, error) {
	f.SubmitCount.Add(1)
	f.ReceivedPayloads = append(f.ReceivedPayloads, payload)

	if f.SubmitFunc != nil {
		return f.SubmitFunc(nil, payload)
	}
	return f.ResultValue, nil
}
