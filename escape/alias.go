package escape

import "jayess-go/ast"

func recordDeclarationAlias(report *Report, stmt *ast.VariableDecl) {
	name, ok := stmt.Pattern.(*ast.BindingName)
	if !ok {
		return
	}
	source, ok := stmt.Value.(*ast.Identifier)
	if !ok {
		return
	}
	report.addAlias(name.Name, source.Name)
}
