// Package store defines the database store abstraction for DevOS. It follows the
// ports/adapters (hexagonal) architecture: core services use Store, and adapters
// (PostgreSQL, in-memory for tests) satisfy it without leaking database SDK types.
//
// Sprint 0 scope:
//   - Store interface (Query, QueryRow, Exec, Begin, Ping, Close)
//   - PostgreSQL adapter (pgx/v5 connection pool)
//   - Environment-driven configuration
//   - Migration runner
//   - Repository helpers (ReadModel)
//   - FakeStore for tests
//
// See Build Order Step 3 (PostgreSQL + migrations + read-model base).
package store

import "context"

// Store is the core database abstraction. Implementations provide connection
// pooling, query execution, and transaction management.
type Store interface {
	// Query executes a query that returns multiple rows.
	Query(ctx context.Context, sql string, args ...any) (Rows, error)

	// QueryRow executes a query that returns at most one row.
	QueryRow(ctx context.Context, sql string, args ...any) Row

	// Exec executes a query that returns no rows (INSERT, UPDATE, DELETE, DDL).
	// Returns the number of rows affected.
	Exec(ctx context.Context, sql string, args ...any) (int64, error)

	// Begin starts a new transaction.
	Begin(ctx context.Context) (Tx, error)

	// Ping verifies the database connection is alive.
	Ping(ctx context.Context) error

	// Close closes the connection pool and releases resources.
	Close(ctx context.Context) error
}

// Tx wraps a database transaction. Commit or Rollback must be called to
// release resources.
type Tx interface {
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Rows is an iterator over query results.
type Rows interface {
	// Next prepares the next row for reading. Returns false when done.
	Next() bool

	// Scan copies the current row's columns into the provided destinations.
	Scan(dest ...any) error

	// Close closes the rows iterator.
	Close()
}

// Row is the result of a QueryRow call.
type Row interface {
	// Scan copies the row's columns into the provided destinations.
	// Returns ErrNoRows if there is no row.
	Scan(dest ...any) error
}
