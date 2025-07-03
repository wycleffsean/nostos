package planner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/vm"
)

func evalExample(name string) (map[string]interface{}, error) {
	path := filepath.Join("..", "..", "examples", name)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_, items := lang.NewStringLexer(string(data))
	p := lang.NewParser(items)
	ast := p.Parse()
	val, err := vm.EvalWithDir(ast, filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	obj, ok := val.(map[string]interface{})
	if !ok {
		return nil, err
	}
	return obj, nil
}

func TestEvaluateOdyssey(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "odyssey.no")
	entries, err := EvaluateOdyssey(path)
	if err != nil {
		t.Fatalf("eval odyssey: %v", err)
	}

	svc, err := evalExample("redis-service.no")
	if err != nil {
		t.Fatalf("eval service: %v", err)
	}
	dep, err := evalExample("redis-deployment.no")
	if err != nil {
		t.Fatalf("eval deployment: %v", err)
	}

	want := map[string]odysseyEntry{
		"do-nyc1-k8s-1-33-1-do-0-nyc1-1750371119772": {
			"default": []interface{}{svc, dep},
		},
	}

	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("unexpected result: %#v", entries)
	}
}
