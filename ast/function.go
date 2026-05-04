package ast

type Parameter struct {
	Name    string
	Pattern BindingPattern
	Rest    bool
	Default Expression
}

type FunctionDecl struct {
	BaseNode
	IsAsync     bool
	IsGenerator bool
	Name        string
	Params      []Parameter
	Body        []Statement
}

func (*FunctionDecl) node()          {}
func (*FunctionDecl) statementNode() {}

type FunctionExpression struct {
	BaseNode
	Name            string
	Params          []Parameter
	Body            []Statement
	ExpressionBody  Expression
	IsArrowFunction bool
	IsAsync         bool
	IsGenerator     bool
}

func (*FunctionExpression) node()           {}
func (*FunctionExpression) expressionNode() {}
