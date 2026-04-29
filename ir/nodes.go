package ir

type Statement interface {
	statementNode()
}

type VariableDecl struct {
	Visibility Visibility
	Kind       DeclarationKind
	Name       string
	Line       int
	Column     int
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

type DoWhileStatement struct {
	Body      []Statement
	Condition Expression
}

func (*DoWhileStatement) statementNode() {}

type BlockStatement struct {
	Body []Statement
}

func (*BlockStatement) statementNode() {}

type ForStatement struct {
	Init      Statement
	Condition Expression
	Update    Statement
	Body      []Statement
}

func (*ForStatement) statementNode() {}

type SwitchCase struct {
	Test       Expression
	Consequent []Statement
}

type SwitchStatement struct {
	Discriminant Expression
	Cases        []SwitchCase
	Default      []Statement
}

func (*SwitchStatement) statementNode() {}

type LabeledStatement struct {
	Label     string
	Statement Statement
}

func (*LabeledStatement) statementNode() {}

type BreakStatement struct {
	Label string
}

func (*BreakStatement) statementNode() {}

type ContinueStatement struct {
	Label string
}

func (*ContinueStatement) statementNode() {}

type DeleteStatement struct {
	Target Expression
}

func (*DeleteStatement) statementNode() {}

type ThrowStatement struct {
	Value Expression
}

func (*ThrowStatement) statementNode() {}

type TryStatement struct {
	TryBody     []Statement
	CatchName   string
	CatchBody   []Statement
	FinallyBody []Statement
}

func (*TryStatement) statementNode() {}

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

type BigIntLiteral struct {
	Value string
}

func (*BigIntLiteral) expressionNode() {}

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
	Key      string
	KeyExpr  Expression
	Value    Expression
	Computed bool
	Spread   bool
	Getter   bool
	Setter   bool
}

type ObjectLiteral struct {
	Properties []ObjectProperty
}

func (*ObjectLiteral) expressionNode() {}

type ArrayLiteral struct {
	Elements []Expression
}

func (*ArrayLiteral) expressionNode() {}

type TemplateLiteral struct {
	Parts  []string
	Values []Expression
}

func (*TemplateLiteral) expressionNode() {}

type SpreadExpression struct {
	Value Expression
}

func (*SpreadExpression) expressionNode() {}

type VariableRef struct {
	Name string
	Kind ValueKind
}

func (*VariableRef) expressionNode() {}

type FunctionValue struct {
	Name        string
	Environment Expression
}

func (*FunctionValue) expressionNode() {}

type CallExpression struct {
	Callee    string
	Arguments []Expression
	Kind      ValueKind
}

func (*CallExpression) expressionNode() {}

type InvokeExpression struct {
	Callee    Expression
	Arguments []Expression
	Kind      ValueKind
	Optional  bool
}

func (*InvokeExpression) expressionNode() {}

type NewTargetExpression struct {
	Kind ValueKind
}

func (*NewTargetExpression) expressionNode() {}

type BinaryExpression struct {
	Operator BinaryOperator
	Left     Expression
	Right    Expression
	Kind     ValueKind
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

type NullishCoalesceExpression struct {
	Left  Expression
	Right Expression
	Kind  ValueKind
}

func (*NullishCoalesceExpression) expressionNode() {}

type CommaExpression struct {
	Left  Expression
	Right Expression
	Kind  ValueKind
}

func (*CommaExpression) expressionNode() {}

type ConditionalExpression struct {
	Condition   Expression
	Consequent  Expression
	Alternative Expression
	Kind        ValueKind
}

func (*ConditionalExpression) expressionNode() {}

type UnaryExpression struct {
	Operator UnaryOperator
	Right    Expression
	Kind     ValueKind
}

func (*UnaryExpression) expressionNode() {}

type TypeofExpression struct {
	Value Expression
}

func (*TypeofExpression) expressionNode() {}

type InstanceofExpression struct {
	Left      Expression
	Right     Expression
	ClassName string
}

func (*InstanceofExpression) expressionNode() {}

type IndexExpression struct {
	Target   Expression
	Index    Expression
	Kind     ValueKind
	Optional bool
}

func (*IndexExpression) expressionNode() {}

type MemberExpression struct {
	Target   Expression
	Property string
	Kind     ValueKind
	Optional bool
}

func (*MemberExpression) expressionNode() {}
