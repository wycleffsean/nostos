package lang

// Shovel represents the infix `<<` operator.
// It simply links a left and right expression.
type Shovel struct {
	Left  node
	Right node
}

func (s *Shovel) Pos() Position { return s.Left.Pos() }

func (s *Shovel) leftExpr() node { return s.Left }

func (s *Shovel) rightExpr() node { return s.Right }
