package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuiltinImport(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "example.no")
	content := "1"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	expr := fmt.Sprintf("foo: import(%s)", file)
	ast := parse(expr)
	result, err := EvalWithDir(ast, ".")
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	wanted := map[string]interface{}{"foo": float64(1)}
	if !reflect.DeepEqual(result, wanted) {
		t.Fatalf("expected %#v got %#v", wanted, result)
	}
}
