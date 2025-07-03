package types

import "sync"

// Registry stores TypeDefinitions in-memory, organized by API group and version (a hierarchical namespace for types).
// It allows thread-safe addition and lookup of both Kubernetes and user-defined types.
type Registry struct {
	mu    sync.RWMutex
	types map[string]map[string]map[string]Type // group -> version -> kind -> Type
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		types: make(map[string]map[string]map[string]Type),
	}
}

// AddType stores a TypeDefinition in the registry. If an entry for the same group/version/kind exists, it is overwritten.
// This method is safe for concurrent use.
func (r *Registry) AddType(td *ObjectType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	grp, ver, kind := td.Group, td.Version, td.Kind
	if r.types[grp] == nil {
		r.types[grp] = make(map[string]map[string]Type)
	}
	if r.types[grp][ver] == nil {
		r.types[grp][ver] = make(map[string]Type)
	}
	r.types[grp][ver][kind] = td
}

// GetType retrieves a TypeDefinition by group, version, and kind.
// It returns the TypeDefinition and true if found, or false if the type is not in the registry.
// This method is safe for concurrent use.
func (r *Registry) GetType(group, version, kind string) (*ObjectType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	verMap, ok := r.types[group]
	if !ok {
		return nil, false
	}
	kindMap, ok := verMap[version]
	if !ok {
		return nil, false
	}
	td, ok := kindMap[kind]
	if !ok {
		return nil, false
	}
	if ot, ok := td.(*ObjectType); ok {
		return ot, true
	}
	return nil, false
}

// ListTypes returns all TypeDefinitions stored in the registry (for inspection or debugging).
// This method is safe for concurrent use.
func (r *Registry) ListTypes() []*ObjectType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*ObjectType
	for _, verMap := range r.types {
		for _, kindMap := range verMap {
			for _, td := range kindMap {
				if ot, ok := td.(*ObjectType); ok {
					result = append(result, ot)
				}
			}
		}
	}
	return result
}

// TypeDefinitions is an alias for ListTypes for clarity in callers.
func (r *Registry) TypeDefinitions() []*ObjectType {
	return r.ListTypes()
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
