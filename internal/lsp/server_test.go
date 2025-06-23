package lsp

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/jsonrpc2/fake"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// testStreamServer adapts the lsp.StartServer logic for in-memory streams.
type testStreamServer struct{ logger *zap.Logger }

// lspTestEnv holds resources for an LSP test instance.
type lspTestEnv struct {
	ctx    context.Context
	cancel context.CancelFunc
       server *fake.PipeServer
       conn   jsonrpc2.Conn
       client protocol.Server
}

// setup spins up a new in-memory LSP server and returns a test environment.
func setup(t *testing.T) *lspTestEnv {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

	var buf bytes.Buffer
	encCfg := zap.NewDevelopmentEncoderConfig()
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), zapcore.AddSync(&buf), zap.DebugLevel))

	server := fake.NewPipeServer(ctx, testStreamServer{logger: logger}, nil)
	conn := server.Connect(ctx)
	conn.Go(ctx, jsonrpc2.MethodNotFoundHandler)

	client := protocol.ServerDispatcher(conn, logger)

	return &lspTestEnv{
		ctx:    ctx,
		cancel: cancel,
		server: server,
		conn:   conn,
		client: client,
	}
}

func (e *lspTestEnv) teardown() {
	_ = e.client.Shutdown(e.ctx)
	_ = e.client.Exit(e.ctx)
	e.conn.Close()
	e.server.Close()
	e.cancel()
}

func (t testStreamServer) ServeStream(ctx context.Context, conn jsonrpc2.Conn) error {
	logger := t.logger
	handler, ctx, err := NewHandler(ctx, protocol.ServerDispatcher(conn, logger), logger)
	if err != nil {
		return err
	}
	handler.state.client = protocol.ClientDispatcher(conn, logger)
	conn.Go(ctx, protocol.ServerHandler(handler, jsonrpc2.MethodNotFoundHandler))
	<-conn.Done()
	return conn.Err()
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
