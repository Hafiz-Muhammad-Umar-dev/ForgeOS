package budget

import (
	"context"
	"testing"
)

func TestBudgetDefaults(t *testing.T) {
	b := Budget{
		OrgID: "org-1", TokenCeiling: 1000,
		TokenUsed: 100, TokenRemaining: 900, Status: BudgetOK,
	}
	if b.OrgID != "org-1" {
		t.Errorf("orgId=%s", b.OrgID)
	}
	if b.TokenRemaining != 900 {
		t.Errorf("remaining=%d", b.TokenRemaining)
	}
}

func TestBudgetExhausted(t *testing.T) {
	b := Budget{
		OrgID: "org-1", TokenCeiling: 100,
		TokenUsed: 100, TokenRemaining: 0, Status: BudgetExhausted,
	}
	if b.Status != BudgetExhausted {
		t.Errorf("status=%s", b.Status)
	}
}

func TestUsageTotal(t *testing.T) {
	u := Usage{InputTokens: 10, OutputTokens: 20}
	if u.TotalTokens() != 30 {
		t.Errorf("total=%d", u.TotalTokens())
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrBudgetExceeded == nil {
		t.Fatal("ErrBudgetExceeded is nil")
	}
}

// ---------------------------------------------------------------------------
// FakeGovernor tests
// ---------------------------------------------------------------------------

func TestFakeGovernorCheckOK(t *testing.T) {
	fg := NewFakeGovernor(1000)
	b, err := fg.Check(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if b.Status != BudgetOK {
		t.Errorf("status=%s", b.Status)
	}
	if b.TokenRemaining != 1000 {
		t.Errorf("remaining=%d", b.TokenRemaining)
	}
	if fg.CheckCount.Load() != 1 {
		t.Errorf("count=%d", fg.CheckCount.Load())
	}
}

func TestFakeGovernorConsume(t *testing.T) {
	fg := NewFakeGovernor(1000)
	err := fg.Consume(context.Background(), "org-1", Usage{InputTokens: 100, OutputTokens: 50})
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if fg.ConsumeCount.Load() != 1 {
		t.Errorf("count=%d", fg.ConsumeCount.Load())
	}
}

func TestFakeGovernorExhausted(t *testing.T) {
	fg := NewFakeGovernor(100)

	fg.Consume(context.Background(), "org-1", Usage{InputTokens: 80, OutputTokens: 20})
	fg.Consume(context.Background(), "org-1", Usage{InputTokens: 50, OutputTokens: 50})

	_, err := fg.Check(context.Background(), "org-1")
	if err == nil {
		t.Fatal("expected budget exceeded error")
	}
}

func TestFakeGovernorMultipleOrgs(t *testing.T) {
	fg := NewFakeGovernor(500)

	fg.Consume(context.Background(), "org-a", Usage{InputTokens: 100})
	fg.Consume(context.Background(), "org-b", Usage{InputTokens: 200})

	bA, _ := fg.Check(context.Background(), "org-a")
	bB, _ := fg.Check(context.Background(), "org-b")

	if bA.TokenUsed != 100 {
		t.Errorf("org-a used=%d", bA.TokenUsed)
	}
	if bB.TokenUsed != 200 {
		t.Errorf("org-b used=%d", bB.TokenUsed)
	}
}

func TestFakeGovernorZeroCeiling(t *testing.T) {
	fg := NewFakeGovernor(0)
	_, err := fg.Check(context.Background(), "org-1")
	if err == nil {
		t.Fatal("expected error with zero ceiling")
	}
}
