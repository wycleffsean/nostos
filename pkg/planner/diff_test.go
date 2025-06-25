package planner

import "testing"

func TestDiffResources(t *testing.T) {
	cluster := []ResourceType{
		{APIVersion: "v1", Kind: "A", Metadata: map[string]interface{}{"name": "a"}, Spec: map[string]interface{}{"x": 1}},
		{APIVersion: "v1", Kind: "B", Metadata: map[string]interface{}{"name": "b"}},
	}

	desired := []ResourceType{
		{APIVersion: "v1", Kind: "A", Metadata: map[string]interface{}{"name": "a"}, Spec: map[string]interface{}{"x": 2}},
		{APIVersion: "v1", Kind: "C", Metadata: map[string]interface{}{"name": "c"}},
	}

	diff := DiffResources(cluster, desired)

	if len(diff.ToCreate) != 1 || ResourceID(diff.ToCreate[0]) != "v1:C::c" {
		t.Fatalf("expected create C got %+v", diff.ToCreate)
	}
	if len(diff.ToUpdate) != 1 || ResourceID(diff.ToUpdate[0].Desired) != "v1:A::a" {
		t.Fatalf("expected update A got %+v", diff.ToUpdate)
	}
	if len(diff.Unmanaged) != 1 || ResourceID(diff.Unmanaged[0]) != "v1:B::b" {
		t.Fatalf("expected unmanaged B got %+v", diff.Unmanaged)
	}
}
