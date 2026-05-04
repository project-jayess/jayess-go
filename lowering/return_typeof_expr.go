package lowering

import "jayess-go/ast"

func evaluateTypeofString(expression ast.Expression, bindings returnScope) (string, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		return typeofIdentifier(expr.Name, bindings), true
	case *ast.NumberLiteral:
		return "number", true
	case *ast.BigIntLiteral:
		return "bigint", true
	case *ast.StringLiteral:
		return "string", true
	case *ast.BooleanLiteral:
		return "boolean", true
	case *ast.NullLiteral:
		return "object", true
	case *ast.UndefinedLiteral:
		return "undefined", true
	case *ast.FunctionExpression:
		return "function", true
	case *ast.ObjectLiteral, *ast.ArrayLiteral:
		return "object", true
	default:
		next := bindings.clone()
		if _, ok := evaluateIntExpression(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return "number", true
		}
		next = bindings.clone()
		if _, ok := evaluateStringExpression(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return "string", true
		}
		next = bindings.clone()
		if _, ok := evaluateBigIntValue(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return "bigint", true
		}
		next = bindings.clone()
		if evaluateFunctionReferenceExpression(expression, next) {
			replaceReturnScopeBindings(bindings, next)
			return "function", true
		}
		next = bindings.clone()
		if evaluateObjectReferenceExpression(expression, next) {
			replaceReturnScopeBindings(bindings, next)
			return "object", true
		}
		next = bindings.clone()
		if kind, ok := evaluateNullishExpression(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			if kind == returnNullKind {
				return "object", true
			}
			return "undefined", true
		}
		next = bindings.clone()
		if _, ok := evaluateBoolExpression(expression, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return "boolean", true
		}
		return "", false
	}
}

func typeofIdentifier(name string, bindings returnScope) string {
	if _, ok := bindings.ints[name]; ok {
		return "number"
	}
	if _, ok := bindings.bigints[name]; ok {
		return "bigint"
	}
	if _, ok := bindings.strings[name]; ok {
		return "string"
	}
	if _, ok := bindings.bools[name]; ok {
		return "boolean"
	}
	if kind, ok := bindings.nullish[name]; ok {
		if kind == returnNullKind {
			return "object"
		}
		return "undefined"
	}
	if _, ok := bindings.funcs[name]; ok {
		return "function"
	}
	if _, ok := bindings.objects[name]; ok {
		return "object"
	}
	return "undefined"
}
