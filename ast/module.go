package ast

type ImportSpecifier struct {
	Imported  string
	Local     string
	Default   bool
	Namespace bool
}

type ImportDecl struct {
	BaseNode
	Source     string
	Specifiers []ImportSpecifier
	SideEffect bool
}

func (*ImportDecl) node()          {}
func (*ImportDecl) statementNode() {}

type ExportSpecifier struct {
	Local    string
	Exported string
}

type ExportDecl struct {
	BaseNode
	Declaration Statement
	Value       Expression
	Specifiers  []ExportSpecifier
	Source      string
	Default     bool
	All         bool
	Namespace   string
}

func (*ExportDecl) node()          {}
func (*ExportDecl) statementNode() {}
