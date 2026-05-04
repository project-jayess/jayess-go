package lowering

import "jayess-go/ast"

func evaluateDiscardExpression(expression ast.Expression, bindings returnScope) bool {
	if _, ok := expression.(*ast.InvokeExpression); ok {
		return evaluateEmptyCallExpression(expression, bindings)
	}
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
	if _, ok := evaluateBigIntValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if evaluateEmptyCallExpression(expression, next) {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if update, ok := expression.(*ast.UpdateExpression); ok && evaluateReferenceUpdateExpression(update, next) {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateObjectLiteralProperties(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateArrayLiteralElements(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if evaluateObjectReferenceExpression(expression, next) {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if evaluateFunctionReferenceExpression(expression, next) {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	next = bindings.clone()
	if _, ok := evaluateBoolExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true
	}
	return false
}
