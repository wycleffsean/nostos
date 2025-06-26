package lang

import (
	"errors"
	"os"

	"github.com/mitchellh/mapstructure"
)

// ParseString parses Nostos source and returns a generic representation.
func ParseString(input string) (interface{}, error) {
	_, items := NewStringLexer(input)
	parser := NewParser(items)
	n := parser.Parse()
	if err, ok := n.(errorNode); ok {
		return nil, errors.New(err.Error())
	}
	return nodeToInterface(n), nil
}

// ParseFile reads and parses a file containing Nostos source.
func ParseFile(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseString(string(data))
}

// DecodeFile parses a file and decodes the result into out using mapstructure.
func DecodeFile(path string, out interface{}) error {
	v, err := ParseFile(path)
	if err != nil {
		return err
	}
	return mapstructure.Decode(v, out)
}

func nodeToInterface(n node) interface{} {
	switch v := n.(type) {
	case *String:
		return v.Text
	case *Path:
		return v.Text
	case *Number:
		return v.Value
	case *Symbol:
		return v.Text
	case *List:
		res := make([]interface{}, len(*v))
		for i, e := range *v {
			res[i] = nodeToInterface(e)
		}
		return res
	case *Map:
		m := make(map[string]interface{})
		for k, val := range *v {
			m[k.Text] = nodeToInterface(val)
		}
		return m
	default:
		return nil
	}
}
