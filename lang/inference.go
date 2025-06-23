package lang

import "github.com/wycleffsean/nostos/pkg/types"

// InferType attempts to infer the TypeDefinition of a parsed map node based on
// the required fields of the types stored in the registry. It returns the
// matching TypeDefinition and true if exactly one match is found.
func InferType(n node, reg *types.Registry) (types.TypeDefinition, bool) {
	m, ok := n.(*Map)
	if !ok {
		return types.TypeDefinition{}, false
	}

	// Build a set of field names present in the map
	fieldNames := make(map[string]struct{})
	for sym := range *m {
		fieldNames[sym.Text] = struct{}{}
	}

	matches := make([]types.TypeDefinition, 0)
	for _, td := range reg.ListTypes() {
		match := true
		for _, f := range td.Fields {
			if f.Required {
				if _, ok := fieldNames[f.Name]; !ok {
					match = false
					break
				}
			}
		}
		if match {
			matches = append(matches, td)
		}
	}

	if len(matches) == 1 {
		return matches[0], true
	}
	return types.TypeDefinition{}, false
}
