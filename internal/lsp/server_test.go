package lsp

import (
	"context"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/jsonrpc2/fake"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// testStreamServer adapts the lsp.StartServer logic for in-memory streams.
type testStreamServer struct{ logger *zap.Logger }

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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logger := zaptest.NewLogger(t)

	server := fake.NewPipeServer(ctx, testStreamServer{logger: logger}, nil)
	defer server.Close()

	conn := server.Connect(ctx)
	conn.Go(ctx, jsonrpc2.MethodNotFoundHandler)
	client := protocol.ServerDispatcher(conn, logger)

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

	_ = client.Shutdown(ctx)
	_ = client.Exit(ctx)
	conn.Close()
}
