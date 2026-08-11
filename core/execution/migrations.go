package execution

import "github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"

// Migrations returns the execution package database migrations.
func Migrations() []store.Migration {
	return []store.Migration{
		{
			Version:     1,
			Description: "create execution tables",
			Up: `
				CREATE TABLE IF NOT EXISTS executions (
					id           TEXT PRIMARY KEY,
					intent_id    TEXT NOT NULL DEFAULT '',
					agent_name   TEXT NOT NULL DEFAULT '',
					status       TEXT NOT NULL DEFAULT 'pending',
					org_id       TEXT NOT NULL DEFAULT 'default',
					created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					started_at   TIMESTAMPTZ,
					completed_at TIMESTAMPTZ
				);
				CREATE INDEX IF NOT EXISTS idx_executions_id ON executions(id);
				CREATE INDEX IF NOT EXISTS idx_executions_org ON executions(org_id);
				CREATE INDEX IF NOT EXISTS idx_executions_intent ON executions(intent_id);
				CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
				CREATE INDEX IF NOT EXISTS idx_executions_created ON executions(created_at DESC);

				CREATE TABLE IF NOT EXISTS execution_nodes (
					id           TEXT PRIMARY KEY,
					execution_id TEXT NOT NULL,
					agent_role   TEXT NOT NULL DEFAULT '',
					label        TEXT NOT NULL DEFAULT '',
					status       TEXT NOT NULL DEFAULT 'pending',
					progress     INTEGER NOT NULL DEFAULT 0,
					runtime      BIGINT NOT NULL DEFAULT 0,
					tokens       INTEGER NOT NULL DEFAULT 0,
					cost         DOUBLE PRECISION NOT NULL DEFAULT 0,
					parent_id    TEXT NOT NULL DEFAULT ''
				);
				CREATE INDEX IF NOT EXISTS idx_execution_nodes_execution ON execution_nodes(execution_id);

				CREATE TABLE IF NOT EXISTS execution_edges (
					id           TEXT PRIMARY KEY,
					execution_id TEXT NOT NULL,
					source       TEXT NOT NULL,
					target       TEXT NOT NULL
				);
				CREATE INDEX IF NOT EXISTS idx_execution_edges_execution ON execution_edges(execution_id);

				CREATE TABLE IF NOT EXISTS execution_events (
					id           TEXT PRIMARY KEY,
					execution_id TEXT NOT NULL,
					type         TEXT NOT NULL,
					agent_id     TEXT NOT NULL DEFAULT '',
					content      TEXT NOT NULL DEFAULT '',
					metadata     JSONB NOT NULL DEFAULT '{}',
					timestamp    TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
				CREATE INDEX IF NOT EXISTS idx_execution_events_execution ON execution_events(execution_id);
				CREATE INDEX IF NOT EXISTS idx_execution_events_timestamp ON execution_events(timestamp);

				CREATE TABLE IF NOT EXISTS execution_metrics (
					execution_id      TEXT PRIMARY KEY,
					total_tokens      INTEGER NOT NULL DEFAULT 0,
					prompt_tokens     INTEGER NOT NULL DEFAULT 0,
					completion_tokens INTEGER NOT NULL DEFAULT 0,
					estimated_cost    DOUBLE PRECISION NOT NULL DEFAULT 0,
					execution_duration BIGINT NOT NULL DEFAULT 0,
					average_latency   DOUBLE PRECISION NOT NULL DEFAULT 0,
					tools_executed    INTEGER NOT NULL DEFAULT 0,
					timestamp         TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
			`,
			Down: `
				DROP TABLE IF EXISTS execution_metrics;
				DROP TABLE IF EXISTS execution_events;
				DROP TABLE IF EXISTS execution_edges;
				DROP TABLE IF EXISTS execution_nodes;
				DROP TABLE IF EXISTS executions;
			`,
		},
	}
}