package lang

import "github.com/wycleffsean/nostos/pkg/urispec"

// Path represents a URI specification literal.
type Path struct {
	Position Position
	Spec     urispec.Spec
}

func (p *Path) Pos() Position { return p.Position }
