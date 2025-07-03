package types

import (
	"fmt"
	"reflect"
)

// Assert validates that the provided value conforms to the given Type.  It
// returns an error describing the mismatch, or nil if the value is valid.
func Assert(val interface{}, t Type) error {
	switch tt := t.(type) {
	case *PrimitiveType:
		if !assertPrimitive(val, tt.N) {
			return fmt.Errorf("expected %s", tt.N)
		}
		return nil
	case *ListType:
		rv := reflect.ValueOf(val)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return fmt.Errorf("expected list")
		}
		for i := 0; i < rv.Len(); i++ {
			if err := Assert(rv.Index(i).Interface(), tt.Elem); err != nil {
				return fmt.Errorf("element %d: %w", i, err)
			}
		}
		return nil
	case *ObjectType:
		m, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected object")
		}
		for name, field := range tt.Fields {
			v, exists := m[name]
			if !exists {
				if field.Required {
					return fmt.Errorf("missing field %s", name)
				}
				continue
			}
			if err := Assert(v, field.Type); err != nil {
				return fmt.Errorf("field %s: %w", name, err)
			}
		}
		if !tt.Open {
			for name := range m {
				if _, ok := tt.Fields[name]; !ok {
					return fmt.Errorf("unexpected field %s", name)
				}
			}
		}
		return nil
	case *FunctionType:
		// Functions are not currently assertable
		return fmt.Errorf("cannot assert function types")
	default:
		return fmt.Errorf("unknown type")
	}
}

func assertPrimitive(val interface{}, name string) bool {
	switch name {
	case "string":
		_, ok := val.(string)
		return ok
	case "number":
		switch val.(type) {
		case int, int64, float64, float32, uint, uint64, int32, uint32:
			return true
		default:
			return false
		}
	case "bool", "boolean":
		_, ok := val.(bool)
		return ok
	default:
		// unknown primitive, accept any
		return true
	}
}
