package types

import "fmt"

func setFieldSince(td *TypeDefinition, version string, since map[string]map[string]map[string]string) {
	if since[td.Group] == nil {
		since[td.Group] = map[string]map[string]string{}
	}
	if since[td.Group][td.Kind] == nil {
		since[td.Group][td.Kind] = map[string]string{}
	}
	for name, f := range td.Fields {
		path := name
		if _, ok := since[td.Group][td.Kind][path]; !ok {
			since[td.Group][td.Kind][path] = version
		}
		f.Since = since[td.Group][td.Kind][path]
		if obj, ok := f.Type.(*ObjectType); ok {
			for subName, sf := range obj.Fields {
				spath := f.Name + "." + subName
				if _, ok := since[td.Group][td.Kind][spath]; !ok {
					since[td.Group][td.Kind][spath] = version
				}
				sf.Since = since[td.Group][td.Kind][spath]
			}
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

func convertSchemaToTypeDefLocal(group, version, kind, scope string, schemaObj map[string]interface{}) ObjectType {
	td := ObjectType{
		Group:       group,
		Version:     version,
		Kind:        kind,
		Scope:       scope,
		Description: getStringFieldLocal(schemaObj, "description"),
		Fields:      map[string]*Field{},
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
		fieldDef := &Field{Name: propName}
		fieldType := getStringFieldLocal(propSchema, "type")
		if fieldType == "" {
			if ref, ok := propSchema["$ref"].(string); ok {
				fieldType = deriveRefTypeNameLocal(ref)
			}
		}
		if fieldType == "object" {
			subFields := map[string]*Field{}
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
					subFields[subName] = &Field{Name: subName, Type: &PrimitiveType{N: subFieldType}}
				}
			}
			fieldDef.Type = &ObjectType{Fields: subFields, Open: propSchema["additionalProperties"] != nil}
			fieldDef.Description = getStringFieldLocal(propSchema, "description")
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
			fieldDef.Type = &ListType{Elem: &PrimitiveType{N: elemType}}
		} else if fieldType != "" {
			fieldDef.Type = &PrimitiveType{N: fieldType}
			fieldDef.Description = getStringFieldLocal(propSchema, "description")
		} else {
			fieldDef.Type = &PrimitiveType{N: "any"}
			fieldDef.Description = getStringFieldLocal(propSchema, "description")
		}
		td.Fields[propName] = fieldDef
	}
	if addProps, ok := schemaObj["additionalProperties"]; ok && addProps != nil {
		td.Open = true
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
