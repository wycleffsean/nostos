package lsp

import (
	"context"
	"sync"
	"sync/atomic"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/types"
)

var log *zap.Logger

type ServerState struct {
	mu               sync.RWMutex
	client           protocol.Client
	projectRoot      uri.URI
	workspaceFolders []uri.URI
	registryReady    chan *types.Registry

	registry    *types.Registry
	documents   map[protocol.DocumentURI]string
	symbolTable atomic.Pointer[lang.SymbolTable]

	indexer *indexer
}

// https://pkg.go.dev/go.lsp.dev/protocol#Server
// this is the interface - implementing methods for this interface
// creates the features for the LSP server

type Handler struct {
	protocol.Server
	state *ServerState
}

func NewHandler(ctx context.Context, server protocol.Server, logger *zap.Logger) (Handler, context.Context, error) {
	log = logger
	// Do initialization logic here, including
	// stuff like setting state variables
	// by returning a new context with
	// context.WithValue(context, ...)
	// instead of just context
	state := &ServerState{
		mu:               sync.RWMutex{},
		documents:        make(map[protocol.DocumentURI]string),
		registryReady:    make(chan *types.Registry),
		workspaceFolders: []uri.URI{},
	}
	state.indexer = newIndexer(state)

	return Handler{Server: server, state: state}, ctx, nil
}

func (h Handler) Initialize(ctx context.Context, params *protocol.InitializeParams) (result *protocol.InitializeResult, err error) {
	h.state.mu.Lock()
	if params.RootURI != "" {
		h.state.projectRoot = uri.URI(params.RootURI)
	} else {
		h.state.projectRoot = uri.URI(params.RootPath)
	}
	h.state.workspaceFolders = nil
	for _, wf := range params.WorkspaceFolders {
		h.state.workspaceFolders = append(h.state.workspaceFolders, uri.URI(wf.URI))
	}
	h.state.mu.Unlock()

	log.Info("LSP initialized", zap.String("projectRoot", string(h.state.projectRoot)))

	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			// CallHierarchyProvider:            h,
			// CodeActionProvider:               nil,
			// CodeLensProvider:                 &protocol.CodeLensOptions{},
			// ColorProvider:                    nil,
			// CompletionProvider:               &protocol.CompletionOptions{},
			// DeclarationProvider:              nil,
			DefinitionProvider: true,
			// DocumentFormattingProvider:       nil,
			// DocumentHighlightProvider:        h,
			// DocumentLinkProvider:             &protocol.DocumentLinkOptions{},
			// DocumentOnTypeFormattingProvider: &protocol.DocumentOnTypeFormattingOptions{},
			// DocumentRangeFormattingProvider:  nil,
			DocumentSymbolProvider: true,
			// ExecuteCommandProvider:           &protocol.ExecuteCommandOptions{},
			// Experimental:                     nil,
			// FoldingRangeProvider:             nil,
			// HoverProvider:                    h,
			// ImplementationProvider:           nil,
			// LinkedEditingRangeProvider:       err,
			// MonikerProvider:                  nil,
			// ReferencesProvider:               nil,
			// RenameProvider:                   nil,
			// SelectionRangeProvider:           nil,
			// SemanticTokensProvider:           nil,
			// SignatureHelpProvider:            &protocol.SignatureHelpOptions{},
			// TextDocumentSync:                 nil,
			// TypeDefinitionProvider:           nil,
			// Workspace:                        &protocol.ServerCapabilitiesWorkspace{},
			// WorkspaceSymbolProvider:          nil,
		},
		ServerInfo: &protocol.ServerInfo{
			Name:    "nostos",
			Version: "0.1.0",
		},
	}, nil
}

func (h Handler) Initialized(ctx context.Context, params *protocol.InitializedParams) (err error) {
	log.Debug("###### Initialized")
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		StartRegistryWorker(ctx, h.state)
		if h.state.indexer != nil {
			h.state.indexer.indexWorkspace(ctx)
		}
		return nil
	}
}

func (h Handler) DidChangeConfiguration(ctx context.Context, params *protocol.DidChangeConfigurationParams) (err error) {
	log.Debug("###### DidChangeConfiguration")
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (h Handler) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) (err error) {
	log.Debug("###### DidOpen")
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if h.state.indexer != nil {
			h.state.indexer.didOpen <- *params
		}
		return nil
	}
}

func (h Handler) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	log.Debug("###### DidChange")
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if h.state.indexer != nil {
			h.state.indexer.didChange <- *params
		}
		return nil
	}
}

func (h Handler) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) ([]interface{}, error) {
	log.Debug("###### DocumentSymbol")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		st := h.state.indexer.currentSymbolTable()
		if st == nil {
			return nil, nil
		}
		entries := st.SymbolsForDocument(params.TextDocument.URI)
		result := make([]interface{}, 0, len(entries))
		for _, e := range entries {
			ds := protocol.DocumentSymbol{
				Name: e.Symbol.Text,
				Kind: protocol.SymbolKindVariable,
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(e.Begin.LineNumber), Character: uint32(e.Begin.CharacterOffset)},
					End:   protocol.Position{Line: uint32(e.End.LineNumber), Character: uint32(e.End.CharacterOffset)},
				},
				SelectionRange: protocol.Range{
					Start: protocol.Position{Line: uint32(e.Begin.LineNumber), Character: uint32(e.Begin.CharacterOffset)},
					End:   protocol.Position{Line: uint32(e.End.LineNumber), Character: uint32(e.End.CharacterOffset)},
				},
			}
			result = append(result, ds)
		}
		return result, nil
	}
}

// IMPORTANT: You _can't_ take a pointer to your handler struct as the receiver,
// your handler will no longer implement protocol.Server if you do that.
func (h Handler) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	log.Debug("###### CALLING Definition")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		st := h.state.indexer.currentSymbolTable()
		if st == nil {
			return nil, nil
		}
		pos := lang.Position{LineNumber: uint(params.Position.Line), CharacterOffset: uint(params.Position.Character)}
		entry, ok := st.SymbolAt(params.TextDocument.URI, pos)
		if !ok {
			return nil, nil
		}
		loc := protocol.Location{
			URI:   protocol.DocumentURI(entry.DefinedIn),
			Range: protocol.Range{Start: protocol.Position{Line: uint32(entry.Begin.LineNumber), Character: uint32(entry.Begin.CharacterOffset)}, End: protocol.Position{Line: uint32(entry.End.LineNumber), Character: uint32(entry.End.CharacterOffset)}},
		}
		return []protocol.Location{loc}, nil
	}
}

func (h Handler) Shutdown(ctx context.Context) (err error) {
	log.Debug("###### Shutdown")
	return nil
}

func (h Handler) Exit(ctx context.Context) (err error) {
	log.Debug("###### Exit")
	return nil
}
