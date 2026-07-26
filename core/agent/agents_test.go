package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/workspace"
)

func TestFrontendAgent(t *testing.T) {
	a := NewFrontendAgent()
	if a.Name() != "frontend" {
		t.Errorf("name=%s", a.Name())
	}
	if !strings.Contains(a.Description(), "Frontend") {
		t.Errorf("desc=%s", a.Description())
	}
}

func TestBackendAgent(t *testing.T) {
	a := NewBackendAgent()
	if a.Name() != "backend" {
		t.Errorf("name=%s", a.Name())
	}
	if !strings.Contains(a.Description(), "Backend") {
		t.Errorf("desc=%s", a.Description())
	}
}

func TestReviewerAgent(t *testing.T) {
	a := NewReviewerAgent()
	if a.Name() != "reviewer" {
		t.Errorf("name=%s", a.Name())
	}
}

func TestTesterAgent(t *testing.T) {
	a := NewTesterAgent()
	if a.Name() != "tester" {
		t.Errorf("name=%s", a.Name())
	}
}

func TestSecurityAgent(t *testing.T) {
	a := NewSecurityAgent()
	if a.Name() != "security" {
		t.Errorf("name=%s", a.Name())
	}
}

func TestDevOpsAgent(t *testing.T) {
	a := NewDevOpsAgent()
	if a.Name() != "devops" {
		t.Errorf("name=%s", a.Name())
	}
}

func TestMonitorAgent(t *testing.T) {
	a := NewMonitorAgent()
	if a.Name() != "monitor" {
		t.Errorf("name=%s", a.Name())
	}
}

func TestProductionAgentRunNoTools(t *testing.T) {
	agents := []Agent{
		NewFrontendAgent(),
		NewBackendAgent(),
		NewReviewerAgent(),
		NewTesterAgent(),
		NewSecurityAgent(),
		NewDevOpsAgent(),
		NewMonitorAgent(),
	}

	llm := NewEchoProvider()

	for _, a := range agents {
		t.Run(a.Name(), func(t *testing.T) {
			ctx := NewContext(context.Background(), Task{
				ID:          "test-" + a.Name(),
				Description: "test " + a.Description(),
			}, ToolSet{}, llm, nil)

			result, err := a.Run(ctx)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if result.Status != ResultSuccess {
				t.Errorf("status=%s", result.Status)
			}
			if result.Summary == "" {
				t.Error("empty summary")
			}
		})
	}
}

func TestProductionAgentRegistration(t *testing.T) {
	llm := NewEchoProvider()
	ws := workspace.NewFakeWorkspace()
	wProvisioned, err := ws.Provision(context.Background(), workspace.WorkspaceSpec{Stack: "test"})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	defer ws.Recycle(context.Background(), wProvisioned.ID)

	rt := NewRuntime(llm, ws, wProvisioned.ID)
	rt.RegisterAgent(NewFrontendAgent())
	rt.RegisterAgent(NewBackendAgent())
	rt.RegisterAgent(NewReviewerAgent())
	rt.RegisterAgent(NewTesterAgent())
	rt.RegisterAgent(NewSecurityAgent())
	rt.RegisterAgent(NewDevOpsAgent())
	rt.RegisterAgent(NewMonitorAgent())

	if rt.Agent("frontend") == nil {
		t.Error("frontend agent not registered")
	}
	if rt.Agent("monitor") == nil {
		t.Error("monitor agent not registered")
	}
	if rt.Agent("nonexistent") != nil {
		t.Error("nonexistent agent should be nil")
	}
}
