package lang

import (
	"sync"

	"github.com/google/btree"
	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/pkg/types"
)

// Symbol represents a language entity (variable, function, type, etc.).
type SymbolEntry struct {
	Name      string
	// Kind      protocol.SymbolKind
	Begin Position
	End Position
	Type      *types.TypeDefinition
	DefinedIn uri.URI
}

func (s *SymbolEntry) Less(than btree.Item) bool {
	o := than.(*Symbol)
	if s.Begin.LineNumber < o.Begin.LineNumber {
		return true
	}
	if s.Begin.LineNumber == o.Begin.LineNumber {
		return s.Begin.CharacterOffset < o.Begin.CharacterOffset
	}
	return false
}


// SymbolTable is a concurrency-safe table of symbols.
type SymbolTable struct {
	mu       sync.RWMutex
	byName   map[string]*SymbolEntry    // Lookup by name
	byPos    *btree.BTree          // Positional lookup
	typeReg  *types.Registry       // Reference to the type registry
	docIndex map[uri.URI][]*SymbolEntry // Tracks symbols by document for easy removal
}

// NewSymbolTable initializes a new symbol table.
func NewSymbolTable(registry *types.Registry) *SymbolTable {
	return &SymbolTable{
		byName:   make(map[string]*SymbolEntry),
		byPos:    btree.New(2),
		typeReg:  registry,
		docIndex: make(map[uri.URI][]*SymbolEntry),
	}
}


// RemoveSymbolsForDocument removes all symbols associated with a given document.
func (s *SymbolTable) RemoveSymbolsForDocument(uri uri.URI) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find all symbols for the document
	symbols, found := s.docIndex[uri]
	if !found {
		return
	}

	// Remove from byName and byPos
	for _, sym := range symbols {
		delete(s.byName, sym.Name)
		s.byPos.Delete(sym)
	}

	// Clear document index
	delete(s.docIndex, uri)
}


// AddSymbol inserts a symbol into the table and tracks it by document.
func (s *SymbolTable) AddSymbol(symbol *Symbol) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.byName[symbol.Name] = symbol
	s.byPos.ReplaceOrInsert(symbol)

	// Track in docIndex for easy removal later
	s.docIndex[symbol.DefinedIn] = append(s.docIndex[symbol.DefinedIn], symbol)
}

func (s *SymbolTable) ProcessAst(ast *Ast) {
	// Clear old symbols for this document
	s.RemoveSymbolsForDocument(ast.Document)

	// Traverse AST and add new symbols
	var collectSymbols func(node node)
	collectSymbols = func(n node) {
		if sym, ok := extractSymbol(n); ok {
			s.AddSymbol(sym)
		}
		for _, child := range getChildren(n) {
			collectSymbols(child)
		}
	}

	collectSymbols(ast.RootNode)
}
