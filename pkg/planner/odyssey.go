package planner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/kube"
	"github.com/wycleffsean/nostos/pkg/workspace"
	"github.com/wycleffsean/nostos/vm"
)

// odysseyEntry represents a single cluster entry in odyssey.no. Each key in the
// map is a namespace name and the value is a list of resources or file paths
// that belong to that namespace. The special "default" namespace mirrors the
// default output in Nix flakes.
type odysseyEntry map[string][]interface{}

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
	_, items := lang.NewStringLexer(string(data))
	p := lang.NewParser(items)
	ast := p.Parse()
	val, err := vm.EvalWithDir(ast, filepath.Dir(odysseyPath))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate odyssey file: %w", err)
	}

	entries := make(map[string]odysseyEntry)
	if err := mapstructure.Decode(val, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode odyssey file: %w", err)
	}

	entry, ok := entries[ctx]
	if !ok {
		return &Plan{Resources: []ResourceType{}}, nil
	}

	var resources []ResourceType

	for ns, items := range entry {
		if ns != "" {
			resources = append(resources, ResourceType{
				APIVersion: "v1",
				Kind:       "Namespace",
				Metadata:   map[string]interface{}{"name": ns},
			})
		}

		var paths []string
		var inlineObjs []map[string]interface{}
		for _, it := range items {
			switch v := it.(type) {
			case string:
				paths = append(paths, filepath.Join(workspace.Dir(), v))
			case map[string]interface{}:
				inlineObjs = append(inlineObjs, v)
			default:
				return nil, fmt.Errorf("unsupported odyssey value %T", it)
			}
		}

		loaded, err := loadResourcesFromFiles(paths, ns)
		if err != nil {
			return nil, err
		}
		resources = append(resources, loaded...)

		for _, obj := range inlineObjs {
			u := &unstructured.Unstructured{Object: obj}
			rt, err := convertUnstructuredToResourceType(u)
			if err != nil {
				return nil, err
			}
			if ns != "" {
				if _, ok := rt.Metadata["namespace"]; !ok {
					rt.Metadata["namespace"] = ns
				}
			}
			resources = append(resources, rt)
		}
	}

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
func loadResourcesFromFiles(paths []string, defaultNS string) ([]ResourceType, error) {
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
			if defaultNS != "" {
				if _, ok := rt.Metadata["namespace"]; !ok {
					rt.Metadata["namespace"] = defaultNS
				}
			}
			resources = append(resources, rt)
		}
	}
	return resources, nil
}
