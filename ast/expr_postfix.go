package ast

type CallExpression struct {
	BaseNode
	Callee    string
	Arguments []Expression
}

func (*CallExpression) node()           {}
func (*CallExpression) expressionNode() {}

type InvokeExpression struct {
	BaseNode
	Callee    Expression
	Arguments []Expression
	Optional  bool
}

func (*InvokeExpression) node()           {}
func (*InvokeExpression) expressionNode() {}

type IndexExpression struct {
	BaseNode
	Target   Expression
	Index    Expression
	Optional bool
}

func (*IndexExpression) node()           {}
func (*IndexExpression) expressionNode() {}

type MemberExpression struct {
	BaseNode
	Target   Expression
	Property string
	Private  bool
	Optional bool
}

func (*MemberExpression) node()           {}
func (*MemberExpression) expressionNode() {}
