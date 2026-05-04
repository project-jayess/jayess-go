package lowering

import (
	"strconv"

	"jayess-go/ast"
)

type returnArrayElement struct {
	intValue     int
	bigIntValue  string
	boolValue    bool
	stringValue  string
	nullishValue returnNullishKind
	funcIdentity int
	objectID     int
	kind         string
}

const (
	returnArrayIntKind     = "int"
	returnArrayBigIntKind  = "bigint"
	returnArrayBoolKind    = "bool"
	returnArrayStringKind  = "string"
	returnArrayNullishKind = "nullish"
	returnArrayFuncKind    = "function"
	returnArrayObjectKind  = "object"
)

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

func evaluateArrayLiteralElements(expression ast.Expression, bindings returnScope) ([]returnArrayElement, bool) {
	array, ok := expression.(*ast.ArrayLiteral)
	if !ok {
		return nil, false
	}
	elements := make([]returnArrayElement, 0, len(array.Elements))
	for _, element := range array.Elements {
		if element == nil {
			elements = append(elements, returnArrayElement{kind: returnArrayNullishKind, nullishValue: returnUndefinedKind})
			continue
		}
		if spread, ok := element.(*ast.SpreadExpression); ok {
			spreadElements, ok := evaluateArrayLiteralElements(spread.Value, bindings)
			if !ok {
				return nil, false
			}
			elements = append(elements, spreadElements...)
			continue
		}
		value, ok := evaluateArrayLiteralElement(element, bindings)
		if !ok {
			return nil, false
		}
		elements = append(elements, value)
	}
	return elements, true
}

func evaluateArrayLiteralElement(expression ast.Expression, bindings returnScope) (returnArrayElement, bool) {
	next := bindings.clone()
	if value, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayIntKind, intValue: value}, true
	}
	next = bindings.clone()
	if value, ok := evaluateBigIntValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayBigIntKind, bigIntValue: value}, true
	}
	next = bindings.clone()
	if value, ok := evaluateStringExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayStringKind, stringValue: value}, true
	}
	next = bindings.clone()
	if value, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayNullishKind, nullishValue: value}, true
	}
	next = bindings.clone()
	if identity, ok := materializeFunctionIdentity(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayFuncKind, funcIdentity: identity}, true
	}
	next = bindings.clone()
	if identity, ok := materializeObjectIdentity(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayObjectKind, objectID: identity}, true
	}
	next = bindings.clone()
	if value, ok := evaluateBoolExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return returnArrayElement{kind: returnArrayBoolKind, boolValue: value}, true
	}
	return returnArrayElement{}, false
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

func evaluateObjectMemberElement(expression *ast.MemberExpression, bindings returnScope) (returnArrayElement, bool) {
	if expression.Property == "" {
		return returnArrayElement{}, false
	}
	next := bindings.clone()
	properties, ok := evaluateObjectLiteralProperties(expression.Target, next)
	if !ok {
		next = bindings.clone()
		properties, ok = evaluateObjectIIFELiteralProperties(expression.Target, next)
		if !ok {
			return returnArrayElement{}, false
		}
	}
	value, ok := properties[expression.Property]
	if !ok {
		value = returnArrayElement{kind: returnArrayNullishKind, nullishValue: returnUndefinedKind}
	}
	replaceReturnScopeBindings(bindings, next)
	return value, true
}

func evaluateObjectIndexElement(expression *ast.IndexExpression, bindings returnScope) (returnArrayElement, bool) {
	next := bindings.clone()
	properties, ok := evaluateObjectLiteralProperties(expression.Target, next)
	if !ok {
		next = bindings.clone()
		properties, ok = evaluateObjectIIFELiteralProperties(expression.Target, next)
		if !ok {
			return returnArrayElement{}, false
		}
	}
	key, ok := evaluateObjectPropertyKey(expression.Index, next)
	if !ok {
		return returnArrayElement{}, false
	}
	value, ok := properties[key]
	if !ok {
		value = returnArrayElement{kind: returnArrayNullishKind, nullishValue: returnUndefinedKind}
	}
	replaceReturnScopeBindings(bindings, next)
	return value, true
}

func evaluateObjectLiteralProperties(expression ast.Expression, bindings returnScope) (map[string]returnArrayElement, bool) {
	object, ok := expression.(*ast.ObjectLiteral)
	if !ok {
		return nil, false
	}
	properties := map[string]returnArrayElement{}
	for _, property := range object.Properties {
		if property.Method || property.Getter || property.Setter {
			return nil, false
		}
		if property.Spread {
			spreadProperties, ok := evaluateObjectLiteralProperties(property.Value, bindings)
			if !ok {
				return nil, false
			}
			for key, value := range spreadProperties {
				properties[key] = value
			}
			continue
		}
		key := property.Key
		if property.Computed {
			var ok bool
			key, ok = evaluateObjectPropertyKey(property.KeyExpr, bindings)
			if !ok {
				return nil, false
			}
		}
		value, ok := evaluateArrayLiteralElement(property.Value, bindings)
		if !ok {
			return nil, false
		}
		properties[key] = value
	}
	return properties, true
}

func evaluateObjectPropertyKey(expression ast.Expression, bindings returnScope) (string, bool) {
	next := bindings.clone()
	if value, ok := evaluateStringCoercion(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return value, true
	}
	return "", false
}
