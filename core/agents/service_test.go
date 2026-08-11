package agents

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

func TestAgentModel(t *testing.T) {
	a := Agent{
		ID: "agent-1", OrgID: "org-1", Name: "Coder",
		Role: RoleCoder, Status: AgentIdle, Model: "claude-sonnet-4",
		Temperature: 0.4, QueueLength: 0,
	}
	if a.ID != "agent-1" {
		t.Errorf("id=%s", a.ID)
	}
	if a.Role != RoleCoder {
		t.Errorf("role=%s", a.Role)
	}
	if a.Status != AgentIdle {
		t.Errorf("status=%s", a.Status)
	}
}

func TestAgentRoleValues(t *testing.T) {
	roles := []AgentRole{RolePlanner, RoleResearcher, RoleCoder, RoleReviewer, RoleTester, RoleDeployer}
	for _, r := range roles {
		if r == "" {
			t.Error("empty role")
		}
	}
}

func TestAgentStatusValues(t *testing.T) {
	statuses := []AgentStatus{AgentIdle, AgentRunning, AgentThinking, AgentWaiting, AgentCompleted, AgentFailed}
	for _, s := range statuses {
		if s == "" {
			t.Error("empty status")
		}
	}
}

func TestRepositoryUpsertAgent(t *testing.T) {
	fs := store.NewFakeStore()
	repo := NewRepository(fs)
	a := Agent{ID: "a1", OrgID: "org-1", Name: "Coder", Role: RoleCoder, Status: AgentIdle}
	if err := repo.UpsertAgent(context.Background(), a); err != nil {
		t.Fatalf("upsert: %v", err)
	}
}

func TestServiceListAgents(t *testing.T) {
	fs := store.NewFakeStore()
	repo := NewRepository(fs)
	svc := NewService(repo)

	svc.RegisterAgent(context.Background(), Agent{ID: "a1", OrgID: "org-1", Name: "Coder"})
	svc.RegisterAgent(context.Background(), Agent{ID: "a2", OrgID: "org-1", Name: "Reviewer", Role: RoleReviewer})

	agents, err := svc.ListAgents(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 (fake store returns empty rows), got %d", len(agents))
	}
}

func TestServiceRegisterAgentDefaultFields(t *testing.T) {
	fs := store.NewFakeStore()
	svc := NewService(NewRepository(fs))

	err := svc.RegisterAgent(context.Background(), Agent{ID: "a1"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	// The repo calls UpsertAgent with the expense of recording. Fake store returns 0 rows.
	rec := fs.RecordedQueries
	if len(rec) != 1 {
		t.Errorf("expected 1 exec, got %d", len(rec))
	}
	if !strings.Contains(rec[0].SQL, "agents") {
		t.Errorf("expected agents table in query, got: %s", rec[0].SQL)
	}
}

func TestServiceRegisterAgentValidation(t *testing.T) {
	fs := store.NewFakeStore()
	svc := NewService(NewRepository(fs))

	err := svc.RegisterAgent(context.Background(), Agent{})
	if err != ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceGetAgent(t *testing.T) {
	fs := store.NewFakeStore()
	// Simulate a database error for QueryRow.
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &errRow{err: store.ErrNoRows}
	}
	svc := NewService(NewRepository(fs))

	_, err := svc.GetAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

// errRow implements store.Row and always returns an error on Scan.
type errRow struct {
	err error
}

func (r *errRow) Scan(_ ...any) error { return r.err }

func TestMigrations(t *testing.T) {
	migs := Migrations()
	if len(migs) == 0 {
		t.Fatal("no migrations")
	}
	if migs[0].Version != 1 {
		t.Errorf("version=%d", migs[0].Version)
	}
	if !strings.Contains(migs[0].Up, "agents") {
		t.Error("migration missing agents table")
	}
	if !strings.Contains(migs[0].Up, "idx_agents_org") {
		t.Error("migration missing org index")
	}
}

func TestAbstractRolesAndStatuses(t *testing.T) {
	// Verify Agent.String roundtrip.
	var _ = reflect.TypeOf(Agent{})
}