package ast

type BinaryExpression struct {
	BaseNode
	Operator BinaryOperator
	Left     Expression
	Right    Expression
}

func (*BinaryExpression) node()           {}
func (*BinaryExpression) expressionNode() {}

type ComparisonExpression struct {
	BaseNode
	Operator ComparisonOperator
	Left     Expression
	Right    Expression
}

func (*ComparisonExpression) node()           {}
func (*ComparisonExpression) expressionNode() {}

type LogicalExpression struct {
	BaseNode
	Operator LogicalOperator
	Left     Expression
	Right    Expression
}

func (*LogicalExpression) node()           {}
func (*LogicalExpression) expressionNode() {}

type NullishCoalesceExpression struct {
	BaseNode
	Left  Expression
	Right Expression
}

func (*NullishCoalesceExpression) node()           {}
func (*NullishCoalesceExpression) expressionNode() {}

type ConditionalExpression struct {
	BaseNode
	Condition   Expression
	Consequent  Expression
	Alternative Expression
}

func (*ConditionalExpression) node()           {}
func (*ConditionalExpression) expressionNode() {}

type CommaExpression struct {
	BaseNode
	Left  Expression
	Right Expression
}

func (*CommaExpression) node()           {}
func (*CommaExpression) expressionNode() {}

type UnaryExpression struct {
	BaseNode
	Operator UnaryOperator
	Right    Expression
}

func (*UnaryExpression) node()           {}
func (*UnaryExpression) expressionNode() {}

type UpdateExpression struct {
	BaseNode
	Operator UpdateOperator
	Target   Expression
	Prefix   bool
}

func (*UpdateExpression) node()           {}
func (*UpdateExpression) expressionNode() {}

type TypeofExpression struct {
	BaseNode
	Value Expression
}

func (*TypeofExpression) node()           {}
func (*TypeofExpression) expressionNode() {}

type AwaitExpression struct {
	BaseNode
	Value Expression
}

func (*AwaitExpression) node()           {}
func (*AwaitExpression) expressionNode() {}

type YieldExpression struct {
	BaseNode
	Value    Expression
	Delegate bool
}

func (*YieldExpression) node()           {}
func (*YieldExpression) expressionNode() {}

type InstanceofExpression struct {
	BaseNode
	Left  Expression
	Right Expression
}

func (*InstanceofExpression) node()           {}
func (*InstanceofExpression) expressionNode() {}
