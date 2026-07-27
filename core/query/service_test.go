package query

import (
	"context"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

func TestRepositoryUpsertIntent(t *testing.T) {
	fs := store.NewFakeStore()
	repo := NewRepository(fs)

	err := repo.UpsertIntent(context.Background(),
		"intent-1", "user-1", "org-1", "proj-1", "trace-1",
		"build an app", "completed", "built", "", time.Now())
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("exec count: got=%d", fs.ExecCount.Load())
	}
}

func TestRepositoryGetIntentNotFound(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(_ context.Context, _ string, _ ...any) store.Row {
		return &errRow{err: store.ErrNoRows}
	}
	repo := NewRepository(fs)
	_, err := repo.GetIntent(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRepositoryUpsertTask(t *testing.T) {
	fs := store.NewFakeStore()
	repo := NewRepository(fs)

	err := repo.UpsertTask(context.Background(),
		"task-1", "intent-1", "coder", "completed", "done", "",
		100, 50, time.Now())
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("exec count: got=%d", fs.ExecCount.Load())
	}
}

func TestRepositoryGetTaskNotFound(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(_ context.Context, _ string, _ ...any) store.Row {
		return &errRow{err: store.ErrNoRows}
	}
	repo := NewRepository(fs)
	_, err := repo.GetTask(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFakeServiceGetIntent(t *testing.T) {
	f := NewFakeQueryService()
	f.AddIntent(IntentView{ID: "i1", OrgID: "org-1", Status: "completed"})
	v, err := f.GetIntent(context.Background(), "i1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v.ID != "i1" {
		t.Errorf("id=%s", v.ID)
	}
}

func TestFakeServiceListIntents(t *testing.T) {
	f := NewFakeQueryService()
	f.AddIntent(IntentView{ID: "i1", OrgID: "org-1"})
	f.AddIntent(IntentView{ID: "i2", OrgID: "org-1"})
	results, err := f.ListIntents(context.Background(), "org-1", 10, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results=%d", len(results))
	}
}

func TestFakeServiceGetTask(t *testing.T) {
	f := NewFakeQueryService()
	f.AddTask(TaskView{ID: "t1", IntentID: "i1", Status: "done"})
	v, err := f.GetTask(context.Background(), "t1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v.ID != "t1" {
		t.Errorf("id=%s", v.ID)
	}
}

func TestFakeServiceListTasks(t *testing.T) {
	f := NewFakeQueryService()
	f.AddTask(TaskView{ID: "t1", IntentID: "i1"})
	f.AddTask(TaskView{ID: "t2", IntentID: "i1"})
	results, err := f.ListTasks(context.Background(), "i1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("results=%d", len(results))
	}
}

// errRow implements store.Row and always returns an error on Scan.
type errRow struct{ err error }

func (r *errRow) Scan(_ ...any) error { return r.err }

func TestMigrationsAreValid(t *testing.T) {
	migs := Migrations()
	if len(migs) == 0 {
		t.Fatal("no migrations defined")
	}
	if migs[0].Version != 1 {
		t.Errorf("version: got=%d", migs[0].Version)
	}
	if migs[0].Up == "" {
		t.Error("empty up migration")
	}
	if migs[0].Down == "" {
		t.Error("empty down migration")
	}
}

func TestEventTypesExist(t *testing.T) {
	if event.TypeIntentCompleted == "" {
		t.Error("TypeIntentCompleted not set")
	}
	if event.TypeIntentFailed == "" {
		t.Error("TypeIntentFailed not set")
	}
	if event.TypeTaskStatus == "" {
		t.Error("TypeTaskStatus not set")
	}
	if event.TypeTaskFailed == "" {
		t.Error("TypeTaskFailed not set")
	}
}

func TestIntentLifecyclePayloadFields(t *testing.T) {
	p := orchestrator.IntentLifecyclePayload{
		IntentID: "i1", Summary: "done", UserID: "u1", OrgID: "o1",
	}
	if p.IntentID != "i1" {
		t.Errorf("id=%s", p.IntentID)
	}
}
