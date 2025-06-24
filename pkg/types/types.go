package types

// TypeDefinition represents a Kubernetes or user-defined API type in a simplified schema form.
// This struct is independent of Kubernetes library types to maintain decoupling from Kubernetes packages.
type TypeDefinition struct {
	Group       string            // API group of the type (empty string for core or user-defined types)
	Version     string            // API version of the type
	Kind        string            // Kind name of the type
	Scope       string            // Scope of the resource: "Namespaced" or "Cluster"
	Fields      []FieldDefinition // Top-level fields of this type (schema of the object)
	Description string
}

// FieldDefinition describes a field (property) in a TypeDefinition.
// If the field is a complex object, SubFields may contain one level of nested fields for introspection.
type FieldDefinition struct {
	Name        string // Name of the field
	Type        string // Data type of the field (e.g., "string", "int", "object", "[]<type>" for arrays)
	Description string
	Required    bool              // Indicates if the field must appear on the object
	Since       string            // Kubernetes version when this field was introduced (empty if unknown)
	SubFields   []FieldDefinition // Nested fields if this field is an object (one level deep)
}

// FieldType represents a simple enumeration for common field types.
// type FieldType string

// const (
// 	FieldTypeString  FieldType = "string"
// 	FieldTypeNumber  FieldType = "number"
// 	FieldTypeBool    FieldType = "boolean"
// 	FieldTypeObject  FieldType = "object"
// 	FieldTypeArray   FieldType = "array"
// 	FieldTypeUnknown FieldType = "unknown"
// )

// // FieldDefinition represents a single field in a type.
// type FieldDefinition struct {
// 	Name       string
// 	Type       FieldType
// 	Properties map[string]*FieldDefinition // only for object types
// }

// // TypeDefinition represents the schema of a resource specification.
// type TypeDefinition struct {
// 	APIVersion string
// 	Kind       string
// 	Fields     map[string]*FieldDefinition
// }
