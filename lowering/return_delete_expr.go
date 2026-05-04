package lowering

import "jayess-go/ast"

func evaluateDeleteExpression(expression ast.Expression, bindings returnScope) (bool, bool) {
	switch expr := expression.(type) {
	case *ast.MemberExpression:
		if !evaluateDeleteTarget(expr.Target, bindings) {
			return false, false
		}
		return true, true
	case *ast.IndexExpression:
		if !evaluateDeleteTarget(expr.Target, bindings) {
			return false, false
		}
		if _, ok := evaluateObjectPropertyKey(expr.Index, bindings); !ok {
			return false, false
		}
		return true, true
	default:
		if !evaluateDeleteTarget(expression, bindings) {
			return false, false
		}
		return true, true
	}
}

func evaluateDeleteTarget(expression ast.Expression, bindings returnScope) bool {
	next := bindings.clone()
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
	if evaluateDiscardExpression(expression, next) {
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
	return false
}
