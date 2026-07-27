package budget

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Compile-time check.
var _ Governor = (*FakeGovernor)(nil)

// FakeGovernor is an in-memory Governor for testing.
type FakeGovernor struct {
	// Ceiling is the token ceiling for all orgs.
	Ceiling int64

	mu         sync.Mutex
	usage      map[string]int64 // orgID → total tokens used
	CheckCount atomic.Int64
	ConsumeCount atomic.Int64
}

// NewFakeGovernor creates a FakeGovernor with the given token ceiling.
func NewFakeGovernor(ceiling int64) *FakeGovernor {
	return &FakeGovernor{
		Ceiling: ceiling,
		usage:   make(map[string]int64),
	}
}

func (f *FakeGovernor) Check(_ context.Context, orgID string) (Budget, error) {
	f.CheckCount.Add(1)
	f.mu.Lock()
	used := f.usage[orgID]
	f.mu.Unlock()

	remaining := f.Ceiling - used
	if remaining <= 0 {
		return Budget{
			OrgID: orgID, TokenCeiling: f.Ceiling,
			TokenUsed: used, TokenRemaining: 0,
			Status: BudgetExhausted,
		}, fmt.Errorf("%w: org=%s ceiling=%d used=%d", ErrBudgetExceeded, orgID, f.Ceiling, used)
	}
	return Budget{
		OrgID: orgID, TokenCeiling: f.Ceiling,
		TokenUsed: used, TokenRemaining: remaining,
		Status: BudgetOK,
	}, nil
}

func (f *FakeGovernor) Consume(_ context.Context, orgID string, usage Usage) error {
	f.ConsumeCount.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.usage[orgID] += usage.TotalTokens()
	return nil
}
