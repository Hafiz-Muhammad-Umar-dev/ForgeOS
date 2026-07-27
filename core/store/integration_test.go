//go:build integration

package store

import (
	"context"
	"os"
	"testing"
)

// getTestDSN returns the PostgreSQL DSN for integration tests.
// If the DATABASE_URL environment variable is not set, the test is skipped.
func getTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DEVOS_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	return dsn
}

// TestIntegrationPGStorePing verifies the PostgreSQL connection.
func TestIntegrationPGStorePing(t *testing.T) {
	dsn := getTestDSN(t)
	cfg := DefaultConfig()
	cfg.DSN = dsn

	pg := NewPGStore(cfg)
	ctx := context.Background()

	if err := pg.Init(ctx); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := pg.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer pg.Stop(ctx)

	if err := pg.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

// TestIntegrationPGStoreExecQuery verifies basic query execution.
func TestIntegrationPGStoreExecQuery(t *testing.T) {
	dsn := getTestDSN(t)
	cfg := DefaultConfig()
	cfg.DSN = dsn

	pg := NewPGStore(cfg)
	ctx := context.Background()

	if err := pg.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer pg.Stop(ctx)

	_, err := pg.Exec(ctx, "CREATE TABLE IF NOT EXISTS _test_integration (id INT PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = pg.Exec(ctx, "INSERT INTO _test_integration VALUES ($1, $2)", 1, "test")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := pg.Query(ctx, "SELECT id, name FROM _test_integration ORDER BY id")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
	}
	if count != 1 {
		t.Errorf("rows: got=%d want=1", count)
	}

	pg.Exec(ctx, "DROP TABLE _test_integration")
}

// TestIntegrationPGStoreTransaction verifies transaction commit and rollback.
func TestIntegrationPGStoreTransaction(t *testing.T) {
	dsn := getTestDSN(t)
	cfg := DefaultConfig()
	cfg.DSN = dsn

	pg := NewPGStore(cfg)
	ctx := context.Background()

	if err := pg.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer pg.Stop(ctx)

	pg.Exec(ctx, "CREATE TABLE IF NOT EXISTS _test_tx (id INT PRIMARY KEY)")

	// Test commit
	tx, err := pg.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	_, err = tx.Exec(ctx, "INSERT INTO _test_tx VALUES (1)")
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Verify
	row := pg.QueryRow(ctx, "SELECT COUNT(*) FROM _test_tx")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != 1 {
		t.Errorf("count: got=%d want=1", count)
	}

	pg.Exec(ctx, "DROP TABLE _test_tx")
}

// TestIntegrationMigrationRun verifies the migration runner against a real database.
func TestIntegrationMigrationRun(t *testing.T) {
	dsn := getTestDSN(t)
	cfg := DefaultConfig()
	cfg.DSN = dsn

	pg := NewPGStore(cfg)
	ctx := context.Background()

	if err := pg.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer pg.Stop(ctx)

	// Clean up previous test artifacts
	pg.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")
	pg.Exec(ctx, "DROP TABLE IF EXISTS _test_migration_users")

	migrations := []Migration{
		{Version: 1, Description: "create users", Up: "CREATE TABLE _test_migration_users (id INT PRIMARY KEY, email TEXT)"},
	}

	mig := NewMigrator(pg, migrations)
	if err := mig.Run(ctx); err != nil {
		t.Fatalf("migration: %v", err)
	}

	// Verify migration was recorded
	row := pg.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations")
	var count int
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != 1 {
		t.Errorf("migrations: got=%d want=1", count)
	}

	// Second run should be idempotent
	if err := mig.Run(ctx); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	row = pg.QueryRow(ctx, "SELECT COUNT(*) FROM schema_migrations")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != 1 {
		t.Errorf("migrations after re-run: got=%d want=1", count)
	}

	pg.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")
	pg.Exec(ctx, "DROP TABLE IF EXISTS _test_migration_users")
}

// TestIntegrationReadModelWithTx verifies transactions through ReadModel.
func TestIntegrationReadModelWithTx(t *testing.T) {
	dsn := getTestDSN(t)
	cfg := DefaultConfig()
	cfg.DSN = dsn

	pg := NewPGStore(cfg)
	ctx := context.Background()

	if err := pg.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer pg.Stop(ctx)

	pg.Exec(ctx, "CREATE TABLE IF NOT EXISTS _test_rm (id INT PRIMARY KEY)")
	defer pg.Exec(ctx, "DROP TABLE IF EXISTS _test_rm")

	rm := NewReadModel(pg)
	err := rm.WithTx(ctx, func(tx Tx) error {
		_, err := tx.Exec(ctx, "INSERT INTO _test_rm VALUES (42)")
		return err
	})
	if err != nil {
		t.Fatalf("withTx: %v", err)
	}

	var val int
	row := rm.QueryRow(ctx, "SELECT id FROM _test_rm WHERE id = 42")
	if err := row.Scan(&val); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if val != 42 {
		t.Errorf("val: got=%d want=42", val)
	}
}
