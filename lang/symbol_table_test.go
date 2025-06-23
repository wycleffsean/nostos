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

func TestSymbolTableLookupByName(t *testing.T) {
	registry := types.NewRegistry()
	st := NewSymbolTable(registry)
	doc := uri.New("file://foo.no")

	st.AddSymbol(&Symbol{Position{}, "cab"}, doc)
	st.AddSymbol(&Symbol{Position{}, "cat"}, doc)
	st.AddSymbol(&Symbol{Position{}, "bat"}, doc)

	expected := []string{"cab", "cat", "bat"}
	for _, name := range expected {
		if _, ok := st.LookupByName(name); !ok {
			t.Fatalf("symbol %q not found in symbol table", name)
		}
	}
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

func TestSymbolTableLookupByPosition(t *testing.T) {
	registry := types.NewRegistry()
	st := NewSymbolTable(registry)
	doc := uri.New("file://foo.no")

	foo := &Symbol{Position{LineNumber: 1, CharacterOffset: 2}, "foo"}
	bar := &Symbol{Position{LineNumber: 3, CharacterOffset: 4}, "bar"}

	st.AddSymbol(foo, doc)
	st.AddSymbol(bar, doc)

	if entry, ok := st.LookupByPosition(foo.Position); !ok || entry.Symbol != foo {
		t.Fatalf("lookup by position failed for foo")
	}
	if entry, ok := st.LookupByPosition(bar.Position); !ok || entry.Symbol != bar {
		t.Fatalf("lookup by position failed for bar")
	}
}

func TestSymbolTableReplaceDocumentSymbols(t *testing.T) {
	registry := types.NewRegistry()
	st := NewSymbolTable(registry)
	docA := uri.New("file://a.no")
	docB := uri.New("file://b.no")

	a1 := &Symbol{Position{LineNumber: 0, CharacterOffset: 0}, "a1"}
	b1 := &Symbol{Position{LineNumber: 0, CharacterOffset: 1}, "b1"}
	st.AddSymbol(a1, docA)
	st.AddSymbol(b1, docB)

	st.RemoveSymbolsForDocument(docA)
	if _, ok := st.LookupByName("a1"); ok {
		t.Fatalf("symbol from docA was not removed")
	}
	if _, ok := st.LookupByName("b1"); !ok {
		t.Fatalf("symbol from docB should remain")
	}

	a2 := &Symbol{Position{LineNumber: 1, CharacterOffset: 0}, "a2"}
	st.AddSymbol(a2, docA)

	if _, ok := st.LookupByName("a2"); !ok {
		t.Fatalf("new symbol for docA not added")
	}
	if _, ok := st.LookupByName("b1"); !ok {
		t.Fatalf("symbol from docB should still exist after replacement")
	}
}
