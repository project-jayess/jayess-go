package ast

type Node interface {
	node()
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type DeclarationKind string

const (
	DeclarationVar   DeclarationKind = "var"
	DeclarationLet   DeclarationKind = "let"
	DeclarationConst DeclarationKind = "const"
)

type BinaryOperator string
type ComparisonOperator string
type LogicalOperator string
type UnaryOperator string

const (
	OperatorAdd BinaryOperator = "+"
	OperatorSub BinaryOperator = "-"
	OperatorMul BinaryOperator = "*"
	OperatorDiv BinaryOperator = "/"
)

const (
	OperatorEq  ComparisonOperator = "=="
	OperatorNe  ComparisonOperator = "!="
	OperatorLt  ComparisonOperator = "<"
	OperatorLte ComparisonOperator = "<="
	OperatorGt  ComparisonOperator = ">"
	OperatorGte ComparisonOperator = ">="
)

const (
	OperatorAnd LogicalOperator = "&&"
	OperatorOr  LogicalOperator = "||"
)

const (
	OperatorNot UnaryOperator = "!"
)

type Program struct {
	Globals         []*VariableDecl
	ExternFunctions []*ExternFunctionDecl
	Functions       []*FunctionDecl
}

func (*Program) node() {}

type Parameter struct {
	Name string
}

type FunctionDecl struct {
	Visibility Visibility
	Name       string
	Params     []Parameter
	Body       []Statement
}

func (*FunctionDecl) node() {}

type ExternFunctionDecl struct {
	Name         string
	NativeSymbol string
	Params       []Parameter
	Variadic     bool
}

func (*ExternFunctionDecl) node() {}

type VariableDecl struct {
	Visibility Visibility
	Kind       DeclarationKind
	Name       string
	Value      Expression
}

func (*VariableDecl) node()          {}
func (*VariableDecl) statementNode() {}

type AssignmentStatement struct {
	Target Expression
	Value  Expression
}

func (*AssignmentStatement) node()          {}
func (*AssignmentStatement) statementNode() {}

type ReturnStatement struct {
	Value Expression
}

func (*ReturnStatement) node()          {}
func (*ReturnStatement) statementNode() {}

type IfStatement struct {
	Condition   Expression
	Consequence []Statement
	Alternative []Statement
}

func (*IfStatement) node()          {}
func (*IfStatement) statementNode() {}

type WhileStatement struct {
	Condition Expression
	Body      []Statement
}

func (*WhileStatement) node()          {}
func (*WhileStatement) statementNode() {}

type ForStatement struct {
	Init      Statement
	Condition Expression
	Update    Statement
	Body      []Statement
}

func (*ForStatement) node()          {}
func (*ForStatement) statementNode() {}

type BreakStatement struct{}

func (*BreakStatement) node()          {}
func (*BreakStatement) statementNode() {}

type ContinueStatement struct{}

func (*ContinueStatement) node()          {}
func (*ContinueStatement) statementNode() {}

type ExpressionStatement struct {
	Expression Expression
}

func (*ExpressionStatement) node()          {}
func (*ExpressionStatement) statementNode() {}

type NumberLiteral struct {
	Value float64
}

func (*NumberLiteral) node()           {}
func (*NumberLiteral) expressionNode() {}

type BooleanLiteral struct {
	Value bool
}

func (*BooleanLiteral) node()           {}
func (*BooleanLiteral) expressionNode() {}

type NullLiteral struct{}

func (*NullLiteral) node()           {}
func (*NullLiteral) expressionNode() {}

type UndefinedLiteral struct{}

func (*UndefinedLiteral) node()           {}
func (*UndefinedLiteral) expressionNode() {}

type StringLiteral struct {
	Value string
}

func (*StringLiteral) node()           {}
func (*StringLiteral) expressionNode() {}

type ObjectProperty struct {
	Key   string
	Value Expression
}

type ObjectLiteral struct {
	Properties []ObjectProperty
}

func (*ObjectLiteral) node()           {}
func (*ObjectLiteral) expressionNode() {}

type ArrayLiteral struct {
	Elements []Expression
}

func (*ArrayLiteral) node()           {}
func (*ArrayLiteral) expressionNode() {}

type Identifier struct {
	Name string
}

func (*Identifier) node()           {}
func (*Identifier) expressionNode() {}

type CallExpression struct {
	Callee    string
	Arguments []Expression
}

func (*CallExpression) node()           {}
func (*CallExpression) expressionNode() {}

type BinaryExpression struct {
	Operator BinaryOperator
	Left     Expression
	Right    Expression
}

func (*BinaryExpression) node()           {}
func (*BinaryExpression) expressionNode() {}

type ComparisonExpression struct {
	Operator ComparisonOperator
	Left     Expression
	Right    Expression
}

func (*ComparisonExpression) node()           {}
func (*ComparisonExpression) expressionNode() {}

type LogicalExpression struct {
	Operator LogicalOperator
	Left     Expression
	Right    Expression
}

func (*LogicalExpression) node()           {}
func (*LogicalExpression) expressionNode() {}

type UnaryExpression struct {
	Operator UnaryOperator
	Right    Expression
}

func (*UnaryExpression) node()           {}
func (*UnaryExpression) expressionNode() {}

type IndexExpression struct {
	Target Expression
	Index  Expression
}

func (*IndexExpression) node()           {}
func (*IndexExpression) expressionNode() {}

type MemberExpression struct {
	Target   Expression
	Property string
}

func (*MemberExpression) node()           {}
func (*MemberExpression) expressionNode() {}
