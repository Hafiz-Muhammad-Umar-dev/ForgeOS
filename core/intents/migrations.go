package intents

import "github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"

// Migrations returns the intents package database migrations.
func Migrations() []store.Migration {
	return []store.Migration{
		{
			Version:     1,
			Description: "create intents and tasks tables",
			Up: `
				CREATE TABLE IF NOT EXISTS intents (
					id         TEXT PRIMARY KEY,
					user_id    TEXT NOT NULL DEFAULT '',
					org_id     TEXT NOT NULL DEFAULT '',
					project_id TEXT NOT NULL DEFAULT '',
					trace_id   TEXT NOT NULL DEFAULT '',
					text       TEXT NOT NULL DEFAULT '',
					status     TEXT NOT NULL DEFAULT 'pending',
					summary    TEXT NOT NULL DEFAULT '',
					error      TEXT NOT NULL DEFAULT '',
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
				CREATE INDEX IF NOT EXISTS idx_intents_org ON intents(org_id);
				CREATE INDEX IF NOT EXISTS idx_intents_created ON intents(created_at DESC);
				CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);

				CREATE TABLE IF NOT EXISTS tasks (
					id            TEXT PRIMARY KEY,
					intent_id     TEXT NOT NULL,
					agent_name    TEXT NOT NULL DEFAULT '',
					status        TEXT NOT NULL DEFAULT '',
					summary       TEXT NOT NULL DEFAULT '',
					error         TEXT NOT NULL DEFAULT '',
					input_tokens  INTEGER NOT NULL DEFAULT 0,
					output_tokens INTEGER NOT NULL DEFAULT 0,
					created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
				CREATE INDEX IF NOT EXISTS idx_tasks_intent ON tasks(intent_id);
			`,
			Down: `
				DROP TABLE IF EXISTS tasks;
				DROP TABLE IF EXISTS intents;
			`,
		},
	}
}
