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

func TestString(t *testing.T) {
	_, items := Lex("yo")
	got := single(t, items)
	assertScalar(t, got, item{itemString, "yo"})
}

func TestUnquotedStringWithSymbols(t *testing.T) {
	_, items := Lex("apps/v1")
	got := single(t, items)
	assertScalar(t, got, item{itemString, "apps/v1"})
}

func TestInteger(t *testing.T) {
	_, items := Lex("123")
	got := single(t, items)
	assertScalar(t, got, item{itemNumber, "123"})
}

func TestFloat(t *testing.T) {
	_, items := Lex("123.99")
	got := single(t, items)
	assertScalar(t, got, item{itemNumber, "123.99"})
}

func TestList(t *testing.T) {
	_, items := Lex("- yo")
	itema, itemb := pair(t, items)
	assertScalar(t, itema, item{itemList, ""})
	assertScalar(t, itemb, item{itemString, "yo"})
}

func TestMap(t *testing.T) {
	_, items := Lex("foo: bar")
	key, value := keyValue(t, items)
	assertScalar(t, key, item{itemString, "foo"})
	assertScalar(t, value, item{itemString, "bar"})
}

// TODO: Should lexer drop escape backslashes?
func TestQuotedString(t *testing.T) {
	_, items := Lex(`foo: "this is a \"quoted\" string"`)
	key, value:= keyValue(t, items)
	assertScalar(t, key, item{itemString, "foo"})
	assertScalar(t, value, item{itemString, `this is a \"quoted\" string`})
}

func TestQuotedStringUnterminated(t *testing.T) {
	_, items := Lex(`foo: "unterminated`)
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, key, item{itemString, "foo"})
	assertScalar(t, colon, item{itemColon, ":"})
	assertScalar(t, value, item{itemError, "EOF reached in unterminated string"})
}

func TestIndent(t *testing.T) {
	_, items := Lex("\n  foo: bar")
	indent := <-items
	key := <-items
	colon := <-items
	value := <-items
	assertScalar(t, indent, item{itemIndent, "  "})
	assertScalar(t, key, item{itemString, "foo"})
	assertScalar(t, colon, item{itemColon, ":"})
	assertScalar(t, value, item{itemString, "bar"})
}

func TestManifest(t *testing.T) {
    manifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "example-deployment"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
      - name: "example-container"
        image: "example-image"
        ports:
        - containerPort: 8080
    `
	_, items := Lex(manifest)
	assertScalar(t, <- items, item{itemString, "apiVersion"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemString, "apps/v1"})
	assertScalar(t, <- items, item{itemString, "kind"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemString, "Deployment"})
	assertScalar(t, <- items, item{itemString, "metadata"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "name"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemString, "example-deployment"})
	assertScalar(t, <- items, item{itemString, "spec"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "replicas"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemNumber, "3"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "selector"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "matchLabels"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "app"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemString, "example"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "template"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "metadata"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "labels"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemIndent, "  "})
	assertScalar(t, <- items, item{itemString, "app"})
	assertScalar(t, <- items, item{itemColon, ":"})
	assertScalar(t, <- items, item{itemString, "example"})
}
