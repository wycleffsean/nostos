package lang

import (
	"go.lsp.dev/uri"
	"testing"
)

func FuzzParse(f *testing.F) {
	seeds := []string{"foo: bar", "x => x", "- foo", "apiVersion: \"v1\""}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, items := NewStringLexer(input)
		p := NewParser(items, uri.URI("fuzz"))
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic: %v", r)
			}
		}()
		_ = p.Parse()
	})
}
