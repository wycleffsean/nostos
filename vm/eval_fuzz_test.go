package vm

import (
	"testing"

	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/lang"
)

func FuzzEval(f *testing.F) {
	seeds := []string{"foo: 1", "x => x", "foo(bar)", "- item"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		_, items := lang.NewStringLexer(input)
		p := lang.NewParser(items, uri.URI("fuzz"))
		ast := p.Parse()
		_, _ = EvalWithDir(ast, ".", uri.URI("fuzz"))
	})
}
