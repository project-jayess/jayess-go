package lowering

import "jayess-go/ast"

func evaluateInExpression(leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	key, ok := evaluateObjectPropertyKey(leftExpression, next)
	if !ok {
		return false, false
	}
	properties, ok := evaluateObjectLiteralProperties(rightExpression, next)
	if !ok {
		next = bindings.clone()
		key, ok = evaluateObjectPropertyKey(leftExpression, next)
		if !ok {
			return false, false
		}
		properties, ok = evaluateObjectIIFELiteralProperties(rightExpression, next)
		if !ok {
			return false, false
		}
	}
	_, exists := properties[key]
	replaceReturnScopeBindings(bindings, next)
	return exists, true
}
