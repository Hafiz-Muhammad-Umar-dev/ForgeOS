package store

import "context"

// ReadModel provides a base set of query helpers for building read models.
// Embed it in projection services to access the store with consistent error handling.
type ReadModel struct {
	store Store
}

// NewReadModel creates a ReadModel backed by the given store.
func NewReadModel(s Store) *ReadModel {
	return &ReadModel{store: s}
}

// Store returns the underlying store.
func (r *ReadModel) Store() Store { return r.store }

// Query executes a read query and returns rows.
func (r *ReadModel) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return r.store.Query(ctx, sql, args...)
}

// QueryRow executes a read query returning a single row.
func (r *ReadModel) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return r.store.QueryRow(ctx, sql, args...)
}

// Exec executes a write query.
func (r *ReadModel) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	return r.store.Exec(ctx, sql, args...)
}

// WithTx executes the given function within a transaction, committing on success
// and rolling back on error.
func (r *ReadModel) WithTx(ctx context.Context, fn func(tx Tx) error) error {
	tx, err := r.store.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		// Only rollback if not already committed.
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
