package lang

import (
	"testing"
)

func parseString(input string) chan *node {
	_, items := NewStringLexer(input)
	parser := NewParser(items)
	return parser.Parse()
}

// Tests

// // Scalars
func TestParseString(t *testing.T) {
	// nodes := parseString("\"yo\"")
	// got := <-nodes
	// assertScalar(t, got, node{})
}

//// Yaml

func TestYamlMap(t *testing.T) {
	// _, items := NewStringLexer("yo")
	// assertScalar(t, got, item{itemString, "yo"})
}
