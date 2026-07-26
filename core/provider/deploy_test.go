package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeployRequestDefaults(t *testing.T) {
	req := DeployRequest{Source: "https://github.com/user/repo", ProjectName: "my-app"}
	if req.Source != "https://github.com/user/repo" {
		t.Errorf("source=%s", req.Source)
	}
	if req.ProjectName != "my-app" {
		t.Errorf("project=%s", req.ProjectName)
	}
}

func TestDeployResponse(t *testing.T) {
	resp := DeployResponse{
		DeploymentID: "dpl-abc123",
		URL:          "https://my-app.vercel.app",
		Status:       DeployStatusReady,
	}
	if resp.DeploymentID != "dpl-abc123" {
		t.Errorf("id=%s", resp.DeploymentID)
	}
	if resp.Status != DeployStatusReady {
		t.Errorf("status=%s", resp.Status)
	}
}

func TestDeployStatusValues(t *testing.T) {
	if DeployStatusPending != "pending" {
		t.Errorf("pending=%s", DeployStatusPending)
	}
	if DeployStatusDeploying != "deploying" {
		t.Errorf("deploying=%s", DeployStatusDeploying)
	}
	if DeployStatusReady != "ready" {
		t.Errorf("ready=%s", DeployStatusReady)
	}
	if DeployStatusFailed != "failed" {
		t.Errorf("failed=%s", DeployStatusFailed)
	}
}

func TestFakeDeployProviderDeploy(t *testing.T) {
	fp := NewFakeDeployProvider()
	resp, err := fp.Deploy(context.Background(), DeployRequest{
		Source:      "https://github.com/user/repo",
		ProjectName: "my-app",
	})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if resp.DeploymentID != "deploy-fake-1" {
		t.Errorf("id=%s", resp.DeploymentID)
	}
	if resp.Status != DeployStatusReady {
		t.Errorf("status=%s", resp.Status)
	}
	if fp.DeployCount.Load() != 1 {
		t.Errorf("count=%d", fp.DeployCount.Load())
	}
}

func TestFakeDeployProviderRecords(t *testing.T) {
	fp := NewFakeDeployProvider()
	fp.Deploy(context.Background(), DeployRequest{Source: "src-1"})
	fp.Deploy(context.Background(), DeployRequest{Source: "src-2"})
	if len(fp.ReceivedRequests) != 2 {
		t.Fatalf("requests=%d", len(fp.ReceivedRequests))
	}
	if fp.ReceivedRequests[0].Source != "src-1" {
		t.Errorf("req0=%s", fp.ReceivedRequests[0].Source)
	}
}

func TestFakeDeployProviderStatus(t *testing.T) {
	fp := NewFakeDeployProvider()
	status, err := fp.Status(context.Background(), "deploy-1")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != DeployStatusReady {
		t.Errorf("status=%s", status)
	}
}

func TestFakeDeployProviderCapabilities(t *testing.T) {
	fp := NewFakeDeployProvider()
	caps := fp.Capabilities()
	if caps.Provider != "fake" {
		t.Errorf("provider=%s", caps.Provider)
	}
}

func TestVercelAdapterHookDeploy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "dpl-hook-1", "url": "https://hook-deploy.vercel.app"})
	}))
	defer srv.Close()

	va := NewVercelAdapter(WithVercelHookURL(srv.URL))
	resp, err := va.Deploy(context.Background(), DeployRequest{
		Source:      "https://github.com/user/repo",
		ProjectName: "test-app",
	})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if resp.DeploymentID == "" {
		t.Error("empty deployment id")
	}
	if resp.Status != DeployStatusDeploying {
		t.Errorf("status=%s", resp.Status)
	}
}

func TestVercelAdapterHookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	va := NewVercelAdapter(WithVercelHookURL(srv.URL))
	_, err := va.Deploy(context.Background(), DeployRequest{Source: "repo"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestVercelAdapterStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"state": "READY"})
	}))
	defer srv.Close()

	va := NewVercelAdapter(WithVercelToken("test-token"), WithVercelAPIBaseURL(srv.URL))
	status, err := va.Status(context.Background(), "dpl-1")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != DeployStatusReady {
		t.Errorf("status=%s", status)
	}
}

func TestVercelAdapterCapabilities(t *testing.T) {
	va := NewVercelAdapter()
	caps := va.Capabilities()
	if caps.Provider != "vercel" {
		t.Errorf("provider=%s", caps.Provider)
	}
	if len(caps.Regions) == 0 {
		t.Error("no regions")
	}
}

func TestFlyAdapterDeploy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"id": "machine-1"})
	}))
	defer srv.Close()

	fa := NewFlyAdapter(WithFlyToken("test-token"), WithFlyApp("test-app"), WithFlyAPIBaseURL(srv.URL))
	resp, err := fa.Deploy(context.Background(), DeployRequest{
		Source:      "https://github.com/user/repo",
		ProjectName: "test-app",
	})
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if resp.DeploymentID == "" {
		t.Error("empty deployment id")
	}
	if resp.URL != "https://test-app.fly.dev" {
		t.Errorf("url=%s", resp.URL)
	}
}

func TestFlyAdapterDeployNoToken(t *testing.T) {
	fa := NewFlyAdapter()
	_, err := fa.Deploy(context.Background(), DeployRequest{Source: "repo"})
	if err == nil {
		t.Fatal("expected error without api token")
	}
}

func TestFlyAdapterStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"state": "started"})
	}))
	defer srv.Close()

	fa := NewFlyAdapter(WithFlyToken("test-token"), WithFlyApp("test-app"), WithFlyAPIBaseURL(srv.URL))
	status, err := fa.Status(context.Background(), "machine-1")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status != DeployStatusDeploying {
		t.Errorf("status=%s", status)
	}
}

func TestFlyAdapterCapabilities(t *testing.T) {
	fa := NewFlyAdapter()
	caps := fa.Capabilities()
	if caps.Provider != "fly" {
		t.Errorf("provider=%s", caps.Provider)
	}
	if len(caps.Regions) == 0 {
		t.Error("no regions")
	}
}

func TestToDeployStatus(t *testing.T) {
	tests := []struct {
		input string
		want  DeployStatus
	}{
		{"BUILDING", DeployStatusDeploying},
		{"DEPLOYING", DeployStatusDeploying},
		{"READY", DeployStatusReady},
		{"ERROR", DeployStatusFailed},
		{"FAILED", DeployStatusFailed},
		{"UNKNOWN", DeployStatusPending},
		{"", DeployStatusPending},
	}
	for _, tt := range tests {
		got := toDeployStatus(tt.input)
		if got != tt.want {
			t.Errorf("toDeployStatus(%q): got=%s want=%s", tt.input, got, tt.want)
		}
	}
}
