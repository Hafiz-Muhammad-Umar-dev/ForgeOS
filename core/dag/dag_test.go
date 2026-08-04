package dag

import (
	"testing"
)

func TestValidDAG(t *testing.T) {
	d := &DAG{
		ID:       "dag-1",
		IntentID: "intent-1",
		Status:   DAGProposed,
		Nodes: []Node{
			{ID: "a", Name: "Node A", Agent: "agent-1", Description: "do A"},
			{ID: "b", Name: "Node B", Agent: "agent-2", Description: "do B", InputIDs: []string{"a"}},
			{ID: "c", Name: "Node C", Agent: "agent-3", Description: "do C", InputIDs: []string{"b"}},
		},
	}

	if err := d.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestEmptyDAG(t *testing.T) {
	d := &DAG{ID: "empty", Nodes: nil}
	if err := d.Validate(); err == nil {
		t.Fatal("expected error for empty DAG")
	}
}

func TestDuplicateNodeIDs(t *testing.T) {
	d := &DAG{
		Nodes: []Node{
			{ID: "a", Agent: "x"},
			{ID: "a", Agent: "y"},
		},
	}
	if err := d.Validate(); err == nil {
		t.Fatal("expected error for duplicate node IDs")
	}
}

func TestCyclicDAG(t *testing.T) {
	d := &DAG{
		Nodes: []Node{
			{ID: "a", Agent: "x", InputIDs: []string{"c"}},
			{ID: "b", Agent: "x", InputIDs: []string{"a"}},
			{ID: "c", Agent: "x", InputIDs: []string{"b"}},
		},
	}
	if err := d.Validate(); err == nil {
		t.Fatal("expected error for cyclic DAG")
	}
}

func TestUnknownEdgeTarget(t *testing.T) {
	d := &DAG{
		Nodes: []Node{
			{ID: "a", Agent: "x"},
		},
		Edges: []Edge{{From: "a", To: "nonexistent"}},
	}
	if err := d.Validate(); err == nil {
		t.Fatal("expected error for unknown edge target")
	}
}

func TestTopologicalSortOrder(t *testing.T) {
	d := &DAG{
		Nodes: []Node{
			{ID: "a", Agent: "x", Description: "A"},
			{ID: "b", Agent: "x", Description: "B", InputIDs: []string{"a"}},
			{ID: "c", Agent: "x", Description: "C", InputIDs: []string{"b"}},
		},
	}

	sorted, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("sort: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(sorted))
	}

	// a must come before b, b before c
	positions := make(map[string]int)
	for i, n := range sorted {
		positions[n.ID] = i
	}
	if positions["a"] >= positions["b"] {
		t.Error("a should come before b")
	}
	if positions["b"] >= positions["c"] {
		t.Error("b should come before c")
	}
}

func TestTopologicalSortNoDeps(t *testing.T) {
	d := &DAG{
		Nodes: []Node{
			{ID: "x", Agent: "a"},
			{ID: "y", Agent: "b"},
			{ID: "z", Agent: "c"},
		},
	}

	sorted, err := d.TopologicalSort()
	if err != nil {
		t.Fatalf("sort: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(sorted))
	}
}

func TestFindNode(t *testing.T) {
	d := &DAG{
		Nodes: []Node{
			{ID: "a", Agent: "x"},
			{ID: "b", Agent: "y"},
		},
	}

	if n := d.FindNode("a"); n == nil || n.ID != "a" {
		t.Error("FindNode('a') failed")
	}
	if n := d.FindNode("nonexistent"); n != nil {
		t.Error("FindNode should return nil for missing node")
	}
}

func TestNodePayload(t *testing.T) {
	p := NodePayload{
		NodeID:  "node-1",
		DAGID:   "dag-1",
		Agent:   "frontend",
		Status:  "completed",
		Summary: "done",
	}
	if p.NodeID != "node-1" {
		t.Errorf("nodeId=%s", p.NodeID)
	}
	if p.Status != "completed" {
		t.Errorf("status=%s", p.Status)
	}
}

func TestDAGStaging(t *testing.T) {
	d := &DAG{
		ID:     "dag-test",
		Status: DAGProposed,
		Nodes:  []Node{{ID: "a", Agent: "x"}},
	}
	if d.Status != DAGProposed {
		t.Errorf("initial status=%s", d.Status)
	}
	d.Status = DAGRunning
	if d.Status != DAGRunning {
		t.Errorf("running status=%s", d.Status)
	}
}
