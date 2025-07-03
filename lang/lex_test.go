package lang

import (
	"testing"
)

// Test Helpers

func assertEOF(t *testing.T, items <-chan item) {
	eof := <-items
	if eof.typ != itemEOF {
		t.Errorf("Expecting EOF but got %s", eof.typ)
	}
	if item, ok := <-items; ok {
		t.Errorf("got item after EOF  %q", item)
	}
}

func single(t *testing.T, items <-chan item) item {
	res := <-items
	assertEOF(t, items)
	return res
}

func pair(t *testing.T, items <-chan item) (item, item) {
	itema := <-items
	itemb := <-items
	assertEOF(t, items)
	return itema, itemb
}

func keyValue(t *testing.T, items <-chan item) (item, item) {
	itema := <-items
	colon := <-items
	itemb := <-items
	assertScalar(t, colon, itemColon, ":", 0)
	assertEOF(t, items)
	return itema, itemb
}

func assertScalar(t *testing.T, got item, wantedType itemType, content string, indent uint) {
	if got.typ != wantedType {
		t.Errorf("got %s, wanted %s", got.typ, wantedType)
	}
	if got.val != content {
		t.Errorf("got %q, wanted %q", got.val, content)
	}
	if got.indent != indent {
		t.Errorf("got indent of %q, wanted %q", got.indent, indent)
	}
}

func assertPosition(t *testing.T, got item, value string, byteOffset, byteLength, lineNumber, characterOffset uint) {
	if got.val != value {
		t.Errorf("got %q, wanted %q", got.val, value)
	}
	if got.position.ByteOffset != byteOffset {
		t.Errorf("ByteOffset: got %d, wanted %d", got.position.ByteOffset, byteOffset)
	}
	if got.position.ByteLength != byteLength {
		t.Errorf("ByteLength: got %d, wanted %d", got.position.ByteLength, byteLength)
	}
	if got.position.LineNumber != lineNumber {
		t.Errorf("LineNumber : got %d, wanted %d", got.position.LineNumber, lineNumber)
	}
	if got.position.CharacterOffset != characterOffset {
		t.Errorf("CharacterOffset for '%s': got %d, wanted %d", value, got.position.CharacterOffset, characterOffset)
	}
}

// Tests

func TestLexSymbol(t *testing.T) {
	_, items := NewStringLexer("yo")
	got := single(t, items)
	assertScalar(t, got, itemSymbol, "yo", 0)
}

func TestLexString(t *testing.T) {
	_, items := NewStringLexer(`"yo"`)
	got := single(t, items)
	assertScalar(t, got, itemString, "yo", 0)
}

func TestLexStringWithSymbols(t *testing.T) {
	_, items := NewStringLexer(`"apps/v1"`)
	got := single(t, items)
	assertScalar(t, got, itemString, "apps/v1", 0)
}

func TestLexPathAbsolute(t *testing.T) {
	_, items := NewStringLexer("/etc/passwd")
	got := single(t, items)
	assertScalar(t, got, itemPath, "/etc/passwd", 0)
}

func TestLexPathRelative(t *testing.T) {
	_, items := NewStringLexer("../foo")
	got := single(t, items)
	assertScalar(t, got, itemPath, "../foo", 0)
}

func TestLexInteger(t *testing.T) {
	_, items := NewStringLexer("123")
	got := single(t, items)
	assertScalar(t, got, itemNumber, "123", 0)
}

func TestLexFloat(t *testing.T) {
	_, items := NewStringLexer("123.99")
	got := single(t, items)
	assertScalar(t, got, itemNumber, "123.99", 0)
}

func TestLexList(t *testing.T) {
	_, items := NewStringLexer("- yo")
	itema, itemb := pair(t, items)
	assertScalar(t, itema, itemList, "", 0)
	assertScalar(t, itemb, itemSymbol, "yo", 1)
}

func TestLexMap(t *testing.T) {
	_, items := NewStringLexer("foo: bar")
	key, value := keyValue(t, items)
	assertScalar(t, key, itemSymbol, "foo", 0)
	assertScalar(t, value, itemSymbol, "bar", 0)
}

// TODO: Should lexer drop escape backslashes?
func TestLexQuotedString(t *testing.T) {
	_, items := NewStringLexer(`foo: "this is a \"quoted\" string"`)
	key, value := keyValue(t, items)
	assertScalar(t, key, itemSymbol, "foo", 0)
	assertScalar(t, value, itemString, `this is a \"quoted\" string`, 0)
}

func TestLexQuotedStringUnterminated(t *testing.T) {
	_, items := NewStringLexer(`foo: "unterminated`)
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, key, itemSymbol, "foo", 0)
	assertScalar(t, colon, itemColon, ":", 0)
	assertScalar(t, value, itemError, "EOF reached in unterminated string", 0)
}

func TestLexIndent(t *testing.T) {
	_, items := NewStringLexer("\n  foo: bar")
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, key, itemSymbol, "foo", 1)
	assertScalar(t, colon, itemColon, ":", 1)
	assertScalar(t, value, itemSymbol, "bar", 1)
	assertEOF(t, items)
}

func TestLexShovel(t *testing.T) {
	_, items := NewStringLexer("a << b")
	left := <-items
	shovel := <-items
	right := <-items
	assertScalar(t, left, itemSymbol, "a", 0)
	assertScalar(t, shovel, itemShovel, "<<", 0)
	assertScalar(t, right, itemSymbol, "b", 0)
	assertEOF(t, items)
}

func TestLexManifest(t *testing.T) {
	manifest := `
apiVersion: "apps/v1"
kind: "Deployment"
metadata:
  name: "example-deployment"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: "example"
  template:
    metadata:
      labels:
        app: "example"
    spec:
      containers:
      - name: "example-container"
        image: "example-image"
        ports:
        - containerPort: 8080
    `
	_, items := NewStringLexer(manifest)
	assertScalar(t, <-items, itemSymbol, "apiVersion", 0)
	assertScalar(t, <-items, itemColon, ":", 0)
	assertScalar(t, <-items, itemString, "apps/v1", 0)
	assertScalar(t, <-items, itemSymbol, "kind", 0)
	assertScalar(t, <-items, itemColon, ":", 0)
	assertScalar(t, <-items, itemString, "Deployment", 0)
	assertScalar(t, <-items, itemSymbol, "metadata", 0)
	assertScalar(t, <-items, itemColon, ":", 0)
	assertScalar(t, <-items, itemSymbol, "name", 1)
	assertScalar(t, <-items, itemColon, ":", 1)
	assertScalar(t, <-items, itemString, "example-deployment", 1)
	assertScalar(t, <-items, itemSymbol, "spec", 0)
	assertScalar(t, <-items, itemColon, ":", 0)
	assertScalar(t, <-items, itemSymbol, "replicas", 1)
	assertScalar(t, <-items, itemColon, ":", 1)
	assertScalar(t, <-items, itemNumber, "3", 1)
	assertScalar(t, <-items, itemSymbol, "selector", 1)
	assertScalar(t, <-items, itemColon, ":", 1)
	assertScalar(t, <-items, itemSymbol, "matchLabels", 2)
	assertScalar(t, <-items, itemColon, ":", 2)
	assertScalar(t, <-items, itemSymbol, "app", 3)
	assertScalar(t, <-items, itemColon, ":", 3)
	assertScalar(t, <-items, itemString, "example", 3)
	assertScalar(t, <-items, itemSymbol, "template", 1)
	assertScalar(t, <-items, itemColon, ":", 1)
	assertScalar(t, <-items, itemSymbol, "metadata", 2)
	assertScalar(t, <-items, itemColon, ":", 2)
	assertScalar(t, <-items, itemSymbol, "labels", 3)
	assertScalar(t, <-items, itemColon, ":", 3)
	assertScalar(t, <-items, itemSymbol, "app", 4)
	assertScalar(t, <-items, itemColon, ":", 4)
	assertScalar(t, <-items, itemString, "example", 4)
	assertScalar(t, <-items, itemSymbol, "spec", 2)
	assertScalar(t, <-items, itemColon, ":", 2)
	assertScalar(t, <-items, itemSymbol, "containers", 3)
	assertScalar(t, <-items, itemColon, ":", 3)
	assertScalar(t, <-items, itemList, "      ", 3) // TODO: this should be "-" or nil
	// Indentation following list items should be one level deeper
	assertScalar(t, <-items, itemSymbol, "name", 4)
	assertScalar(t, <-items, itemColon, ":", 4)
	assertScalar(t, <-items, itemString, "example-container", 4)
	assertScalar(t, <-items, itemSymbol, "image", 4)
	assertScalar(t, <-items, itemColon, ":", 4)
	assertScalar(t, <-items, itemString, "example-image", 4)
	assertScalar(t, <-items, itemSymbol, "ports", 4)
	assertScalar(t, <-items, itemColon, ":", 4)
	assertScalar(t, <-items, itemList, "        ", 4) // TODO: this should be "-" or nil
	assertScalar(t, <-items, itemSymbol, "containerPort", 5)
	assertScalar(t, <-items, itemColon, ":", 5)
	assertScalar(t, <-items, itemNumber, "8080", 5)
	assertEOF(t, items)
}

func TestLexPosition(t *testing.T) {
	manifest := `
apiVersion: "apps/v1"
kind: "Deplöyment"
metadata:
  name: "schön"
    `
	// Note, with the heredoc - the very first character is a newline
	_, items := NewStringLexer(manifest)
	// offset, length, line, character offset
	assertPosition(t, <-items, "apiVersion", 1, 10, 1, 0)
	assertPosition(t, <-items, ":", 11, 1, 1, 10)
	assertPosition(t, <-items, "apps/v1", 14, 7, 1, 12)
	assertPosition(t, <-items, "kind", 23, 4, 2, 0)
	assertPosition(t, <-items, ":", 27, 1, 2, 4)
	assertPosition(t, <-items, "Deplöyment", 30, 11, 2, 6)
	assertPosition(t, <-items, "metadata", 43, 8, 3, 0)
	assertPosition(t, <-items, ":", 51, 1, 3, 8)
	assertPosition(t, <-items, "name", 55, 4, 4, 2)
	assertPosition(t, <-items, ":", 59, 1, 4, 6)
	assertPosition(t, <-items, "schön", 62, 6, 4, 8)
	assertEOF(t, items)
}
