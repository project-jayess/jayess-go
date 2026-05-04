package lowering

import "jayess-go/ast"

func leftIsNullish(expression ast.Expression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	if _, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return true, true
	}
	next = bindings.clone()
	if evaluateDiscardExpression(expression, next) {
		replaceReturnScopeBindings(bindings, next)
		return false, true
	}
	return false, false
}
