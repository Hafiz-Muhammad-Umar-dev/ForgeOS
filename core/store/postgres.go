package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/lifecycle"
)

// Compile-time checks.
var (
	_ Store              = (*PGStore)(nil)
	_ lifecycle.Component = (*PGStore)(nil)
)

// PGStore is a PostgreSQL adapter implementing the Store port.
// It uses pgx/v5 connection pooling and implements lifecycle.Component.
type PGStore struct {
	config Config
	pool   *pgxpool.Pool
}

// NewPGStore creates a new PostgreSQL store adapter.
func NewPGStore(cfg Config) *PGStore {
	return &PGStore{config: cfg}
}

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

// Query executes a query returning multiple rows.
func (p *PGStore) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, MapPGError(err)
	}
	return &pgxRowsAdapter{rows: rows}, nil
}

// QueryRow executes a query returning at most one row.
func (p *PGStore) QueryRow(ctx context.Context, sql string, args ...any) Row {
	row := p.pool.QueryRow(ctx, sql, args...)
	return &pgxRowAdapter{row: row}
}

// Exec executes a query that returns no rows.
func (p *PGStore) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := p.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, MapPGError(err)
	}
	return tag.RowsAffected(), nil
}

// Begin starts a new transaction.
func (p *PGStore) Begin(ctx context.Context) (Tx, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: begin tx: %w", MapPGError(err))
	}
	return &pgxTxAdapter{tx: tx}, nil
}

// Ping verifies the database connection.
func (p *PGStore) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close closes the connection pool.
func (p *PGStore) Close(_ context.Context) error {
	if p.pool != nil {
		p.pool.Close()
	}
	return nil
}

// Pool returns the underlying pgxpool.Pool for use by migrations and raw access.
func (p *PGStore) Pool() *pgxpool.Pool {
	return p.pool
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// Name returns "store" for the lifecycle manager.
func (p *PGStore) Name() string { return "store" }

// Init validates configuration.
func (p *PGStore) Init(_ context.Context) error {
	if p.config.DSN == "" {
		return fmt.Errorf("store: %w: DSN is required", ErrConnectionFailed)
	}
	return nil
}

// Start connects to PostgreSQL and initializes the pool.
func (p *PGStore) Start(ctx context.Context) error {
	connCfg, err := pgxpool.ParseConfig(p.config.DSN)
	if err != nil {
		return fmt.Errorf("store: parse dsn: %w", err)
	}

	connCfg.MaxConns = int32(p.config.MaxOpenConns)
	connCfg.MinConns = int32(p.config.MaxIdleConns)
	connCfg.MaxConnLifetime = p.config.ConnMaxLifetime
	connCfg.MaxConnIdleTime = p.config.ConnMaxIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, connCfg)
	if err != nil {
		return fmt.Errorf("store: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("store: ping: %w", err)
	}

	p.pool = pool
	return nil
}

// Stop closes the connection pool.
func (p *PGStore) Stop(ctx context.Context) error {
	return p.Close(ctx)
}

// Health reports whether the database is reachable.
func (p *PGStore) Health() lifecycle.Health {
	if p.pool == nil {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := p.pool.Ping(ctx); err != nil {
		return lifecycle.Health{Status: lifecycle.StatusDown, Since: time.Now(), Message: err.Error()}
	}
	return lifecycle.Health{Status: lifecycle.StatusUp, Since: time.Now()}
}

// ---------------------------------------------------------------------------
// pgx adapter types
// ---------------------------------------------------------------------------

type pgxRowsAdapter struct {
	rows pgx.Rows
}

func (r *pgxRowsAdapter) Next() bool { return r.rows.Next() }

func (r *pgxRowsAdapter) Scan(dest ...any) error {
	err := r.rows.Scan(dest...)
	return MapPGError(err)
}

func (r *pgxRowsAdapter) Close() { r.rows.Close() }

type pgxRowAdapter struct {
	row pgx.Row
}

func (r *pgxRowAdapter) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	return MapPGError(err)
}

type pgxTxAdapter struct {
	tx pgx.Tx
}

func (t *pgxTxAdapter) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, MapPGError(err)
	}
	return &pgxRowsAdapter{rows: rows}, nil
}

func (t *pgxTxAdapter) QueryRow(ctx context.Context, sql string, args ...any) Row {
	row := t.tx.QueryRow(ctx, sql, args...)
	return &pgxRowAdapter{row: row}
}

func (t *pgxTxAdapter) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	tag, err := t.tx.Exec(ctx, sql, args...)
	if err != nil {
		return 0, MapPGError(err)
	}
	return tag.RowsAffected(), nil
}

func (t *pgxTxAdapter) Commit(ctx context.Context) error {
	err := t.tx.Commit(ctx)
	return MapPGError(err)
}

func (t *pgxTxAdapter) Rollback(ctx context.Context) error {
	err := t.tx.Rollback(ctx)
	return MapPGError(err)
}

// Compile-time checks for adapter types.
var (
	_ Rows = (*pgxRowsAdapter)(nil)
	_ Row  = (*pgxRowAdapter)(nil)
	_ Tx   = (*pgxTxAdapter)(nil)
)
