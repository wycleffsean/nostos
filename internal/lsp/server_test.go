package lsp

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

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

// waitForDiagnostic polls the handler's diagnostics store until a diagnostic for uri
// contains the expected substring or the timeout expires.
func waitForDiagnostic(t *testing.T, h *Handler, uri protocol.DocumentURI, expect string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		h.state.mu.RLock()
		diags := h.state.diagnostics[uri]
		h.state.mu.RUnlock()
		for _, d := range diags {
			if strings.Contains(d.Message, expect) {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("diagnostic for %s did not contain %q", uri, expect)
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

func TestOdysseyEvaluation(t *testing.T) {
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

	docURI := protocol.DocumentURI("file:///odyssey.no")
	text := "cluster:\n  default:\n    - svc.yaml\n"
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

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		env.handler.state.mu.RLock()
		val := env.handler.state.odyssey
		env.handler.state.mu.RUnlock()
		if m, ok := val.(map[string]interface{}); ok {
			if _, ok := m["cluster"]; ok {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("odyssey value not evaluated")
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

	locs, err := client.Definition(ctx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: protocol.DocumentURI("file:///foo.no")},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	})
	if err != nil {
		t.Fatalf("Definition failed: %v", err)
	}
	want := []protocol.Location{{
		URI:   uri.File("foo.no"),
		Range: protocol.Range{Start: protocol.Position{Line: 0, Character: 0}, End: protocol.Position{Line: 0, Character: 0}},
	}}
	if len(locs) != 1 || locs[0] != want[0] {
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
	if !strings.Contains(hover.Contents.Value, "string") {
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

func TestCodeActionInsertNewline(t *testing.T) {
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
	text := "apiVersion: v1\nkind: Service"
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
	edits := acts[0].Edit.Changes[docURI]
	if len(edits) == 0 {
		t.Fatalf("expected edits from code action")
	}
	if !strings.HasPrefix(edits[0].NewText, "\n") {
		t.Fatalf("expected first edit to start with newline: %#v", edits)
	}
}

func TestPublishDiagnostics(t *testing.T) {
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

	docURI := protocol.DocumentURI("file:///bad.no")
	openParams := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        docURI,
			LanguageID: "nostos",
			Version:    1,
			Text:       "foo:",
		},
	}
	if err := client.DidOpen(ctx, openParams); err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	waitForDocument(t, env.handler, docURI, "foo:")
	waitForDiagnostic(t, env.handler, docURI, "ParseError")
}
