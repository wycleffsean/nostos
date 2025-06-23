package types

// DefaultRegistry returns a Registry populated with a minimal set of built-in
// Kubernetes type definitions. The data here is intentionally small so the
// language server can provide basic completions and hover information even when
// connecting to a cluster is not possible.
func DefaultRegistry() *Registry {
	r := NewRegistry()

	r.AddType(TypeDefinition{
		Group:       "",
		Version:     "v1",
		Kind:        "Service",
		Scope:       "Namespaced",
		Description: "Service exposes a set of Pods as a network service.",
		Fields: []FieldDefinition{
			{Name: "apiVersion", Type: "string", Required: true},
			{Name: "kind", Type: "string", Required: true},
			{
				Name:      "metadata",
				Type:      "object",
				Required:  true,
				SubFields: []FieldDefinition{{Name: "name", Type: "string", Required: true}},
			},
			{
				Name: "spec",
				Type: "object",
				SubFields: []FieldDefinition{
					{Name: "type", Type: "string"},
					{Name: "selector", Type: "object"},
					{Name: "ports", Type: "[]object"},
				},
			},
		},
	})

	return r
}
