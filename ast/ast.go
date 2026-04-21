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

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Pattern interface {
	Node
	patternNode()
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
type AssignmentOperator string

const (
	OperatorAdd BinaryOperator = "+"
	OperatorSub BinaryOperator = "-"
	OperatorMul BinaryOperator = "*"
	OperatorDiv BinaryOperator = "/"
)

const (
	OperatorEq       ComparisonOperator = "=="
	OperatorNe       ComparisonOperator = "!="
	OperatorStrictEq ComparisonOperator = "==="
	OperatorStrictNe ComparisonOperator = "!=="
	OperatorLt       ComparisonOperator = "<"
	OperatorLte      ComparisonOperator = "<="
	OperatorGt       ComparisonOperator = ">"
	OperatorGte      ComparisonOperator = ">="
)

const (
	OperatorAnd LogicalOperator = "&&"
	OperatorOr  LogicalOperator = "||"
)

const (
	OperatorNot UnaryOperator = "!"
)

const (
	AssignmentAssign        AssignmentOperator = "="
	AssignmentAddAssign     AssignmentOperator = "+="
	AssignmentSubAssign     AssignmentOperator = "-="
	AssignmentMulAssign     AssignmentOperator = "*="
	AssignmentDivAssign     AssignmentOperator = "/="
	AssignmentNullishAssign AssignmentOperator = "??="
	AssignmentOrAssign      AssignmentOperator = "||="
	AssignmentAndAssign     AssignmentOperator = "&&="
)

type Program struct {
	BaseNode
	Globals         []*VariableDecl
	ExternFunctions []*ExternFunctionDecl
	Functions       []*FunctionDecl
	Classes         []*ClassDecl
}

func (*Program) node() {}

type Parameter struct {
	Name           string
	Pattern        Pattern
	Rest           bool
	Default        Expression
	TypeAnnotation string
}

type ClassMember interface {
	Node
	classMemberNode()
}

type FunctionDecl struct {
	BaseNode
	Visibility Visibility
	Name       string
	Params     []Parameter
	ReturnType string
	IsAsync    bool
	Body       []Statement
}

func (*FunctionDecl) node() {}

type FunctionExpression struct {
	BaseNode
	Params          []Parameter
	ReturnType      string
	IsAsync         bool
	Body            []Statement
	ExpressionBody  Expression
	IsArrowFunction bool
}

func (*FunctionExpression) node()           {}
func (*FunctionExpression) expressionNode() {}

type ClosureExpression struct {
	BaseNode
	FunctionName string
	Environment  Expression
}

func (*ClosureExpression) node()           {}
func (*ClosureExpression) expressionNode() {}

type ExternFunctionDecl struct {
	BaseNode
	Name         string
	NativeSymbol string
	Params       []Parameter
	Variadic     bool
}

func (*ExternFunctionDecl) node() {}

type ClassDecl struct {
	BaseNode
	Name       string
	SuperClass string
	Members    []ClassMember
}

func (*ClassDecl) node() {}

type ClassFieldDecl struct {
	BaseNode
	Name        string
	Private     bool
	Static      bool
	Initializer Expression
}

func (*ClassFieldDecl) node()            {}
func (*ClassFieldDecl) classMemberNode() {}

type ClassMethodDecl struct {
	BaseNode
	Name          string
	Private       bool
	Static        bool
	IsConstructor bool
	Params        []Parameter
	Body          []Statement
}

func (*ClassMethodDecl) node()            {}
func (*ClassMethodDecl) classMemberNode() {}

type VariableDecl struct {
	BaseNode
	Visibility     Visibility
	Kind           DeclarationKind
	Name           string
	TypeAnnotation string
	Value          Expression
}

func (*VariableDecl) node()          {}
func (*VariableDecl) statementNode() {}

type DestructuringDecl struct {
	BaseNode
	Visibility Visibility
	Kind       DeclarationKind
	Pattern    Pattern
	Value      Expression
}

func (*DestructuringDecl) node()          {}
func (*DestructuringDecl) statementNode() {}

type AssignmentStatement struct {
	BaseNode
	Target   Expression
	Operator AssignmentOperator
	Value    Expression
}

func (*AssignmentStatement) node()          {}
func (*AssignmentStatement) statementNode() {}

type DestructuringAssignment struct {
	BaseNode
	Pattern Pattern
	Value   Expression
}

func (*DestructuringAssignment) node()          {}
func (*DestructuringAssignment) statementNode() {}

type ReturnStatement struct {
	BaseNode
	Value Expression
}

func (*ReturnStatement) node()          {}
func (*ReturnStatement) statementNode() {}

type IfStatement struct {
	BaseNode
	Condition   Expression
	Consequence []Statement
	Alternative []Statement
}

func (*IfStatement) node()          {}
func (*IfStatement) statementNode() {}

type WhileStatement struct {
	BaseNode
	Condition Expression
	Body      []Statement
}

func (*WhileStatement) node()          {}
func (*WhileStatement) statementNode() {}

type ForStatement struct {
	BaseNode
	Init      Statement
	Condition Expression
	Update    Statement
	Body      []Statement
}

func (*ForStatement) node()          {}
func (*ForStatement) statementNode() {}

type ForOfStatement struct {
	BaseNode
	Kind     DeclarationKind
	Name     string
	Iterable Expression
	Body     []Statement
}

func (*ForOfStatement) node()          {}
func (*ForOfStatement) statementNode() {}

type ForInStatement struct {
	BaseNode
	Kind     DeclarationKind
	Name     string
	Iterable Expression
	Body     []Statement
}

func (*ForInStatement) node()          {}
func (*ForInStatement) statementNode() {}

type SwitchCase struct {
	Test       Expression
	Consequent []Statement
}

type SwitchStatement struct {
	BaseNode
	Discriminant Expression
	Cases        []SwitchCase
	Default      []Statement
}

func (*SwitchStatement) node()          {}
func (*SwitchStatement) statementNode() {}

type BreakStatement struct{ BaseNode }

func (*BreakStatement) node()          {}
func (*BreakStatement) statementNode() {}

type ContinueStatement struct{ BaseNode }

func (*ContinueStatement) node()          {}
func (*ContinueStatement) statementNode() {}

type DeleteStatement struct {
	BaseNode
	Target Expression
}

func (*DeleteStatement) node()          {}
func (*DeleteStatement) statementNode() {}

type ThrowStatement struct {
	BaseNode
	Value Expression
}

func (*ThrowStatement) node()          {}
func (*ThrowStatement) statementNode() {}

type TryStatement struct {
	BaseNode
	TryBody     []Statement
	CatchName   string
	CatchBody   []Statement
	FinallyBody []Statement
}

func (*TryStatement) node()          {}
func (*TryStatement) statementNode() {}

type ExpressionStatement struct {
	BaseNode
	Expression Expression
}

func (*ExpressionStatement) node()          {}
func (*ExpressionStatement) statementNode() {}

type NumberLiteral struct {
	BaseNode
	Value float64
}

func (*NumberLiteral) node()           {}
func (*NumberLiteral) expressionNode() {}

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

type StringLiteral struct {
	BaseNode
	Value string
}

func (*StringLiteral) node()           {}
func (*StringLiteral) expressionNode() {}

type ObjectProperty struct {
	Key      string
	KeyExpr  Expression
	Value    Expression
	Computed bool
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

type TemplateLiteral struct {
	BaseNode
	Parts  []string
	Values []Expression
}

func (*TemplateLiteral) node()           {}
func (*TemplateLiteral) expressionNode() {}

type SpreadExpression struct {
	BaseNode
	Value Expression
}

func (*SpreadExpression) node()           {}
func (*SpreadExpression) expressionNode() {}

type Identifier struct {
	BaseNode
	Name string
}

func (*Identifier) node()           {}
func (*Identifier) expressionNode() {}

type IdentifierPattern struct {
	BaseNode
	Name string
}

func (*IdentifierPattern) node()        {}
func (*IdentifierPattern) patternNode() {}

type ObjectPatternProperty struct {
	Key     string
	Pattern Pattern
	Default Expression
}

type ObjectPattern struct {
	BaseNode
	Properties []ObjectPatternProperty
	Rest       string
}

func (*ObjectPattern) node()        {}
func (*ObjectPattern) patternNode() {}

type ArrayPattern struct {
	BaseNode
	Elements []ArrayPatternElement
}

type ArrayPatternElement struct {
	Pattern Pattern
	Default Expression
	Rest    bool
}

func (*ArrayPattern) node()        {}
func (*ArrayPattern) patternNode() {}

type ThisExpression struct{ BaseNode }

func (*ThisExpression) node()           {}
func (*ThisExpression) expressionNode() {}

type SuperExpression struct{ BaseNode }

func (*SuperExpression) node()           {}
func (*SuperExpression) expressionNode() {}

type BoundSuperExpression struct {
	BaseNode
	OwnerClass string
	BaseClass  string
	IsStatic   bool
	Receiver   Expression
}

func (*BoundSuperExpression) node()           {}
func (*BoundSuperExpression) expressionNode() {}

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

type NewExpression struct {
	BaseNode
	Callee    Expression
	Arguments []Expression
}

func (*NewExpression) node()           {}
func (*NewExpression) expressionNode() {}

type NewTargetExpression struct{ BaseNode }

func (*NewTargetExpression) node()           {}
func (*NewTargetExpression) expressionNode() {}

type AwaitExpression struct {
	BaseNode
	Value Expression
}

func (*AwaitExpression) node()           {}
func (*AwaitExpression) expressionNode() {}

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

type UnaryExpression struct {
	BaseNode
	Operator UnaryOperator
	Right    Expression
}

func (*UnaryExpression) node()           {}
func (*UnaryExpression) expressionNode() {}

type TypeofExpression struct {
	BaseNode
	Value Expression
}

func (*TypeofExpression) node()           {}
func (*TypeofExpression) expressionNode() {}

type InstanceofExpression struct {
	BaseNode
	Left  Expression
	Right Expression
}

func (*InstanceofExpression) node()           {}
func (*InstanceofExpression) expressionNode() {}

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
