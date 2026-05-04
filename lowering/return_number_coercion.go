package lowering

import "jayess-go/ast"

func evaluateNumericCoercion(expression ast.Expression, bindings returnScope) (int, bool) {
	next := bindings.clone()
	if value, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return value, true
	}
	next = bindings.clone()
	if text, ok := evaluateStringExpression(expression, next); ok {
		number, ok := parseLooseStringNumber(text)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(bindings, next)
		return int(number), true
	}
	next = bindings.clone()
	if value, ok := evaluateBoolValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		if value {
			return 1, true
		}
		return 0, true
	}
	next = bindings.clone()
	if kind, ok := evaluateNullishExpression(expression, next); ok {
		if kind == returnNullKind {
			replaceReturnScopeBindings(bindings, next)
			return 0, true
		}
		return 0, false
	}
	return 0, false
}
