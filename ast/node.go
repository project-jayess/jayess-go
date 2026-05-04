package ast

type Node interface {
	node()
}

type SourcePos struct {
	Line   int
	Column int
}

type BaseNode struct {
	Pos SourcePos
}

func (b BaseNode) SourcePosition() SourcePos {
	return b.Pos
}

type Positioned interface {
	SourcePosition() SourcePos
}

func PositionOf(value any) SourcePos {
	if positioned, ok := value.(Positioned); ok {
		return positioned.SourcePosition()
	}
	return SourcePos{}
}

type Expression interface {
	Node
	expressionNode()
}
