package ast

type WhileStatement struct {
	BaseNode
	Condition Expression
	Body      []Statement
}

func (*WhileStatement) node()          {}
func (*WhileStatement) statementNode() {}

type DoWhileStatement struct {
	BaseNode
	Body      []Statement
	Condition Expression
}

func (*DoWhileStatement) node()          {}
func (*DoWhileStatement) statementNode() {}

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
	Pattern  BindingPattern
	Target   Expression
	Iterable Expression
	Body     []Statement
	Await    bool
}

func (*ForOfStatement) node()          {}
func (*ForOfStatement) statementNode() {}

type ForInStatement struct {
	BaseNode
	Kind    DeclarationKind
	Name    string
	Pattern BindingPattern
	Target  Expression
	Object  Expression
	Body    []Statement
}

func (*ForInStatement) node()          {}
func (*ForInStatement) statementNode() {}

type ThrowStatement struct {
	BaseNode
	Value Expression
}

func (*ThrowStatement) node()          {}
func (*ThrowStatement) statementNode() {}

type TryStatement struct {
	BaseNode
	TryBody      []Statement
	CatchName    string
	CatchPattern BindingPattern
	CatchBody    []Statement
	FinallyBody  []Statement
}

func (*TryStatement) node()          {}
func (*TryStatement) statementNode() {}
