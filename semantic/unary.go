package semantic

import "jayess-go/ast"

func analyzeUnaryExpressionWithContext(scope *scope, context controlContext, expr *ast.UnaryExpression) error {
	if expr.Operator == ast.OperatorDelete {
		if _, ok := expr.Right.(*ast.Identifier); ok {
			return errorAt(expr, "delete of identifier is not allowed")
		}
		if member, ok := expr.Right.(*ast.MemberExpression); ok && member.Private {
			return errorAt(expr, "delete of private member is not allowed")
		}
	}
	return analyzeExpressionWithContext(scope, context, expr.Right)
}
