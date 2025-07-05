package lang

import (
	"sync"

	"go.lsp.dev/uri"
)

type Ast struct {
	Document uri.URI
	RootNode node
}

func NewAst(input string, uri uri.URI) Ast {
	lexer := &lexer{input: input, items: make(chan item)}
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		lexer.run()
	}()

	parser := NewParser(lexer.items, uri)
	parsedItem := parser.Parse()

	return Ast{uri, parsedItem}
}

func (a *Ast) ExtractSymbols() []*Symbol {
	return extractSymbols(a.RootNode)
}

func extractSymbols(ast_node node) []*Symbol {
	symbols := make([]*Symbol, 0)
	switch node := ast_node.(type) {
	case *Symbol:
		symbols = append(symbols, node)
	case binaryOpNode:
		symbols = append(symbols, extractSymbolsBinary(node)...)
	case collectionNode:
		symbols = append(symbols, extractSymbolsCollection(node)...)
	case *Let:
		for k, v := range *node.Bindings {
			key := k
			symbols = append(symbols, &key)
			symbols = append(symbols, extractSymbols(v)...)
		}
		symbols = append(symbols, extractSymbols(node.Body)...)
	default:
	}
	return symbols
}

func extractSymbolsBinary(binary binaryOpNode) []*Symbol {
	symbols := make([]*Symbol, 0)
	// symbols = append(symbols, extractSymbols(node.leftExpr())...)
	// symbols = append(symbols, extractSymbols(node.rightExpr())...)
	return symbols
}

func extractSymbolsCollection(collection collectionNode) []*Symbol {
	symbols := make([]*Symbol, 0)
	for _, child := range collection.Symbols() {
		symbols = append(symbols, extractSymbols(child)...)
	}
	return symbols
}
