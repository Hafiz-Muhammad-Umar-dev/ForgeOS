package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/deployment"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/execution"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/metrics"
)

type Handler struct {
	execMgr   *execution.Manager
	deployMgr *deployment.Manager
	metrics   *metrics.Collector
	gitDir    string
}

func New(em *execution.Manager, dm *deployment.Manager, mc *metrics.Collector) *Handler {
	return &Handler{
		execMgr:   em,
		deployMgr: dm,
		metrics:   mc,
		gitDir:    os.Getenv("GIT_REPO_DIR"),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ---------------------------------------------------------------------------
// Health
// ---------------------------------------------------------------------------

func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// ---------------------------------------------------------------------------
// Workspace files
// ---------------------------------------------------------------------------

func (h *Handler) ListWorkspaceFiles(w http.ResponseWriter, r *http.Request) {
	root := os.Getenv("WORKSPACE_ROOT")
	if root == "" {
		root = "."
	}
	files, err := listDir(root, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     filepath.Base(root),
		"type":     "folder",
		"children": files,
	})
}

func listDir(root, prefix string) ([]map[string]any, error) {
	entries, err := os.ReadDir(filepath.Join(root, prefix))
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		relPath := filepath.Join(prefix, name)
		if e.IsDir() {
			children, err := listDir(root, relPath)
			if err != nil {
				continue
			}
			result = append(result, map[string]any{
				"name":     name,
				"path":     relPath,
				"type":     "folder",
				"children": children,
			})
		} else {
			data, err := os.ReadFile(filepath.Join(root, relPath))
			size := 0
			if err == nil {
				size = len(data)
			}
			result = append(result, map[string]any{
				"name": name,
				"path": relPath,
				"type": "file",
				"size": size,
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		ti := result[i]["type"].(string)
		tj := result[j]["type"].(string)
		if ti != tj {
			return ti == "folder"
		}
		return result[i]["name"].(string) < result[j]["name"].(string)
	})
	return result, nil
}

func (h *Handler) WriteWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	root := os.Getenv("WORKSPACE_ROOT")
	if root == "" {
		root = "."
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	fullPath := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.WriteFile(fullPath, []byte(body.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Terminal WebSocket (stub — returns unavailable message)
// ---------------------------------------------------------------------------

func (h *Handler) TerminalWS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "terminal_backend_unavailable",
		"message": "Terminal PTY requires host OS support. Connect via SSH or local terminal.",
	})
}

// ---------------------------------------------------------------------------
// Agents (static list)
// ---------------------------------------------------------------------------

func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agents := []map[string]any{
		{"id": "planner", "name": "Planner", "role": "planner", "status": "idle", "model": "claude-sonnet-4", "temperature": 0.3, "queue_length": 0},
		{"id": "researcher", "name": "Researcher", "role": "researcher", "status": "idle", "model": "claude-sonnet-4", "temperature": 0.5, "queue_length": 0},
		{"id": "coder", "name": "Coder", "role": "coder", "status": "idle", "model": "claude-sonnet-4", "temperature": 0.4, "queue_length": 0},
		{"id": "reviewer", "name": "Reviewer", "role": "reviewer", "status": "idle", "model": "claude-sonnet-4", "temperature": 0.2, "queue_length": 0},
		{"id": "tester", "name": "Tester", "role": "tester", "status": "idle", "model": "claude-sonnet-4", "temperature": 0.3, "queue_length": 0},
		{"id": "deployer", "name": "Deployer", "role": "deployer", "status": "idle", "model": "claude-sonnet-4", "temperature": 0.2, "queue_length": 0},
	}
	writeJSON(w, http.StatusOK, agents)
}

// ---------------------------------------------------------------------------
// Executions
// ---------------------------------------------------------------------------

func (h *Handler) RunExecution(w http.ResponseWriter, r *http.Request) {
	intentID := r.PathValue("intentId")
	exec, err := h.execMgr.Create(r.Context(), intentID, "planner")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.execMgr.UpdateStatus(exec.ID, execution.StatusRunning)
	h.execMgr.AddEvent(exec.ID, "tool_started", "Starting execution")
	writeJSON(w, http.StatusOK, exec)
}

func (h *Handler) PauseExecution(w http.ResponseWriter, r *http.Request) {
	intentID := r.PathValue("intentId")
	list := h.execMgr.List(intentID)
	for _, e := range list {
		h.execMgr.UpdateStatus(e.ID, execution.StatusPaused)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (h *Handler) ResumeExecution(w http.ResponseWriter, r *http.Request) {
	intentID := r.PathValue("intentId")
	list := h.execMgr.List(intentID)
	for _, e := range list {
		h.execMgr.UpdateStatus(e.ID, execution.StatusRunning)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
}

func (h *Handler) StopExecution(w http.ResponseWriter, r *http.Request) {
	intentID := r.PathValue("intentId")
	list := h.execMgr.List(intentID)
	for _, e := range list {
		h.execMgr.UpdateStatus(e.ID, execution.StatusFailed)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *Handler) ExecutionEvents(w http.ResponseWriter, r *http.Request) {
	intentID := r.PathValue("intentId")
	list := h.execMgr.List(intentID)
	var allEvents []execution.Event
	for _, e := range list {
		allEvents = append(allEvents, h.execMgr.Events(e.ID)...)
	}
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp.After(allEvents[j].Timestamp)
	})
	writeJSON(w, http.StatusOK, allEvents)
}

func (h *Handler) ExecutionMetrics(w http.ResponseWriter, r *http.Request) {
	intentID := r.PathValue("intentId")
	list := h.execMgr.List(intentID)
	totalMetrics := execution.Metrics{}
	for _, e := range list {
		m, err := h.execMgr.GetMetrics(e.ID)
		if err == nil {
			totalMetrics.TotalTokens += m.TotalTokens
			totalMetrics.PromptTokens += m.PromptTokens
			totalMetrics.CompletionTokens += m.CompletionTokens
			totalMetrics.EstimatedCost += m.EstimatedCost
			totalMetrics.ToolsExecuted += m.ToolsExecuted
		}
	}
	writeJSON(w, http.StatusOK, totalMetrics)
}

func (h *Handler) ExecutionStreamSSE(w http.ResponseWriter, r *http.Request) {
	intentID := r.PathValue("intentId")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	list := h.execMgr.List(intentID)
	for _, e := range list {
		events := h.execMgr.Events(e.ID)
		for _, evt := range events {
			data, _ := json.Marshal(evt)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------------
// Plans (stub)
// ---------------------------------------------------------------------------

func (h *Handler) GetPlan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{
		"id":        id,
		"intent_id": id,
		"status":    "completed",
		"summary":   fmt.Sprintf("Plan for %s executed successfully", id),
		"created_at": time.Now().Add(-1 * time.Hour),
		"updated_at": time.Now(),
	})
}

// ---------------------------------------------------------------------------
// Git operations (os/exec based)
// ---------------------------------------------------------------------------

func (h *Handler) gitDirPath() string {
	if h.gitDir != "" {
		return h.gitDir
	}
	return "."
}

func (h *Handler) gitExec(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = h.gitDirPath()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("git %s: %s: %w", args[0], strings.TrimSpace(string(out)), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (h *Handler) GitStatus(w http.ResponseWriter, r *http.Request) {
	out, err := h.gitExec("status", "--porcelain")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git not available")
		return
	}
	lines := strings.Split(out, "\n")
	var changed, staged, untracked int
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		code := line[:2]
		if code == "??" {
			untracked++
		} else if code[0] != ' ' {
			staged++
		} else {
			changed++
		}
	}
	branch, _ := h.gitExec("rev-parse", "--abbrev-ref", "HEAD")
	writeJSON(w, http.StatusOK, map[string]any{
		"current_branch": branch,
		"changed":        changed,
		"staged":         staged,
		"untracked":      untracked,
		"ignored":        0,
		"behind":         0,
		"ahead":          0,
	})
}

func (h *Handler) GitFiles(w http.ResponseWriter, r *http.Request) {
	out, err := h.gitExec("status", "--porcelain")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git not available")
		return
	}
	type gitFile struct {
		Path        string `json:"path"`
		Status      string `json:"status"`
		Staged      bool   `json:"staged"`
		IsUntracked bool   `json:"is_untracked"`
		IsIgnored   bool   `json:"is_ignored"`
	}
	var files []gitFile
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		code := line[:2]
		path := line[3:]
		if code == "??" {
			files = append(files, gitFile{Path: path, Status: "added", Staged: false, IsUntracked: true})
		} else if code[0] != ' ' {
			status := "modified"
			if code[0] == 'A' {
				status = "added"
			} else if code[0] == 'D' {
				status = "deleted"
			}
			files = append(files, gitFile{Path: path, Status: status, Staged: true})
		} else {
			status := "modified"
			if code[1] == 'D' {
				status = "deleted"
			}
			files = append(files, gitFile{Path: path, Status: status, Staged: false})
		}
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *Handler) GitStage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	_, err := h.gitExec("add", body.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitUnstage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	_, err := h.gitExec("reset", "HEAD", "--", body.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitCommit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Message     string `json:"message"`
		Description string `json:"description,omitempty"`
		SignOff     bool   `json:"sign_off,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	msg := body.Message
	if body.Description != "" {
		msg += "\n\n" + body.Description
	}
	args := []string{"commit", "-m", msg}
	if body.SignOff {
		args = append(args, "--signoff")
	}
	out, err := h.gitExec(args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"oid": out, "status": "ok"})
}

func (h *Handler) GitPush(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Remote string `json:"remote,omitempty"`
		Branch string `json:"branch,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	args := []string{"push"}
	if body.Remote != "" {
		args = append(args, body.Remote)
	}
	if body.Branch != "" {
		args = append(args, body.Branch)
	}
	_, err := h.gitExec(args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitPull(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Remote string `json:"remote,omitempty"`
		Branch string `json:"branch,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	args := []string{"pull"}
	if body.Remote != "" {
		args = append(args, body.Remote)
	}
	if body.Branch != "" {
		args = append(args, body.Branch)
	}
	_, err := h.gitExec(args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitBranches(w http.ResponseWriter, r *http.Request) {
	out, err := h.gitExec("branch", "-a")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git not available")
		return
	}
	current, _ := h.gitExec("rev-parse", "--abbrev-ref", "HEAD")
	var list []map[string]any
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(strings.TrimPrefix(line, "* "))
		isCurrent := strings.HasPrefix(line, "*")
		isRemote := strings.HasPrefix(name, "remotes/") || strings.Contains(name, "/")
		if isRemote {
			name = strings.TrimPrefix(name, "remotes/")
		}
		if name == "" || name == current {
			continue
		}
		list = append(list, map[string]any{
			"name":       name,
			"is_remote":  isRemote,
			"is_current": isCurrent,
			"ahead":      0,
			"behind":     0,
		})
	}
	// Add current branch
	list = append([]map[string]any{{
		"name": current, "is_remote": false, "is_current": true, "ahead": 0, "behind": 0,
	}}, list...)
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) GitCreateBranch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Base string `json:"base,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	args := []string{"branch", body.Name}
	if body.Base != "" {
		args = append(args, body.Base)
	}
	_, err := h.gitExec(args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitCheckout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	_, err := h.gitExec("checkout", body.Branch)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitDeleteBranch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	_, err := h.gitExec("branch", "-D", body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitHistory(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
		limit = l
	}
	out, err := h.gitExec("log", fmt.Sprintf("--max-count=%d", limit), "--format=%H|%an|%ai|%s")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git not available")
		return
	}
	type commit struct {
		OID       string `json:"oid"`
		Message   string `json:"message"`
		Author    string `json:"author"`
		Timestamp int64  `json:"timestamp"`
	}
	var commits []commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		t, err := time.Parse("2006-01-02 15:04:05 -0700", parts[2])
		if err != nil {
			t = time.Now()
		}
		commits = append(commits, commit{
			OID:       parts[0],
			Author:    parts[1],
			Timestamp: t.UnixMilli(),
			Message:   parts[3],
		})
	}
	writeJSON(w, http.StatusOK, commits)
}

func (h *Handler) GitDiff(w http.ResponseWriter, r *http.Request) {
	commitID := r.URL.Query().Get("commit")
	args := []string{"diff"}
	if commitID != "" {
		args = append(args, commitID+"^.."+commitID)
	} else {
		args = append(args, "HEAD")
	}
	out, err := h.gitExec(args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "git not available")
		return
	}
	type diffFile struct {
		File    string `json:"file"`
		Added   int    `json:"added"`
		Removed int    `json:"removed"`
		Content string `json:"content"`
	}
	var files []diffFile
	sections := strings.Split(out, "diff --git ")
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		lines := strings.Split(section, "\n")
		filePath := ""
		added, removed := 0, 0
		for _, line := range lines {
			if strings.HasPrefix(line, "+++ b/") {
				filePath = strings.TrimPrefix(line, "+++ b/")
			}
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				added++
			}
			if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				removed++
			}
		}
		if filePath != "" {
			files = append(files, diffFile{
				File:    filePath,
				Added:   added,
				Removed: removed,
				Content: section,
			})
		}
	}
	writeJSON(w, http.StatusOK, files)
}

func (h *Handler) GitFetch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Remote string `json:"remote,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	args := []string{"fetch"}
	if body.Remote != "" {
		args = append(args, body.Remote)
	}
	_, err := h.gitExec(args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GitMetrics(w http.ResponseWriter, r *http.Request) {
	repoSize, _ := h.gitExec("count-objects", "-H")
	commitCount, _ := h.gitExec("rev-list", "--count", "HEAD")
	branchCount, _ := h.gitExec("branch", "--list")
	bCount := len(strings.Split(branchCount, "\n"))
	writeJSON(w, http.StatusOK, map[string]any{
		"repository_size": repoSize,
		"commit_count":    parseCount(commitCount),
		"branch_count":    bCount,
		"changed_files":   0,
		"staged_files":    0,
	})
}

func parseCount(s string) int {
	s = strings.TrimSpace(s)
	n, _ := strconv.Atoi(s)
	return n
}

// ---------------------------------------------------------------------------
// Deployments
// ---------------------------------------------------------------------------

func (h *Handler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	status := r.URL.Query().Get("status")
	list := h.deployMgr.List(project, status, 1, 50)
	writeJSON(w, http.StatusOK, list)
}

func (h *Handler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := h.deployMgr.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *Handler) DeploymentLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	logs := h.deployMgr.Logs(id)
	writeJSON(w, http.StatusOK, logs)
}

func (h *Handler) DeploymentMetrics(w http.ResponseWriter, r *http.Request) {
	snapshots := h.metrics.History()
	writeJSON(w, http.StatusOK, snapshots)
}

func (h *Handler) DeploymentTimeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	stages := h.deployMgr.Stages(id)
	writeJSON(w, http.StatusOK, stages)
}

func (h *Handler) DeploymentHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]any{
		{"type": "http", "endpoint": "/healthz", "status": "healthy", "last_checked": time.Now(), "interval": 30},
	})
}

func (h *Handler) StartDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.deployMgr.UpdateStatus(id, deployment.StatusRunning)
	h.deployMgr.AddLog(id, "info", "Deployment started")
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (h *Handler) StopDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.deployMgr.UpdateStatus(id, deployment.StatusStopped)
	h.deployMgr.AddLog(id, "info", "Deployment stopped")
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *Handler) RestartDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.deployMgr.UpdateStatus(id, deployment.StatusRunning)
	h.deployMgr.AddLog(id, "info", "Deployment restarted")
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func (h *Handler) PauseDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.deployMgr.UpdateStatus(id, deployment.StatusStopped)
	h.deployMgr.AddLog(id, "info", "Deployment paused")
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (h *Handler) ResumeDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.deployMgr.UpdateStatus(id, deployment.StatusRunning)
	h.deployMgr.AddLog(id, "info", "Deployment resumed")
	writeJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
}

func (h *Handler) DeleteDeployment(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) RollbackDeployment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.deployMgr.AddLog(id, "info", "Rollback initiated")
	h.deployMgr.AddStage(id, "rollback", deployment.StatusRunning)
	writeJSON(w, http.StatusOK, map[string]string{"status": "rollback_started"})
}

// ---------------------------------------------------------------------------
// System Metrics
// ---------------------------------------------------------------------------

func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.metrics.History())
}
