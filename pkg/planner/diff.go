package planner

import "reflect"

// DiffResult represents differences between desired resources and cluster state.
type DiffResult struct {
	ToCreate []ResourceType
	ToUpdate []ResourceType
	ToDelete []ResourceType
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
			diff.ToUpdate = append(diff.ToUpdate, dr)
		}
	}

	for id, cr := range clusterMap {
		if _, ok := desiredMap[id]; !ok {
			diff.ToDelete = append(diff.ToDelete, cr)
		}
	}

	return diff
}

// BuildPlanFromDiff returns resources to apply (creates and updates) sorted
// topologically based on their dependencies.
func BuildPlanFromDiff(diff DiffResult) ([]ResourceType, error) {
	toApply := append([]ResourceType{}, diff.ToCreate...)
	toApply = append(toApply, diff.ToUpdate...)
	return TopologicalSort(toApply)
}
