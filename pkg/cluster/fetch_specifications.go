package cluster

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// ClientConfig is an alias for rest.Config so that all k8s client configuration stays here.
type ClientConfig = rest.Config

// FetchCRDs fetches all CustomResourceDefinitions (CRDs) from the cluster.
func FetchCRDs(config *ClientConfig) ([]apiextensionsv1.CustomResourceDefinition, error) {
	clientset, err := apiextensionsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRD client: %w", err)
	}

	crdList, err := clientset.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list CRDs: %w", err)
	}

	return crdList.Items, nil
}
