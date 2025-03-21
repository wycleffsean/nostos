package lang

import (
	"go.lsp.dev/uri"
	"sync"
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

	parser := NewParser(lexer.items)
	parsedItem := parser.Parse()

	return Ast{uri, parsedItem}
}
