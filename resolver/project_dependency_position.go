package resolver

import "jayess-go/ast"

func dependencyPosition(program *ast.Program, dependency ast.ModuleDependency) ast.SourcePos {
	if program == nil {
		return ast.SourcePos{}
	}
	for _, statement := range program.Statements {
		switch stmt := statement.(type) {
		case *ast.ImportDecl:
			if stmt.Source == dependency.Source && stmt.SideEffect == dependency.SideEffect {
				return ast.PositionOf(stmt)
			}
		case *ast.ExportDecl:
			if stmt.Source == dependency.Source && dependency.ReExport {
				return ast.PositionOf(stmt)
			}
		}
	}
	return ast.SourcePos{}
}
