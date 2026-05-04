package ast

type ObjectProperty struct {
	Key       string
	KeyExpr   Expression
	Value     Expression
	Computed  bool
	Spread    bool
	Method    bool
	Getter    bool
	Setter    bool
	Shorthand bool
}

type ObjectLiteral struct {
	BaseNode
	Properties []ObjectProperty
}

func (*ObjectLiteral) node()           {}
func (*ObjectLiteral) expressionNode() {}

type ArrayLiteral struct {
	BaseNode
	Elements []Expression
}

func (*ArrayLiteral) node()           {}
func (*ArrayLiteral) expressionNode() {}

type SpreadExpression struct {
	BaseNode
	Value Expression
}

func (*SpreadExpression) node()           {}
func (*SpreadExpression) expressionNode() {}
