package lang

import (
	"reflect"
	"strings"
	"testing"

	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/pkg/urispec"
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
	parser := NewParser(items, uri.URI("test"))
	return parser.Parse()
}

func zeroPositions(n node) {
	switch v := n.(type) {
	case *String:
		v.Position = Position{}
	case *Path:
		v.Position = Position{}
	case *Number:
		v.Position = Position{}
	case *Symbol:
		v.Position = Position{}
	case *List:
		for i, item := range *v {
			zeroPositions(item)
			(*v)[i] = item
		}
	case *Map:
		newMap := make(Map)
		for k, val := range *v {
			zeroPositions(val)
			k.Position = Position{}
			newMap[k] = val
		}
		*v = newMap
	case *Function:
		zeroPositions(v.Param)
		zeroPositions(v.Body)
	case *Call:
		zeroPositions(v.Func)
		zeroPositions(v.Arg)
	case *Shovel:
		zeroPositions(v.Left)
		zeroPositions(v.Right)
	case *Let:
		if v.Bindings != nil {
			zeroPositions(v.Bindings)
		}
		zeroPositions(v.Body)
		v.Position = Position{}
	}
}

// Tests

// // Scalars
func TestParseString(t *testing.T) {
	got := parseString("\"yo\"")
	// assertScalar(t, got, node{})
	wanted := String{Position{}, "yo"}
	zeroPositions(got)
	if str, ok := got.(*String); ok {
		if *str != wanted {
			t.Errorf("got %q, wanted %q", *str, wanted)
		}
	} else {
		t.Errorf("can't cast to String: %v", got)
	}
}

func TestParsePath(t *testing.T) {
	got := parseString("../foo")
	wanted := Path{Position{}, urispec.Parse("../foo")}
	zeroPositions(got)
	if p, ok := got.(*Path); ok {
		if !reflect.DeepEqual(*p, wanted) {
			t.Errorf("got %#v, wanted %#v", *p, wanted)
		}
	} else {
		t.Errorf("can't cast to Path: %v", got)
	}
}

func TestParsePathAbsolute(t *testing.T) {
	got := parseString("/etc/passwd")
	wanted := Path{Position{}, urispec.Parse("/etc/passwd")}
	zeroPositions(got)
	if p, ok := got.(*Path); ok {
		if !reflect.DeepEqual(*p, wanted) {
			t.Errorf("got %#v, wanted %#v", *p, wanted)
		}
	} else {
		t.Errorf("can't cast to Path: %v", got)
	}
}

//// Yaml

func TestParseYamlSimpleMap(t *testing.T) {
	got := parseString(`foo: "bar"`)

	key := Symbol{Position{}, "foo"}
	var wanted Map = make(map[Symbol]node)
	value := &String{Position{}, "bar"}
	wanted[key] = value

	zeroPositions(got)
	if m, ok := got.(*Map); ok {
		if !reflect.DeepEqual(*m, wanted) {
			t.Errorf("maps aren't equal")
		}
	} else {
		t.Errorf("can't cast to Map: %v", got)
	}
}

func TestParseMultiYamlMap(t *testing.T) {
	assert := func(version, code string) {
		got := parseString(code)
		foo := Symbol{Position{}, "foo"}
		bar := Symbol{Position{}, "baz"}
		var wanted Map = make(map[Symbol]node)
		value := &String{Position{}, "bar"}
		wanted[foo] = value
		wanted[bar] = value

		zeroPositions(got)
		if m, ok := got.(*Map); ok {
			if !reflect.DeepEqual(*m, wanted) {
				t.Errorf("%s: maps aren't equal - expected: %v got: %v", version, wanted, *m)
			}
		} else {
			t.Errorf("can't cast to Map: %v", got)
		}
	}
	assert("unindented", `
foo: "bar"
baz: "bar"`)
	assert("leading indent", `
  foo: "bar"
  baz: "bar"`)
}

func TestParseYamlNestedMaps(t *testing.T) {
	got := parseString(`
  foo: "bar"
  baz:
    foo: "bar"`)

	foo := Symbol{Position{}, "foo"}
	baz := Symbol{Position{}, "baz"}
	var wanted Map = make(map[Symbol]node)
	var child Map = make(map[Symbol]node)
	bar := &String{Position{}, "bar"}
	wanted[foo] = bar
	wanted[baz] = &child
	child[foo] = bar

	zeroPositions(got)
	if m, ok := got.(*Map); ok {
		if !reflect.DeepEqual(*m, wanted) {
			t.Errorf("maps aren't equal - expected: %v got: %v", wanted, *m)
		}
	} else {
		t.Errorf("can't cast to Map: %v", got)
	}
}

func TestParseYamlSimpleList(t *testing.T) {
	got := parseString(`- "foo"`)

	wanted := List{&String{Position{}, "foo"}}

	zeroPositions(got)
	if l, ok := got.(*List); ok {
		if !reflect.DeepEqual(*l, wanted) {
			t.Errorf("lists aren't equal - expected: %v got: %v", wanted, *l)
		}
	} else {
		t.Errorf("can't cast to List: %v", got)
	}
}

func TestParseYamlMultiList(t *testing.T) {
	got := parseString(`- "foo"
- "bar"`)

	wanted := List{&String{Position{}, "foo"}, &String{Position{}, "bar"}}

	zeroPositions(got)
	if l, ok := got.(*List); ok {
		if !reflect.DeepEqual(*l, wanted) {
			t.Errorf("lists aren't equal - expected: %v got: %v", wanted, *l)
		}
	} else {
		t.Errorf("can't cast to List: %v", got)
	}
}

func TestParseYamlListOfMaps(t *testing.T) {
	got := parseString("list:\n- foo: 123\n  bar: 678")

	listSym := Symbol{Position{}, "list"}
	fooSym := Symbol{Position{}, "foo"}
	barSym := Symbol{Position{}, "bar"}

	var item Map = make(map[Symbol]node)
	item[fooSym] = &Number{Position{}, 123}
	item[barSym] = &Number{Position{}, 678}

	wantedList := List{&item}
	var wanted Map = make(map[Symbol]node)
	wanted[listSym] = &wantedList

	zeroPositions(got)
	if m, ok := got.(*Map); ok {
		if !reflect.DeepEqual(*m, wanted) {
			t.Errorf("unexpected parse result: %#v", *m)
		}
	} else {
		t.Errorf("can't cast to Map: %T", got)
	}
}

func TestParseFunction(t *testing.T) {
	got := parseString("x => x")
	wanted := &Function{Param: &Symbol{Position{}, "x"}, Body: &Symbol{Position{}, "x"}}
	zeroPositions(got)
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("function parse mismatch - expected: %#v got: %#v", wanted, got)
	}
}

func TestParseShovel(t *testing.T) {
	got := parseString("a << b")
	wanted := &Shovel{Left: &Symbol{Position{}, "a"}, Right: &Symbol{Position{}, "b"}}
	zeroPositions(got)
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("shovel parse mismatch - expected: %#v got: %#v", wanted, got)
	}
}

func TestParseCall(t *testing.T) {
	got := parseString("foo(bar)")
	wanted := &Call{Func: &Symbol{Position{}, "foo"}, Arg: &Symbol{Position{}, "bar"}}
	zeroPositions(got)
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("call parse mismatch - expected: %#v got: %#v", wanted, got)
	}
}

func TestParseLet(t *testing.T) {
	got := parseString("let foo: 1 in foo")
	foo := Symbol{Position{}, "foo"}
	bindings := make(Map)
	bindings[foo] = &Number{Position{}, 1}
	wanted := &Let{Bindings: &bindings, Body: &Symbol{Position{}, "foo"}}
	zeroPositions(got)
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("let parse mismatch - expected: %#v got: %#v", wanted, got)
	}
}

func TestParseHandlesLexerErrors(t *testing.T) {
	// Input with an unterminated string should not cause panics
	got := parseString("apiVersion: \"v1\"\nkind: \"Service")
	err, ok := got.(*ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T", got)
	}
	if !strings.Contains(err.Error(), "unterminated string") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseHandlesTabsGracefully(t *testing.T) {
	got := parseString("foo:\tbar")
	err, ok := got.(*ParseError)
	if !ok {
		t.Fatalf("expected ParseError, got %T", got)
	}
	if !strings.Contains(err.Error(), "horizontal tabs") {
		t.Errorf("unexpected error: %v", err)
	}
}
