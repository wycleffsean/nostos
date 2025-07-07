package lang_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	lang "github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/types"
	"github.com/wycleffsean/nostos/vm"
	"go.lsp.dev/uri"
)

type Program string

func (Program) Generate(r *rand.Rand, size int) reflect.Value {
	depth := 1 + r.Intn(3)
	s := strings.TrimSpace(randomValue(r, depth, 0))
	return reflect.ValueOf(Program(s))
}

func randomPrimitive(r *rand.Rand) string {
	switch r.Intn(3) {
	case 0:
		return fmt.Sprintf("\"s%d\"", r.Intn(100))
	case 1:
		return fmt.Sprintf("%d", r.Intn(10))
	default:
		return fmt.Sprintf("sym%d", r.Intn(10))
	}
}

func randomList(r *rand.Rand, depth, indent int) string {
	n := r.Intn(3) + 1
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(strings.Repeat("  ", indent))
		b.WriteString("- ")
		val := randomValue(r, depth-1, indent+1)
		if strings.Contains(val, "\n") {
			b.WriteString("\n")
			b.WriteString(val)
		} else {
			b.WriteString(val)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func randomMap(r *rand.Rand, depth, indent int) string {
	n := r.Intn(3) + 1
	var b strings.Builder
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key%d", r.Intn(10))
		b.WriteString(strings.Repeat("  ", indent))
		b.WriteString(key)
		b.WriteString(": ")
		val := randomValue(r, depth-1, indent+1)
		if strings.Contains(val, "\n") {
			b.WriteString("\n")
			b.WriteString(val)
		} else {
			b.WriteString(val)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func randomValue(r *rand.Rand, depth, indent int) string {
	if depth <= 0 {
		return randomPrimitive(r)
	}
	switch r.Intn(3) {
	case 0:
		return randomPrimitive(r)
	case 1:
		return randomList(r, depth, indent)
	default:
		return randomMap(r, depth, indent)
	}
}

func TestQuickCheckPrograms(t *testing.T) {
	reg := types.NewRegistry()
	reg.AddType(&types.ObjectType{
		Group:   "",
		Version: "v1",
		Kind:    "Example",
		Fields: map[string]*types.Field{
			"apiVersion": {Name: "apiVersion", Type: &types.PrimitiveType{N: "string"}, Required: true},
			"kind":       {Name: "kind", Type: &types.PrimitiveType{N: "string"}, Required: true},
		},
	})
	cfg := &quick.Config{MaxCount: 20}
	if err := quick.Check(func(p Program) bool {
		_, items := lang.NewStringLexer(string(p))
		parser := lang.NewParser(items, uri.URI("quick"))
		ast := parser.Parse()
		if errs := lang.CollectParseErrors(ast); len(errs) > 0 {
			return false
		}
		if _, err := vm.Eval(ast); err != nil {
			return false
		}
		lang.InferType(ast, reg)
		return true
	}, cfg); err != nil {
		t.Fatal(err)
	}
}
