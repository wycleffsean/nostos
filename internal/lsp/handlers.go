package lsp

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/types"
	"github.com/wycleffsean/nostos/pkg/workspace"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

var log *zap.Logger

type ServerState struct {
	mu            sync.RWMutex
	client        protocol.Client
	projectRoot   uri.URI
	registryReady chan *types.Registry

	registry    *types.Registry
	documents   map[protocol.DocumentURI]string
	diagnostics map[protocol.DocumentURI][]protocol.Diagnostic
	symbolTable atomic.Pointer[lang.SymbolTable]

	// worker infrastructure
	DidChangeChan      chan DocumentChangeMsg
	diagnosticSnapshot atomic.Pointer[DiagnosticSnapshot]

	odyssey interface{}

	indexer *indexer
	logger  *zap.Logger
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
		mu:            sync.RWMutex{},
		documents:     make(map[protocol.DocumentURI]string),
		diagnostics:   make(map[protocol.DocumentURI][]protocol.Diagnostic),
		registryReady: make(chan *types.Registry),
		DidChangeChan: make(chan DocumentChangeMsg, 32),
		logger:        logger,
	}
	state.indexer = newIndexer(state)

	return Handler{Server: server, state: state}, ctx, nil
}

func (h Handler) Initialize(ctx context.Context, params *protocol.InitializeParams) (result *protocol.InitializeResult, err error) {
	h.state.mu.Lock()
	// TODO - rootURI is deprecated, use WorkspaceFolders instead
	//nolint:staticcheck // keep compatibility with older clients
	if params.RootURI != "" {
		h.state.projectRoot = uri.URI(params.RootURI)
	} else {
		h.state.projectRoot = uri.URI(params.RootPath)
	}
	h.state.mu.Unlock()

	if h.state.projectRoot != "" {
		workspace.Set(h.state.projectRoot.Filename())
	}

	log.Info("LSP initialized", zap.String("projectRoot", string(h.state.projectRoot)))

	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			// CallHierarchyProvider:            h,
			CodeActionProvider: true,
			// CodeLensProvider:                 &protocol.CodeLensOptions{},
			// ColorProvider:                    nil,
			CompletionProvider: &protocol.CompletionOptions{},
			// DeclarationProvider:              nil,
			DefinitionProvider: true,
			// DocumentFormattingProvider:       nil,
			// DocumentHighlightProvider:        h,
			// DocumentLinkProvider:             &protocol.DocumentLinkOptions{},
			// DocumentOnTypeFormattingProvider: &protocol.DocumentOnTypeFormattingOptions{},
			// DocumentRangeFormattingProvider:  nil,
			// DocumentSymbolProvider:           nil,
			// ExecuteCommandProvider:           &protocol.ExecuteCommandOptions{},
			// Experimental:                     nil,
			// FoldingRangeProvider:             nil,
			HoverProvider: true,
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
	var latest string
	for _, change := range params.ContentChanges {
		latest = change.Text
	}

	msg := DocumentChangeMsg{
		URI:     params.TextDocument.URI,
		Version: params.TextDocument.Version,
		Text:    latest,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case h.state.DidChangeChan <- msg:
		// forwarded
	default:
		h.state.logger.Sugar().Warnf("Dropping DidChange for %s due to full channel", msg.URI)
	}
	return nil
}

// IMPORTANT: You _can't_ take a pointer to your handler struct as the receiver,
// your handler will no longer implement protocol.Server if you do that.
func (h Handler) Definition(ctx context.Context, params *protocol.DefinitionParams) ([]protocol.Location, error) {
	log.Debug("###### CALLING Definition")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		_ = h.state.symbolTable.Load()
		return []protocol.Location{
			{
				URI: uri.File("foo.no"),
				Range: protocol.Range{
					Start: protocol.Position{Line: 0, Character: 0},
					End:   protocol.Position{Line: 0, Character: 0},
				},
			},
		}, nil
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

// Hover returns basic information about the Kubernetes resource at the current position.
func (h Handler) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	log.Debug("###### Hover")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	h.state.mu.RLock()
	text, ok := h.state.documents[params.TextDocument.URI]
	h.state.mu.RUnlock()
	if !ok {
		return nil, nil
	}

	ast := lang.NewAst(text, uri.URI(params.TextDocument.URI))
	diags := diagnosticsFromParseErrors(ast.RootNode)

	var msg string
	if len(diags) > 0 {
		msg = diags[0].Message
	} else {
		evalDiags, val := evalForDiagnostics(ast.RootNode, filepath.Dir(uri.URI(params.TextDocument.URI).Filename()), uri.URI(params.TextDocument.URI))
		if len(evalDiags) > 0 {
			msg = evalDiags[0].Message
		} else {
			msg = fmt.Sprintf("%v", val)
		}
	}

	contents := protocol.MarkupContent{Kind: protocol.PlainText, Value: msg}
	return &protocol.Hover{Contents: contents}, nil
}

// Completion provides completions using parsed symbols from the document.
func (h Handler) Completion(ctx context.Context, params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	log.Debug("###### Completion")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	h.state.mu.RLock()
	text, ok := h.state.documents[params.TextDocument.URI]
	h.state.mu.RUnlock()
	if !ok {
		return nil, nil
	}

	ast := lang.NewAst(text, uri.URI(params.TextDocument.URI))
	syms := ast.ExtractSymbols()
	seen := make(map[string]struct{})
	items := []protocol.CompletionItem{}
	for _, s := range syms {
		if _, ok := seen[s.Text]; ok {
			continue
		}
		seen[s.Text] = struct{}{}
		items = append(items, protocol.CompletionItem{Label: s.Text})
	}
	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// CodeAction is currently a stub as no language-specific actions are implemented.
func (h Handler) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	log.Debug("###### CodeAction")
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return nil, nil
}
