package lowering

import "jayess-go/ast"

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
				spreadElements, ok = evaluateArrayIIFELiteralElements(spread.Value, bindings)
				if !ok {
					return nil, false
				}
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
