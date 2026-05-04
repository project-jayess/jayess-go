package ast

type NewExpression struct {
	BaseNode
	Callee    Expression
	Arguments []Expression
}

func (*NewExpression) node()           {}
func (*NewExpression) expressionNode() {}

type NewTargetExpression struct {
	BaseNode
}

func (*NewTargetExpression) node()           {}
func (*NewTargetExpression) expressionNode() {}
