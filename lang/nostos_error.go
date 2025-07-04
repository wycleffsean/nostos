package lang

import "go.lsp.dev/uri"

// NostosError is the common error type returned by the lexer, parser and virtual machine.
// It exposes location information and optional stack traces for evaluation errors.
type NostosError interface {
	error
	// URI of the document where the error occurred.
	URI() uri.URI
	// Pos returns the position associated with the error.
	Pos() Position
	// StackTrace returns a stack trace for runtime errors. Parsing or lexing
	// errors may return nil.
	StackTrace() []string
}
