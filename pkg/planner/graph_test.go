package planner

import "testing"

func TestTopologicalSort(t *testing.T) {
	a := ResourceType{APIVersion: "v1", Kind: "A", Metadata: map[string]interface{}{"name": "a"}, Dependencies: []string{"v1:B::b", "v1:C::c"}}
	b := ResourceType{APIVersion: "v1", Kind: "B", Metadata: map[string]interface{}{"name": "b"}, Dependencies: []string{"v1:C::c"}}
	c := ResourceType{APIVersion: "v1", Kind: "C", Metadata: map[string]interface{}{"name": "c"}}

	sorted, err := TopologicalSort([]ResourceType{a, b, c})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 3 {
		t.Fatalf("expected 3 resources got %d", len(sorted))
	}
	if ResourceID(sorted[0]) != ResourceID(c) || ResourceID(sorted[1]) != ResourceID(b) || ResourceID(sorted[2]) != ResourceID(a) {
		t.Fatalf("unexpected order: %v", sorted)
	}
}
