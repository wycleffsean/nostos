package types

// ---------------------------------------------------------------------------
// Generic type system
// ---------------------------------------------------------------------------

// Type is the common interface implemented by all types in the system.
// It exposes a Name method which should return a human readable name for the
// type and an Inspect method which renders a value of this type in a
// YAML-like format.
type Type interface {
	Name() string
	Inspect(v interface{}) string
}

// PrimitiveType represents a built in primitive such as string, number or bool.
type PrimitiveType struct{ N string }

func (p *PrimitiveType) Name() string                 { return p.N }
func (p *PrimitiveType) Inspect(v interface{}) string { return InspectValue(v) }

// ListType represents a list of another type.
type ListType struct{ Elem Type }

func (l *ListType) Name() string                 { return "[]" + l.Elem.Name() }
func (l *ListType) Inspect(v interface{}) string { return InspectValue(v) }

// Field describes an object field.  The Type field references another Type in
// the system which may itself be an ObjectType, ListType etc.
type Field struct {
	Name        string
	Type        Type
	Description string
	Required    bool
	Since       string
}

// ObjectType represents a Kubernetes (or user defined) resource.  It contains
// the standard group/version/kind metadata as well as structural information
// about its fields.  When Open is true additional unknown fields are allowed
// when validating values against this type.
type ObjectType struct {
	Group       string
	Version     string
	Kind        string
	Scope       string
	Description string
	Fields      map[string]*Field
	Open        bool
}

func (o *ObjectType) Name() string                 { return o.Kind }
func (o *ObjectType) Inspect(v interface{}) string { return InspectValue(v) }

// FunctionType represents a function that takes a list of parameter types and
// returns another type.
type FunctionType struct {
	Params []Type
	Result Type
}

func (f *FunctionType) Name() string                 { return "func" }
func (f *FunctionType) Inspect(v interface{}) string { return "<function>" }

// Extend merges fields from the provided ObjectType into the receiver.  Fields
// with the same name must be compatible – currently this means they have the
// same type name.  Required flags are ORed together.  The receiver's Open flag
// becomes true if either object is open.
func (o *ObjectType) Extend(other *ObjectType) {
	if o.Fields == nil {
		o.Fields = make(map[string]*Field)
	}
	for name, f := range other.Fields {
		if existing, ok := o.Fields[name]; ok {
			if existing.Type.Name() != f.Type.Name() {
				// Incompatible, prefer existing - real
				// implementation could return error, but for
				// now keep the original field.
				continue
			}
			if f.Required {
				existing.Required = true
			}
		} else {
			o.Fields[name] = f
		}
	}
	if other.Open {
		o.Open = true
	}
}

// ---------------------------------------------------------------------------
// Backwards compatibility aliases
// ---------------------------------------------------------------------------

// For now keep the old names used throughout the codebase.  They are simple
// type aliases to the new structures so existing callers continue to compile
// while we migrate the code.

type FieldDefinition = Field
type TypeDefinition = ObjectType

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
