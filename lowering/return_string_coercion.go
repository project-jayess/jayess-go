package lowering

import (
	"strconv"

	"jayess-go/ast"
)

func evaluateStringCoercion(expression ast.Expression, bindings returnScope) (string, bool) {
	next := bindings.clone()
	if value, ok := evaluateStringExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return value, true
	}
	next = bindings.clone()
	if value, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return strconv.Itoa(value), true
	}
	next = bindings.clone()
	if value, ok := evaluateBoolValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		if value {
			return "true", true
		}
		return "false", true
	}
	next = bindings.clone()
	if value, ok := evaluateBigIntStringCoercion(expression); ok {
		replaceReturnScopeBindings(bindings, next)
		return value, true
	}
	next = bindings.clone()
	if value, ok := evaluateBigIntValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return value, true
	}
	next = bindings.clone()
	if kind, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		if kind == returnNullKind {
			return "null", true
		}
		return "undefined", true
	}
	return "", false
}
