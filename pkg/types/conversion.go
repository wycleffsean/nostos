package types

// import (
// 	"fmt"
// 	"strings"

// 	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
// )

// // -----------------------------------------------------------------------------
// // Conversion for CRDs (using apiextensionsv1.JSONSchemaProps)
// // -----------------------------------------------------------------------------

// // ConvertJSONSchemaPropsToTypeDefinition converts a CRD's JSONSchemaProps (OpenAPI v3)
// // into a TypeDefinition.
// func ConvertJSONSchemaPropsToTypeDefinition(apiVersion, kind string, schema *apiextensionsv1.JSONSchemaProps) *TypeDefinition {
// 	if schema == nil {
// 		return nil
// 	}

// 	td := &TypeDefinition{
// 		APIVersion: apiVersion,
// 		Kind:       kind,
// 		Fields:     make(map[string]*FieldDefinition),
// 	}

// 	if schema.Properties == nil {
// 		return td
// 	}

// 	for propName, propValue := range schema.Properties {
// 		fieldType := inferFieldTypeFromSchema(propValue)
// 		fd := &FieldDefinition{
// 			Name: propName,
// 			Type: fieldType,
// 		}

// 		if fieldType == FieldTypeObject {
// 			fd.Properties = convertPropertiesFromSchema(propValue.Properties)
// 		}
// 		td.Fields[propName] = fd
// 	}

// 	return td
// }

// // Convert OpenAPI spec to TypeDefinitions
// func ConvertOpenAPISpecToTypeDefinitions(spec map[string]interface{}) (map[string]*TypeDefinition, error) {
// 	result := make(map[string]*TypeDefinition)
// 	var defs map[string]interface{}

// 	if d, ok := spec["definitions"].(map[string]interface{}); ok {
// 		defs = d
// 	} else if comp, ok := spec["components"].(map[string]interface{}); ok {
// 		if s, ok := comp["schemas"].(map[string]interface{}); ok {
// 			defs = s
// 		}
// 	}

// 	if defs == nil {
// 		return nil, fmt.Errorf("OpenAPI spec does not contain 'definitions' or 'components.schemas'")
// 	}

// 	// for key := range defs["additional_properties"].(map[string]interface{}) {
//     		// fmt.Println(defs)
// 	// }

//     	fmt.Printf("Total Definitions Found: %d\n", len(defs))
// 	for defName, defValue := range defs {
// 		defMap, ok := defValue.(map[string]interface{})
// 		if !ok {
//                 	// fmt.Printf("casting in %s Properties: %+v\n", defName, props)
//                 	fmt.Printf("xcasting in %s failed\n", defName)
// 			continue
// 		}

// 		props, ok := defMap["properties"].(map[string]interface{})

// 		if !ok {
// 			continue
// 		}

// 		td := &TypeDefinition{
// 			APIVersion: "built-in",
// 			Kind:       simpleKind(defName),
// 			Fields:     convertPropertiesFromMap(props),
// 		}
// 		result[defName] = td
// 	}
// 	return result, nil
// }

// func convertPropertiesFromSchema(props map[string]apiextensionsv1.JSONSchemaProps) map[string]*FieldDefinition {
// 	fields := make(map[string]*FieldDefinition)
// 	for propName, propValue := range props {
// 		fd := &FieldDefinition{
// 			Name: propName,
// 			Type: inferFieldTypeFromSchema(propValue),
// 		}
// 		if fd.Type == FieldTypeObject {
// 			fd.Properties = convertPropertiesFromSchema(propValue.Properties)
// 		}
// 		fields[propName] = fd
// 	}
// 	return fields
// }

// func convertPropertiesFromMap(props map[string]interface{}) map[string]*FieldDefinition {
// 	fields := make(map[string]*FieldDefinition)
// 	for propName, propValue := range props {
// 		propMap, ok := propValue.(map[string]interface{})
// 		if !ok {
// 			continue
// 		}
// 		fd := &FieldDefinition{
// 			Name: propName,
// 			Type: inferFieldTypeFromMap(propMap),
// 		}
// 		if fd.Type == FieldTypeObject {
// 			if subProps, ok := propMap["properties"].(map[string]interface{}); ok {
// 				fd.Properties = convertPropertiesFromMap(subProps)
// 			}
// 		}
// 		fields[propName] = fd
// 	}
// 	return fields
// }

// func inferFieldTypeFromSchema(prop apiextensionsv1.JSONSchemaProps) FieldType {
// 	if prop.Type != "" {
// 		switch strings.ToLower(prop.Type) {
// 		case "string":
// 			return FieldTypeString
// 		case "integer", "number":
// 			return FieldTypeNumber
// 		case "boolean":
// 			return FieldTypeBool
// 		case "object":
// 			return FieldTypeObject
// 		case "array":
// 			return FieldTypeArray
// 		default:
// 			return FieldTypeUnknown
// 		}
// 	}
// 	return FieldTypeUnknown
// }

// func inferFieldTypeFromMap(propMap map[string]interface{}) FieldType {
// 	if t, ok := propMap["type"].(string); ok {
// 		switch strings.ToLower(t) {
// 		case "string":
// 			return FieldTypeString
// 		case "integer", "number":
// 			return FieldTypeNumber
// 		case "boolean":
// 			return FieldTypeBool
// 		case "object":
// 			return FieldTypeObject
// 		case "array":
// 			return FieldTypeArray
// 		default:
// 			return FieldTypeUnknown
// 		}
// 	}
// 	return FieldTypeUnknown
// }

// func simpleKind(defName string) string {
// 	parts := strings.Split(defName, ".")
// 	if len(parts) > 0 {
// 		return parts[len(parts)-1]
// 	}
// 	return defName
// }
