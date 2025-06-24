package lsp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"

	"gopkg.in/yaml.v3"

	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/pkg/types"
	"github.com/wycleffsean/nostos/pkg/workspace"
)

var log *zap.Logger

type ServerState struct {
	mu            sync.RWMutex
	client        protocol.Client
	projectRoot   uri.URI
	registryReady chan *types.Registry

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
		mu:            sync.RWMutex{},
		documents:     make(map[protocol.DocumentURI]string),
		registryReady: make(chan *types.Registry),
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

	kind, apiVersion := extractKindAPIVersion(text)
	reg := h.state.registry
	if reg == nil {
		reg = types.DefaultRegistry()
	}
	td, found := reg.GetType("", apiVersion, kind)
	if !found {
		return nil, nil
	}

	msg := td.Kind + " (" + td.Version + ")"
	if td.Description != "" {
		msg += " - " + td.Description
	}
	contents := protocol.MarkupContent{Kind: protocol.Markdown, Value: msg}
	return &protocol.Hover{Contents: contents}, nil
}

// Completion provides simple completions for kind and apiVersion fields.
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

	lines := strings.Split(text, "\n")
	if int(params.Position.Line) >= len(lines) {
		return nil, nil
	}
	line := lines[params.Position.Line][:int(params.Position.Character)]
	line = strings.TrimSpace(line)

	reg := h.state.registry
	if reg == nil {
		reg = types.DefaultRegistry()
	}

	items := []protocol.CompletionItem{}
	if strings.HasPrefix(line, "kind:") {
		for _, td := range reg.TypeDefinitions() {
			items = append(items, protocol.CompletionItem{Label: td.Kind})
		}
	} else if strings.HasPrefix(line, "apiVersion:") {
		kind, _ := extractKindAPIVersion(text)
		versions := []string{}
		if kind != "" {
			for _, td := range reg.TypeDefinitions() {
				if td.Kind == kind {
					versions = append(versions, td.Version)
				}
			}
		}
		if len(versions) == 0 {
			for _, td := range reg.TypeDefinitions() {
				versions = append(versions, td.Version)
			}
		}
		for _, v := range versions {
			items = append(items, protocol.CompletionItem{Label: v})
		}
	}
	return &protocol.CompletionList{IsIncomplete: false, Items: items}, nil
}

// CodeAction inserts any missing required fields for the detected resource type.
func (h Handler) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	log.Debug("###### CodeAction")
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

	kind, apiVersion := extractKindAPIVersion(text)
	reg := h.state.registry
	if reg == nil {
		reg = types.DefaultRegistry()
	}
	td, found := reg.GetType("", apiVersion, kind)
	if !found {
		return nil, nil
	}

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(text), &root); err != nil {
		return nil, nil
	}

	path, node := findPathAndNode(&root, int(params.Range.Start.Line))
	if node.Kind != yaml.MappingNode {
		if node.Kind == yaml.ScalarNode && node.Value == "" && len(node.Content) == 0 && len(path) > 0 {
			// treat empty scalar as an empty mapping node
		} else {
			if len(path) == 0 {
				return nil, nil
			}
			path = path[:len(path)-1]
			node = findNodeByPath(&root, path)
			if node == nil || node.Kind != yaml.MappingNode {
				return nil, nil
			}
		}
	}

	fields := getFields(td, path)
	if len(fields) == 0 {
		return nil, nil
	}

	obj := map[string]interface{}{}
	if node.Kind == yaml.MappingNode {
		_ = node.Decode(&obj)
	}

	_, end := nodeRange(node)
	indent := node.Column - 1
	insertLine := uint32(end + 1)
	edits := []protocol.TextEdit{}

	// If we're inserting after the last line and the document doesn't end
	// with a newline, prefix the first insertion with one to avoid
	// concatenating with the previous line's text.
	lines := strings.Split(text, "\n")
	prefix := ""
	if int(insertLine) >= len(lines) && !strings.HasSuffix(text, "\n") {
		prefix = "\n"
	}

	for _, f := range fields {
		if _, ok := obj[f.Name]; ok {
			continue
		}
		insertText := prefix + strings.Repeat(" ", indent) + f.Name + ":\n"
		prefix = ""
		edits = append(edits, protocol.TextEdit{
			Range:   protocol.Range{Start: protocol.Position{Line: insertLine, Character: 0}, End: protocol.Position{Line: insertLine, Character: 0}},
			NewText: insertText,
		})
	}
	if len(edits) == 0 {
		return nil, nil
	}
	changeMap := map[protocol.DocumentURI][]protocol.TextEdit{params.TextDocument.URI: edits}
	ca := protocol.CodeAction{
		Title: "Fill required fields",
		Kind:  protocol.QuickFix,
		Edit:  &protocol.WorkspaceEdit{Changes: changeMap},
	}
	return []protocol.CodeAction{ca}, nil
}

// extractKindAPIVersion parses a manifest and returns its kind and apiVersion.
func extractKindAPIVersion(text string) (string, string) {
	var m map[string]interface{}
	if err := yaml.Unmarshal([]byte(text), &m); err != nil {
		return "", ""
	}
	kind, _ := m["kind"].(string)
	apiVersion, _ := m["apiVersion"].(string)
	return kind, apiVersion
}

// nodeRange returns the inclusive start and end line numbers (0-indexed) that a yaml node spans.
func nodeRange(n *yaml.Node) (start, end int) {
	start = n.Line - 1
	end = start
	var walk func(*yaml.Node)
	walk = func(nd *yaml.Node) {
		if nd.Line-1 > end {
			end = nd.Line - 1
		}
		for _, c := range nd.Content {
			walk(c)
		}
	}
	walk(n)
	return
}

// findNodeByPath traverses the yaml AST and returns the node for the given field path.
func findNodeByPath(root *yaml.Node, path []string) *yaml.Node {
	if len(root.Content) == 0 {
		return nil
	}
	node := root.Content[0]
	for _, p := range path {
		if node.Kind != yaml.MappingNode {
			return nil
		}
		found := false
		for i := 0; i < len(node.Content); i += 2 {
			k := node.Content[i]
			v := node.Content[i+1]
			if k.Value == p {
				node = v
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return node
}

// findPathAndNode locates the field path and node at a given line number.
func findPathAndNode(root *yaml.Node, line int) ([]string, *yaml.Node) {
	var resPath []string
	var resNode *yaml.Node
	var found bool
	var search func(n *yaml.Node, path []string)
	search = func(n *yaml.Node, path []string) {
		if found {
			return
		}
		if n.Kind == yaml.MappingNode {
			for i := 0; i < len(n.Content); i += 2 {
				k := n.Content[i]
				v := n.Content[i+1]
				if line == k.Line-1 {
					resPath = append(path, k.Value)
					resNode = v
					found = true
					return
				}
				s, e := nodeRange(v)
				if line >= s && line <= e {
					search(v, append(path, k.Value))
					if found {
						return
					}
				}
			}
		} else if n.Kind == yaml.SequenceNode {
			for i, c := range n.Content {
				s, e := nodeRange(c)
				if line >= s && line <= e {
					search(c, append(path, fmt.Sprintf("[%d]", i)))
					if found {
						return
					}
				}
			}
		}
	}
	for _, doc := range root.Content {
		s, e := nodeRange(doc)
		if line >= s && line <= e {
			search(doc, nil)
			if found {
				break
			}
		}
	}
	if !found {
		return nil, root
	}
	return resPath, resNode
}

// getFields returns the FieldDefinitions for a nested path within a TypeDefinition.
func getFields(td types.TypeDefinition, path []string) []types.FieldDefinition {
	fields := td.Fields
	for _, p := range path {
		found := false
		for _, f := range fields {
			if f.Name == p {
				fields = f.SubFields
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return fields
}
