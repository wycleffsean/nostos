package planner

import "fmt"

// ResourceID creates a deterministic ID for a resource using
// apiVersion, kind, namespace and name.
func ResourceID(r ResourceType) string {
	name, _ := r.Metadata["name"].(string)
	ns, _ := r.Metadata["namespace"].(string)
	return fmt.Sprintf("%s:%s:%s:%s", r.APIVersion, r.Kind, ns, name)
}

// TopologicalSort orders resources so that dependencies come first.
// It returns an error if a cycle is detected.
func TopologicalSort(resources []ResourceType) ([]ResourceType, error) {
	nodes := make(map[string]ResourceType)
	indegree := make(map[string]int)
	adj := make(map[string][]string)

	for _, r := range resources {
		id := ResourceID(r)
		nodes[id] = r
		if len(r.Dependencies) == 0 {
			indegree[id] = indegree[id]
		}
		for _, dep := range r.Dependencies {
			indegree[id]++
			adj[dep] = append(adj[dep], id)
		}
	}

	var queue []string
	for id := range nodes {
		if indegree[id] == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []ResourceType
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		sorted = append(sorted, nodes[n])
		for _, m := range adj[n] {
			indegree[m]--
			if indegree[m] == 0 {
				queue = append(queue, m)
			}
		}
	}

	if len(sorted) != len(nodes) {
		return nil, fmt.Errorf("dependency cycle detected")
	}

	return sorted, nil
}
