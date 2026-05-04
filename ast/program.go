package ast

type Program struct {
	BaseNode
	Statements []Statement
}

func (*Program) node() {}
