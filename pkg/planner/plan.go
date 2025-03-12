package planner

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/wycleffsean/nostos/pkg/kube"
)

// ResourceType represents an internal abstraction of a Kubernetes resource.
type ResourceType struct {
	APIVersion string
	Kind       string
	Metadata   map[string]interface{}
	Spec       map[string]interface{}
}

// Plan represents a unified plan graph that includes both the current cluster state and user-defined resources.
type Plan struct {
	Resources []ResourceType
}

// BuildPlanFromCluster fetches all resources from the cluster and converts them into internal ResourceType representations.
func BuildPlanFromCluster() (*Plan, error) {
	clusterResources, err := kube.FetchAllResources()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cluster resources: %w", err)
	}

	var plan Plan

	// Convert each unstructured object into a ResourceType.
	for _, resourceList := range clusterResources {
		for _, u := range resourceList {
			rt, err := convertUnstructuredToResourceType(u)
			if err != nil {
				// Optionally log error and skip this resource.
				fmt.Printf("Warning: failed to convert resource: %v\n", err)
				continue
			}
			plan.Resources = append(plan.Resources, rt)
		}
	}

	return &plan, nil
}

// MergeUserDefinedResources takes a plan built from the cluster and merges it with resources defined in user code.
func MergeUserDefinedResources(clusterPlan *Plan, userResources []ResourceType) *Plan {
	mergedPlan := &Plan{
		Resources: append(clusterPlan.Resources, userResources...),
	}
	// Further merging logic could resolve conflicts, order operations, etc.
	return mergedPlan
}

// convertUnstructuredToResourceType converts an unstructured Kubernetes object into a ResourceType.
func convertUnstructuredToResourceType(u *unstructured.Unstructured) (ResourceType, error) {
	metadata, found, err := unstructured.NestedMap(u.Object, "metadata")
	if err != nil || !found {
		return ResourceType{}, fmt.Errorf("metadata not found")
	}
	spec, found, err := unstructured.NestedMap(u.Object, "spec")
	if err != nil {
		return ResourceType{}, fmt.Errorf("spec error: %v", err)
	}
	if !found {
		// Some resources may not have a spec.
		spec = make(map[string]interface{})
	}

	return ResourceType{
		APIVersion: u.GetAPIVersion(),
		Kind:       u.GetKind(),
		Metadata:   metadata,
		Spec:       spec,
	}, nil
}
