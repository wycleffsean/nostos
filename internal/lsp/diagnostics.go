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

func collectDiagnostics(n interface{}, diags *[]protocol.Diagnostic) {
	switch t := n.(type) {
	case *lang.ParseError:
		*diags = append(*diags, diagnosticFromParseError(t))
	case *lang.List:
		for _, c := range *t {
			collectDiagnostics(c, diags)
		}
	case *lang.Map:
		for k, v := range *t {
			collectDiagnostics(&k, diags)
			collectDiagnostics(v, diags)
		}
	case *lang.Call:
		collectDiagnostics(t.Func, diags)
		collectDiagnostics(t.Arg, diags)
	case *lang.Function:
		collectDiagnostics(t.Param, diags)
		collectDiagnostics(t.Body, diags)
	case *lang.Shovel:
		collectDiagnostics(t.Left, diags)
		collectDiagnostics(t.Right, diags)
	}
}

func evalForDiagnostics(n interface{}, base string) ([]protocol.Diagnostic, interface{}) {
	val, err := vm.EvalWithDir(n, base)
	if err != nil {
		return []protocol.Diagnostic{diagnosticFromError(err)}, nil
	}
	return nil, val
}
