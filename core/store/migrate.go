package store

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Migration represents a database migration.
type Migration struct {
	// Version is a unique, monotonically increasing migration number.
	Version int

	// Description describes what the migration does.
	Description string

	// Up is the SQL to apply the migration.
	Up string

	// Down is the SQL to roll back the migration (can be empty).
	Down string
}

// Migrator runs database migrations.
type Migrator struct {
	store      Store
	migrations []Migration
}

// NewMigrator creates a migrator with the given migrations.
func NewMigrator(store Store, migrations []Migration) *Migrator {
	return &Migrator{
		store:      store,
		migrations: migrations,
	}
}

// Run applies all pending migrations in version order.
func (m *Migrator) Run(ctx context.Context) error {
	if err := m.ensureTable(ctx); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	appliedSet := make(map[int]bool)
	for _, v := range applied {
		appliedSet[v] = true
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	for _, mig := range m.migrations {
		if appliedSet[mig.Version] {
			continue
		}

		tx, err := m.store.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate %d: begin: %w", mig.Version, err)
		}

		if _, err := tx.Exec(ctx, mig.Up); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migrate %d %s: %w", mig.Version, mig.Description, MapPGError(err))
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version, description, applied_at) VALUES ($1, $2, $3)`,
			mig.Version, mig.Description, time.Now(),
		); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migrate %d: record: %w", mig.Version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate %d: commit: %w", mig.Version, err)
		}
	}

	return nil
}

// appliedVersions returns the list of already-applied migration versions.
func (m *Migrator) appliedVersions(ctx context.Context) ([]int, error) {
	rows, err := m.store.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, nil
}

// ensureTable creates the schema_migrations tracking table if it doesn't exist.
func (m *Migrator) ensureTable(ctx context.Context) error {
	_, err := m.store.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version     INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}
