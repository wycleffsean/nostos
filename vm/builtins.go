package vm

import (
	"fmt"
	"os"

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
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, items := lang.NewStringLexer(string(data))
	p := lang.NewParser(items)
	ast := p.Parse()
	res, err := Eval(ast)
	if err != nil {
		return err
	}
	v.push(res)
	return nil
}
