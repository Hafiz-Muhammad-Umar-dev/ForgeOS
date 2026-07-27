package hitl

import (
	"context"
	"testing"
	"time"
)

func TestApprovalTypes(t *testing.T) {
	if ApprovalPlan != "plan" {
		t.Errorf("ApprovalPlan=%s", ApprovalPlan)
	}
	if ApprovalDeploy != "deploy" {
		t.Errorf("ApprovalDeploy=%s", ApprovalDeploy)
	}
}

func TestApprovalStatusValues(t *testing.T) {
	if ApprovalPending != "pending" {
		t.Errorf("pending=%s", ApprovalPending)
	}
	if ApprovalApproved != "approved" {
		t.Errorf("approved=%s", ApprovalApproved)
	}
	if ApprovalRejected != "rejected" {
		t.Errorf("rejected=%s", ApprovalRejected)
	}
	if ApprovalExpired != "expired" {
		t.Errorf("expired=%s", ApprovalExpired)
	}
}

func TestApprovalRequestDefaults(t *testing.T) {
	req := ApprovalRequest{
		ID: "req-1", IntentID: "intent-1", Type: ApprovalPlan,
		Summary: "approve plan", Status: ApprovalPending,
	}
	if req.ID != "req-1" {
		t.Errorf("id=%s", req.ID)
	}
	if req.Status != ApprovalPending {
		t.Errorf("status=%s", req.Status)
	}
}

func TestApprovalResult(t *testing.T) {
	res := ApprovalResult{
		RequestID: "req-1", Status: ApprovalApproved,
		ApprovedBy: "user-1", Reason: "looks good",
	}
	if res.Status != ApprovalApproved {
		t.Errorf("status=%s", res.Status)
	}
	if res.ApprovedBy != "user-1" {
		t.Errorf("by=%s", res.ApprovedBy)
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrTimeout == nil {
		t.Fatal("ErrTimeout is nil")
	}
}

// ---------------------------------------------------------------------------
// FakeHITLGate tests
// ---------------------------------------------------------------------------

func TestFakeHITLGateAutoApprove(t *testing.T) {
	g := NewFakeHITLGate()
	res, err := g.RequestApproval(context.Background(), ApprovalRequest{
		ID: "req-1", Type: ApprovalPlan, Summary: "test",
	})
	if err != nil {
		t.Fatalf("approval: %v", err)
	}
	if res.Status != ApprovalApproved {
		t.Errorf("status=%s", res.Status)
	}
	if g.RequestCount.Load() != 1 {
		t.Errorf("count=%d", g.RequestCount.Load())
	}
}

func TestFakeHITLGateAutoReject(t *testing.T) {
	g := NewFakeHITLGate()
	g.AutoApprove = false
	g.AutoReject = true

	res, err := g.RequestApproval(context.Background(), ApprovalRequest{
		ID: "req-1", Type: ApprovalPlan,
	})
	if err != nil {
		t.Fatalf("approval: %v", err)
	}
	if res.Status != ApprovalRejected {
		t.Errorf("status=%s", res.Status)
	}
}

func TestFakeHITLGateTimeout(t *testing.T) {
	g := NewFakeHITLGate()
	g.SimulateTimeout = true

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := g.RequestApproval(ctx, ApprovalRequest{ID: "req-1", Type: ApprovalPlan})
	if err != ErrTimeout && err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestFakeHITLGateRecordsRequests(t *testing.T) {
	g := NewFakeHITLGate()
	g.RequestApproval(context.Background(), ApprovalRequest{ID: "req-1", Type: ApprovalPlan})
	g.RequestApproval(context.Background(), ApprovalRequest{ID: "req-2", Type: ApprovalDeploy})

	if len(g.ReceivedRequests) != 2 {
		t.Fatalf("requests=%d", len(g.ReceivedRequests))
	}
	if g.ReceivedRequests[0].ID != "req-1" {
		t.Errorf("first=%s", g.ReceivedRequests[0].ID)
	}
	if g.ReceivedRequests[1].Type != ApprovalDeploy {
		t.Errorf("second type=%s", g.ReceivedRequests[1].Type)
	}
}

func TestFakeHITLGateDelay(t *testing.T) {
	g := NewFakeHITLGate()
	g.Delay = 10 * time.Millisecond

	start := time.Now()
	res, err := g.RequestApproval(context.Background(), ApprovalRequest{
		ID: "req-1", Type: ApprovalPlan,
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("approval: %v", err)
	}
	if res.Status != ApprovalApproved {
		t.Errorf("status=%s", res.Status)
	}
	if elapsed < 10*time.Millisecond {
		t.Errorf("elapsed=%s", elapsed)
	}
}

func TestMigrations(t *testing.T) {
	migs := Migrations()
	if len(migs) == 0 {
		t.Fatal("no migrations")
	}
	if migs[0].Version != 1 {
		t.Errorf("version=%d", migs[0].Version)
	}
	if migs[0].Up == "" {
		t.Error("empty up migration")
	}
}
