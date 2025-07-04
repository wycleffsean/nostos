package planner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

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

// EvaluateOdyssey reads and evaluates an odyssey.no file. It returns the
// fully evaluated structure where any import() calls have been resolved.
func EvaluateOdyssey(path string) (map[string]odysseyEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read odyssey file: %w", err)
	}
	_, items := lang.NewStringLexer(string(data))
	p := lang.NewParser(items)
	ast := p.Parse()
	if perrs := lang.CollectParseErrors(ast); len(perrs) > 0 {
		return nil, fmt.Errorf("failed to parse odyssey file: %s", perrs[0].Error())
	}
	val, err := vm.EvalWithDir(ast, filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate odyssey file: %w", err)
	}
	entries := make(map[string]odysseyEntry)
	if err := mapstructure.Decode(val, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode odyssey file: %w", err)
	}
	return entries, nil
}

// BuildPlanFromOdyssey loads the workspace odyssey.no file and returns a plan
// for the current Kubernetes context.
func BuildPlanFromOdyssey(ignoreSystemNamespace, ignoreClusterScoped bool) (*Plan, error) {
	ctx, err := kube.CurrentContext()
	if err != nil {
		return nil, err
	}

	odysseyPath := filepath.Join(workspace.Dir(), "odyssey.no")
	entries, err := EvaluateOdyssey(odysseyPath)
	if err != nil {
		return nil, err
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
				fmt.Printf("Warning: failed to convert resource: %v\n", err)
				continue
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
			_, items := lang.NewStringLexer(string(d))
			parser := lang.NewParser(items)
			ast := parser.Parse()
			if perrs := lang.CollectParseErrors(ast); len(perrs) > 0 {
				return nil, fmt.Errorf("parse %s: %s", p, perrs[0].Error())
			}
			val, err := vm.EvalWithDir(ast, filepath.Dir(p))
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", p, err)
			}
			obj, ok := val.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected map in %s", p)
			}
			u := &unstructured.Unstructured{Object: obj}
			rt, err := convertUnstructuredToResourceType(u)
			if err != nil {
				fmt.Printf("Warning: failed to convert resource: %v\n", err)
				continue
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
