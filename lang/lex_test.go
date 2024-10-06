package lang

import (
	"testing"
)

// Test Helpers

func assertChannelClosed(t *testing.T, items chan item) {
	i := <-items
	if i.typ != itemEOF {
		t.Errorf("expected EOF")
	}
	_, ok := <-items
	if ok {
		t.Errorf("got item after EOF: %q", i)
	}
}

func single(t *testing.T, items chan item) item {
	res := <-items
	assertChannelClosed(t, items)
	return res
}

func pair(t *testing.T, items chan item) (item, item) {
	itema := <-items
	itemb := <-items
	assertChannelClosed(t, items)
	return itema, itemb
}

func assertScalar(t *testing.T, got, wanted item) {
	if got != wanted {
		t.Errorf("got %q, wanted %q", got, wanted)
	}
}

// Tests

// func TestMapKeyValue(t *testing.T) {
// 	_, items := Lex("foo: bar")
// 	gotkey, gotvalue := pair(t, items)
// 	assertScalar(t, gotkey, item{itemString, "foo"})
// 	assertScalar(t, gotvalue, item{itemString, "bar"})
// }

func TestString(t *testing.T) {
	_, items := Lex("yo")
	got := single(t, items)
	assertScalar(t, got, item{itemString, "yo"})
}

// TODO: Should lexer drop escape backslashes?
func TestQuotedString(t *testing.T) {
	_, items := Lex(`"this is a \"quoted\" string"`)
	got := single(t, items)
	assertScalar(t, got, item{itemString, `this is a \"quoted\" string`})
}

func TestQuotedStringUnterminated(t *testing.T) {
	_, items := Lex(`"unterminated`)
	got := single(t, items)
	assertScalar(t, got, item{itemError, "EOF reached in unterminated string"})
}

// func TestInteger(t *testing.T) {
// 	_, items := Lex("123")
// 	got := single(t, items)
// 	assertScalar(t, got, item{itemNumber, "123"})
// }

// func TestFloat(t *testing.T) {
// 	_, items := Lex("123.99")
// 	got := single(t, items)
// 	assertScalar(t, got, item{itemNumber, "123.99"})
// }

func TestList(t *testing.T) {
	_, items := Lex("- yo")
	itema, itemb := pair(t, items)
	assertScalar(t, itema, item{itemList, ""})
	assertScalar(t, itemb, item{itemString, "yo"})
}
