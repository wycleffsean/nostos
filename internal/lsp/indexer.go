package lsp

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/types"
)

// indexer processes document events in the background and keeps the symbol
// table up to date.
type indexer struct {
	state     *ServerState
	didOpen   chan protocol.DidOpenTextDocumentParams
	didChange chan protocol.DidChangeTextDocumentParams
}

func newIndexer(state *ServerState) *indexer {
	return &indexer{
		state:     state,
		didOpen:   make(chan protocol.DidOpenTextDocumentParams, 16),
		didChange: make(chan protocol.DidChangeTextDocumentParams, 32),
	}
}

func (a *indexer) start(ctx context.Context) {
	go a.loop(ctx)
}

func (a *indexer) indexWorkspace(ctx context.Context) {
	var roots []uri.URI
	a.state.mu.RLock()
	if a.state.projectRoot != "" {
		roots = append(roots, a.state.projectRoot)
	}
	roots = append(roots, a.state.workspaceFolders...)
	a.state.mu.RUnlock()

	for _, r := range roots {
		path := r.Filename()
		filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".no") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			u := uri.File(p)
			a.state.mu.Lock()
			a.state.documents[u] = string(data)
			a.state.mu.Unlock()
			return nil
		})
	}

	a.reindex()
}

func (a *indexer) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-a.didOpen:
			a.handleDidOpen(p)
		case p := <-a.didChange:
			// drain successive change events for debouncing
		drain:
			for {
				select {
				case p = <-a.didChange:
				default:
					break drain
				}
			}
			a.handleDidChange(p)
		}
	}
}

func (a *indexer) handleDidOpen(p protocol.DidOpenTextDocumentParams) {
	a.state.mu.Lock()
	a.state.documents[p.TextDocument.URI] = p.TextDocument.Text
	a.state.mu.Unlock()
	a.reindex()
}

func (a *indexer) handleDidChange(p protocol.DidChangeTextDocumentParams) {
	if len(p.ContentChanges) == 0 {
		return
	}
	latest := p.ContentChanges[len(p.ContentChanges)-1].Text
	a.state.mu.Lock()
	a.state.documents[p.TextDocument.URI] = latest
	a.state.mu.Unlock()
	a.reindex()
}

func (a *indexer) ensureRegistry() *types.Registry {
	a.state.mu.RLock()
	reg := a.state.registry
	a.state.mu.RUnlock()
	if reg != nil {
		return reg
	}
	select {
	case reg, _ = <-a.state.registryReady:
		a.state.mu.Lock()
		a.state.registry = reg
		a.state.mu.Unlock()
		return reg
	default:
		return nil
	}
}

func (a *indexer) reindex() {
	reg := a.ensureRegistry()
	if reg == nil {
		return
	}

	a.state.mu.RLock()
	docs := make(map[uri.URI]string, len(a.state.documents))
	for u, t := range a.state.documents {
		docs[u] = t
	}
	a.state.mu.RUnlock()

	st := lang.NewSymbolTable(reg)
	for u, text := range docs {
		ast := lang.NewAst(text, u)
		st.ProcessAst(&ast)
	}
	a.state.symbolTable.Store(st)
}

func (a *indexer) currentSymbolTable() *lang.SymbolTable {
	st := a.state.symbolTable.Load()
	if st != nil {
		return st
	}
	return nil
}
