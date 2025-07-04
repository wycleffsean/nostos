package vm

import (
	"fmt"
	"os"
	"path/filepath"

	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/lang"
)

type builtinFunc func(*VM, ...interface{}) error

var builtins map[string]builtinFunc

func init() {
	builtins = map[string]builtinFunc{
		"import": builtinImport,
	}
}

func builtinImport(v *VM, args ...interface{}) error {
	if len(args) != 1 {
		return fmt.Errorf("import expects 1 argument, got %d", len(args))
	}
	path, ok := args[0].(string)
	if !ok {
		return fmt.Errorf("import expects a path argument")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(v.baseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, items := lang.NewStringLexer(string(data))
	p := lang.NewParser(items, uri.File(path))
	ast := p.Parse()
	if perrs := lang.CollectParseErrors(ast); len(perrs) > 0 {
		return perrs[0]
	}
	res, err := EvalWithDir(ast, filepath.Dir(path), uri.File(path))
	if err != nil {
		return err
	}
	v.push(res)
	return nil
}
