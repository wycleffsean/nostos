package lang

import (
	"testing"
)

var comments = []struct {
    list []string
    text string
}{
    {[]string{"#"}, ""},
    {[]string{"# "}, ""},
    {[]string{"# foo   "}, "foo\n"},
    {[]string{"#", "#", "# foo"}, "foo\n"},
    {[]string{"# foo bar  "}, "foo bar\n"},
    {[]string{"# foo", "#", "#", "#", "# bar"}, "foo\n\nbar\n"},
    {[]string{"# ", "#", "#foo", "#", "#"}, "foo\n"},
}

func TestCommentText(t *testing.T) {
    for i, c := range comments {
        list := make([]*Comment, len(c.list))
        for i, s := range c.list {
            list[i] = &Comment{Text: s}
        }

        text := (&CommentGroup{list}).Text()
        if text != c.text {
            t.Errorf("case %d: got %q; expected %q", i, text, c.text)
        }
    }
}

func parseString(input string) chan *node {
	_, items := NewStringLexer(input)
	parser := NewParser(items)
	return parser.Parse()
}

// Tests

// // Scalars
func TestParseString(t *testing.T) {
	nodes := parseString("\"yo\"")
	got := <-nodes
	// assertScalar(t, got, node{})
	wanted := String{0, "yo"}
	if got.(String) != wanted {
		t.Errorf("got %q, wanted %q", got, wanted)
	}
}

//// Yaml

func TestYamlMap(t *testing.T) {
	// _, items := NewStringLexer("yo")
	// assertScalar(t, got, item{itemString, "yo"})
}
