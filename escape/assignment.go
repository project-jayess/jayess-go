package escape

import "jayess-go/ast"

func markOuterAssignmentEscaping(report *Report, scope *scope, stmt *ast.AssignmentStatement) {
	identifier, ok := stmt.Target.(*ast.Identifier)
	if !ok {
		return
	}
	if scope.hasLocal(identifier.Name) {
		return
	}
	if scope.hasOuter(identifier.Name) {
		markExpressionIdentifiersEscaping(report, stmt.Value)
	}
}
