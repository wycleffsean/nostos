package lang

// Function represents a lambda expression with a single parameter.
type Function struct {
	Param *Symbol
	Body  node
}

func (f *Function) Pos() Position { return f.Param.Pos() }

func (f *Function) Symbols() []node {
	return []node{f.Param, f.Body}
}
