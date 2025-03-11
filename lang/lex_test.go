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
	assertScalar(t, colon, item{itemColon, ":"})
	assertChannelClosed(t, items)
	return itema, itemb
}

func assertScalar(t *testing.T, got, wanted item) {
	if got != wanted {
		t.Errorf("got %q, wanted %q", got, wanted)
	}
}

// Tests

func TestSymbol(t *testing.T) {
	_, items := NewStringLexer("yo")
	got := single(t, items)
	assertScalar(t, got, item{itemSymbol, "yo"})
}

func TestString(t *testing.T) {
	_, items := NewStringLexer(`"yo"`)
	got := single(t, items)
	assertScalar(t, got, item{itemString, "yo"})
}

func TestStringWithSymbols(t *testing.T) {
	_, items := NewStringLexer(`"apps/v1"`)
	got := single(t, items)
	assertScalar(t, got, item{itemString, "apps/v1"})
}

func TestInteger(t *testing.T) {
	_, items := NewStringLexer("123")
	got := single(t, items)
	assertScalar(t, got, item{itemNumber, "123"})
}

func TestFloat(t *testing.T) {
	_, items := NewStringLexer("123.99")
	got := single(t, items)
	assertScalar(t, got, item{itemNumber, "123.99"})
}

func TestList(t *testing.T) {
	_, items := NewStringLexer("- yo")
	itema, itemb := pair(t, items)
	assertScalar(t, itema, item{itemList, ""})
	assertScalar(t, itemb, item{itemSymbol, "yo"})
}

func TestMap(t *testing.T) {
	_, items := NewStringLexer("foo: bar")
	key, value := keyValue(t, items)
	assertScalar(t, key, item{itemSymbol, "foo"})
	assertScalar(t, value, item{itemSymbol, "bar"})
}

// TODO: Should lexer drop escape backslashes?
func TestQuotedString(t *testing.T) {
	_, items := NewStringLexer(`foo: "this is a \"quoted\" string"`)
	key, value := keyValue(t, items)
	assertScalar(t, key, item{itemSymbol, "foo"})
	assertScalar(t, value, item{itemString, `this is a \"quoted\" string`})
}

func TestQuotedStringUnterminated(t *testing.T) {
	_, items := NewStringLexer(`foo: "unterminated`)
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, key, item{itemSymbol, "foo"})
	assertScalar(t, colon, item{itemColon, ":"})
	assertScalar(t, value, item{itemError, "EOF reached in unterminated string"})
}

func TestIndent(t *testing.T) {
	_, items := NewStringLexer("\n  foo: bar")
	indent := <-items
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, indent, item{itemIndent, "  "})
	assertScalar(t, key, item{itemSymbol, "foo"})
	assertScalar(t, colon, item{itemColon, ":"})
	assertScalar(t, value, item{itemSymbol, "bar"})
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
	assertScalar(t, <-items, item{itemSymbol, "apiVersion"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemString, "apps/v1"})
	assertScalar(t, <-items, item{itemSymbol, "kind"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemString, "Deployment"})
	assertScalar(t, <-items, item{itemSymbol, "metadata"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "name"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemString, "example-deployment"})
	assertScalar(t, <-items, item{itemSymbol, "spec"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "replicas"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemNumber, "3"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "selector"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "matchLabels"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "app"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemString, "example"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "template"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "metadata"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "labels"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemIndent, "  "})
	assertScalar(t, <-items, item{itemSymbol, "app"})
	assertScalar(t, <-items, item{itemColon, ":"})
	assertScalar(t, <-items, item{itemString, "example"})
}
