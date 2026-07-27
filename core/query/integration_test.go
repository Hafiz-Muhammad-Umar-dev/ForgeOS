//go:build integration

package query

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/event"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/orchestrator"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

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

func TestIntegrationMigrations(t *testing.T) {
	dsn := getTestDSN(t)
	cfg := store.DefaultConfig()
	cfg.DSN = dsn

	pg := store.NewPGStore(cfg)
	ctx := context.Background()
	if err := pg.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer pg.Stop(ctx)

	// Clean up from previous runs
	pg.Exec(ctx, "DROP TABLE IF EXISTS query_tasks")
	pg.Exec(ctx, "DROP TABLE IF EXISTS query_intents")
	pg.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")

	mig := store.NewMigrator(pg, Migrations())
	if err := mig.Run(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Verify tables exist
	rows, err := pg.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_name IN ('query_intents', 'query_tasks') ORDER BY table_name")
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tables = append(tables, name)
	}
	if len(tables) != 2 {
		t.Errorf("tables found: %v", tables)
	}

	// Verify indexes exist
	idxRows, err := pg.Query(ctx, "SELECT indexname FROM pg_indexes WHERE indexname LIKE 'idx_query_%' ORDER BY indexname")
	if err != nil {
		t.Fatalf("query indexes: %v", err)
	}
	defer idxRows.Close()

	var indexes []string
	for idxRows.Next() {
		var name string
		if err := idxRows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		indexes = append(indexes, name)
	}
	if len(indexes) != 4 {
		t.Errorf("indexes found: %v", indexes)
	}

	pg.Exec(ctx, "DROP TABLE IF EXISTS query_tasks CASCADE")
	pg.Exec(ctx, "DROP TABLE IF EXISTS query_intents CASCADE")
	pg.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations CASCADE")
}

func TestIntegrationProjectIntentAndQuery(t *testing.T) {
	dsn := getTestDSN(t)
	cfg := store.DefaultConfig()
	cfg.DSN = dsn

	pg := store.NewPGStore(cfg)
	ctx := context.Background()
	if err := pg.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer pg.Stop(ctx)

	pg.Exec(ctx, "DROP TABLE IF EXISTS query_tasks")
	pg.Exec(ctx, "DROP TABLE IF EXISTS query_intents")
	pg.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")

	mig := store.NewMigrator(pg, Migrations())
	if err := mig.Run(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Project an intent
	env := event.RawEnvelope{
		ID:         "evt-int-1",
		Type:       event.TypeIntentCompleted,
		OrgID:      "org-test",
		ProjectID:  "proj-test",
		TraceID:    "trace-test",
		ProducedAt: time.Now().UnixNano(),
		Payload:    mustMarshalPayload(t, orchestrator.IntentLifecyclePayload{IntentID: "intent-integration", Summary: "integration test"}),
	}

	proj := NewIntentProjection(pg)
	if err := proj.Handle(ctx, env); err != nil {
		t.Fatalf("project: %v", err)
	}

	// Query the projected intent
	s := NewService(pg)
	fetched, err := s.GetIntent(ctx, "intent-integration")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fetched.OrgID != "org-test" {
		t.Errorf("orgId: got=%s", fetched.OrgID)
	}
	if fetched.Status != "completed" {
		t.Errorf("status: got=%s", fetched.Status)
	}

	pg.Exec(ctx, "DROP TABLE IF EXISTS query_tasks CASCADE")
	pg.Exec(ctx, "DROP TABLE IF EXISTS query_intents CASCADE")
	pg.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations CASCADE")
}
