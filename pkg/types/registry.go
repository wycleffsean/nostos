package types

import "sync"

// TypeDefinition represents a Kubernetes or user-defined API type in a simplified schema form.
// This struct is independent of Kubernetes library types to maintain decoupling from Kubernetes packages.
type TypeDefinition struct {
	Group   string            // API group of the type (empty string for core or user-defined types)
	Version string            // API version of the type
	Kind    string            // Kind name of the type
	Scope   string            // Scope of the resource: "Namespaced" or "Cluster"
	Fields  []FieldDefinition // Top-level fields of this type (schema of the object)
}

// FieldDefinition describes a field (property) in a TypeDefinition.
// If the field is a complex object, SubFields may contain one level of nested fields for introspection.
type FieldDefinition struct {
	Name      string            // Name of the field
	Type      string            // Data type of the field (e.g., "string", "int", "object", "[]<type>" for arrays)
	SubFields []FieldDefinition // Nested fields if this field is an object (one level deep)
}

// Registry stores TypeDefinitions in-memory, organized by API group and version (a hierarchical namespace for types).
// It allows thread-safe addition and lookup of both Kubernetes and user-defined types.
type Registry struct {
	mu    sync.RWMutex
	types map[string]map[string]map[string]TypeDefinition // group -> version -> kind -> TypeDefinition
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		types: make(map[string]map[string]map[string]TypeDefinition),
	}
}

// AddType stores a TypeDefinition in the registry. If an entry for the same group/version/kind exists, it is overwritten.
// This method is safe for concurrent use.
func (r *Registry) AddType(td TypeDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	grp, ver, kind := td.Group, td.Version, td.Kind
	if r.types[grp] == nil {
		r.types[grp] = make(map[string]map[string]TypeDefinition)
	}
	if r.types[grp][ver] == nil {
		r.types[grp][ver] = make(map[string]TypeDefinition)
	}
	r.types[grp][ver][kind] = td
}

// GetType retrieves a TypeDefinition by group, version, and kind.
// It returns the TypeDefinition and true if found, or false if the type is not in the registry.
// This method is safe for concurrent use.
func (r *Registry) GetType(group, version, kind string) (TypeDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	verMap, ok := r.types[group]
	if !ok {
		return TypeDefinition{}, false
	}
	kindMap, ok := verMap[version]
	if !ok {
		return TypeDefinition{}, false
	}
	td, ok := kindMap[kind]
	if !ok {
		return TypeDefinition{}, false
	}
	return td, true
}

// ListTypes returns all TypeDefinitions stored in the registry (for inspection or debugging).
// This method is safe for concurrent use.
func (r *Registry) ListTypes() []TypeDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []TypeDefinition
	for _, verMap := range r.types {
		for _, kindMap := range verMap {
			for _, td := range kindMap {
				result = append(result, td)
			}
		}
	}
	return result
}

/*
Evaluation of k8s.io/apimachinery/pkg/util/managedfields.TypeConverter:

The Kubernetes `TypeConverter` (from the managedfields package) can convert Kubernetes objects
to a "typed" representation using an OpenAPI schema. In theory, we could use `TypeConverter`
by providing it with the cluster’s OpenAPI schemas to get structured type information.
However, `TypeConverter` is primarily designed for managing object fields and server-side apply
field tracking, not for building a general-purpose type registry.

In our use case, we need to store and query type definitions independent of the Kubernetes runtime.
Using `TypeConverter` would introduce additional complexity and a dependency on Kubernetes internals
without clear benefit. Instead, by parsing the OpenAPI v3 and CRD schemas directly into our own
TypeDefinition format, we keep `pkg/types` self-contained and free of direct Kubernetes library dependencies.

In summary, while `TypeConverter` is a powerful tool for its intended purpose (field management),
it is not ideally suited for populating a type registry for a compiler. Our custom schema conversion
approach is simpler and more appropriate for maintaining an independent type registry.
*/
