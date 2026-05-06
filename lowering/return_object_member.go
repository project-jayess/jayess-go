package lowering

import "jayess-go/ast"

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
				spreadProperties, ok = evaluateObjectIIFELiteralProperties(property.Value, bindings)
				if !ok {
					return nil, false
				}
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
