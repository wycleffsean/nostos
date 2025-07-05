package lang

import "testing"

func FuzzLex(f *testing.F) {
	seeds := []string{"", "foo: bar", "apiVersion: \"v1\"", "- item"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, items := NewStringLexer(input)
		for range items {
			// drain items until closed
		}
	})
}
