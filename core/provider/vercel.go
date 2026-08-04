package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Compile-time check.
var _ DeployProvider = (*VercelAdapter)(nil)

// VercelAdapter is a DeployProvider that deploys to Vercel using Deploy Hooks.
type VercelAdapter struct {
	hookURL string
	token   string
	teamID  string
	baseURL string
	client  *http.Client
}

// VercelConfig configures the Vercel deploy adapter.
type VercelConfig struct {
	// HookURL is the Vercel Deploy Hook URL.
	HookURL string
	// APIToken is the Vercel API token for status checks.
	APIToken string
	// TeamID is the Vercel team ID (optional).
	TeamID string
	// Timeout for HTTP requests.
	Timeout time.Duration
	// APIBaseURL is the Vercel API base URL (overridable for testing).
	APIBaseURL string
}

// DefaultVercelConfig returns a default Vercel configuration.
func DefaultVercelConfig() VercelConfig {
	return VercelConfig{
		Timeout:    30 * time.Second,
		APIBaseURL: "https://api.vercel.com",
	}
}

// VercelOption configures the Vercel adapter.
type VercelOption func(*VercelConfig)

func WithVercelHookURL(url string) VercelOption {
	return func(c *VercelConfig) { c.HookURL = url }
}
func WithVercelToken(token string) VercelOption {
	return func(c *VercelConfig) { c.APIToken = token }
}
func WithVercelTeamID(id string) VercelOption {
	return func(c *VercelConfig) { c.TeamID = id }
}
func WithVercelAPIBaseURL(url string) VercelOption {
	return func(c *VercelConfig) { c.APIBaseURL = url }
}

// NewVercelAdapter creates a new Vercel deploy adapter.
func NewVercelAdapter(opts ...VercelOption) *VercelAdapter {
	cfg := DefaultVercelConfig()
	for _, fn := range opts {
		fn(&cfg)
	}
	return &VercelAdapter{
		hookURL: cfg.HookURL,
		token:   cfg.APIToken,
		teamID:  cfg.TeamID,
		baseURL: cfg.APIBaseURL,
		client:  &http.Client{Timeout: cfg.Timeout},
	}
}

// Name returns "vercel".
func (v *VercelAdapter) Name() string { return "vercel" }

// Deploy triggers a Vercel deployment via Deploy Hook or API.
func (v *VercelAdapter) Deploy(ctx context.Context, req DeployRequest) (DeployResponse, error) {
	if v.hookURL != "" {
		return v.deployViaHook(ctx, req)
	}
	return v.deployViaAPI(ctx, req)
}

// Status checks deployment status via Vercel API.
func (v *VercelAdapter) Status(ctx context.Context, deploymentID string) (DeployStatus, error) {
	if v.token == "" {
		return DeployStatusUnknown, fmt.Errorf("%w: api token required for status", ErrDeployFailed)
	}

	apiURL := v.getAPIBase() + "/v1/deployments/" + deploymentID
	if v.teamID != "" {
		apiURL += "?teamId=" + v.teamID
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return DeployStatusUnknown, fmt.Errorf("vercel: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+v.token)

	resp, err := v.client.Do(req)
	if err != nil {
		return DeployStatusUnknown, fmt.Errorf("vercel: status request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeployStatusUnknown, fmt.Errorf("vercel: decode: %w", err)
	}

	return toDeployStatus(result.State), nil
}

func (v *VercelAdapter) getAPIBase() string {
	if v.baseURL != "" {
		return v.baseURL
	}
	return "https://api.vercel.com"
}

// Capabilities returns Vercel's capabilities.
func (v *VercelAdapter) Capabilities() DeployCapabilities {
	return DeployCapabilities{
		Provider: "vercel",
		Regions:  []string{"iad1", "sfo1"},
	}
}

func (v *VercelAdapter) deployViaHook(ctx context.Context, req DeployRequest) (DeployResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, v.hookURL, nil)
	if err != nil {
		return DeployResponse{}, fmt.Errorf("vercel: create hook request: %w", err)
	}

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return DeployResponse{}, fmt.Errorf("vercel: hook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return DeployResponse{}, fmt.Errorf("%w: vercel hook returned %s", ErrDeployFailed, resp.Status)
	}

	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeployResponse{}, fmt.Errorf("vercel: decode hook response: %w", err)
	}
	if result.ID == "" {
		result.ID = "vercel-" + fmt.Sprintf("%d", time.Now().Unix())
	}

	return DeployResponse{
		DeploymentID: result.ID,
		URL:          result.URL,
		Status:       DeployStatusDeploying,
	}, nil
}

func (v *VercelAdapter) deployViaAPI(ctx context.Context, req DeployRequest) (DeployResponse, error) {
	if v.token == "" {
		return DeployResponse{}, fmt.Errorf("%w: vercel api token required", ErrDeployFailed)
	}

	body := map[string]any{
		"name":                 req.ProjectName,
		"gitSource":            map[string]string{"type": "github", "repoUrl": req.Source},
		"environmentVariables": req.Env,
	}
	data, _ := json.Marshal(body)

	apiURL := v.getAPIBase() + "/v1/deployments"
	if v.teamID != "" {
		apiURL += "?teamId=" + v.teamID
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(data))
	if err != nil {
		return DeployResponse{}, fmt.Errorf("vercel: create api request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+v.token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return DeployResponse{}, fmt.Errorf("vercel: api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return DeployResponse{}, fmt.Errorf("%w: vercel api returned %s", ErrDeployFailed, resp.Status)
	}

	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeployResponse{}, fmt.Errorf("vercel: decode api response: %w", err)
	}

	return DeployResponse{
		DeploymentID: result.ID,
		URL:          result.URL,
		Status:       DeployStatusDeploying,
	}, nil
}

func toDeployStatus(s string) DeployStatus {
	switch s {
	case "BUILDING", "DEPLOYING":
		return DeployStatusDeploying
	case "READY":
		return DeployStatusReady
	case "ERROR", "FAILED":
		return DeployStatusFailed
	default:
		return DeployStatusPending
	}
}

// DeployStatusUnknown is returned when status cannot be determined.
const DeployStatusUnknown DeployStatus = "unknown"
