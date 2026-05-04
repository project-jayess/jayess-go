package lowering

import "jayess-go/ast"

func applyReferenceAssignmentStatement(statement *ast.AssignmentStatement, scope returnScope) {
	if statement == nil || statement.Value == nil {
		return
	}
	next := scope.clone()
	if !evaluateAssignmentTargetReference(statement.Target, next) {
		return
	}
	if !evaluateDiscardExpression(statement.Value, next) {
		return
	}
	replaceReturnScopeBindings(scope, next)
}

func evaluateAssignmentTargetReference(expression ast.Expression, scope returnScope) bool {
	switch expr := expression.(type) {
	case *ast.MemberExpression:
		return !expr.Optional && evaluateDiscardExpression(expr.Target, scope)
	case *ast.IndexExpression:
		if expr.Optional || !evaluateDiscardExpression(expr.Target, scope) {
			return false
		}
		_, ok := evaluateObjectPropertyKey(expr.Index, scope)
		return ok
	default:
		return false
	}
}
