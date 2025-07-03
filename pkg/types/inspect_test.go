package types

import "testing"

func TestInspectValue(t *testing.T) {
	obj := map[string]interface{}{"foo": "bar", "num": 1.0}
	got := InspectValue(obj)
	expected1 := "foo: \"bar\"\nnum: 1\n"
	// map order not guaranteed; check both possibilities
	if got != expected1 && got != "num: 1\nfoo: \"bar\"\n" {
		t.Fatalf("unexpected inspect output:\n%s", got)
	}
}
