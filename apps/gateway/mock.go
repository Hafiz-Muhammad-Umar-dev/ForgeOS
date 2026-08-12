package gateway

import (
	"net/http"
	"strings"
)

// handleV1Mock dispatches to v1MockResponse and writes JSON or 200 default.
func (g *Gateway) handleV1Mock(w http.ResponseWriter, r *http.Request, path string, method string) {
	mock := v1MockResponse(path, method)
	if mock != nil {
		writeJSON(w, http.StatusOK, mock)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "mock"})
}

// v1MockResponse returns realistic mock data for common frontend API paths
// that do not have dedicated handlers yet. This bridges the gap until a
// full database-backed implementation is deployed.
func v1MockResponse(path string, method string) any {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return nil
	}
	resource := parts[1]

	switch resource {
	case "tasks":
		return []map[string]any{
			{"id": "task-1", "intent_id": "intent-1", "agent_name": "Planner", "status": "completed", "summary": "Design system architecture", "input_tokens": 450, "output_tokens": 120, "created_at": "2026-07-29T10:00:00Z", "updated_at": "2026-07-29T11:00:00Z"},
			{"id": "task-2", "intent_id": "intent-1", "agent_name": "Coder", "status": "running", "summary": "Implement API routes", "input_tokens": 320, "output_tokens": 890, "created_at": "2026-07-29T12:00:00Z", "updated_at": "2026-07-29T12:30:00Z"},
		}

	case "plans":
		return map[string]any{"id": path, "intent_id": path, "status": "completed", "summary": "Plan executed successfully", "created_at": "2026-07-29T10:00:00Z", "updated_at": "2026-07-29T11:00:00Z"}

	case "deployments":
		if len(parts) >= 3 && parts[2] == "stream" {
			return nil
		}
		if len(parts) > 3 {
			switch parts[3] {
			case "logs":
				return []map[string]any{
					{"timestamp": "2026-07-29T10:00:00Z", "level": "info", "message": "Build started"},
					{"timestamp": "2026-07-29T10:01:00Z", "level": "info", "message": "Installing dependencies"},
					{"timestamp": "2026-07-29T10:02:00Z", "level": "info", "message": "Build complete"},
				}
			case "metrics":
				return []map[string]any{
					{"timestamp": "2026-07-29T10:00:00Z", "cpu": 0.45, "memory": 0.62, "disk": 0.3, "network": 1.2, "requests_per_sec": 150, "latency": 45, "errors": 0},
				}
			case "timeline":
				return []map[string]any{
					{"name": "Queued", "status": "completed", "started_at": "2026-07-29T10:00:00Z", "finished_at": "2026-07-29T10:00:05Z", "duration": 5000},
					{"name": "Building", "status": "completed", "started_at": "2026-07-29T10:00:05Z", "finished_at": "2026-07-29T10:02:00Z", "duration": 115000},
					{"name": "Deploying", "status": "running", "started_at": "2026-07-29T10:02:00Z"},
				}
			case "health":
				return []map[string]any{
					{"type": "http", "endpoint": "/healthz", "status": "healthy", "last_checked": "2026-07-29T10:05:00Z", "interval": 30},
				}
			}
		}
		return []map[string]any{
			{"id": "deploy-1", "project_name": parts[2], "status": "running", "branch": "main", "commit": "abc1234", "region": "us-east", "url": "https://forgeos.app", "created_by": "admin", "created_at": "2026-07-29T10:00:00Z", "started_at": "2026-07-29T10:00:05Z"},
			{"id": "deploy-2", "project_name": parts[2], "status": "healthy", "branch": "main", "commit": "def5678", "region": "us-east", "url": "https://forgeos.app", "created_by": "admin", "created_at": "2026-07-28T10:00:00Z", "started_at": "2026-07-28T10:00:05Z", "finished_at": "2026-07-28T10:02:00Z"},
		}

	case "git":
		if len(parts) > 2 {
			switch parts[2] {
			case "status":
				return map[string]any{"current_branch": "main", "changed": 3, "staged": 1, "untracked": 2, "ignored": 0, "behind": 0, "ahead": 1}
			case "files":
				return []map[string]any{
					{"path": "src/main.ts", "status": "modified", "staged": true, "is_untracked": false, "is_ignored": false},
					{"path": "src/utils.ts", "status": "modified", "staged": false, "is_untracked": false, "is_ignored": false},
					{"path": "README.md", "status": "added", "staged": false, "is_untracked": true, "is_ignored": false},
				}
			case "branches":
				return []map[string]any{
					{"name": "main", "is_remote": false, "is_current": true, "ahead": 1, "behind": 0},
					{"name": "develop", "is_remote": false, "is_current": false, "ahead": 0, "behind": 2},
					{"name": "origin/main", "is_remote": true, "is_current": false, "ahead": 0, "behind": 0},
				}
			case "history":
				return []map[string]any{
					{"oid": "abc1234", "message": "Add authentication flow", "author": "admin", "timestamp": int64(1722240000000), "parent_count": 1},
					{"oid": "def5678", "message": "Implement API routes", "author": "admin", "timestamp": int64(1722153600000), "parent_count": 1},
					{"oid": "ghi9012", "message": "Initial project setup", "author": "admin", "timestamp": int64(1722067200000), "parent_count": 0},
				}
			case "diff":
				return []map[string]any{
					{"file": "src/main.ts", "added": 15, "removed": 3, "content": "+ import { auth } from './auth';"},
				}
			case "metrics":
				return map[string]any{"repository_size": "2.3 MB", "commit_count": 42, "branch_count": 3, "changed_files": 3, "staged_files": 1, "last_fetch": "2026-07-29T09:00:00Z"}
			}
		}
		return map[string]string{"status": "ok"}

	case "metrics":
		return []map[string]any{
			{"cpu": 0.35, "memory": 0.55, "disk": 0.28, "network": 0.8, "requests_per_sec": 120, "latency": 35, "errors": 0, "timestamp": "2026-07-29T10:00:00Z"},
		}

	case "terminal":
		return map[string]any{"status": "terminal_backend_unavailable", "message": "Terminal requires host OS support."}
	}

	return nil
}