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

func keyValue(t *testing.T, items chan item) (item, item) {
	itema := <-items
	colon := <-items
	itemb := <-items
	assertScalar(t, colon, itemColon, ":")
	assertChannelClosed(t, items)
	return itema, itemb
}

func assertScalar(t *testing.T, got item, wantedType itemType, content string) {
	if got.typ != wantedType {
		t.Errorf("got %s, wanted %s", got.typ, wantedType)
	}
	if got.val != content {
		t.Errorf("got %q, wanted %q", got.val, content)
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

func TestSymbol(t *testing.T) {
	_, items := NewStringLexer("yo")
	got := single(t, items)
	assertScalar(t, got, itemSymbol, "yo")
}

func TestString(t *testing.T) {
	_, items := NewStringLexer(`"yo"`)
	got := single(t, items)
	assertScalar(t, got, itemString, "yo")
}

func TestStringWithSymbols(t *testing.T) {
	_, items := NewStringLexer(`"apps/v1"`)
	got := single(t, items)
	assertScalar(t, got, itemString, "apps/v1")
}

func TestInteger(t *testing.T) {
	_, items := NewStringLexer("123")
	got := single(t, items)
	assertScalar(t, got, itemNumber, "123")
}

func TestFloat(t *testing.T) {
	_, items := NewStringLexer("123.99")
	got := single(t, items)
	assertScalar(t, got, itemNumber, "123.99")
}

func TestList(t *testing.T) {
	_, items := NewStringLexer("- yo")
	itema, itemb := pair(t, items)
	assertScalar(t, itema, itemList, "")
	assertScalar(t, itemb, itemSymbol, "yo")
}

func TestMap(t *testing.T) {
	_, items := NewStringLexer("foo: bar")
	key, value := keyValue(t, items)
	assertScalar(t, key, itemSymbol, "foo")
	assertScalar(t, value, itemSymbol, "bar")
}

// TODO: Should lexer drop escape backslashes?
func TestQuotedString(t *testing.T) {
	_, items := NewStringLexer(`foo: "this is a \"quoted\" string"`)
	key, value := keyValue(t, items)
	assertScalar(t, key, itemSymbol, "foo")
	assertScalar(t, value, itemString, `this is a \"quoted\" string`)
}

func TestQuotedStringUnterminated(t *testing.T) {
	_, items := NewStringLexer(`foo: "unterminated`)
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, key, itemSymbol, "foo")
	assertScalar(t, colon, itemColon, ":")
	assertScalar(t, value, itemError, "EOF reached in unterminated string")
}

func TestIndent(t *testing.T) {
	_, items := NewStringLexer("\n  foo: bar")
	indent := <-items
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, indent, itemIndent, "  ")
	assertScalar(t, key, itemSymbol, "foo")
	assertScalar(t, colon, itemColon, ":")
	assertScalar(t, value, itemSymbol, "bar")
}

func TestManifest(t *testing.T) {
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
	assertScalar(t, <-items, itemSymbol, "apiVersion")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemString, "apps/v1")
	assertScalar(t, <-items, itemSymbol, "kind")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemString, "Deployment")
	assertScalar(t, <-items, itemSymbol, "metadata")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "name")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemString, "example-deployment")
	assertScalar(t, <-items, itemSymbol, "spec")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "replicas")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemNumber, "3")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "selector")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "matchLabels")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "app")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemString, "example")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "template")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "metadata")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "labels")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemIndent, "  ")
	assertScalar(t, <-items, itemSymbol, "app")
	assertScalar(t, <-items, itemColon, ":")
	assertScalar(t, <-items, itemString, "example")
}

func TestPosition(t *testing.T) {
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
	assertPosition(t, <-items, "  ", 53, 2, 4, 0)
	assertPosition(t, <-items, "name", 55, 4, 4, 2)
	assertPosition(t, <-items, ":", 59, 1, 4, 6)
	assertPosition(t, <-items, "schön", 62, 6, 4, 8)
}
