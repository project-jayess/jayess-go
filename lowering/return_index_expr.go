package lowering

import (
	"strconv"

	"jayess-go/ast"
)

func evaluateStringIndexExpression(expression *ast.IndexExpression, bindings returnScope) (string, bool) {
	next := bindings.clone()
	value, ok := evaluateStringExpression(expression.Target, next)
	if !ok {
		return "", false
	}
	index, ok := evaluateIndexPropertyNumber(expression.Index, next)
	if !ok || index < 0 || index >= len(value) {
		return "", false
	}
	replaceReturnScopeBindings(bindings, next)
	return value[index : index+1], true
}

func evaluateNullishIndexExpression(expression *ast.IndexExpression, bindings returnScope) (returnNullishKind, bool) {
	if expression.Optional {
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expression.Target, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return returnUndefinedKind, true
		}
	}
	next := bindings.clone()
	value, ok := evaluateStringExpression(expression.Target, next)
	if !ok {
		return "", false
	}
	index, ok := evaluateIndexPropertyNumber(expression.Index, next)
	if !ok || index >= 0 && index < len(value) {
		return "", false
	}
	replaceReturnScopeBindings(bindings, next)
	return returnUndefinedKind, true
}

func evaluateArrayNullishIndexExpression(expression *ast.IndexExpression, bindings returnScope) (returnNullishKind, bool) {
	value, ok := evaluateArrayIndexElement(expression, bindings)
	if ok && value.kind == returnArrayNullishKind {
		return value.nullishValue, true
	}
	value, ok = evaluateObjectIndexElement(expression, bindings)
	if !ok || value.kind != returnArrayNullishKind {
		return "", false
	}
	return value.nullishValue, true
}

func evaluateArrayIndexElement(expression *ast.IndexExpression, bindings returnScope) (returnArrayElement, bool) {
	next := bindings.clone()
	elements, ok := evaluateArrayLiteralElements(expression.Target, next)
	if !ok {
		next = bindings.clone()
		elements, ok = evaluateArrayIIFELiteralElements(expression.Target, next)
		if !ok {
			return returnArrayElement{}, false
		}
	}
	index, ok := evaluateIndexPropertyNumber(expression.Index, next)
	if !ok {
		return returnArrayElement{}, false
	}
	if index < 0 || index >= len(elements) {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayNullishKind, nullishValue: returnUndefinedKind}, true
	}
	replaceReturnScopeBindings(bindings, next)
	return elements[index], true
}

func evaluateIndexPropertyNumber(expression ast.Expression, bindings returnScope) (int, bool) {
	next := bindings.clone()
	if value, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return value, true
	}
	next = bindings.clone()
	value, ok := evaluateBigIntValue(expression, next)
	if !ok {
		return 0, false
	}
	index, err := strconv.Atoi(value)
	if err == nil {
		replaceReturnScopeBindings(bindings, next)
		return index, true
	}
	parsed, ok := parseBigIntValue(value)
	if !ok {
		return 0, false
	}
	replaceReturnScopeBindings(bindings, next)
	if parsed.Sign() < 0 {
		return -1, true
	}
	return maxReturnIndexPropertyNumber(), true
}

func maxReturnIndexPropertyNumber() int {
	return int(^uint(0) >> 1)
}
