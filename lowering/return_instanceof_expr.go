package lowering

import "jayess-go/ast"

func evaluateInstanceofExpression(expression *ast.InstanceofExpression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	if !evaluatePrimitiveInstanceofLeft(expression.Left, next) {
		return false, false
	}
	if !evaluateFunctionReferenceExpression(expression.Right, next) {
		return false, false
	}
	replaceReturnScopeBindings(bindings, next)
	return false, true
}

func evaluatePrimitiveInstanceofLeft(expression ast.Expression, bindings returnScope) bool {
	next := bindings.clone()
	if _, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateStringExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateBigIntValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateBoolValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	return false
}
