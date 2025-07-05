package lang

// Let represents a let-in expression.
type Let struct {
	Position Position
	Bindings *Map
	Body     node
}

func (l *Let) Pos() Position { return l.Position }
