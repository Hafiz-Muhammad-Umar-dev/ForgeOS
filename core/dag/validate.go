package dag

import (
	"fmt"
)

// Validate checks the DAG for structural integrity:
// - No duplicate node IDs
// - All edge references point to existing nodes
// - No cycles (via topological sort)
// - At least one node exists
func (d *DAG) Validate() error {
	if len(d.Nodes) == 0 {
		return fmt.Errorf("dag: at least one node is required")
	}

	// Check for duplicate node IDs.
	seen := make(map[string]bool)
	for _, n := range d.Nodes {
		if seen[n.ID] {
			return fmt.Errorf("dag: duplicate node id %q", n.ID)
		}
		seen[n.ID] = true
	}

	// Build adjacency list and in-degree map for cycle detection.
	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	for _, n := range d.Nodes {
		adj[n.ID] = nil
		inDegree[n.ID] = 0
	}

	// Check edges reference existing nodes.
	for _, e := range d.Edges {
		if !seen[e.From] {
			return fmt.Errorf("dag: edge references unknown from node %q", e.From)
		}
		if !seen[e.To] {
			return fmt.Errorf("dag: edge references unknown to node %q", e.To)
		}
		adj[e.From] = append(adj[e.From], e.To)
		inDegree[e.To]++
	}

	// Check InputIDs reference existing nodes.
	for _, n := range d.Nodes {
		for _, inputID := range n.InputIDs {
			if !seen[inputID] {
				return fmt.Errorf("dag: node %q references unknown input %q", n.ID, inputID)
			}
			// Add edge from input to this node for topological sort.
			adj[inputID] = append(adj[inputID], n.ID)
			inDegree[n.ID]++
		}
	}

	// Kahn's algorithm for topological sort and cycle detection.
	var queue []string
	for _, n := range d.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	sorted := 0
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted++

		for _, neighbor := range adj[id] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if sorted != len(d.Nodes) {
		return fmt.Errorf("dag: cycle detected: %d of %d nodes reachable", sorted, len(d.Nodes))
	}

	return nil
}

// TopologicalSort returns nodes in execution order (dependencies first).
// Returns an error if the DAG has a cycle.
func (d *DAG) TopologicalSort() ([]Node, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	// Build adjacency list and in-degree map.
	adj := make(map[string][]string)
	inDegree := make(map[string]int)
	for _, n := range d.Nodes {
		adj[n.ID] = nil
		inDegree[n.ID] = 0
	}

	for _, n := range d.Nodes {
		for _, inputID := range n.InputIDs {
			adj[inputID] = append(adj[inputID], n.ID)
			inDegree[n.ID]++
		}
	}

	// Kahn's algorithm.
	var queue []string
	for _, n := range d.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var result []Node
	nodeMap := make(map[string]Node)
	for _, n := range d.Nodes {
		nodeMap[n.ID] = n
	}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, nodeMap[id])

		for _, neighbor := range adj[id] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return result, nil
}
