package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/dag"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/provider"
)

// Compile-time check.
var _ Planner = (*LLMPlanner)(nil)

// LLMPlanner implements Planner by calling an LLM to decompose the intent
// into a structured DAG. The LLM response is parsed as JSON.
type LLMPlanner struct {
	llm      provider.LLMProvider
	model    string
}

// NewLLMPlanner creates an LLMPlanner that uses the given LLM provider.
func NewLLMPlanner(llm provider.LLMProvider, model string) *LLMPlanner {
	return &LLMPlanner{llm: llm, model: model}
}

// plannerPrompt is the system prompt that instructs the LLM to produce a DAG.
const plannerPrompt = `You are a software project planner. Decompose the given user request into a DAG of tasks that can be executed by specialized agents.

Available agents and their capabilities:
- frontend: Builds React/TypeScript UIs, components, and styling
- backend: Builds Go/Node.js APIs, services, and data models
- reviewer: Reviews code for correctness, security, and quality
- tester: Writes unit, integration, and E2E tests
- security: Audits code for vulnerabilities
- devops: Configures CI/CD, Docker, and deployment
- monitor: Sets up logging, metrics, and alerting
- coder: General-purpose development tasks

Respond with a JSON object in this exact format:
{
  "nodes": [
    {
      "id": "unique-node-id",
      "name": "Human-readable name",
      "agent: "agent-name",
      "description": "What this agent should do",
      "input_ids": ["id-of-prerequisite-node"]
    }
  ]
}

Rules:
- Each node must have exactly one agent assigned.
- Use "input_ids" to express dependencies (empty for root nodes).
- Keep the DAG flat and parallel where possible.
- Do NOT include duplicate or circular dependencies.`

// Plan sends the intent to the LLM and parses the DAG from the response.
func (p *LLMPlanner) Plan(ctx context.Context, intent ingress.IntentPayload) (*dag.DAG, error) {
	resp, err := p.llm.Complete(ctx, provider.CompletionRequest{
		Model:    p.model,
		System:   plannerPrompt,
		Messages: []provider.Message{{Role: provider.RoleUser, Content: intent.Text}},
	})
	if err != nil {
		return nil, fmt.Errorf("planner: llm: %w", err)
	}

	var plan struct {
		Nodes []struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			Agent       string   `json:"agent"`
			Description string   `json:"description"`
			InputIDs    []string `json:"input_ids"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(resp.Message.Content), &plan); err != nil {
		return nil, fmt.Errorf("planner: parse: %w", err)
	}
	if len(plan.Nodes) == 0 {
		return nil, fmt.Errorf("planner: empty plan")
	}

	d := &dag.DAG{
		Status:    dag.DAGProposed,
		CreatedAt: time.Now(),
	}
	for _, n := range plan.Nodes {
		if n.ID == "" || n.Agent == "" {
			return nil, fmt.Errorf("planner: node missing id or agent")
		}
		d.Nodes = append(d.Nodes, dag.Node{
			ID:          n.ID,
			Name:        n.Name,
			Agent:       n.Agent,
			Description: n.Description,
			InputIDs:    n.InputIDs,
			Status:      dag.NodePending,
		})
	}

	if err := d.Validate(); err != nil {
		return nil, fmt.Errorf("planner: invalid dag: %w", err)
	}

	return d, nil
}
