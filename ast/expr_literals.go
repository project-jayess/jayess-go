package ast

type Identifier struct {
	BaseNode
	Name string
}

func (*Identifier) node()           {}
func (*Identifier) expressionNode() {}

type ThisExpression struct{ BaseNode }

func (*ThisExpression) node()           {}
func (*ThisExpression) expressionNode() {}

type SuperExpression struct{ BaseNode }

func (*SuperExpression) node()           {}
func (*SuperExpression) expressionNode() {}

type NumberLiteral struct {
	BaseNode
	Value string
}

func (*NumberLiteral) node()           {}
func (*NumberLiteral) expressionNode() {}

type BigIntLiteral struct {
	BaseNode
	Value string
}

func (*BigIntLiteral) node()           {}
func (*BigIntLiteral) expressionNode() {}

type StringLiteral struct {
	BaseNode
	Value string
}

func (*StringLiteral) node()           {}
func (*StringLiteral) expressionNode() {}

type TemplateLiteral struct {
	BaseNode
	Value       string
	Expressions []Expression
}

func (*TemplateLiteral) node()           {}
func (*TemplateLiteral) expressionNode() {}

type BooleanLiteral struct {
	BaseNode
	Value bool
}

func (*BooleanLiteral) node()           {}
func (*BooleanLiteral) expressionNode() {}

type NullLiteral struct{ BaseNode }

func (*NullLiteral) node()           {}
func (*NullLiteral) expressionNode() {}

type UndefinedLiteral struct{ BaseNode }

func (*UndefinedLiteral) node()           {}
func (*UndefinedLiteral) expressionNode() {}
