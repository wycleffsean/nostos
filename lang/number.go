package lang

type Number struct {
	Position Position
	Value    float64
}

func (n *Number) Pos() Position { return n.Position }
