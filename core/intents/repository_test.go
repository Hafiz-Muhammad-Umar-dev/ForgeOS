package intents

import (
	"context"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

// newFakeStoreWithCreate returns a FakeStore configured to return a valid
// intent on QueryRow and empty slices on Query.
func newFakeStoreWithCreate() *store.FakeStore {
	fs := store.NewFakeStore()

	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &intentRow{id: args[0].(string)}
	}
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &emptyRows{}, nil
	}
	return fs
}

// intentRow is a store.Row that scans a fixed intent.
type intentRow struct {
	id  string
	org string
}

func (r *intentRow) Scan(dest ...any) error {
	// dest[0..] correspond to the intent fields.
	org := r.org
	if org == "" {
		org = "org-1"
	}
	now := time.Now()
	*(dest[0].(*string)) = r.id
	*(dest[1].(*string)) = "dev-admin"
	*(dest[2].(*string)) = org
	*(dest[3].(*string)) = ""
	*(dest[4].(*string)) = ""
	*(dest[5].(*string)) = "Build auth"
	*(dest[6].(*string)) = "pending"
	*(dest[7].(*string)) = ""
	*(dest[8].(*string)) = ""
	*(dest[9].(*time.Time)) = now
	*(dest[10].(*time.Time)) = now
	return nil
}

// emptyRows is a store.Rows that yields no rows.
type emptyRows struct{}

func (r *emptyRows) Next() bool             { return false }
func (r *emptyRows) Scan(dest ...any) error { return nil }
func (r *emptyRows) Close()                 {}

// TestRepositoryCreateIntent verifies the repository issues an INSERT.
func TestRepositoryCreateIntent(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &intentRow{id: args[0].(string)}
	}
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &emptyRows{}, nil
	}
	repo := NewRepository(fs)

	intent, err := repo.CreateIntent(context.Background(), NewIntentRequest{
		Text: "hello", UserID: "u1", OrgID: "org-1",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if intent.ID == "" {
		t.Error("expected generated id")
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("exec count: got=%d want=1", fs.ExecCount.Load())
	}
}

// TestRepositoryGetIntent verifies the repository issues a SELECT and scans it.
func TestRepositoryGetIntent(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &intentRow{id: args[0].(string)}
	}
	repo := NewRepository(fs)

	intent, err := repo.GetIntent(context.Background(), "intent-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if intent.ID != "intent-1" {
		t.Errorf("id: got=%s", intent.ID)
	}
	if intent.Status != "pending" {
		t.Errorf("status: got=%s", intent.Status)
	}
}

// TestRepositoryGetIntentRecorded verifies the SELECT SQL was issued.
func TestRepositoryGetIntentRecorded(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryRowFunc = func(ctx context.Context, sql string, args ...any) store.Row {
		return &intentRow{id: args[0].(string)}
	}
	repo := NewRepository(fs)

	_, err := repo.GetIntent(context.Background(), "intent-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	found := false
	for _, q := range fs.RecordedQueries {
		if q.SQL == sqlGetIntent {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected sqlGetIntent to be issued")
	}
}

// TestRepositoryListIntents verifies list issues a query.
func TestRepositoryListIntents(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &emptyRows{}, nil
	}
	repo := NewRepository(fs)

	intents, err := repo.ListIntents(context.Background(), "org-1", 20, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if intents == nil {
		t.Error("expected non-nil slice")
	}
}

// TestRepositoryListTasks verifies task listing.
func TestRepositoryListTasks(t *testing.T) {
	fs := store.NewFakeStore()
	fs.QueryFunc = func(ctx context.Context, sql string, args ...any) (store.Rows, error) {
		return &emptyRows{}, nil
	}
	repo := NewRepository(fs)

	tasks, err := repo.ListTasks(context.Background(), "intent-1")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if tasks == nil {
		t.Error("expected non-nil slice")
	}
}
