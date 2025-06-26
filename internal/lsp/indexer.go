package lsp

import (
	"context"
	"path/filepath"

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
	case reg = <-a.state.registryReady:
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

	a.state.mu.RLock()
	docs := make(map[uri.URI]string, len(a.state.documents))
	for u, t := range a.state.documents {
		docs[u] = t
	}
	a.state.mu.RUnlock()

	var st *lang.SymbolTable
	if reg != nil {
		st = lang.NewSymbolTable(reg)
	}

	for u, text := range docs {
		ast := lang.NewAst(text, u)
		if st != nil {
			st.ProcessAst(&ast)
		}

		diags := []protocol.Diagnostic{}
		collectDiagnostics(ast.RootNode, &diags)

		evalDiags, val := evalForDiagnostics(ast.RootNode, filepath.Dir(u.Filename()))
		if len(evalDiags) > 0 {
			diags = append(diags, evalDiags...)
		}

		if filepath.Base(u.Filename()) == "odyssey.no" && len(evalDiags) == 0 {
			a.state.mu.Lock()
			a.state.odyssey = val
			a.state.mu.Unlock()
		}

		a.state.mu.Lock()
		a.state.diagnostics[protocol.DocumentURI(u)] = diags
		a.state.mu.Unlock()

		if a.state.client != nil {
			_ = a.state.client.PublishDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{
				URI:         protocol.DocumentURI(u),
				Diagnostics: diags,
			})
		}
	}
	if st != nil {
		a.state.symbolTable.Store(st)
	}
}
