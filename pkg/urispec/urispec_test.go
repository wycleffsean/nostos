package urispec

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in  string
		typ string
	}{
		{"examples", "path"},
		{"./examples/odyssey.no", "path"},
		{"https://github.com/wycleffsean/nostos.git", "git"},
		{"github:wycleffsean/nostos", "git"},
	}
	for _, c := range cases {
		s := Parse(c.in)
		if s.Type != c.typ {
			t.Fatalf("%s expected type %s got %s", c.in, c.typ, s.Type)
		}
	}
}
