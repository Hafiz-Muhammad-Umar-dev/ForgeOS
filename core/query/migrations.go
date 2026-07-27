package query

import "github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"

// Migrations returns the Query Service's database migrations.
func Migrations() []store.Migration {
	return []store.Migration{
		{
			Version:     1,
			Description: "create query_intents and query_tasks read models",
			Up: `
				CREATE TABLE IF NOT EXISTS query_intents (
					id         TEXT PRIMARY KEY,
					user_id    TEXT NOT NULL DEFAULT '',
					org_id     TEXT NOT NULL DEFAULT '',
					project_id TEXT NOT NULL DEFAULT '',
					trace_id   TEXT NOT NULL DEFAULT '',
					text       TEXT NOT NULL DEFAULT '',
					status     TEXT NOT NULL DEFAULT '',
					summary    TEXT NOT NULL DEFAULT '',
					error      TEXT NOT NULL DEFAULT '',
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
				CREATE INDEX IF NOT EXISTS idx_query_intents_org ON query_intents(org_id);
				CREATE INDEX IF NOT EXISTS idx_query_intents_created ON query_intents(created_at DESC);
				CREATE INDEX IF NOT EXISTS idx_query_intents_status ON query_intents(status);

				CREATE TABLE IF NOT EXISTS query_tasks (
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
				CREATE INDEX IF NOT EXISTS idx_query_tasks_intent ON query_tasks(intent_id);
			`,
			Down: `
				DROP TABLE IF EXISTS query_tasks;
				DROP TABLE IF EXISTS query_intents;
			`,
		},
	}
}
