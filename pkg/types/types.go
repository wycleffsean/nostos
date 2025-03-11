package types

// FieldType represents a simple enumeration for common field types.
type FieldType string

const (
	FieldTypeString  FieldType = "string"
	FieldTypeNumber  FieldType = "number"
	FieldTypeBool    FieldType = "boolean"
	FieldTypeObject  FieldType = "object"
	FieldTypeArray   FieldType = "array"
	FieldTypeUnknown FieldType = "unknown"
)

// FieldDefinition represents a single field in a type.
type FieldDefinition struct {
	Name       string
	Type       FieldType
	Properties map[string]*FieldDefinition // only for object types
}

// TypeDefinition represents the schema of a resource specification.
type TypeDefinition struct {
	APIVersion string
	Kind       string
	Fields     map[string]*FieldDefinition
}
