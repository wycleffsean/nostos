package types

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"strings"
)

// ConvertJSONSchemaPropsToTypeDefinition converts a CRD's JSON schema into a TypeDefinition.
// This is a basic example – you can expand it to handle nested objects and arrays recursively.
func ConvertJSONSchemaPropsToTypeDefinition(apiVersion, kind string, schema *apiextensionsv1.JSONSchemaProps) *TypeDefinition {
	td := &TypeDefinition{
		APIVersion: apiVersion,
		Kind:       kind,
		Fields:     make(map[string]*FieldDefinition),
	}
	// Iterate over the schema properties.
	for propName, propSchema := range schema.Properties {
		fieldType := FieldTypeUnknown
		switch strings.ToLower(propSchema.Type) {
		case "string":
			fieldType = FieldTypeString
		case "integer", "number":
			fieldType = FieldTypeNumber
		case "boolean":
			fieldType = FieldTypeBool
		case "object":
			fieldType = FieldTypeObject
		case "array":
			fieldType = FieldTypeArray
		}
		fd := &FieldDefinition{
			Name: propName,
			Type: fieldType,
		}
		// If it's an object, you might want to recursively process its properties.
		if fieldType == FieldTypeObject && propSchema.Properties != nil {
			fd.Properties = convertProperties(propSchema.Properties)
		}
		td.Fields[propName] = fd
	}
	return td
}

// convertProperties converts a map of JSONSchemaProps to FieldDefinitions.
func convertProperties(props map[string]apiextensionsv1.JSONSchemaProps) map[string]*FieldDefinition {
	fields := make(map[string]*FieldDefinition)
	for propName, propSchema := range props {
		fieldType := FieldTypeUnknown
		switch strings.ToLower(propSchema.Type) {
		case "string":
			fieldType = FieldTypeString
		case "integer", "number":
			fieldType = FieldTypeNumber
		case "boolean":
			fieldType = FieldTypeBool
		case "object":
			fieldType = FieldTypeObject
		case "array":
			fieldType = FieldTypeArray
		}
		fd := &FieldDefinition{
			Name: propName,
			Type: fieldType,
		}
		// Recurse if needed.
		if fieldType == FieldTypeObject && propSchema.Properties != nil {
			fd.Properties = convertProperties(propSchema.Properties)
		}
		fields[propName] = fd
	}
	return fields
}
