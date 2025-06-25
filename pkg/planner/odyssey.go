package planner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/wycleffsean/nostos/pkg/kube"
	"github.com/wycleffsean/nostos/pkg/workspace"
)

// odysseyEntry represents a single cluster entry in odyssey.no.
type odysseyEntry struct {
	Namespaces []string `yaml:"namespaces"`
	Resources  []string `yaml:"resources"`
}

// BuildPlanFromOdyssey loads the workspace odyssey.no file and returns a plan
// for the current Kubernetes context.
func BuildPlanFromOdyssey(ignoreSystemNamespace, ignoreClusterScoped bool) (*Plan, error) {
	ctx, err := kube.CurrentContext()
	if err != nil {
		return nil, err
	}

	odysseyPath := filepath.Join(workspace.Dir(), "odyssey.no")
	data, err := os.ReadFile(odysseyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read odyssey file: %w", err)
	}

	entries := make(map[string]odysseyEntry)
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse odyssey file: %w", err)
	}

	entry, ok := entries[ctx]
	if !ok {
		return &Plan{Resources: []ResourceType{}}, nil
	}

	var resources []ResourceType
	for _, ns := range entry.Namespaces {
		resources = append(resources, ResourceType{
			APIVersion: "v1",
			Kind:       "Namespace",
			Metadata:   map[string]interface{}{"name": ns},
		})
	}

	filePaths := make([]string, 0, len(entry.Resources))
	for _, r := range entry.Resources {
		filePaths = append(filePaths, filepath.Join(workspace.Dir(), r))
	}
	loaded, err := loadResourcesFromFiles(filePaths)
	if err != nil {
		return nil, err
	}
	resources = append(resources, loaded...)

	if ignoreSystemNamespace {
		resources = FilterSystemNamespace(resources)
	}
	if ignoreClusterScoped {
		resources = FilterClusterScoped(resources)
	}

	return &Plan{Resources: resources}, nil
}

// loadResourcesFromFiles parses Kubernetes YAML manifests from the given paths
// and converts them to ResourceType values.
func loadResourcesFromFiles(paths []string) ([]ResourceType, error) {
	var resources []ResourceType
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		docs := bytes.Split(data, []byte("\n---"))
		for _, d := range docs {
			if len(bytes.TrimSpace(d)) == 0 {
				continue
			}
			var obj map[string]interface{}
			if err := yaml.Unmarshal(d, &obj); err != nil {
				return nil, fmt.Errorf("parse %s: %w", p, err)
			}
			u := &unstructured.Unstructured{Object: obj}
			rt, err := convertUnstructuredToResourceType(u)
			if err != nil {
				return nil, err
			}
			resources = append(resources, rt)
		}
	}
	return resources, nil
}
