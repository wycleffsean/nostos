package lang

import (
	"sync"

	"github.com/google/btree"
	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/pkg/types"
)

// Symbol represents a language entity (variable, function, type, etc.).
type SymbolEntry struct {
	Symbol *Symbol
	// Name      string
	// Kind      protocol.SymbolKind
	Begin     Position
	End       Position
	Type      *types.TypeDefinition
	DefinedIn uri.URI
}

func (s *SymbolEntry) Less(than btree.Item) bool {
	o := than.(*SymbolEntry)
	return s.Begin.Less(o.Begin)
}

// SymbolTable is a concurrency-safe table of symbols.
type SymbolTable struct {
	mu       sync.RWMutex
	byName   map[string]*SymbolEntry    // Lookup by name
	byPos    *btree.BTree               // Positional lookup
	typeReg  *types.Registry            // Reference to the type registry
	docIndex map[uri.URI][]*SymbolEntry // Tracks symbols by document for easy removal
}

// LookupByName retrieves a symbol entry by its name.
func (s *SymbolTable) LookupByName(name string) (*SymbolEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.byName[name]
	return entry, ok
}

// LookupByPosition retrieves a symbol entry by its starting position.
func (s *SymbolTable) LookupByPosition(pos Position) (*SymbolEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item := &SymbolEntry{Begin: pos}
	found := s.byPos.Get(item)
	if found == nil {
		return nil, false
	}
	return found.(*SymbolEntry), true
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
		delete(s.byName, sym.Symbol.Text)
		s.byPos.Delete(sym)
	}

	// Clear document index
	delete(s.docIndex, uri)
}

// AddSymbol inserts a symbol into the table and tracks it by document.
func (s *SymbolTable) AddSymbol(symbol *Symbol, doc uri.URI) {
	s.mu.Lock()
	defer s.mu.Unlock()

	symbolEntry := SymbolEntry{
		Symbol:    symbol,
		Begin:     symbol.Position,
		DefinedIn: doc,
	}

	s.byName[symbol.Text] = &symbolEntry
	s.byPos.ReplaceOrInsert(&symbolEntry)

	// Track in docIndex for easy removal later
	s.docIndex[doc] = append(s.docIndex[doc], &symbolEntry)
}

func (s *SymbolTable) ProcessAst(ast *Ast) {
	// Clear old symbols for this document
	s.RemoveSymbolsForDocument(ast.Document)

	// Traverse AST and add new symbols
	for _, symbol := range ast.ExtractSymbols() {
		s.AddSymbol(symbol, ast.Document)
	}
}
func (s *SymbolTable) SymbolsForDocument(doc uri.URI) []*SymbolEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.docIndex[doc]
	out := make([]*SymbolEntry, len(entries))
	copy(out, entries)
	return out
}

func (s *SymbolTable) SymbolAt(doc uri.URI, pos Position) (*SymbolEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, entry := range s.docIndex[doc] {
		if entry.Begin == pos {
			return entry, true
		}
	}
	return nil, false
}
