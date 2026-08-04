package store

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

// Compile-time check.
var _ Store = (*FakeStore)(nil)

// FakeStore is an in-memory Store implementation for testing.
// It records all queries and returns configurable results.
type FakeStore struct {
	// QueryFunc overrides Query behavior.
	QueryFunc func(ctx context.Context, sql string, args ...any) (Rows, error)
	// QueryRowFunc overrides QueryRow behavior.
	QueryRowFunc func(ctx context.Context, sql string, args ...any) Row
	// ExecFunc overrides Exec behavior.
	ExecFunc func(ctx context.Context, sql string, args ...any) (int64, error)
	// BeginFunc overrides Begin behavior.
	BeginFunc func(ctx context.Context) (Tx, error)

	mu              sync.Mutex
	QueryCount      atomic.Int64
	ExecCount       atomic.Int64
	RecordedQueries []recordedQuery
}

type recordedQuery struct {
	SQL  string
	Args []any
}

// NewFakeStore creates a FakeStore.
func NewFakeStore() *FakeStore {
	return &FakeStore{}
}

func (f *FakeStore) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	f.QueryCount.Add(1)
	f.mu.Lock()
	f.RecordedQueries = append(f.RecordedQueries, recordedQuery{SQL: sql, Args: args})
	f.mu.Unlock()
	if f.QueryFunc != nil {
		return f.QueryFunc(ctx, sql, args...)
	}
	return &fakeRows{}, nil
}

func (f *FakeStore) QueryRow(ctx context.Context, sql string, args ...any) Row {
	f.mu.Lock()
	f.RecordedQueries = append(f.RecordedQueries, recordedQuery{SQL: sql, Args: args})
	f.mu.Unlock()
	if f.QueryRowFunc != nil {
		return f.QueryRowFunc(ctx, sql, args...)
	}
	return &fakeRow{}
}

func (f *FakeStore) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	f.ExecCount.Add(1)
	f.mu.Lock()
	f.RecordedQueries = append(f.RecordedQueries, recordedQuery{SQL: sql, Args: args})
	f.mu.Unlock()
	if f.ExecFunc != nil {
		return f.ExecFunc(ctx, sql, args...)
	}
	return 0, nil
}

func (f *FakeStore) Begin(ctx context.Context) (Tx, error) {
	if f.BeginFunc != nil {
		return f.BeginFunc(ctx)
	}
	return &fakeTx{}, nil
}

func (f *FakeStore) Ping(_ context.Context) error  { return nil }
func (f *FakeStore) Close(_ context.Context) error { return nil }

// ---------------------------------------------------------------------------
// Fake rows and tx
// ---------------------------------------------------------------------------

type fakeRows struct {
	index int
	data  []map[string]any
}

func (r *fakeRows) Next() bool { r.index++; return r.index <= len(r.data) }
func (r *fakeRows) Scan(dest ...any) error {
	if r.index-1 >= len(r.data) {
		return fmt.Errorf("fake: no rows")
	}
	return nil
}
func (r *fakeRows) Close() {}

type fakeRow struct{}

func (r *fakeRow) Scan(dest ...any) error { return nil }

type fakeTx struct{}

func (t *fakeTx) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return &fakeRows{}, nil
}
func (t *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) Row        { return &fakeRow{} }
func (t *fakeTx) Exec(ctx context.Context, sql string, args ...any) (int64, error) { return 0, nil }
func (t *fakeTx) Commit(ctx context.Context) error                                 { return nil }
func (t *fakeTx) Rollback(ctx context.Context) error                               { return nil }
