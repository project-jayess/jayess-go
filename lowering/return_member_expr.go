package lowering

import "jayess-go/ast"

func evaluateIntMemberExpression(expression *ast.MemberExpression, bindings returnScope) (int, bool) {
	if expression.Property != "length" {
		value, ok := evaluateObjectMemberElement(expression, bindings)
		if !ok || value.kind != returnArrayIntKind {
			return 0, false
		}
		return value.intValue, true
	}
	next := bindings.clone()
	value, ok := evaluateStringExpression(expression.Target, next)
	if ok {
		replaceReturnScopeBindings(bindings, next)
		return len(value), true
	}
	next = bindings.clone()
	elements, ok := evaluateArrayLiteralElements(expression.Target, next)
	if ok {
		replaceReturnScopeBindings(bindings, next)
		return len(elements), true
	}
	next = bindings.clone()
	elements, ok = evaluateArrayIIFELiteralElements(expression.Target, next)
	if ok {
		replaceReturnScopeBindings(bindings, next)
		return len(elements), true
	}
	return 0, false
}

func evaluateIntIndexExpression(expression *ast.IndexExpression, bindings returnScope) (int, bool) {
	value, ok := evaluateArrayIndexElement(expression, bindings)
	if ok && value.kind == returnArrayIntKind {
		return value.intValue, true
	}
	value, ok = evaluateObjectIndexElement(expression, bindings)
	if !ok || value.kind != returnArrayIntKind {
		return 0, false
	}
	return value.intValue, true
}

func evaluateBoolIndexExpression(expression *ast.IndexExpression, bindings returnScope) (bool, bool) {
	value, ok := evaluateArrayIndexElement(expression, bindings)
	if ok && value.kind == returnArrayBoolKind {
		return value.boolValue, true
	}
	value, ok = evaluateObjectIndexElement(expression, bindings)
	if !ok || value.kind != returnArrayBoolKind {
		return false, false
	}
	return value.boolValue, true
}

func evaluateStringMemberExpression(expression *ast.MemberExpression, bindings returnScope) (string, bool) {
	value, ok := evaluateObjectMemberElement(expression, bindings)
	if !ok || value.kind != returnArrayStringKind {
		return "", false
	}
	return value.stringValue, true
}

func evaluateBoolMemberExpression(expression *ast.MemberExpression, bindings returnScope) (bool, bool) {
	value, ok := evaluateObjectMemberElement(expression, bindings)
	if !ok || value.kind != returnArrayBoolKind {
		return false, false
	}
	return value.boolValue, true
}

func evaluateNullishMemberExpression(expression *ast.MemberExpression, bindings returnScope) (returnNullishKind, bool) {
	if expression.Optional {
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expression.Target, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return returnUndefinedKind, true
		}
	}
	value, ok := evaluateObjectMemberElement(expression, bindings)
	if !ok || value.kind != returnArrayNullishKind {
		return "", false
	}
	return value.nullishValue, true
}

func evaluateFunctionMemberIdentity(expression *ast.MemberExpression, bindings returnScope) (int, bool) {
	value, ok := evaluateObjectMemberElement(expression, bindings)
	if !ok || value.kind != returnArrayFuncKind {
		return 0, false
	}
	return value.funcIdentity, true
}

func evaluateFunctionIndexIdentity(expression *ast.IndexExpression, bindings returnScope) (int, bool) {
	value, ok := evaluateArrayIndexElement(expression, bindings)
	if ok && value.kind == returnArrayFuncKind {
		return value.funcIdentity, true
	}
	value, ok = evaluateObjectIndexElement(expression, bindings)
	if !ok || value.kind != returnArrayFuncKind {
		return 0, false
	}
	return value.funcIdentity, true
}

func evaluateObjectMemberIdentity(expression *ast.MemberExpression, bindings returnScope) (int, bool) {
	value, ok := evaluateObjectMemberElement(expression, bindings)
	if !ok || value.kind != returnArrayObjectKind {
		return 0, false
	}
	return value.objectID, true
}

func evaluateObjectIndexIdentity(expression *ast.IndexExpression, bindings returnScope) (int, bool) {
	value, ok := evaluateArrayIndexElement(expression, bindings)
	if ok && value.kind == returnArrayObjectKind {
		return value.objectID, true
	}
	value, ok = evaluateObjectIndexElement(expression, bindings)
	if !ok || value.kind != returnArrayObjectKind {
		return 0, false
	}
	return value.objectID, true
}
