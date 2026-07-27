package hitl

import (
	"context"
	"sync/atomic"
	"time"
)

// Compile-time check.
var _ HITLGate = (*FakeHITLGate)(nil)

// FakeHITLGate is an in-memory HITLGate for testing.
// It can auto-approve, auto-reject, or simulate timeout.
type FakeHITLGate struct {
	// AutoApprove makes all requests succeed immediately.
	AutoApprove bool

	// AutoReject makes all requests fail immediately.
	AutoReject bool

	// SimulateTimeout makes RequestApproval return ErrTimeout.
	SimulateTimeout bool

	// Delay simulates human response delay.
	Delay time.Duration

	// RequestCount tracks the number of RequestApproval calls.
	RequestCount atomic.Int64

	// ReceivedRequests records every request received.
	ReceivedRequests []ApprovalRequest
}

// NewFakeHITLGate creates a FakeHITLGate that auto-approves.
func NewFakeHITLGate() *FakeHITLGate {
	return &FakeHITLGate{AutoApprove: true}
}

// RequestApproval records the call and returns based on configuration.
func (f *FakeHITLGate) RequestApproval(ctx context.Context, req ApprovalRequest) (ApprovalResult, error) {
	f.RequestCount.Add(1)
	f.ReceivedRequests = append(f.ReceivedRequests, req)

	if f.SimulateTimeout {
		<-ctx.Done()
		return ApprovalResult{}, ErrTimeout
	}
	if f.Delay > 0 {
		select {
		case <-ctx.Done():
			return ApprovalResult{}, ErrTimeout
		case <-time.After(f.Delay):
		}
	}
	if f.AutoReject {
		return ApprovalResult{
			RequestID:  req.ID,
			Status:     ApprovalRejected,
			ApprovedBy: "fake-user",
			Reason:     "rejected by test",
			DecidedAt:  time.Now(),
		}, nil
	}
	return ApprovalResult{
		RequestID:  req.ID,
		Status:     ApprovalApproved,
		ApprovedBy: "fake-user",
		Reason:     "approved by test",
		DecidedAt:  time.Now(),
	}, nil
}
