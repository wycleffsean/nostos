package lang

// List represents a YAML list/sequence.
type List []node

func (l *List) Pos() Position {
	if l == nil || len(*l) == 0 {
		return Position{}
	}
	pos := (*l)[0].Pos()
	for _, child := range *l {
		childPos := child.Pos()
		if childPos.Less(pos) {
			pos = childPos
		}
	}
	return pos
}

func (l *List) Symbols() []node {
	if l == nil {
		return nil
	}
	return *l
}
