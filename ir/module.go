package ir

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

type ValueKind string

const (
	ValueNumber    ValueKind = "number"
	ValueBoolean   ValueKind = "boolean"
	ValueString    ValueKind = "string"
	ValueNull      ValueKind = "null"
	ValueUndefined ValueKind = "undefined"
	ValueArgsArray ValueKind = "args_array"
	ValueArray     ValueKind = "array"
	ValueObject    ValueKind = "object"
	ValueDynamic   ValueKind = "dynamic"
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

type Module struct {
	Globals         []VariableDecl
	ExternFunctions []ExternFunction
	Functions       []Function
}

type Parameter struct {
	Name string
	Kind ValueKind
}

type Function struct {
	Visibility Visibility
	Name       string
	Params     []Parameter
	Body       []Statement
}

type ExternFunction struct {
	Name       string
	SymbolName string
	Params     []Parameter
	Variadic   bool
}

type Statement interface {
	statementNode()
}

type VariableDecl struct {
	Visibility Visibility
	Kind       DeclarationKind
	Name       string
	Value      Expression
}

func (*VariableDecl) statementNode() {}

type AssignmentStatement struct {
	Target Expression
	Value  Expression
}

func (*AssignmentStatement) statementNode() {}

type ReturnStatement struct {
	Value Expression
}

func (*ReturnStatement) statementNode() {}

type IfStatement struct {
	Condition   Expression
	Consequence []Statement
	Alternative []Statement
}

func (*IfStatement) statementNode() {}

type WhileStatement struct {
	Condition Expression
	Body      []Statement
}

func (*WhileStatement) statementNode() {}

type ForStatement struct {
	Init      Statement
	Condition Expression
	Update    Statement
	Body      []Statement
}

func (*ForStatement) statementNode() {}

type BreakStatement struct{}

func (*BreakStatement) statementNode() {}

type ContinueStatement struct{}

func (*ContinueStatement) statementNode() {}

type ExpressionStatement struct {
	Expression Expression
}

func (*ExpressionStatement) statementNode() {}

type Expression interface {
	expressionNode()
}

type NumberLiteral struct {
	Value float64
}

func (*NumberLiteral) expressionNode() {}

type BooleanLiteral struct {
	Value bool
}

func (*BooleanLiteral) expressionNode() {}

type NullLiteral struct{}

func (*NullLiteral) expressionNode() {}

type UndefinedLiteral struct{}

func (*UndefinedLiteral) expressionNode() {}

type StringLiteral struct {
	Value string
}

func (*StringLiteral) expressionNode() {}

type ObjectProperty struct {
	Key   string
	Value Expression
}

type ObjectLiteral struct {
	Properties []ObjectProperty
}

func (*ObjectLiteral) expressionNode() {}

type ArrayLiteral struct {
	Elements []Expression
}

func (*ArrayLiteral) expressionNode() {}

type VariableRef struct {
	Name string
	Kind ValueKind
}

func (*VariableRef) expressionNode() {}

type CallExpression struct {
	Callee    string
	Arguments []Expression
	Kind      ValueKind
}

func (*CallExpression) expressionNode() {}

type BinaryExpression struct {
	Operator BinaryOperator
	Left     Expression
	Right    Expression
}

func (*BinaryExpression) expressionNode() {}

type ComparisonExpression struct {
	Operator ComparisonOperator
	Left     Expression
	Right    Expression
}

func (*ComparisonExpression) expressionNode() {}

type LogicalExpression struct {
	Operator LogicalOperator
	Left     Expression
	Right    Expression
}

func (*LogicalExpression) expressionNode() {}

type UnaryExpression struct {
	Operator UnaryOperator
	Right    Expression
}

func (*UnaryExpression) expressionNode() {}

type IndexExpression struct {
	Target Expression
	Index  Expression
	Kind   ValueKind
}

func (*IndexExpression) expressionNode() {}

type MemberExpression struct {
	Target   Expression
	Property string
	Kind     ValueKind
}

func (*MemberExpression) expressionNode() {}
