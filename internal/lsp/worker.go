package lsp

import (
	"context"
	"path/filepath"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/lang"
)

// DocumentChangeMsg represents a text document change delivered to the worker.
type DocumentChangeMsg struct {
	URI     protocol.DocumentURI
	Version int32
	Text    string
}

// DiagnosticSnapshot holds the latest processed AST and diagnostics.
type DiagnosticSnapshot struct {
	AST         *lang.Ast
	Diagnostics []protocol.Diagnostic
}

// StartWorkerLoop launches a goroutine that processes document change events and
// publishes diagnostics to the client.
func StartWorkerLoop(ctx context.Context, state *ServerState) {
	go func() {
		logger := state.logger.Sugar()
		documents := make(map[protocol.DocumentURI]string)
		for {
			select {
			case <-ctx.Done():
				return
			case change := <-state.DidChangeChan:
				logger.Infof("Processing change for %s", change.URI)
				documents[change.URI] = change.Text

				ast := lang.NewAst(change.Text, uri.URI(change.URI))
				diagnostics := diagnosticsFromParseErrors(ast.RootNode)
				evalDiags, val := evalForDiagnostics(ast.RootNode, filepath.Dir(uri.URI(change.URI).Filename()), uri.URI(change.URI))
				if len(evalDiags) > 0 {
					diagnostics = append(diagnostics, evalDiags...)
				}
				if filepath.Base(uri.URI(change.URI).Filename()) == "odyssey.no" && len(evalDiags) == 0 {
					state.mu.Lock()
					state.odyssey = val
					state.mu.Unlock()
				}

				snapshot := &DiagnosticSnapshot{AST: &ast, Diagnostics: diagnostics}
				state.diagnosticSnapshot.Store(snapshot)

				state.mu.Lock()
				state.documents[change.URI] = change.Text
				state.diagnostics[change.URI] = diagnostics
				state.mu.Unlock()

				if state.client != nil {
					_ = state.client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
						URI:         change.URI,
						Diagnostics: diagnostics,
					})
				}
			}
		}
	}()
}
