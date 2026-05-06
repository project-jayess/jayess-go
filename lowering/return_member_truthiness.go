package lowering

import "jayess-go/ast"

func evaluateMemberTruthiness(expression *ast.MemberExpression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	value, ok := evaluateObjectMemberElement(expression, next)
	if ok {
		truthy, ok := evaluateArrayElementTruthiness(value)
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return truthy, true
	}
	next = bindings.clone()
	length, ok := evaluateIntMemberExpression(expression, next)
	if !ok {
		return false, false
	}
	replaceReturnScopeBindings(bindings, next)
	return length != 0, true
}

func evaluateIndexTruthiness(expression *ast.IndexExpression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	value, ok := evaluateArrayIndexElement(expression, next)
	if ok {
		truthy, ok := evaluateArrayElementTruthiness(value)
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return truthy, true
	}
	next = bindings.clone()
	value, ok = evaluateObjectIndexElement(expression, next)
	if ok {
		truthy, ok := evaluateArrayElementTruthiness(value)
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return truthy, true
	}
	next = bindings.clone()
	stringValue, ok := evaluateStringIndexExpression(expression, next)
	if ok {
		replaceReturnScopeBindings(bindings, next)
		return stringValue != "", true
	}
	next = bindings.clone()
	if _, ok := evaluateNullishIndexExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return false, true
	}
	return false, false
}

func evaluateArrayElementTruthiness(value returnArrayElement) (bool, bool) {
	switch value.kind {
	case returnArrayIntKind:
		return value.intValue != 0, true
	case returnArrayBigIntKind:
		return isTruthyBigIntValue(value.bigIntValue), true
	case returnArrayBoolKind:
		return value.boolValue, true
	case returnArrayStringKind:
		return value.stringValue != "", true
	case returnArrayNullishKind:
		return false, true
	case returnArrayFuncKind, returnArrayObjectKind:
		return true, true
	default:
		return false, false
	}
}
