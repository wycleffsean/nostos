package planner

import (
	"reflect"

	"github.com/pmezard/go-difflib/difflib"
	"sigs.k8s.io/yaml"
)

// DiffResult represents differences between desired resources and cluster state.
type Update struct {
	Current ResourceType
	Desired ResourceType
}

type DiffResult struct {
	ToCreate  []ResourceType
	ToUpdate  []Update
	Unmanaged []ResourceType
}

// DiffResources computes the difference between cluster and desired resources.
func DiffResources(cluster, desired []ResourceType) DiffResult {
	clusterMap := make(map[string]ResourceType)
	for _, r := range cluster {
		clusterMap[ResourceID(r)] = r
	}
	desiredMap := make(map[string]ResourceType)
	for _, r := range desired {
		desiredMap[ResourceID(r)] = r
	}

	var diff DiffResult

	for id, dr := range desiredMap {
		cr, ok := clusterMap[id]
		if !ok {
			diff.ToCreate = append(diff.ToCreate, dr)
			continue
		}
		if !reflect.DeepEqual(cr.Spec, dr.Spec) || cr.APIVersion != dr.APIVersion || cr.Kind != dr.Kind {
			diff.ToUpdate = append(diff.ToUpdate, Update{Current: cr, Desired: dr})
		}
	}

	for id, cr := range clusterMap {
		if _, ok := desiredMap[id]; !ok {
			diff.Unmanaged = append(diff.Unmanaged, cr)
		}
	}

	return diff
}

// BuildPlanFromDiff returns resources to apply (creates and updates) sorted
// topologically based on their dependencies.
func BuildPlanFromDiff(diff DiffResult) ([]ResourceType, error) {
	toApply := append([]ResourceType{}, diff.ToCreate...)
	for _, u := range diff.ToUpdate {
		toApply = append(toApply, u.Desired)
	}
	return TopologicalSort(toApply)
}

// DiffString returns a unified diff of two resources focusing on their specs and metadata.
func DiffString(current, desired ResourceType) string {
	currYAML, _ := yaml.Marshal(map[string]interface{}{
		"apiVersion": current.APIVersion,
		"kind":       current.Kind,
		"metadata":   current.Metadata,
		"spec":       current.Spec,
	})
	desiredYAML, _ := yaml.Marshal(map[string]interface{}{
		"apiVersion": desired.APIVersion,
		"kind":       desired.Kind,
		"metadata":   desired.Metadata,
		"spec":       desired.Spec,
	})

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(currYAML)),
		B:        difflib.SplitLines(string(desiredYAML)),
		FromFile: "cluster",
		ToFile:   "desired",
		Context:  3,
	}
	out, _ := difflib.GetUnifiedDiffString(diff)
	return out
}
