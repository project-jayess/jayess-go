package lowering

import "jayess-go/ast"

func applyVariableDecl(statement *ast.VariableDecl, scope returnScope) {
	if statement.Name == "" || statement.Value == nil {
		return
	}
	if applyReferenceVariableDecl(statement.Name, statement.Value, scope) {
		return
	}
	if applyScalarVariableDecl(statement.Name, statement.Value, scope) {
		return
	}
	evaluateDiscardExpression(statement.Value, scope)
}

func applyScalarVariableDecl(name string, expression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if value, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.ints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateStringExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.strings[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBigIntValue(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bigints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.nullish[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBoolExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bools[name] = value
		return true
	}
	return false
}

func applyReferenceVariableDecl(name string, expression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if identity, ok := materializeFunctionIdentity(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignFunctionIdentity(scope, name, identity)
		return true
	}
	next = scope.clone()
	if identity, ok := materializeObjectIdentity(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignObjectIdentity(scope, name, identity)
		return true
	}
	return false
}
