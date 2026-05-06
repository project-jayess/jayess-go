package lowering

import "jayess-go/ast"

func applyLogicalAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	switch operator {
	case ast.AssignmentNullishAssign:
		if _, ok := scope.nullish[name]; !ok {
			return bindingKnown(scope, name)
		}
		return applyScalarAssignment(name, valueExpression, scope)
	case ast.AssignmentOrAssign:
		value, ok := evaluateBoolExpression(&ast.Identifier{Name: name}, scope)
		if !ok {
			return false
		}
		if value {
			return true
		}
		return applyScalarAssignment(name, valueExpression, scope)
	case ast.AssignmentAndAssign:
		value, ok := evaluateBoolExpression(&ast.Identifier{Name: name}, scope)
		if !ok {
			return false
		}
		if !value {
			return true
		}
		return applyScalarAssignment(name, valueExpression, scope)
	default:
		return false
	}
}

func bindingKnown(scope returnScope, name string) bool {
	if _, ok := scope.ints[name]; ok {
		return true
	}
	if _, ok := scope.strings[name]; ok {
		return true
	}
	if _, ok := scope.bigints[name]; ok {
		return true
	}
	if _, ok := scope.nullish[name]; ok {
		return true
	}
	if _, ok := scope.funcs[name]; ok {
		return true
	}
	if _, ok := scope.objects[name]; ok {
		return true
	}
	if _, ok := scope.bools[name]; ok {
		return true
	}
	return false
}

func applyScalarAssignment(name string, valueExpression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if value, ok := evaluateIntExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.ints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateStringExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.strings[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBigIntValue(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bigints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateNullishExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.nullish[name] = value
		return true
	}
	next = scope.clone()
	if identity, ok := materializeFunctionIdentity(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignFunctionIdentity(scope, name, identity)
		return true
	}
	next = scope.clone()
	if identity, ok := materializeObjectIdentity(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignObjectIdentity(scope, name, identity)
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBoolExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bools[name] = value
		return true
	}
	return false
}
