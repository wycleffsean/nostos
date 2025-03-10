package cluster

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// FetchAllResources retrieves all resources available in the cluster.
// It returns a map keyed by GroupVersionResource with a slice of unstructured objects.
func FetchAllResources(config *rest.Config) (map[schema.GroupVersionResource][]*unstructured.Unstructured, error) {
	// Create a discovery client.
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery client: %w", err)
	}

	// Get all API group resources (returns a slice).
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get API group resources: %w", err)
	}

	// Build a RESTMapper (if needed later).
	rm := restmapper.NewDiscoveryRESTMapper(apiGroupResources)

	// Create a dynamic client.
	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	resources := make(map[schema.GroupVersionResource][]*unstructured.Unstructured)

	// Iterate over each API group resource.
	for _, groupResource := range apiGroupResources {
		groupName := groupResource.Group.Name // group name as string
		for version, resourceList := range groupResource.VersionedResources {
			for _, r := range resourceList {
				// Skip subresources if the resource name contains a "/"
				if containsSubresource(r.Name) {
					continue
				}

				// Build the GroupVersionResource.
				gvr := schema.GroupVersionResource{
					Group:    groupName,
					Version:  version,
					Resource: r.Name,
				}

				// List the resource.
				list, err := dyn.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
				if err != nil {
					// Log the error and continue with other resources.
					fmt.Printf("Warning: failed to list resource %s: %v\n", gvr.String(), err)
					continue
				}

				// Append each item from the list.
				for i := range list.Items {
					resources[gvr] = append(resources[gvr], list.Items[i].DeepCopy())
				}
			}
		}
	}

	// Optionally, you could use rm (the RESTMapper) for further processing.
	_ = rm

	return resources, nil
}

// containsSubresource returns true if the resource name indicates a subresource.
func containsSubresource(name string) bool {
	// A simple check: if the name contains a "/" it is likely a subresource.
	return strings.Contains(name, "/")
}
