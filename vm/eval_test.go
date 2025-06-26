package vm

import (
	"os"
	"path/filepath"
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

func TestEvalOdysseyExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "examples", "odyssey.no"))
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	_, items := lang.NewStringLexer(string(data))
	p := lang.NewParser(items)
	ast := p.Parse()

	got, err := Eval(ast)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	want := map[string]interface{}{
		"do-nyc1-k8s-1-33-1-do-0-nyc1-1750371119772": map[string]interface{}{
			"default": []interface{}{"./redis-service.no", "./redis-deployment.no"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected eval result: %#v", got)
	}
}
