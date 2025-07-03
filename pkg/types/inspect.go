package types

import (
	"fmt"
	"sort"
	"strings"
)

// InspectValue returns a YAML-like representation of v.
func InspectValue(v interface{}) string {
	return inspectValue(v, 0)
}

func inspectValue(v interface{}, indent int) string {
	indentStr := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			return indentStr + "{}\n"
		}
		var sb strings.Builder
		for _, k := range keys {
			sb.WriteString(indentStr)
			sb.WriteString(k)
			sb.WriteString(": ")
			child := val[k]
			switch child.(type) {
			case map[string]interface{}, []interface{}:
				sb.WriteString("\n")
				sb.WriteString(inspectValue(child, indent+1))
			default:
				sb.WriteString(formatScalar(child))
				sb.WriteString("\n")
			}
		}
		return sb.String()
	case []interface{}:
		if len(val) == 0 {
			return indentStr + "[]\n"
		}
		var sb strings.Builder
		for _, item := range val {
			sb.WriteString(indentStr)
			sb.WriteString("- ")
			switch item.(type) {
			case map[string]interface{}, []interface{}:
				sb.WriteString("\n")
				sb.WriteString(inspectValue(item, indent+1))
			default:
				sb.WriteString(formatScalar(item))
				sb.WriteString("\n")
			}
		}
		return sb.String()
	default:
		return indentStr + formatScalar(val) + "\n"
	}
}

func formatScalar(v interface{}) string {
	switch s := v.(type) {
	case string:
		return fmt.Sprintf("%q", s)
	case nil:
		return "null"
	default:
		return fmt.Sprint(s)
	}
}
