package agents

import "github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"

// Migrations returns the agents package database migrations.
func Migrations() []store.Migration {
	return []store.Migration{
		{
			Version:     1,
			Description: "create agents table",
			Up: `
				CREATE TABLE IF NOT EXISTS agents (
					id                 TEXT PRIMARY KEY,
					org_id             TEXT NOT NULL DEFAULT 'default',
					name               TEXT NOT NULL DEFAULT '',
					role               TEXT NOT NULL DEFAULT 'coder',
					status             TEXT NOT NULL DEFAULT 'idle',
					model              TEXT NOT NULL DEFAULT '',
					temperature        DOUBLE PRECISION NOT NULL DEFAULT 0.4,
					current_tool       TEXT NOT NULL DEFAULT '',
					reasoning          TEXT NOT NULL DEFAULT '',
					memory             TEXT NOT NULL DEFAULT '',
					output             TEXT NOT NULL DEFAULT '',
					queue_length       INTEGER NOT NULL DEFAULT 0,
					execution_time     BIGINT NOT NULL DEFAULT 0,
					prompt_tokens      INTEGER NOT NULL DEFAULT 0,
					completion_tokens  INTEGER NOT NULL DEFAULT 0,
					cost               DOUBLE PRECISION NOT NULL DEFAULT 0,
					created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
				CREATE INDEX IF NOT EXISTS idx_agents_id ON agents(id);
				CREATE INDEX IF NOT EXISTS idx_agents_org ON agents(org_id);
				CREATE INDEX IF NOT EXISTS idx_agents_role ON agents(role);
				CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);
			`,
			Down: `DROP TABLE IF EXISTS agents;`,
		},
	}
}