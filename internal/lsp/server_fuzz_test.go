package lsp

import (
	"testing"
	"time"

	"go.lsp.dev/protocol"
)

// FuzzDidChangeDebounce verifies that a burst of DidChange events results in
// the document reflecting only the final change.
func FuzzDidChangeDebounce(f *testing.F) {
	seeds := []struct {
		open    string
		change1 string
		change2 string
	}{
		{"foo: bar", "foo: baz", "foo: qux"},
		{"a: \"1\"\n", "a: \"2\"\n", "a: \"3\"\n"},
	}
	for _, s := range seeds {
		f.Add(s.open, s.change1, s.change2, byte(0), byte(0))
	}

	f.Fuzz(func(t *testing.T, openText, change1, change2 string, d1, d2 byte) {
		env := setup(t)
		defer env.teardown()

		client := env.client
		ctx := env.ctx

		if _, err := client.Initialize(ctx, &protocol.InitializeParams{RootURI: "file:///tmp"}); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		if err := client.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
			t.Fatalf("Initialized failed: %v", err)
		}

		docURI := protocol.DocumentURI("file:///fuzz.no")
		openParams := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        docURI,
				LanguageID: "nostos",
				Version:    1,
				Text:       openText,
			},
		}
		if err := client.DidOpen(ctx, openParams); err != nil {
			t.Fatalf("DidOpen failed: %v", err)
		}
		waitForDocument(t, env.handler, docURI, openText)

		changeParams1 := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: change1}},
		}
		if err := client.DidChange(ctx, changeParams1); err != nil {
			t.Fatalf("DidChange failed: %v", err)
		}

		time.Sleep(time.Duration(d1) * time.Millisecond)

		changeParams2 := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: docURI},
				Version:                3,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: change2}},
		}
		if err := client.DidChange(ctx, changeParams2); err != nil {
			t.Fatalf("DidChange failed: %v", err)
		}

		time.Sleep(time.Duration(d2) * time.Millisecond)

		waitForDocument(t, env.handler, docURI, change2)

		// Exercise additional LSP requests on the final document.
		_, _ = client.Hover(ctx, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		})
		_, _ = client.Completion(ctx, &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		})
	})
}
