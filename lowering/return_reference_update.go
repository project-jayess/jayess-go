package lowering

import "jayess-go/ast"

func evaluateReferenceUpdateExpression(expression *ast.UpdateExpression, scope returnScope) bool {
	if expression == nil {
		return false
	}
	if _, ok := expression.Target.(*ast.Identifier); ok {
		return false
	}
	return evaluateAssignmentTargetReference(expression.Target, scope)
}
