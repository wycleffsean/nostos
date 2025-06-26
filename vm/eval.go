package vm

import (
	"fmt"

	"github.com/wycleffsean/nostos/lang"
)

type VM struct {
	stack []interface{}
}

func New() *VM { return &VM{stack: make([]interface{}, 0)} }

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
	vm := New()
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
		return fmt.Errorf("functions not supported in evaluation")
	case *lang.Shovel:
		return fmt.Errorf("shovel operator not supported in evaluation")
	case *lang.ParseError:
		return fmt.Errorf(node.Error())
	default:
		return fmt.Errorf("unknown node type %T", node)
	}
	return nil
}
