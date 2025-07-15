package planner

// Node represents a resource in the plan graph with links to its parents and children.
type Node struct {
	ID       string
	Resource ResourceType
	Parents  []*Node
	Children []*Node
}

// DAG represents a directed acyclic graph of resources.
type DAG struct {
	Roots []*Node
	Nodes map[string]*Node
	Order []*Node
}

// BuildDAG constructs a DAG from the given resources. It ensures the graph is acyclic
// by running a topological sort and returns an error if a cycle is detected.
func BuildDAG(resources []ResourceType) (*DAG, error) {
	sorted, err := TopologicalSort(resources)
	if err != nil {
		return nil, err
	}

	dag := &DAG{Nodes: make(map[string]*Node)}
	// create nodes
	for _, r := range resources {
		id := ResourceID(r)
		dag.Nodes[id] = &Node{ID: id, Resource: r}
	}
	// build edges
	for _, r := range resources {
		n := dag.Nodes[ResourceID(r)]
		for _, dep := range r.Dependencies {
			parent, ok := dag.Nodes[dep]
			if !ok {
				// unknown dependency, create placeholder node
				parent = &Node{ID: dep}
				dag.Nodes[dep] = parent
			}
			n.Parents = append(n.Parents, parent)
			parent.Children = append(parent.Children, n)
		}
	}
	for id, n := range dag.Nodes {
		if len(n.Parents) == 0 {
			dag.Roots = append(dag.Roots, dag.Nodes[id])
		}
	}
	// preserve topological order
	for _, r := range sorted {
		dag.Order = append(dag.Order, dag.Nodes[ResourceID(r)])
	}
	return dag, nil
}
