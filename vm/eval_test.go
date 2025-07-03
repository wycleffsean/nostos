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
	result, err := EvalWithDir(ast, ".")
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	wanted := map[string]interface{}{"foo": float64(1), "bar": "example"}
	if !reflect.DeepEqual(result, wanted) {
		t.Fatalf("expected %#v got %#v", wanted, result)
	}
}

func TestEvalOdysseyExample(t *testing.T) {
	path := filepath.Join("..", "examples", "odyssey.no")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example: %v", err)
	}

	_, items := lang.NewStringLexer(string(data))
	p := lang.NewParser(items)
	ast := p.Parse()

	got, err := EvalWithDir(ast, filepath.Dir(path))
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}

	svcPath := filepath.Join("..", "examples", "redis-service.no")
	svcData, err := os.ReadFile(svcPath)
	if err != nil {
		t.Fatalf("read service: %v", err)
	}
	_, items = lang.NewStringLexer(string(svcData))
	p = lang.NewParser(items)
	svcAST := p.Parse()
	svcObj, err := EvalWithDir(svcAST, filepath.Dir(svcPath))
	if err != nil {
		t.Fatalf("eval service: %v", err)
	}

	depPath := filepath.Join("..", "examples", "redis-deployment.no")
	depData, err := os.ReadFile(depPath)
	if err != nil {
		t.Fatalf("read deployment: %v", err)
	}
	_, items = lang.NewStringLexer(string(depData))
	p = lang.NewParser(items)
	depAST := p.Parse()
	depObj, err := EvalWithDir(depAST, filepath.Dir(depPath))
	if err != nil {
		t.Fatalf("eval deployment: %v", err)
	}

	want := map[string]interface{}{
		"do-nyc1-k8s-1-33-1-do-0-nyc1-1750371119772": map[string]interface{}{
			"default": []interface{}{svcObj, depObj},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected eval result: %#v", got)
	}
}
