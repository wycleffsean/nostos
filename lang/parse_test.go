package lang

import (
	"reflect"
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

func parseString(input string) node {
	_, items := NewStringLexer(input)
	parser := NewParser(items)
	return parser.Parse()
}

// Tests

// // Scalars
func TestParseString(t *testing.T) {
	got:= parseString("\"yo\"")
	// assertScalar(t, got, node{})
	wanted := String{Position{}, "yo"}
	if str, ok := got.(*String); ok {
		if *str != wanted {
			t.Errorf("got %q, wanted %q", *str, wanted)
		}
	} else {
		t.Errorf("can't cast to String: %v", got)
	}
}

//// Yaml

func TestParseYamlSimpleMap(t *testing.T) {
	got := parseString(`foo: "bar"`)

	key := Symbol{Position{}, "foo"}
	var wanted Map = make(map[Symbol]node)
	value := &String{Position{}, "bar"}
	wanted[key] = value

	if m, ok := got.(*Map); ok {
		if !reflect.DeepEqual(*m, wanted) {
			t.Errorf("maps aren't equal")
		}
	} else {
		t.Errorf("can't cast to Map: %v", got)
	}
}

func TestParseYamlMap(t *testing.T) {
	got:= parseString(`
  foo: "bar"
  baz: "bar"
    	`)

	foo := Symbol{Position{}, "foo"}
	bar := Symbol{Position{}, "baz"}
	var wanted Map = make(map[Symbol]node)
	value := &String{Position{}, "bar"}
	wanted[foo] = value
	wanted[bar] = value

	if m, ok := got.(*Map); ok {
		if !reflect.DeepEqual(*m, wanted) {
			t.Errorf("maps aren't equal - expected: %v got: %v", wanted, *m)
		}
	} else {
		t.Errorf("can't cast to Map: %v", got)
	}
}

func TestParseYamlNestedMaps(t *testing.T) {
	got := parseString(`
  foo: "bar"
  baz:
    foo: "bar"
    	`)

	foo := Symbol{Position{}, "foo"}
	baz := Symbol{Position{}, "baz"}
	var wanted Map = make(map[Symbol]node)
	var child Map = make(map[Symbol]node)
	bar := &String{Position{}, "bar"}
	wanted[foo] = bar
	wanted[baz] = &child
	child[foo] = bar

	if m, ok := got.(*Map); ok {
		if !reflect.DeepEqual(*m, wanted) {
			t.Errorf("maps aren't equal - expected: %v got: %v", wanted, *m)
		}
	} else {
		t.Errorf("can't cast to Map: %v", got)
	}
}
