package planner

import "testing"

func TestBuildDAG(t *testing.T) {
	a := ResourceType{APIVersion: "v1", Kind: "A", Metadata: map[string]interface{}{"name": "a"}, Dependencies: []string{"v1:B::b"}}
	b := ResourceType{APIVersion: "v1", Kind: "B", Metadata: map[string]interface{}{"name": "b"}}

	dag, err := BuildDAG([]ResourceType{a, b})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dag.Roots) != 1 || dag.Roots[0].ID != ResourceID(b) {
		t.Fatalf("unexpected roots: %+v", dag.Roots)
	}
	nodeA := dag.Nodes[ResourceID(a)]
	if len(nodeA.Parents) != 1 || nodeA.Parents[0].ID != ResourceID(b) {
		t.Fatalf("unexpected parents for a: %+v", nodeA.Parents)
	}
	if len(dag.Order) != 2 || dag.Order[0].ID != ResourceID(b) || dag.Order[1].ID != ResourceID(a) {
		t.Fatalf("unexpected order: %+v", dag.Order)
	}
}

func TestBuildDAGCycle(t *testing.T) {
	a := ResourceType{APIVersion: "v1", Kind: "A", Metadata: map[string]interface{}{"name": "a"}, Dependencies: []string{"v1:B::b"}}
	b := ResourceType{APIVersion: "v1", Kind: "B", Metadata: map[string]interface{}{"name": "b"}, Dependencies: []string{"v1:A::a"}}
	_, err := BuildDAG([]ResourceType{a, b})
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}
