package kube

import (
	"context"
	"encoding/json"
	"fmt"

	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// ApplyResources applies the given unstructured objects to the Kubernetes cluster
// using server-side apply. Resources are applied sequentially in the order provided.
func ApplyResources(objs []*unstructured.Unstructured) error {
	config, err := LoadKubeConfig()
	if err != nil {
		return err
	}

	dyn, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("failed to get API resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	for _, obj := range objs {
		if err := ApplyResource(context.TODO(), dyn, mapper, obj); err != nil {
			return err
		}
	}
	return nil
}

// ApplyResource performs server-side apply of a single unstructured object.
func ApplyResource(ctx context.Context, dyn dynamic.Interface, mapper meta.RESTMapper, obj *unstructured.Unstructured) error {
	gv, err := schema.ParseGroupVersion(obj.GetAPIVersion())
	if err != nil {
		return err
	}
	gvk := gv.WithKind(obj.GetKind())
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to map GVK %s: %w", gvk.String(), err)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	force := true
	_, err = dyn.Resource(mapping.Resource).Namespace(obj.GetNamespace()).Patch(
		ctx,
		obj.GetName(),
		types.ApplyPatchType,
		data,
		metav1.PatchOptions{FieldManager: "nostos", Force: &force},
	)
	if err != nil {
		return fmt.Errorf("apply %s/%s failed: %w", mapping.Resource.Resource, obj.GetName(), err)
	}
	return nil
}
