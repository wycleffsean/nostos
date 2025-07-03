package lsp

import (
	"github.com/wycleffsean/nostos/lang"
	"github.com/wycleffsean/nostos/vm"
	"go.lsp.dev/protocol"
)

func langPosToProtocol(p lang.Position) protocol.Position {
	return protocol.Position{Line: uint32(p.LineNumber), Character: uint32(p.CharacterOffset)}
}

func diagnosticFromParseError(e *lang.ParseError) protocol.Diagnostic {
	pos := langPosToProtocol(e.Pos())
	rng := protocol.Range{Start: pos, End: pos}
	return protocol.Diagnostic{
		Range:    rng,
		Severity: protocol.DiagnosticSeverityError,
		Source:   "nostos",
		Message:  e.Error(),
	}
}

func diagnosticFromError(err error) protocol.Diagnostic {
	if pe, ok := err.(*lang.ParseError); ok {
		return diagnosticFromParseError(pe)
	}
	// fallback with zero range
	return protocol.Diagnostic{
		Range:    protocol.Range{Start: protocol.Position{}, End: protocol.Position{}},
		Severity: protocol.DiagnosticSeverityError,
		Source:   "nostos",
		Message:  err.Error(),
	}
}

func diagnosticsFromParseErrors(n interface{}) []protocol.Diagnostic {
	errs := lang.CollectParseErrors(n)
	diags := make([]protocol.Diagnostic, 0, len(errs))
	for _, e := range errs {
		diags = append(diags, diagnosticFromParseError(e))
	}
	return diags
}

func evalForDiagnostics(n interface{}, base string) ([]protocol.Diagnostic, interface{}) {
	val, err := vm.EvalWithDir(n, base)
	if err != nil {
		return []protocol.Diagnostic{diagnosticFromError(err)}, nil
	}
	return nil, val
}
