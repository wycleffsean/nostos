package kube

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// List of resources that are known to trigger deprecation warnings from the
// Kubernetes API server. We simply skip them when fetching resources since they
// are not typically useful in diffs and avoiding them keeps output clean.
var deprecatedGVRs = []schema.GroupVersionResource{
	{Group: "", Version: "v1", Resource: "componentstatuses"},
	{Group: "", Version: "v1", Resource: "endpoints"},
	{Group: "cilium.io", Version: "v2alpha1", Resource: "ciliumnodeconfigs"},
}

// FetchAllResources retrieves all resources available in the cluster.
// It returns a map keyed by GroupVersionResource with a slice of unstructured objects.
func FetchAllResources() (map[schema.GroupVersionResource][]*unstructured.Unstructured, error) {
	config, err := LoadKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client config: %w", err)
	}
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
				// Skip resources that do not support the "list" verb
				if !supportsListVerb(r) {
					continue
				}
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

				// Skip deprecated resources to avoid API server warnings.
				if isDeprecatedResource(gvr) {
					continue
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

// supportsListVerb checks whether the API resource supports the "list" verb.
func supportsListVerb(r metav1.APIResource) bool {
	for _, v := range r.Verbs {
		if strings.EqualFold(v, "list") {
			return true
		}
	}
	return false
}

// isDeprecatedResource returns true if the given GVR is in the deprecated list.
func isDeprecatedResource(gvr schema.GroupVersionResource) bool {
	for _, d := range deprecatedGVRs {
		if gvr.Group == d.Group && gvr.Version == d.Version && gvr.Resource == d.Resource {
			return true
		}
	}
	return false
}
