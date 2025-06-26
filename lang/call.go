package lang

// Call represents a function call with a single argument.
type Call struct {
	Func node
	Arg  node
}

func (c *Call) Pos() Position { return c.Func.Pos() }

func (c *Call) leftExpr() node { return c.Func }

func (c *Call) rightExpr() node { return c.Arg }
