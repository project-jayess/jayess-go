package ast

type ClassDecl struct {
	BaseNode
	Name       string
	SuperClass Expression
	Members    []ClassMember
}

func (*ClassDecl) node()          {}
func (*ClassDecl) statementNode() {}

type ClassMember struct {
	BaseNode
	Name        string
	KeyExpr     Expression
	Params      []Parameter
	Body        []Statement
	Value       Expression
	Computed    bool
	StaticBlock bool
	Constructor bool
	Field       bool
	Getter      bool
	Setter      bool
	Private     bool
	Static      bool
	IsAsync     bool
	IsGenerator bool
}
