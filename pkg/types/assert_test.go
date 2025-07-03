package types

import "testing"

func TestAssertPrimitive(t *testing.T) {
	if err := Assert("foo", &PrimitiveType{"string"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Assert(1, &PrimitiveType{"string"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAssertList(t *testing.T) {
	lt := &ListType{Elem: &PrimitiveType{"number"}}
	if err := Assert([]interface{}{1, 2.5}, lt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Assert("notlist", lt); err == nil {
		t.Fatalf("expected list error")
	}
	if err := Assert([]interface{}{1, "a"}, lt); err == nil {
		t.Fatalf("expected element error")
	}
}

func TestAssertObject(t *testing.T) {
	obj := &ObjectType{Fields: map[string]*Field{
		"name": {Name: "name", Type: &PrimitiveType{"string"}, Required: true},
		"age":  {Name: "age", Type: &PrimitiveType{"number"}},
	}}
	if err := Assert(map[string]interface{}{"name": "bob", "age": 30}, obj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Assert(map[string]interface{}{"age": 30}, obj); err == nil {
		t.Fatalf("expected missing field error")
	}
	if err := Assert(map[string]interface{}{"name": 1}, obj); err == nil {
		t.Fatalf("expected field type error")
	}
	if err := Assert(map[string]interface{}{"name": "bob", "extra": true}, obj); err == nil {
		t.Fatalf("expected unexpected field error")
	}
}

func TestAssertObjectOpen(t *testing.T) {
	obj := &ObjectType{Fields: map[string]*Field{
		"name": {Name: "name", Type: &PrimitiveType{"string"}},
	}, Open: true}
	if err := Assert(map[string]interface{}{"name": "bob", "extra": true}, obj); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertNested(t *testing.T) {
	inner := &ObjectType{Fields: map[string]*Field{
		"bar": {Name: "bar", Type: &PrimitiveType{"string"}, Required: true},
	}}
	outer := &ObjectType{Fields: map[string]*Field{
		"foo": {Name: "foo", Type: inner},
	}}
	if err := Assert(map[string]interface{}{"foo": map[string]interface{}{"bar": "x"}}, outer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := Assert(map[string]interface{}{"foo": map[string]interface{}{}}, outer); err == nil {
		t.Fatalf("expected nested error")
	}
}

func TestAssertFunction(t *testing.T) {
	if err := Assert(nil, &FunctionType{}); err == nil {
		t.Fatalf("expected function error")
	}
}
