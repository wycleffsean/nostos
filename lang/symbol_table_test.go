package lang

import (
	// "reflect"
	"testing"

	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/pkg/types"
)

func setup(asts ...*Ast) *SymbolTable {
	registry := types.NewRegistry()
	st := NewSymbolTable(registry)

	for _, ast := range asts {
		st.ProcessAst(ast)
	}
	return st
}

const simpleCode string = `
  cab: "taxi"
  cat:
    bat: "baseball"
`

func TestSymbolTable(t *testing.T) {
	ast := NewAst(simpleCode, uri.New("file://foo.no"))
	_ = setup(&ast)
	// for i, c := range comments {
	// 	list := make([]*Comment, len(c.list))
	// 	for i, s := range c.list {
	// 		list[i] = &Comment{Text: s}
	// 	}

	// 	text := (&CommentGroup{list}).Text()
	// 	if text != c.text {
	// 		t.Errorf("case %d: got %q; expected %q", i, text, c.text)
	// 	}
	// }
}
