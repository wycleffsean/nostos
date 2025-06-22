package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	// Kubernetes API clients for discovery and CRDs
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/openapi3"
	// "k8s.io/client-go/rest"
	// "k8s.io/client-go/tools/clientcmd"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/wycleffsean/nostos/pkg/types" // Import the types package (no direct Kubernetes deps inside it)
)

// Ensure we fetch specifications only once.
var fetchOnce sync.Once
var cachedFetchErr error

// FetchSpecifications connects to the Kubernetes API server, retrieves the OpenAPI v3
// schema for all Kubernetes built-in types and Custom Resource Definitions (CRDs),
// and stores their definitions in the provided type registry.
// It isolates all Kubernetes API interactions within this package.
// Subsequent calls use a cached result to avoid repeated fetches.
func FetchSpecifications(registry *types.Registry) error {
	fetchOnce.Do(func() {
		cachedFetchErr = fetchAndStoreSpecifications(registry)
	})
	return cachedFetchErr
}

func FetchAndFillRegistry() (*types.Registry, error) {
		// Create type registry
		registry := types.NewRegistry()

		// Fetch Kubernetes specifications and populate the registry
		err := FetchSpecifications(registry)
		return registry, err
}

// fetchAndStoreSpecifications performs the actual retrieval of schemas and populates the registry.
// This function is intended to be called only once (via FetchSpecifications).
func fetchAndStoreSpecifications(registry *types.Registry) error {
	config, err := LoadKubeConfig()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client config: %w", err)
	}

	// Create core API and CRD clientsets with the loaded config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	discoveryClient := clientset.Discovery() // for discovery and OpenAPI
	crdClient, err := apiextclientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create API extensions client: %w", err)
	}

	// Initialize OpenAPI V3 client to fetch schemas from the API server
	openapiClient := openapi.NewClient(discoveryClient.RESTClient())
	openapiRoot := openapi3.NewRoot(openapiClient)

	// Get all GroupVersions that have an OpenAPI V3 document
	groupVersions, err := openapiRoot.GroupVersions()
	if err != nil {
		// If OpenAPI V3 is not available, log a warning and continue (we will still fetch CRDs)
		fmt.Printf("Warning: OpenAPI V3 schema retrieval failed: %v\n", err)
		groupVersions = []schema.GroupVersion{}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex // protects the errs slice
	var errs []error

	// Fetch and process OpenAPI schemas for each group-version concurrently
	for _, gv := range groupVersions {
		wg.Add(1)
		go func(gv schema.GroupVersion) {
			defer wg.Done()
			// Fetch OpenAPI V3 spec as a map for this group-version
			specMap, err := openapiRoot.GVSpecAsMap(gv)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("failed to fetch OpenAPI spec for %s: %w", gv.String(), err))
				mu.Unlock()
				return
			}
			// Extract definitions (schemas) from the OpenAPI spec
			definitions, _ := extractDefinitions(specMap)
			if definitions == nil {
				return // no schemas found for this group-version
			}
			// Iterate through each definition in this group-version's OpenAPI schema
			for defName, schemaData := range definitions {
				_ = defName
				schemaObj, ok := schemaData.(map[string]interface{})
				if !ok {
					continue // skip if not in expected map format
				}
				// Check for the Kubernetes group-version-kind marker to identify top-level resource types
				if gvkList, hasGVK := schemaObj["x-kubernetes-group-version-kind"]; hasGVK {
					gvkSlice, ok := gvkList.([]interface{})
					if !ok {
						continue
					}
					for _, gvk := range gvkSlice {
						gvkMap, ok := gvk.(map[string]interface{})
						if !ok {
							continue
						}
						grp := getStringField(gvkMap, "group")
						ver := getStringField(gvkMap, "version")
						kind := getStringField(gvkMap, "kind")
						// Ensure the GVK matches the current group-version (it should, given how we fetched it)
						if grp == gv.Group && ver == gv.Version && kind != "" {
							// Determine if this resource is namespaced by checking discovery data
							namespaced := lookupNamespaced(discoveryClient, gv, kind)
							scope := "Cluster"
							if namespaced {
								scope = "Namespaced"
							}
							// Convert the OpenAPI schema to our internal TypeDefinition format
							typeDef := convertSchemaToTypeDef(grp, ver, kind, scope, schemaObj)
							registry.AddType(typeDef)
						}
					}
				}
			}
		}(gv)
	}

	// Fetch and process CustomResourceDefinition schemas concurrently as well
	wg.Add(1)
	go func() {
		defer wg.Done()
		crdList, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), v1.ListOptions{})
		if err != nil {
			mu.Lock()
			errs = append(errs, fmt.Errorf("failed to list CRDs: %w", err))
			mu.Unlock()
			return
		}
		for _, crd := range crdList.Items {
			grp := crd.Spec.Group
			kind := crd.Spec.Names.Kind
			scope := "Cluster"
			if crd.Spec.Scope == "Namespaced" {
				scope = "Namespaced"
			}
			// Each CRD may have multiple versions with distinct schemas
			for _, ver := range crd.Spec.Versions {
				verName := ver.Name
				if ver.Schema != nil && ver.Schema.OpenAPIV3Schema != nil {
					// Convert the CRD's OpenAPIV3Schema (JSONSchemaProps) to a generic map for reuse of conversion logic
					schemaBytes, _ := json.Marshal(ver.Schema.OpenAPIV3Schema)
					var schemaObj map[string]interface{}
					_ = json.Unmarshal(schemaBytes, &schemaObj)
					if schemaObj == nil {
						continue
					}
					// Inject the GVK extension if not present, for consistency
					schemaObj["x-kubernetes-group-version-kind"] = []interface{}{map[string]interface{}{
						"group":   grp,
						"version": verName,
						"kind":    kind,
					}}
					typeDef := convertSchemaToTypeDef(grp, verName, kind, scope, schemaObj)
					registry.AddType(typeDef)
				} else {
					// If no schema is provided (should not happen for v1 CRDs with structural schemas), create an empty definition
					typeDef := types.TypeDefinition{
						Group:   grp,
						Version: verName,
						Kind:    kind,
						Scope:   scope,
						Fields:  []types.FieldDefinition{}, // no field information available
					}
					registry.AddType(typeDef)
				}
			}
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Aggregate any errors that occurred during retrieval
	if len(errs) > 0 {
		errMsg := "errors occurred during spec fetch: "
		for i, e := range errs {
			if i > 0 {
				errMsg += " | "
			}
			errMsg += e.Error()
		}
		return fmt.Errorf(errMsg)
	}
	return nil
}

// extractDefinitions finds the "components.schemas" section in an OpenAPI v3 spec map.
func extractDefinitions(specMap map[string]interface{}) (map[string]interface{}, bool) {
	components, ok := specMap["components"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		return nil, false
	}
	return schemas, true
}

// convertSchemaToTypeDef converts an OpenAPI v3 schema (as a generic map) into a TypeDefinition.
// It extracts top-level fields and their types, including one level of nested fields for object types.
func convertSchemaToTypeDef(group, version, kind, scope string, schemaObj map[string]interface{}) types.TypeDefinition {
	td := types.TypeDefinition{
		Group:       group,
		Version:     version,
		Kind:        kind,
		Scope:       scope,
		Description: getStringField(schemaObj, "description"),
		Fields:      []types.FieldDefinition{},
	}
	// Only proceed if the schema has defined properties (i.e., it's an object schema)
	properties, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		return td
	}
	for propName, propVal := range properties {
		propSchema, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}
		fieldDef := types.FieldDefinition{Name: propName}
		// Determine the field's type
		fieldType := getStringField(propSchema, "type")
		if fieldType == "" && propSchema["$ref"] != nil {
			// If type is not directly given, it might be a reference to another schema
			ref := fmt.Sprintf("%v", propSchema["$ref"])
			fieldType = deriveRefTypeName(ref)
		}
		// Handle object and array types specifically
		if fieldType == "object" {
			// If the field is an object, capture one level of its sub-fields (properties)
			subFields := []types.FieldDefinition{}
			if subProps, ok := propSchema["properties"].(map[string]interface{}); ok {
				for subName, subVal := range subProps {
					subSchema, _ := subVal.(map[string]interface{})
					if subSchema == nil {
						continue
					}
					subFieldType := getStringField(subSchema, "type")
					if subFieldType == "" && subSchema["$ref"] != nil {
						ref := fmt.Sprintf("%v", subSchema["$ref"])
						subFieldType = deriveRefTypeName(ref)
					}
					subFields = append(subFields, types.FieldDefinition{
						Name: subName,
						Type: subFieldType,
					})
				}
			}
			fieldDef.Type = "object"
			fieldDef.Description = getStringField(propSchema, "description")
			fieldDef.SubFields = subFields
		} else if fieldType == "array" {
			// If the field is an array, determine the element type
			elemType := "any"
			if items, ok := propSchema["items"].(map[string]interface{}); ok {
				elemType = getStringField(items, "type")
				if elemType == "" && items["$ref"] != nil {
					ref := fmt.Sprintf("%v", items["$ref"])
					elemType = deriveRefTypeName(ref)
				}
				if elemType == "" {
					elemType = "object"
				}
			}
			fieldDef.Type = "[]" + elemType
		} else if fieldType != "" {
			// Primitive type (string, integer, boolean, etc.)
			fieldDef.Type = fieldType
			fieldDef.Description = getStringField(propSchema, "description")
		} else {
			// Fallback if type is unspecified
			fieldDef.Type = "any"
			fieldDef.Description = getStringField(propSchema, "description")
		}
		td.Fields = append(td.Fields, fieldDef)
	}
	return td
}

// lookupNamespaced checks via discovery if a given kind in a group-version is namespaced.
func lookupNamespaced(discoveryClient discovery.DiscoveryInterface, gv schema.GroupVersion, kind string) bool {
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(gv.String())
	if err != nil {
		return false
	}
	for _, res := range resourceList.APIResources {
		if res.Kind == kind {
			return res.Namespaced
		}
	}
	return false
}

// deriveRefTypeName derives a type name from an OpenAPI $ref string.
// e.g. "#/components/schemas/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta" -> "ObjectMeta"
func deriveRefTypeName(ref string) string {
	// Take substring after the last '/'
	idx := lastIndex(ref, '/')
	fullName := ref
	if idx != -1 {
		fullName = ref[idx+1:]
	}
	// If the full name contains dots (package path), return only the last segment
	if dotIdx := lastIndex(fullName, '.'); dotIdx != -1 {
		return fullName[dotIdx+1:]
	}
	return fullName
}

// getStringField safely retrieves a string value from a map for the given key.
func getStringField(m map[string]interface{}, field string) string {
	if val, ok := m[field]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// lastIndex returns the index of the last occurrence of sep in s, or -1 if not found.
func lastIndex(s string, sep byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep {
			return i
		}
	}
	return -1
}
