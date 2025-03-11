package registry

import (
	"fmt"
	"sync"

	"github.com/wycleffsean/nostos/pkg/cluster"
	"github.com/wycleffsean/nostos/pkg/types"
)

// TypeRegistry uses a sync.Map to allow concurrent registration of type definitions.
type TypeRegistry struct {
	Types sync.Map // map[string]*types.TypeDefinition
}

// NewTypeRegistry creates a new empty TypeRegistry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{}
}

// AppendRegistryFromCRDs fetches CRDs using the cluster package and appends the resulting
// type definitions to the registry. The key used is "group/version/kind".
func (r *TypeRegistry) AppendRegistryFromCRDs(config *cluster.ClientConfig) error {
	crds, err := cluster.FetchCRDs(config)
	if err != nil {
		return fmt.Errorf("failed to fetch CRDs: %w", err)
	}

	for _, crd := range crds {
		for _, version := range crd.Spec.Versions {
			// Only process versions with an OpenAPI v3 schema.
			if version.Schema != nil && version.Schema.OpenAPIV3Schema != nil {
				td := types.ConvertJSONSchemaPropsToTypeDefinition(
					fmt.Sprintf("%s/%s", crd.Spec.Group, version.Name),
					crd.Spec.Names.Kind,
					version.Schema.OpenAPIV3Schema,
				)
				// Construct a unique key (e.g., "group/version/kind").
				key := fmt.Sprintf("%s/%s/%s", crd.Spec.Group, version.Name, crd.Spec.Names.Kind)
				r.Types.Store(key, td)
			}
		}
	}
	return nil
}
