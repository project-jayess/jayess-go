package lowering

import "jayess-go/ast"

func evaluateFoldedExpressionTruthiness(expression ast.Expression, bindings returnScope) (bool, bool) {
	switch expression.(type) {
	case *ast.BinaryExpression, *ast.UnaryExpression:
		next := bindings.clone()
		if value, ok := evaluateIntExpression(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value != 0, true
		}
		next = bindings.clone()
		if _, ok := evaluateNullishExpression(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return false, true
		}
		next = bindings.clone()
		if value, ok := evaluateBigIntValue(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return isTruthyBigIntValue(value), true
		}
		return evaluateStringExpressionTruthiness(expression, bindings)
	case *ast.TemplateLiteral, *ast.TypeofExpression:
		return evaluateStringExpressionTruthiness(expression, bindings)
	default:
		return false, false
	}
}

func evaluateStringExpressionTruthiness(expression ast.Expression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	value, ok := evaluateStringExpression(expression, next)
	if !ok {
		return false, false
	}
	replaceReturnScopeBindings(bindings, next)
	return value != "", true
}
