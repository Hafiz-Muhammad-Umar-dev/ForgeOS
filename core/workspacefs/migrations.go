package workspacefs

import "github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"

// Migrations returns the workspacefs package database migrations.
func Migrations() []store.Migration {
	return []store.Migration{
		{
			Version:     2,
			Description: "create workspaces and workspace_files tables",
			Up: `
				CREATE TABLE IF NOT EXISTS workspaces (
					id         TEXT PRIMARY KEY,
					name       TEXT NOT NULL DEFAULT '',
					root       TEXT NOT NULL DEFAULT '/',
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);

				CREATE TABLE IF NOT EXISTS workspace_files (
					id           TEXT PRIMARY KEY,
					workspace_id TEXT NOT NULL,
					name         TEXT NOT NULL,
					path         TEXT NOT NULL,
					size         INTEGER NOT NULL DEFAULT 0,
					content      TEXT NOT NULL DEFAULT '',
					is_folder    BOOLEAN NOT NULL DEFAULT FALSE,
					created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					UNIQUE (workspace_id, path)
				);
				CREATE INDEX IF NOT EXISTS idx_workspace_files_ws ON workspace_files(workspace_id);
				CREATE INDEX IF NOT EXISTS idx_workspace_files_path ON workspace_files(path);
			`,
			Down: `
				DROP TABLE IF EXISTS workspace_files;
				DROP TABLE IF EXISTS workspaces;
			`,
		},
	}
}
