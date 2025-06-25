package planner

import "testing"

func TestFilterSystemNamespace(t *testing.T) {
	resources := []ResourceType{
		{APIVersion: "v1", Kind: "A", Metadata: map[string]interface{}{"name": "a", "namespace": "default"}},
		{APIVersion: "v1", Kind: "B", Metadata: map[string]interface{}{"name": "b", "namespace": "kube-system"}},
	}
	filtered := FilterSystemNamespace(resources)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 resource got %d", len(filtered))
	}
	if ResourceID(filtered[0]) != "v1:A:default:a" {
		t.Fatalf("unexpected resource: %+v", filtered[0])
	}
}

func TestFilterClusterScoped(t *testing.T) {
	resources := []ResourceType{
		{APIVersion: "v1", Kind: "Namespace", Metadata: map[string]interface{}{"name": "default"}},
		{APIVersion: "v1", Kind: "ConfigMap", Metadata: map[string]interface{}{"name": "cfg", "namespace": "default"}},
	}
	filtered := FilterClusterScoped(resources)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 resource got %d", len(filtered))
	}
	if ResourceID(filtered[0]) != "v1:ConfigMap:default:cfg" {
		t.Fatalf("unexpected resource: %+v", filtered[0])
	}
}
