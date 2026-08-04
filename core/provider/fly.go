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
var _ DeployProvider = (*FlyAdapter)(nil)

// FlyAdapter is a DeployProvider that deploys to Fly.io using the Machines API.
type FlyAdapter struct {
	apiToken string
	appName  string
	orgSlug  string
	baseURL  string
	client   *http.Client
}

// FlyConfig configures the Fly.io deploy adapter.
type FlyConfig struct {
	APIToken   string
	AppName    string
	OrgSlug    string
	Timeout    time.Duration
	APIBaseURL string
}

// DefaultFlyConfig returns a default Fly.io configuration.
func DefaultFlyConfig() FlyConfig {
	return FlyConfig{Timeout: 60 * time.Second}
}

// FlyOption configures the Fly adapter.
type FlyOption func(*FlyConfig)

func WithFlyToken(token string) FlyOption {
	return func(c *FlyConfig) { c.APIToken = token }
}
func WithFlyApp(name string) FlyOption {
	return func(c *FlyConfig) { c.AppName = name }
}
func WithFlyOrg(slug string) FlyOption {
	return func(c *FlyConfig) { c.OrgSlug = slug }
}
func WithFlyAPIBaseURL(url string) FlyOption {
	return func(c *FlyConfig) { c.APIBaseURL = url }
}

// NewFlyAdapter creates a new Fly.io deploy adapter.
func NewFlyAdapter(opts ...FlyOption) *FlyAdapter {
	cfg := DefaultFlyConfig()
	for _, fn := range opts {
		fn(&cfg)
	}
	return &FlyAdapter{
		apiToken: cfg.APIToken,
		baseURL:  cfg.APIBaseURL,
		appName:  cfg.AppName,
		orgSlug:  cfg.OrgSlug,
		client:   &http.Client{Timeout: cfg.Timeout},
	}
}

// Name returns "fly".
func (f *FlyAdapter) Name() string { return "fly" }

// Deploy triggers a deployment on Fly.io by creating a new machine release.
func (f *FlyAdapter) Deploy(ctx context.Context, req DeployRequest) (DeployResponse, error) {
	if f.apiToken == "" {
		return DeployResponse{}, fmt.Errorf("%w: fly api token required", ErrDeployFailed)
	}

	appName := f.appName
	if req.ProjectName != "" {
		appName = req.ProjectName
	}

	// For Fly.io, trigger a deploy by creating a new release via API.
	body := map[string]any{
		"app":      appName,
		"strategy": "immediate",
		"source":   req.Source,
	}
	if len(req.Env) > 0 {
		body["env"] = req.Env
	}

	data, _ := json.Marshal(body)

	apiURL := f.getAPIBase() + "/v1/apps/" + appName + "/machines"
	if f.orgSlug != "" {
		apiURL += "?org=" + f.orgSlug
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(data))
	if err != nil {
		return DeployResponse{}, fmt.Errorf("fly: create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+f.apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return DeployResponse{}, fmt.Errorf("fly: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return DeployResponse{}, fmt.Errorf("%w: fly api returned %s", ErrDeployFailed, resp.Status)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeployResponse{}, fmt.Errorf("fly: decode: %w", err)
	}

	return DeployResponse{
		DeploymentID: result.ID,
		URL:          fmt.Sprintf("https://%s.fly.dev", appName),
		Status:       DeployStatusDeploying,
	}, nil
}

// Status checks the status of a Fly.io machine.
func (f *FlyAdapter) Status(ctx context.Context, machineID string) (DeployStatus, error) {
	if f.apiToken == "" {
		return DeployStatusUnknown, fmt.Errorf("%w: fly api token required", ErrDeployFailed)
	}

	apiURL := f.getAPIBase() + "/v1/apps/" + f.appName + "/machines/" + machineID

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return DeployStatusUnknown, fmt.Errorf("fly: create status request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+f.apiToken)

	resp, err := f.client.Do(httpReq)
	if err != nil {
		return DeployStatusUnknown, fmt.Errorf("fly: status request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DeployStatusUnknown, fmt.Errorf("fly: decode: %w", err)
	}

	switch result.State {
	case "started", "starting":
		return DeployStatusDeploying, nil
	case "running":
		return DeployStatusReady, nil
	case "stopped":
		return DeployStatusFailed, nil
	default:
		return DeployStatusPending, nil
	}
}
func (f *FlyAdapter) getAPIBase() string {
	if f.baseURL != "" {
		return f.baseURL
	}
	return "https://api.machines.dev"
}

// Capabilities returns Fly.io's capabilities.
func (f *FlyAdapter) Capabilities() DeployCapabilities {
	return DeployCapabilities{
		Provider: "fly",
		Regions:  []string{"ams", "iad", "sjc", "lhr", "fra", "sin", "hkg"},
	}
}
