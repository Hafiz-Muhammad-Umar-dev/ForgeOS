package planner

import (
	"context"
	"strings"
	"testing"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
)

func TestFakePlannerDefaults(t *testing.T) {
	fp := NewFakePlanner()
	d, err := fp.Plan(nil, ingress.IntentPayload{Text: "build an app"})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if d == nil {
		t.Fatal("nil DAG")
	}
	if len(d.Nodes) != 1 {
		t.Errorf("nodes: got=%d", len(d.Nodes))
	}
	if fp.PlanCount.Load() != 1 {
		t.Errorf("count=%d", fp.PlanCount.Load())
	}
}

func TestFakePlannerRecordsIntents(t *testing.T) {
	fp := NewFakePlanner()
	fp.Plan(nil, ingress.IntentPayload{Text: "first"})
	fp.Plan(nil, ingress.IntentPayload{Text: "second"})

	if len(fp.ReceivedIntents) != 2 {
		t.Fatalf("intents=%d", len(fp.ReceivedIntents))
	}
	if fp.ReceivedIntents[0].Text != "first" {
		t.Errorf("intent0=%s", fp.ReceivedIntents[0].Text)
	}
}

func TestFakePlannerCustomResult(t *testing.T) {
	fp := NewFakePlanner()
	fp.Result = &dag.DAG{
		Nodes: []dag.Node{
			{ID: "a", Agent: "frontend", Description: "build UI"},
			{ID: "b", Agent: "backend", Description: "build API", InputIDs: []string{"a"}},
		},
	}

	d, err := fp.Plan(nil, ingress.IntentPayload{Text: "build app"})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(d.Nodes) != 2 {
		t.Fatalf("nodes=%d", len(d.Nodes))
	}
}

func TestFakePlannerError(t *testing.T) {
	fp := NewFakePlanner()
	fp.PlanError = planError("planning failed")

	_, err := fp.Plan(nil, ingress.IntentPayload{Text: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFakePlannerCustomFunc(t *testing.T) {
	fp := NewFakePlanner()
	fp.PlanFunc = func(_ context.Context, intent ingress.IntentPayload) (*dag.DAG, error) {
		return &dag.DAG{
			Nodes: []dag.Node{{ID: "custom", Agent: "coder", Description: intent.Text}},
		}, nil
	}

	d, err := fp.Plan(nil, ingress.IntentPayload{Text: "custom task"})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if d.Nodes[0].Description != "custom task" {
		t.Errorf("desc=%s", d.Nodes[0].Description)
	}
}

func TestLLMPlannerPrompt(t *testing.T) {
	if !strings.Contains(plannerPrompt, "frontend") {
		t.Error("prompt missing agent descriptions")
	}
	if !strings.Contains(plannerPrompt, "nodes") {
		t.Error("prompt missing JSON format")
	}
}

// planError is a simple error for testing.
type planError string

func (e planError) Error() string { return string(e) }
