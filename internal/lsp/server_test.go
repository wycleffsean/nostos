package lsp

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wycleffsean/nostos/pkg/types"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/jsonrpc2/fake"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// testStreamServer adapts the lsp.StartServer logic for in-memory streams.
type testStreamServer struct {
	logger  *zap.Logger
	handler *Handler
}

// lspTestEnv holds resources for an LSP test instance.
type lspTestEnv struct {
	ctx     context.Context
	cancel  context.CancelFunc
	server  *fake.PipeServer
	conn    jsonrpc2.Conn
	client  protocol.Server
	handler *Handler
}

// setup spins up a new in-memory LSP server and returns a test environment.
func setup(t *testing.T) *lspTestEnv {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	var buf bytes.Buffer
	encCfg := zap.NewDevelopmentEncoderConfig()
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(&buf), zap.DebugLevel))

	var handler Handler
	server := fake.NewPipeServer(ctx, testStreamServer{logger: logger, handler: &handler}, nil)
	conn := server.Connect(ctx)
	conn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

	client := protocol.ServerDispatcher(conn, logger)

	return &lspTestEnv{
		ctx:     ctx,
		cancel:  cancel,
		server:  server,
		conn:    conn,
		client:  client,
		handler: &handler,
	}
}

func (e *lspTestEnv) teardown() {
	_ = e.client.Shutdown(e.ctx)
	_ = e.client.Exit(e.ctx)
	_ = e.conn.Close()
	_ = e.server.Close()
	e.cancel()
}

func (t testStreamServer) ServeStream(ctx context.Context, conn jsonrpc2.Conn) error {
	logger := t.logger
	handler, ctx, err := NewHandler(ctx, protocol.ServerDispatcher(conn, logger), logger)
	if err != nil {
		return err
	}
	handler.state.client = protocol.ClientDispatcher(conn, logger)
	handler.state.indexer.start(ctx)
	if t.handler != nil {
		*t.handler = handler
	}
	conn.Go(ctx, protocol.ServerHandler(handler, jsonrpc2.MethodNotFoundHandler))
	<-conn.Done()
	return conn.Err()
}

// waitForDocument polls the handler's document store until the text for uri
// matches expect or the timeout expires.
func waitForDocument(t *testing.T, h *Handler, uri protocol.DocumentURI, expect string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		h.state.mu.RLock()
		got, ok := h.state.documents[uri]
		h.state.mu.RUnlock()
		if ok && got == expect {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("document %s did not reach expected text", uri)
}

func TestInitializeAndInitialized(t *testing.T) {
	env := setup(t)
	defer env.teardown()

	client := env.client
	ctx := env.ctx

	params := &protocol.InitializeParams{RootURI: "file:///tmp"}
	res, err := client.Initialize(ctx, params)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if res == nil {
		t.Fatal("initialize returned nil")
	}
	dp, ok := res.Capabilities.DefinitionProvider.(bool)
	if !ok || !dp {
		t.Fatalf("unexpected initialize result: %#v", res)
	}

	if err := client.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("Initialized failed: %v", err)
	}

}

func TestDidChange(t *testing.T) {
	env := setup(t)
	defer env.teardown()

	client := env.client
	ctx := env.ctx

	_, err := client.Initialize(ctx, &protocol.InitializeParams{RootURI: "file:///tmp"})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := client.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("Initialized failed: %v", err)
	}

	env.handler.state.mu.Lock()
	env.handler.state.registry = types.DefaultRegistry()
	env.handler.state.mu.Unlock()
	env.handler.state.indexer.reindex()

	docURI := protocol.DocumentURI("file:///foo.no")
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "nostos",
			Version:    1,
			Text:       "a: \"1\"\n",
		},
	}
	if err := client.DidOpen(ctx, openParams); err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}
	waitForDocument(t, env.handler, docURI, openParams.TextDocument.Text)
	waitForDocument(t, env.handler, docURI, "a: \"1\"\n")

	changeParams := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
			Version:                2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: "a: \"2\"\n"}},
	}
	if err := client.DidChange(ctx, changeParams); err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}

	waitForDocument(t, env.handler, docURI, "a: \"2\"\n")
}

func TestDefinition(t *testing.T) {
	env := setup(t)
	defer env.teardown()

	client := env.client
	ctx := env.ctx

	_, err := client.Initialize(ctx, &protocol.InitializeParams{RootURI: "file:///tmp"})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := client.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("Initialized failed: %v", err)
	}

	docURI := protocol.DocumentURI("file:///foo.no")
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "nostos",
			Version:    1,
			Text:       "foo: \"1\"\n",
		},
	}
	if err := client.DidOpen(ctx, openParams); err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}
	env.handler.state.indexer.reindex()
	waitForSymbol(t, env.handler, "foo")

	locs, err := client.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	})
	if err != nil {
		t.Fatalf("Definition failed: %v", err)
	}
	if len(locs) != 1 || locs[0].URI != docURI {
		t.Fatalf("unexpected definition result: %#v", locs)
	}
}

func TestHoverCompletionAndCodeAction(t *testing.T) {
	env := setup(t)
	defer env.teardown()

	client := env.client
	ctx := env.ctx

	_, err := client.Initialize(ctx, &protocol.InitializeParams{RootURI: "file:///tmp"})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := client.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("Initialized failed: %v", err)
	}

	docURI := protocol.DocumentURI("file:///svc.yaml")
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "nostos",
			Version:    1,
			Text:       "apiVersion: v1\nkind: Service\n",
		},
	}
	if err := client.DidOpen(ctx, openParams); err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}
	waitForDocument(t, env.handler, docURI, openParams.TextDocument.Text)

	hover, err := client.Hover(ctx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	})
	if err != nil || hover == nil {
		t.Fatalf("Hover failed: %v", err)
	}
	if !strings.Contains(hover.Contents.Value, "Service") {
		t.Fatalf("unexpected hover: %#v", hover)
	}

	comp, err := client.Completion(ctx, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 1, Character: 6},
		},
	})
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}
	found := false
	for _, item := range comp.Items {
		if item.Label == "Service" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("completion did not contain Service: %#v", comp.Items)
	}

	acts, err := client.CodeAction(ctx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		},
	})
	if err != nil {
		t.Fatalf("CodeAction failed: %v", err)
	}
	if len(acts) == 0 {
		t.Fatalf("expected code action")
	}
}

func TestNestedCodeAction(t *testing.T) {
	env := setup(t)
	defer env.teardown()

	client := env.client
	ctx := env.ctx

	_, err := client.Initialize(ctx, &protocol.InitializeParams{RootURI: "file:///tmp"})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := client.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("Initialized failed: %v", err)
	}

	docURI := protocol.DocumentURI("file:///svc.yaml")
	text := "apiVersion: v1\nkind: Service\nmetadata:\n  name: foo\nspec:\n"
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "nostos",
			Version:    1,
			Text:       text,
		},
	}
	if err := client.DidOpen(ctx, openParams); err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}
	waitForDocument(t, env.handler, docURI, text)

	acts, err := client.CodeAction(ctx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 0},
			End:   protocol.Position{Line: 4, Character: 0},
		},
	})
	if err != nil {
		t.Fatalf("CodeAction failed: %v", err)
	}
	if len(acts) == 0 {
		t.Fatalf("expected code action")
	}
	edits := acts[0].Edit.Changes[docURI]
	if len(edits) == 0 || !strings.Contains(edits[0].NewText, "type:") {
		t.Fatalf("unexpected code action edits: %#v", edits)
	}
}

func waitForSymbol(t *testing.T, h *Handler, name string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st := h.state.symbolTable.Load()
		if st != nil {
			if _, ok := st.LookupByName(name); ok {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("symbol %s not indexed", name)
}

func TestWorkspaceIndexingAndSymbols(t *testing.T) {
	env := setup(t)
	defer env.teardown()

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "a.yaml"), []byte("foo: \"1\"\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, "b.yaml"), []byte("bar: \"2\"\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	client := env.client
	ctx := env.ctx

	rootURI := protocol.DocumentURI(uri.File(filepath.ToSlash(dir)))
	_, err = client.Initialize(ctx, &protocol.InitializeParams{RootURI: rootURI})
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}
	if err := client.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("Initialized failed: %v", err)
	}

	env.handler.state.mu.Lock()
	env.handler.state.registry = types.DefaultRegistry()
	env.handler.state.mu.Unlock()
	env.handler.state.indexer.reindex()

	waitForSymbol(t, env.handler, "foo")
	waitForSymbol(t, env.handler, "bar")

	docURI := protocol.DocumentURI(uri.File(filepath.Join(dir, "a.yaml")))
	raw, err := client.DocumentSymbol(ctx, &protocol.DocumentSymbolParams{TextDocument: protocol.TextDocumentIdentifier{URI: docURI}})
	if err != nil {
		t.Fatalf("DocumentSymbol failed: %v", err)
	}
	if len(raw) != 1 {
		t.Fatalf("unexpected document symbols: %#v", raw)
	}
	m, ok := raw[0].(map[string]interface{})
	if !ok || m["name"] != "foo" {
		t.Fatalf("unexpected document symbols: %#v", raw)
	}

	ws, err := client.Symbols(ctx, &protocol.WorkspaceSymbolParams{Query: ""})
	if err != nil {
		t.Fatalf("WorkspaceSymbol failed: %v", err)
	}
	if len(ws) < 2 {
		t.Fatalf("expected workspace symbols, got %#v", ws)
	}

	// modify a file and ensure symbols update
	change := &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
			Version:                2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: "baz: \"3\"\n"}},
	}
	if err := client.DidChange(ctx, change); err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}
	waitForSymbol(t, env.handler, "baz")
	st := env.handler.state.symbolTable.Load()
	if st == nil {
		t.Fatalf("symbol table nil")
	}
	if _, ok := st.LookupByName("foo"); ok {
		t.Fatalf("old symbol should be removed")
	}
}
