package types

import "testing"

func TestLastIndexLocal(t *testing.T) {
	if idx := lastIndexLocal("a/b/c", '/'); idx != 3 {
		t.Fatalf("got %d", idx)
	}
	if idx := lastIndexLocal("abc", '/'); idx != -1 {
		t.Fatalf("expected -1 got %d", idx)
	}
}

func TestDeriveRefTypeNameLocal(t *testing.T) {
	if n := deriveRefTypeNameLocal("#/definitions/io.k8s.api.core.v1.Pod"); n != "Pod" {
		t.Fatalf("unexpected %s", n)
	}
	if n := deriveRefTypeNameLocal("io.k8s.api.core.v1.Pod"); n != "Pod" {
		t.Fatalf("unexpected %s", n)
	}
	if n := deriveRefTypeNameLocal("pkg.Type"); n != "Type" {
		t.Fatalf("unexpected %s", n)
	}
}

func TestGetStringFieldLocal(t *testing.T) {
	m := map[string]interface{}{"a": 1, "b": true, "c": "x"}
	if getStringFieldLocal(m, "a") != "1" {
		t.Fatalf("int conversion failed")
	}
	if getStringFieldLocal(m, "b") != "true" {
		t.Fatalf("bool conversion failed")
	}
	if getStringFieldLocal(m, "c") != "x" {
		t.Fatalf("string conversion failed")
	}
	if getStringFieldLocal(m, "d") != "" {
		t.Fatalf("missing field")
	}
}

func TestConvertSchemaToTypeDefLocal(t *testing.T) {
	schema := map[string]interface{}{
		"description": "example",
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{"type": "string", "description": "f"},
			"bar": map[string]interface{}{
				"type":        "object",
				"description": "b",
				"properties": map[string]interface{}{
					"baz": map[string]interface{}{"type": "integer"},
				},
				"additionalProperties": map[string]interface{}{},
			},
			"items": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "boolean"},
			},
			"ref": map[string]interface{}{
				"$ref": "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta",
			},
		},
		"additionalProperties": true,
	}
	td := convertSchemaToTypeDefLocal("apps", "v1", "Demo", "Namespaced", schema)
	if td.Group != "apps" || td.Version != "v1" || td.Kind != "Demo" || td.Scope != "Namespaced" {
		t.Fatalf("metadata mismatch: %+v", td)
	}
	if !td.Open {
		t.Fatalf("expected open object")
	}
	if td.Fields["foo"].Description != "f" || td.Fields["foo"].Type.Name() != "string" {
		t.Fatalf("foo field incorrect")
	}
	bar, ok := td.Fields["bar"].Type.(*ObjectType)
	if !ok || !bar.Open || bar.Fields["baz"].Type.Name() != "integer" {
		t.Fatalf("bar field incorrect: %+v", td.Fields["bar"])
	}
	if td.Fields["items"].Type.(*ListType).Elem.Name() != "boolean" {
		t.Fatalf("items element type")
	}
	if td.Fields["ref"].Type.Name() != "ObjectMeta" {
		t.Fatalf("ref type name %s", td.Fields["ref"].Type.Name())
	}
}
