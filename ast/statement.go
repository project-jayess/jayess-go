package ast

type Statement interface {
	Node
	statementNode()
}

type EmptyStatement struct {
	BaseNode
}

func (*EmptyStatement) node()          {}
func (*EmptyStatement) statementNode() {}

type DebuggerStatement struct {
	BaseNode
}

func (*DebuggerStatement) node()          {}
func (*DebuggerStatement) statementNode() {}

type DeclarationKind string

const (
	DeclarationVar   DeclarationKind = "var"
	DeclarationConst DeclarationKind = "const"
)

type VariableDecl struct {
	BaseNode
	Kind    DeclarationKind
	Name    string
	Pattern BindingPattern
	Value   Expression
}

func (*VariableDecl) node()          {}
func (*VariableDecl) statementNode() {}

type ExpressionStatement struct {
	BaseNode
	Expression Expression
}

func (*ExpressionStatement) node()          {}
func (*ExpressionStatement) statementNode() {}

type AssignmentStatement struct {
	BaseNode
	Target   Expression
	Operator AssignmentOperator
	Value    Expression
}

func (*AssignmentStatement) node()          {}
func (*AssignmentStatement) statementNode() {}

type BlockStatement struct {
	BaseNode
	Statements []Statement
}

func (*BlockStatement) node()          {}
func (*BlockStatement) statementNode() {}

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

type LabeledStatement struct {
	BaseNode
	Label     string
	Statement Statement
}

func (*LabeledStatement) node()          {}
func (*LabeledStatement) statementNode() {}

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

type BreakStatement struct {
	BaseNode
	Label string
}

func (*BreakStatement) node()          {}
func (*BreakStatement) statementNode() {}

type ContinueStatement struct {
	BaseNode
	Label string
}

func (*ContinueStatement) node()          {}
func (*ContinueStatement) statementNode() {}
