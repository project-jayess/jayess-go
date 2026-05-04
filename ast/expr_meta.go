package ast

type ImportMetaExpression struct {
	BaseNode
}

func (*ImportMetaExpression) node()           {}
func (*ImportMetaExpression) expressionNode() {}
