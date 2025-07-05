package types

import (
	"os"
	"testing"
)

func TestRegistryAddGetList(t *testing.T) {
	r := NewRegistry()
	a := &ObjectType{Group: "g", Version: "v1", Kind: "A", Fields: map[string]*Field{"f": {Name: "f", Type: &PrimitiveType{"string"}}}}
	b := &ObjectType{Group: "g", Version: "v1", Kind: "B"}
	r.AddType(a)
	r.AddType(b)

	if got, ok := r.GetType("g", "v1", "A"); !ok || got != a {
		t.Fatalf("get A failed")
	}
	if got, ok := r.GetType("g", "v1", "B"); !ok || got != b {
		t.Fatalf("get B failed")
	}
	types := r.ListTypes()
	if len(types) != 2 {
		t.Fatalf("expected 2 types got %d", len(types))
	}
}

func TestObjectTypeExtend(t *testing.T) {
	base := &ObjectType{Fields: map[string]*Field{
		"a": {Name: "a", Type: &PrimitiveType{"string"}},
	}}
	ext := &ObjectType{Fields: map[string]*Field{
		"a": {Name: "a", Type: &PrimitiveType{"string"}, Required: true},
		"b": {Name: "b", Type: &PrimitiveType{"number"}},
	}, Open: true}
	base.Extend(ext)
	if !base.Open {
		t.Fatalf("expected open")
	}
	if f, ok := base.Fields["a"]; !ok || !f.Required {
		t.Fatalf("field a merge failed: %+v", f)
	}
	if _, ok := base.Fields["b"]; !ok {
		t.Fatalf("field b not merged")
	}

	incompatible := &ObjectType{Fields: map[string]*Field{
		"a": {Name: "a", Type: &PrimitiveType{"number"}},
	}}
	base.Extend(incompatible)
	if base.Fields["a"].Type.Name() != "string" {
		t.Fatalf("incompatible merge replaced type")
	}
}

func TestDefaultRegistry(t *testing.T) {
	_ = os.Unsetenv("NOSTOS_USE_KUBESPEC")
	r := DefaultRegistry()
	svc, ok := r.GetType("", "v1", "Service")
	if !ok || svc == nil {
		t.Fatalf("service not found")
	}
	if len(r.ListTypes()) != 1 {
		t.Fatalf("expected minimal registry")
	}
}

func TestDefaultRegistryUsesKubespec(t *testing.T) {
	orig := os.Getenv("NOSTOS_USE_KUBESPEC")
	_ = os.Setenv("NOSTOS_USE_KUBESPEC", "1")
	defer func() { _ = os.Setenv("NOSTOS_USE_KUBESPEC", orig) }()
	r := DefaultRegistry()
	if len(r.ListTypes()) <= 1 {
		t.Fatalf("expected kubespec types loaded")
	}
	if _, ok := r.GetType("", "v1", "Pod"); !ok {
		t.Fatalf("pod type missing")
	}
}

func TestKubespecRegistry(t *testing.T) {
	r, err := KubespecRegistry()
	if err != nil {
		t.Fatalf("kubespec registry error: %v", err)
	}
	pod, ok := r.GetType("", "v1", "Pod")
	if !ok {
		t.Fatalf("pod not found")
	}
	if pod.Fields["apiVersion"].Since == "" {
		t.Fatalf("since not set")
	}
}
