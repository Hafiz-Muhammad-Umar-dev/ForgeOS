package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/api/handlers"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/deployment"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/execution"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/metrics"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8082"
	}

	execMgr := execution.NewManager()
	deployMgr := deployment.NewManager()
	metricsCol := metrics.NewCollector()

	h := handlers.New(execMgr, deployMgr, metricsCol)

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /healthz", h.Healthz)
	mux.HandleFunc("GET /readyz", h.Readyz)

	// Workspace files
	mux.HandleFunc("GET /v1/workspace/files", h.ListWorkspaceFiles)
	mux.HandleFunc("PUT /v1/workspace/files/{path...}", h.WriteWorkspaceFile)

	// Terminal
	mux.HandleFunc("GET /v1/terminal", h.TerminalWS)

	// Agents
	mux.HandleFunc("GET /v1/agents", h.ListAgents)

	// Executions
	mux.HandleFunc("POST /v1/executions/{intentId}/run", h.RunExecution)
	mux.HandleFunc("POST /v1/executions/{intentId}/pause", h.PauseExecution)
	mux.HandleFunc("POST /v1/executions/{intentId}/resume", h.ResumeExecution)
	mux.HandleFunc("POST /v1/executions/{intentId}/stop", h.StopExecution)
	mux.HandleFunc("GET /v1/executions/{intentId}/events", h.ExecutionEvents)
	mux.HandleFunc("GET /v1/executions/{intentId}/metrics", h.ExecutionMetrics)
	mux.HandleFunc("GET /v1/executions/{intentId}/stream", h.ExecutionStreamSSE)

	// Plans
	mux.HandleFunc("GET /v1/plans/{id}", h.GetPlan)

	// Git
	mux.HandleFunc("GET /v1/git/status", h.GitStatus)
	mux.HandleFunc("GET /v1/git/files", h.GitFiles)
	mux.HandleFunc("POST /v1/git/stage", h.GitStage)
	mux.HandleFunc("POST /v1/git/unstage", h.GitUnstage)
	mux.HandleFunc("POST /v1/git/commit", h.GitCommit)
	mux.HandleFunc("POST /v1/git/push", h.GitPush)
	mux.HandleFunc("POST /v1/git/pull", h.GitPull)
	mux.HandleFunc("GET /v1/git/branches", h.GitBranches)
	mux.HandleFunc("POST /v1/git/branches", h.GitCreateBranch)
	mux.HandleFunc("POST /v1/git/checkout", h.GitCheckout)
	mux.HandleFunc("DELETE /v1/git/branches", h.GitDeleteBranch)
	mux.HandleFunc("GET /v1/git/history", h.GitHistory)
	mux.HandleFunc("GET /v1/git/diff", h.GitDiff)
	mux.HandleFunc("POST /v1/git/fetch", h.GitFetch)
	mux.HandleFunc("GET /v1/git/metrics", h.GitMetrics)

	// Deployments
	mux.HandleFunc("GET /v1/deployments", h.ListDeployments)
	mux.HandleFunc("GET /v1/deployments/{id}", h.GetDeployment)
	mux.HandleFunc("GET /v1/deployments/{id}/logs", h.DeploymentLogs)
	mux.HandleFunc("GET /v1/deployments/{id}/metrics", h.DeploymentMetrics)
	mux.HandleFunc("GET /v1/deployments/{id}/timeline", h.DeploymentTimeline)
	mux.HandleFunc("GET /v1/deployments/{id}/health", h.DeploymentHealth)
	mux.HandleFunc("POST /v1/deployments/{id}/start", h.StartDeployment)
	mux.HandleFunc("POST /v1/deployments/{id}/stop", h.StopDeployment)
	mux.HandleFunc("POST /v1/deployments/{id}/restart", h.RestartDeployment)
	mux.HandleFunc("POST /v1/deployments/{id}/pause", h.PauseDeployment)
	mux.HandleFunc("POST /v1/deployments/{id}/resume", h.ResumeDeployment)
	mux.HandleFunc("DELETE /v1/deployments/{id}", h.DeleteDeployment)
	mux.HandleFunc("POST /v1/deployments/{id}/rollback", h.RollbackDeployment)

	// Metrics
	mux.HandleFunc("GET /v1/metrics", h.GetMetrics)

	// CORS
	handler := corsMiddleware(mux)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("api-service: listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("api-service: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
