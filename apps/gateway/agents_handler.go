package gateway

import (
	"log"
	"net/http"
	"strings"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/middleware"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/agents"
)

// handleListAgents serves GET /v1/agents.
func (g *Gateway) handleListAgents(w http.ResponseWriter, r *http.Request) {
	claims, _ := middleware.ClaimsFromContext(r.Context())
	orgID := claims.OrgID

	agentList, err := g.agentsSvc.ListAgents(r.Context(), orgID)
	if err != nil {
		log.Printf("gateway: list agents: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	writeJSON(w, http.StatusOK, agentList)
}

// handleGetAgent serves GET /v1/agents/{id}.
func (g *Gateway) handleGetAgent(w http.ResponseWriter, r *http.Request, path string) {
	id := strings.TrimPrefix(path, "/v1/agents/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "agent id is required")
		return
	}
	agent, err := g.agentsSvc.GetAgent(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

// compile-time guard for the agents service type.
var _ = (*agents.Service)(nil)