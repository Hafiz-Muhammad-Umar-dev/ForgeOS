package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/query"
)

var errMissingAuth = errors.New("missing authorization header")

// writeGatewayError writes a JSON error response.
func writeGatewayError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// QueryHandler serves read-model data to clients via REST.
type QueryHandler struct {
	service  query.QueryService
	provider auth.AuthProvider
}

// NewQueryHandler creates a query handler.
func NewQueryHandler(service query.QueryService, provider auth.AuthProvider) *QueryHandler {
	return &QueryHandler{service: service, provider: provider}
}

// HandleListIntents serves GET /v1/intents with pagination.
func (h *QueryHandler) HandleListIntents(w http.ResponseWriter, r *http.Request) {
	claims, err := h.authenticate(r)
	if err != nil {
		writeGatewayError(w, http.StatusUnauthorized, err.Error())
		return
	}

	orgID := claims.OrgID
	if orgID == "" {
		orgID = r.URL.Query().Get("orgId")
	}

	limit := parseIntDefault(r.URL.Query().Get("limit"), 50)
	offset := parseIntDefault(r.URL.Query().Get("offset"), 0)

	intents, err := h.service.ListIntents(r.Context(), orgID, limit, offset)
	if err != nil {
		writeGatewayError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"intents": intents,
		"count":   len(intents),
	})
}

// HandleGetIntent serves GET /v1/intents/:id.
func (h *QueryHandler) HandleGetIntent(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authenticate(r); err != nil {
		writeGatewayError(w, http.StatusUnauthorized, err.Error())
		return
	}

	id := r.PathValue("id")
	intent, err := h.service.GetIntent(r.Context(), id)
	if err != nil {
		writeGatewayError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(intent)
}

// HandleListTasks serves GET /v1/tasks?intentId=.
func (h *QueryHandler) HandleListTasks(w http.ResponseWriter, r *http.Request) {
	if _, err := h.authenticate(r); err != nil {
		writeGatewayError(w, http.StatusUnauthorized, err.Error())
		return
	}

	intentID := r.URL.Query().Get("intentId")
	if intentID == "" {
		writeGatewayError(w, http.StatusBadRequest, "missing intentId")
		return
	}

	tasks, err := h.service.ListTasks(r.Context(), intentID)
	if err != nil {
		writeGatewayError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"tasks": tasks,
		"count": len(tasks),
	})
}

func (h *QueryHandler) authenticate(r *http.Request) (auth.Claims, error) {
	token := r.Header.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	} else {
		return auth.Claims{}, errMissingAuth
	}
	return h.provider.Authenticate(r.Context(), token)
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}
