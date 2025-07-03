package types

import "os"

// DefaultRegistry returns a Registry populated with a minimal set of built-in
// Kubernetes type definitions. The data here is intentionally small so the
// language server can provide basic completions and hover information even when
// connecting to a cluster is not possible.
func DefaultRegistry() *Registry {
	// Load the baked kubespec dataset when requested.
	if os.Getenv("NOSTOS_USE_KUBESPEC") != "" {
		if r, err := KubespecRegistry(); err == nil {
			return r
		}
	}

	r := NewRegistry()

	svc := &ObjectType{
		Group:       "",
		Version:     "v1",
		Kind:        "Service",
		Scope:       "Namespaced",
		Description: "Service exposes a set of Pods as a network service.",
		Fields: map[string]*Field{
			"apiVersion": {Name: "apiVersion", Type: &PrimitiveType{"string"}, Required: true},
			"kind":       {Name: "kind", Type: &PrimitiveType{"string"}, Required: true},
			"metadata": {
				Name:     "metadata",
				Type:     &ObjectType{Fields: map[string]*Field{"name": {Name: "name", Type: &PrimitiveType{"string"}, Required: true}}, Open: true},
				Required: true,
			},
			"spec": {
				Name: "spec",
				Type: &ObjectType{Fields: map[string]*Field{
					"type":     {Name: "type", Type: &PrimitiveType{"string"}},
					"selector": {Name: "selector", Type: &ObjectType{Open: true, Fields: map[string]*Field{}}},
					"ports":    {Name: "ports", Type: &ListType{Elem: &ObjectType{Open: true, Fields: map[string]*Field{}}}},
				}, Open: true},
			},
		},
	}
	r.AddType(svc)

	return r
}
