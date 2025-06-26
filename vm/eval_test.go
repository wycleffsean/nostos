package vm

import (
	"reflect"
	"testing"

	"github.com/wycleffsean/nostos/lang"
)

func parse(input string) interface{} {
	_, items := lang.NewStringLexer(input)
	p := lang.NewParser(items)
	return p.Parse()
}

func TestEvalSimpleMap(t *testing.T) {
	ast := parse("foo: 1\nbar: \"example\"")
	result, err := Eval(ast)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	wanted := map[string]interface{}{"foo": float64(1), "bar": "example"}
	if !reflect.DeepEqual(result, wanted) {
		t.Fatalf("expected %#v got %#v", wanted, result)
	}
}
