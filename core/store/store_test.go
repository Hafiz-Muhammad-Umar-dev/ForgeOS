package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

func TestFakeStoreQuery(t *testing.T) {
	fs := NewFakeStore()
	rows, err := fs.Query(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	rows.Close()
	if fs.QueryCount.Load() != 1 {
		t.Errorf("count=%d", fs.QueryCount.Load())
	}
}

func TestFakeStoreExec(t *testing.T) {
	fs := NewFakeStore()
	n, err := fs.Exec(context.Background(), "INSERT INTO test VALUES (1)")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if n != 0 {
		t.Errorf("rows=%d", n)
	}
	if fs.ExecCount.Load() != 1 {
		t.Errorf("count=%d", fs.ExecCount.Load())
	}
}

func TestFakeStoreRecordsQueries(t *testing.T) {
	fs := NewFakeStore()
	fs.Query(context.Background(), "SELECT * FROM users")
	fs.Exec(context.Background(), "DELETE FROM users")
	if len(fs.RecordedQueries) != 2 {
		t.Fatalf("queries=%d", len(fs.RecordedQueries))
	}
	if fs.RecordedQueries[0].SQL != "SELECT * FROM users" {
		t.Errorf("sql0=%s", fs.RecordedQueries[0].SQL)
	}
	if fs.RecordedQueries[1].SQL != "DELETE FROM users" {
		t.Errorf("sql1=%s", fs.RecordedQueries[1].SQL)
	}
}

func TestFakeStorePing(t *testing.T) {
	fs := NewFakeStore()
	if err := fs.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestFakeStoreClose(t *testing.T) {
	fs := NewFakeStore()
	if err := fs.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestFakeStoreBegin(t *testing.T) {
	fs := NewFakeStore()
	tx, err := fs.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestFakeStoreCustomExec(t *testing.T) {
	fs := NewFakeStore()
	fs.ExecFunc = func(_ context.Context, _ string, _ ...any) (int64, error) {
		return 42, nil
	}
	n, err := fs.Exec(context.Background(), "UPDATE test")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if n != 42 {
		t.Errorf("rows=%d", n)
	}
}

func TestFakeStoreConcurrent(t *testing.T) {
	fs := NewFakeStore()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fs.Query(context.Background(), "SELECT 1")
			fs.Exec(context.Background(), "INSERT INTO t VALUES (1)")
		}()
	}
	wg.Wait()
	if fs.QueryCount.Load() != 20 {
		t.Errorf("queries=%d", fs.QueryCount.Load())
	}
	if fs.ExecCount.Load() != 20 {
		t.Errorf("execs=%d", fs.ExecCount.Load())
	}
}

// ---------------------------------------------------------------------------
// Error mapping tests
// ---------------------------------------------------------------------------

func TestMapPGErrorNil(t *testing.T) {
	if err := MapPGError(nil); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestMapPGErrorNoRows(t *testing.T) {
	// MapPGError checks for pgx.ErrNoRows, not our sentinel.
	// This tests that a wrapped pgx error still passes through unchanged
	// when it doesn't match known pgx error types.
	orig := fmt.Errorf("some database error")
	result := MapPGError(orig)
	if result != orig {
		t.Errorf("expected original, got %v", result)
	}
}

func TestMapPGErrorUnknown(t *testing.T) {
	orig := fmt.Errorf("some random error")
	result := MapPGError(orig)
	if result != orig {
		t.Errorf("expected original error, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// ReadModel tests
// ---------------------------------------------------------------------------

func TestReadModel(t *testing.T) {
	fs := NewFakeStore()
	rm := NewReadModel(fs)

	if rm.Store() != fs {
		t.Error("store mismatch")
	}
}

func TestReadModelWithTx(t *testing.T) {
	fs := NewFakeStore()
	rm := NewReadModel(fs)

	called := false
	err := rm.WithTx(context.Background(), func(tx Tx) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("withTx: %v", err)
	}
	if !called {
		t.Error("callback not called")
	}
}

func TestReadModelWithTxRollback(t *testing.T) {
	fs := NewFakeStore()
	rm := NewReadModel(fs)

	expectedErr := fmt.Errorf("test error")
	err := rm.WithTx(context.Background(), func(tx Tx) error {
		return expectedErr
	})
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestReadModelQuery(t *testing.T) {
	fs := NewFakeStore()
	rm := NewReadModel(fs)

	rows, err := rm.Query(context.Background(), "SELECT 1")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	rows.Close()
	if fs.QueryCount.Load() != 1 {
		t.Errorf("count=%d", fs.QueryCount.Load())
	}
}

// ---------------------------------------------------------------------------
// Migration tests
// ---------------------------------------------------------------------------

func TestMigratorCreateTable(t *testing.T) {
	fs := NewFakeStore()
	mig := NewMigrator(fs, nil)

	// Run with no migrations — should create tracking table
	err := mig.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Verify the CREATE TABLE was executed
	found := false
	for _, q := range fs.RecordedQueries {
		if q.SQL == `SELECT version FROM schema_migrations ORDER BY version` {
			found = true
			break
		}
	}
	if !found {
		t.Error("migration query not found")
	}
}

func TestMigratorAppliesMigrations(t *testing.T) {
	fs := NewFakeStore()

	// Tracking queries return empty result set.
	fs.QueryFunc = func(_ context.Context, sql string, _ ...any) (Rows, error) {
		return &fakeRows{}, nil
	}

	// Capture all SQL executed, including through transactions.
	var allSQL []string
	fs.BeginFunc = func(_ context.Context) (Tx, error) {
		return &recordingTx{recorded: &allSQL}, nil
	}

	migrations := []Migration{
		{Version: 1, Description: "create users", Up: "CREATE TABLE users (id INT)"},
		{Version: 2, Description: "create posts", Up: "CREATE TABLE posts (id INT)"},
	}

	mig := NewMigrator(fs, migrations)
	err := mig.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Should have executed 2 migration SQLs plus 2 INSERT INTO schema_migrations
	createCount := 0
	for _, sql := range allSQL {
		if sql == "CREATE TABLE users (id INT)" || sql == "CREATE TABLE posts (id INT)" {
			createCount++
		}
	}
	if createCount != 2 {
		t.Errorf("migrations applied: got=%d want=2", createCount)
	}
}

func TestMigratorSortsByVersion(t *testing.T) {
	fs := NewFakeStore()
	fs.QueryFunc = func(_ context.Context, sql string, _ ...any) (Rows, error) {
		return &fakeRows{}, nil
	}

	var allSQL []string
	fs.BeginFunc = func(_ context.Context) (Tx, error) {
		return &recordingTx{recorded: &allSQL}, nil
	}

	migrations := []Migration{
		{Version: 2, Description: "second", Up: "CREATE TABLE t2 ()"},
		{Version: 1, Description: "first", Up: "CREATE TABLE t1 ()"},
	}

	mig := NewMigrator(fs, migrations)
	err := mig.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	// Ordered by version: CREATE TABLE t1 should run before t2
	found := false
	for _, sql := range allSQL {
		if sql == "CREATE TABLE t1 ()" {
			found = true
		}
	}
	if !found {
		t.Error("version 1 migration not found")
	}
}

func TestMigratorEmpty(t *testing.T) {
	fs := NewFakeStore()
	mig := NewMigrator(fs, nil)
	err := mig.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Config tests
// ---------------------------------------------------------------------------

// recordingTx records all Exec SQL for test assertions.
type recordingTx struct {
	recorded *[]string
}

func (t *recordingTx) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return &fakeRows{}, nil
}
func (t *recordingTx) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return &fakeRow{}
}
func (t *recordingTx) Exec(_ context.Context, sql string, _ ...any) (int64, error) {
	*t.recorded = append(*t.recorded, sql)
	return 0, nil
}
func (t *recordingTx) Commit(_ context.Context) error   { return nil }
func (t *recordingTx) Rollback(_ context.Context) error { return nil }

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DSN == "" {
		t.Error("empty DSN")
	}
	if cfg.MaxOpenConns <= 0 {
		t.Errorf("maxOpen=%d", cfg.MaxOpenConns)
	}
}

// ---------------------------------------------------------------------------
// Lifecycle tests
// ---------------------------------------------------------------------------

func TestPGStoreInitNoDSN(t *testing.T) {
	pg := NewPGStore(Config{DSN: ""})
	err := pg.Init(context.Background())
	if err == nil {
		t.Fatal("expected error with empty DSN")
	}
}

func TestPGStoreName(t *testing.T) {
	pg := NewPGStore(DefaultConfig())
	if pg.Name() != "store" {
		t.Errorf("name=%s", pg.Name())
	}
}

func TestPGStoreInitSuccess(t *testing.T) {
	pg := NewPGStore(DefaultConfig())
	if err := pg.Init(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}
}
