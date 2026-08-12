package gateway

import (
	"log"
	"net/http"
	"strings"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/execution"
)

// handleListExecutions serves GET /v1/executions (scope by org/intent).
func (g *Gateway) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	intentID := r.URL.Query().Get("intentId")
	orgID := r.URL.Query().Get("org_id")
	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	execs, err := g.execSvc.ListExecutions(r.Context(), orgID, intentID, limit, offset)
	if err != nil {
		log.Printf("gateway: list executions: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list executions")
		return
	}
	writeJSON(w, http.StatusOK, execs)
}

// handleGetExecution serves GET /v1/executions/{id}.
func (g *Gateway) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/executions/")
	id = strings.TrimSuffix(id, "/")
	exec, err := g.execSvc.GetExecution(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "execution not found")
		return
	}
	writeJSON(w, http.StatusOK, exec)
}

// handleExecutionAction serves POST /v1/executions/{id}/{action}.
func (g *Gateway) handleExecutionAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/executions/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "execution id and action required")
		return
	}
	id := parts[0]
	actionName := parts[1]

	action := execution.Action(actionName)
	switch action {
	case execution.ActionRun, execution.ActionPause, execution.ActionResume, execution.ActionStop:
	default:
		writeError(w, http.StatusBadRequest, "unknown action: "+actionName)
		return
	}

	exec, err := g.execSvc.ApplyAction(r.Context(), id, action)
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, exec)
}

// handleExecutionGetOrSub dispatches GET /v1/executions/{id} vs sub-resources.
func (g *Gateway) handleExecutionGetOrSub(w http.ResponseWriter, r *http.Request, path string) {
	rest := strings.TrimPrefix(path, "/v1/executions/")
	if strings.Contains(rest, "/metrics") {
		g.handleExecutionMetrics(w, r)
		return
	}
	if strings.Contains(rest, "/events") {
		g.handleExecutionEvents(w, r)
		return
	}
	if strings.Contains(rest, "/plan") {
		g.handleExecutionPlan(w, r)
		return
	}
	g.handleGetExecution(w, r)
}

// handleExecutionMetrics serves GET /v1/executions/{id}/metrics.
func (g *Gateway) handleExecutionMetrics(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/executions/"), "/")
	if len(parts) < 1 {
		writeError(w, http.StatusBadRequest, "execution id required")
		return
	}
	id := parts[0]
	metrics, err := g.execSvc.GetMetrics(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get metrics")
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

// handleExecutionEvents serves GET /v1/executions/{id}/events.
func (g *Gateway) handleExecutionEvents(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/executions/"), "/")
	if len(parts) < 1 {
		writeError(w, http.StatusBadRequest, "execution id required")
		return
	}
	id := parts[0]
	events, err := g.execSvc.ListEvents(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list events")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// handleExecutionPlan serves GET /v1/executions/{id}/plan.
func (g *Gateway) handleExecutionPlan(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/executions/"), "/")
	if len(parts) < 1 {
		writeError(w, http.StatusBadRequest, "execution id required")
		return
	}
	id := parts[0]
	plan, err := g.execSvc.GetPlan(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get plan")
		return
	}
	writeJSON(w, http.StatusOK, plan)
}