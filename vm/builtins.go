package vm

import (
	"fmt"
	"os"
	"path/filepath"

	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/urispec"
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
	var spec urispec.Spec
	switch v := args[0].(type) {
	case urispec.Spec:
		spec = v
	case string:
		spec = urispec.Parse(v)
	default:
		return fmt.Errorf("import expects a path argument")
	}
	var path string
	if spec.Type == "path" {
		path = spec.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(v.baseDir, path)
		}
	} else {
		var err error
		path, err = spec.LocalPath()
		if err != nil {
			return err
		}
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		path = filepath.Join(path, "odyssey.no")
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
