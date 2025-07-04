package vm

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/lang"
)

type VM struct {
	stack   []interface{}
	baseDir string
	uri     uri.URI
}

// EvalError represents runtime errors produced during evaluation. It implements
// lang.NostosError so callers can inspect file position and stack traces.
type EvalError struct {
	File     uri.URI
	Position lang.Position
	Msg      string
	Stack    []string
}

func (e *EvalError) Error() string        { return e.Msg }
func (e *EvalError) URI() uri.URI         { return e.File }
func (e *EvalError) Pos() lang.Position   { return e.Position }
func (e *EvalError) StackTrace() []string { return e.Stack }

func newVM(dir string, u uri.URI) *VM {
	return &VM{stack: make([]interface{}, 0), baseDir: dir, uri: u}
}

func New() *VM { return newVM(".", uri.URI("")) }

func (v *VM) wrapError(n interface{}, err error) error {
	if _, ok := err.(lang.NostosError); ok {
		return err
	}
	pos := lang.Position{}
	if p, ok := n.(interface{ Pos() lang.Position }); ok {
		pos = p.Pos()
	}
	return &EvalError{
		File:     v.uri,
		Position: pos,
		Msg:      err.Error(),
		Stack:    strings.Split(string(debug.Stack()), "\n"),
	}
}

func (v *VM) push(x interface{}) { v.stack = append(v.stack, x) }

func (v *VM) pop() interface{} {
	if len(v.stack) == 0 {
		return nil
	}
	x := v.stack[len(v.stack)-1]
	v.stack = v.stack[:len(v.stack)-1]
	return x
}

func (v *VM) peek() interface{} {
	if len(v.stack) == 0 {
		return nil
	}
	return v.stack[len(v.stack)-1]
}

// Stack operations
func (v *VM) createMap()         { v.push(make(map[string]interface{})) }
func (v *VM) pushKey(key string) { v.push(key) }
func (v *VM) pushValueToMap() {
	val := v.pop()
	key := v.pop().(string)
	m := v.peek().(map[string]interface{})
	m[key] = val
}
func (v *VM) createList() { v.push(make([]interface{}, 0)) }
func (v *VM) appendItem() {
	val := v.pop()
	list := v.peek().([]interface{})
	list = append(list, val)
	v.stack[len(v.stack)-1] = list
}

func Eval(n interface{}) (interface{}, error) {
	return EvalWithDir(n, ".", uri.URI(""))
}

func EvalWithDir(n interface{}, dir string, u uri.URI) (interface{}, error) {
	vm := newVM(dir, u)
	if err := vm.evalNode(n); err != nil {
		return nil, err
	}
	return vm.pop(), nil
}

func (v *VM) evalNode(n interface{}) error {
	switch node := n.(type) {
	case *lang.String:
		v.push(node.Text)
	case *lang.Path:
		v.push(node.Text)
	case *lang.Number:
		v.push(node.Value)
	case *lang.Symbol:
		v.push(node.Text)
	case *lang.List:
		v.createList()
		for _, item := range *node {
			if err := v.evalNode(item); err != nil {
				return err
			}
			v.appendItem()
		}
	case *lang.Map:
		v.createMap()
		for k, val := range *node {
			v.pushKey(k.Text)
			if err := v.evalNode(val); err != nil {
				return err
			}
			v.pushValueToMap()
		}
	case *lang.Function:
		return v.wrapError(node, fmt.Errorf("functions not supported in evaluation"))
	case *lang.Call:
		if err := v.evalNode(node.Func); err != nil {
			return err
		}
		fn := v.pop()
		if err := v.evalNode(node.Arg); err != nil {
			return err
		}
		arg := v.pop()
		name, ok := fn.(string)
		if !ok {
			return v.wrapError(node, fmt.Errorf("function name must be a symbol"))
		}
		builtin, ok := builtins[name]
		if !ok {
			return v.wrapError(node, fmt.Errorf("unknown builtin %s", name))
		}
		if err := builtin(v, arg); err != nil {
			return v.wrapError(node, err)
		}
		return nil
	case *lang.Shovel:
		return v.wrapError(node, fmt.Errorf("shovel operator not supported in evaluation"))
	case *lang.ParseError:
		return errors.New(node.Error())
	default:
		return v.wrapError(node, fmt.Errorf("unknown node type %T", node))
	}
	return nil
}
