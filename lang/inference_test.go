package lang

import (
	"testing"

	"github.com/wycleffsean/nostos/pkg/types"
)

func parseManifest(input string) node {
	_, items := NewStringLexer(input)
	parser := NewParser(items)
	return parser.Parse()
}

func TestInferType(t *testing.T) {
	registry := types.NewRegistry()
	td := types.TypeDefinition{
		Group:   "",
		Version: "v1",
		Kind:    "MyKind",
		Fields: []types.FieldDefinition{
			{Name: "apiVersion", Required: true},
			{Name: "kind", Required: true},
			{Name: "metadata", Required: true},
		},
	}
	registry.AddType(td)

	manifest := `
apiVersion: v1
kind: MyKind
metadata: something
`
	node := parseManifest(manifest)
	got, ok := InferType(node, registry)
	if !ok {
		t.Fatalf("expected match")
	}
	if got.Kind != td.Kind || got.Version != td.Version || got.Group != td.Group {
		t.Fatalf("unexpected type definition: %+v", got)
	}
}

func TestInferTypeNoMatch(t *testing.T) {
	registry := types.NewRegistry()
	td := types.TypeDefinition{
		Group:   "",
		Version: "v1",
		Kind:    "MyKind",
		Fields: []types.FieldDefinition{
			{Name: "apiVersion", Required: true},
			{Name: "kind", Required: true},
		},
	}
	registry.AddType(td)

	manifest := `kind: Other`
	node := parseManifest(manifest)
	if _, ok := InferType(node, registry); ok {
		t.Fatalf("unexpected match")
	}
}

func TestInferTypeMultipleMatch(t *testing.T) {
	registry := types.NewRegistry()
	common := []types.FieldDefinition{{Name: "apiVersion", Required: true}, {Name: "kind", Required: true}}
	registry.AddType(types.TypeDefinition{Group: "", Version: "v1", Kind: "A", Fields: common})
	registry.AddType(types.TypeDefinition{Group: "", Version: "v1", Kind: "B", Fields: common})

	manifest := `apiVersion: v1
kind: A`
	node := parseManifest(manifest)
	if _, ok := InferType(node, registry); ok {
		t.Fatalf("expected inference failure with multiple matches")
	}
}
