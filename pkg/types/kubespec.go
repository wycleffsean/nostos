package types

import "fmt"

func setFieldSince(td *TypeDefinition, version string, since map[string]map[string]map[string]string) {
	if since[td.Group] == nil {
		since[td.Group] = map[string]map[string]string{}
	}
	if since[td.Group][td.Kind] == nil {
		since[td.Group][td.Kind] = map[string]string{}
	}
	for i, f := range td.Fields {
		path := f.Name
		if _, ok := since[td.Group][td.Kind][path]; !ok {
			since[td.Group][td.Kind][path] = version
		}
		td.Fields[i].Since = since[td.Group][td.Kind][path]
		for j, sf := range f.SubFields {
			spath := f.Name + "." + sf.Name
			if _, ok := since[td.Group][td.Kind][spath]; !ok {
				since[td.Group][td.Kind][spath] = version
			}
			td.Fields[i].SubFields[j].Since = since[td.Group][td.Kind][spath]
		}
	}
}

func extractDefinitionsLocal(specMap map[string]interface{}) (map[string]interface{}, bool) {
	components, ok := specMap["definitions"].(map[string]interface{})
	if ok {
		return components, true
	}
	if comp, ok := specMap["components"].(map[string]interface{}); ok {
		if schemas, ok := comp["schemas"].(map[string]interface{}); ok {
			return schemas, true
		}
	}
	return nil, false
}

func convertSchemaToTypeDefLocal(group, version, kind, scope string, schemaObj map[string]interface{}) TypeDefinition {
	td := TypeDefinition{
		Group:       group,
		Version:     version,
		Kind:        kind,
		Scope:       scope,
		Description: getStringFieldLocal(schemaObj, "description"),
		Fields:      []FieldDefinition{},
	}
	properties, ok := schemaObj["properties"].(map[string]interface{})
	if !ok {
		return td
	}
	for propName, propVal := range properties {
		propSchema, ok := propVal.(map[string]interface{})
		if !ok {
			continue
		}
		fieldDef := FieldDefinition{Name: propName}
		fieldType := getStringFieldLocal(propSchema, "type")
		if fieldType == "" {
			if ref, ok := propSchema["$ref"].(string); ok {
				fieldType = deriveRefTypeNameLocal(ref)
			}
		}
		if fieldType == "object" {
			subFields := []FieldDefinition{}
			if subProps, ok := propSchema["properties"].(map[string]interface{}); ok {
				for subName, subVal := range subProps {
					subSchema, _ := subVal.(map[string]interface{})
					if subSchema == nil {
						continue
					}
					subFieldType := getStringFieldLocal(subSchema, "type")
					if subFieldType == "" {
						if ref, ok := subSchema["$ref"].(string); ok {
							subFieldType = deriveRefTypeNameLocal(ref)
						}
					}
					subFields = append(subFields, FieldDefinition{Name: subName, Type: subFieldType})
				}
			}
			fieldDef.Type = "object"
			fieldDef.Description = getStringFieldLocal(propSchema, "description")
			fieldDef.SubFields = subFields
		} else if fieldType == "array" {
			elemType := "any"
			if items, ok := propSchema["items"].(map[string]interface{}); ok {
				elemType = getStringFieldLocal(items, "type")
				if elemType == "" {
					if ref, ok := items["$ref"].(string); ok {
						elemType = deriveRefTypeNameLocal(ref)
					}
				}
				if elemType == "" {
					elemType = "object"
				}
			}
			fieldDef.Type = "[]" + elemType
		} else if fieldType != "" {
			fieldDef.Type = fieldType
			fieldDef.Description = getStringFieldLocal(propSchema, "description")
		} else {
			fieldDef.Type = "any"
			fieldDef.Description = getStringFieldLocal(propSchema, "description")
		}
		td.Fields = append(td.Fields, fieldDef)
	}
	return td
}

func getStringFieldLocal(m map[string]interface{}, field string) string {
	if val, ok := m[field]; ok {
		return fmt.Sprintf("%v", val)
	}
	return ""
}

func deriveRefTypeNameLocal(ref string) string {
	idx := lastIndexLocal(ref, '/')
	fullName := ref
	if idx != -1 {
		fullName = ref[idx+1:]
	}
	if dotIdx := lastIndexLocal(fullName, '.'); dotIdx != -1 {
		return fullName[dotIdx+1:]
	}
	return fullName
}

func lastIndexLocal(s string, sep byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep {
			return i
		}
	}
	return -1
}
