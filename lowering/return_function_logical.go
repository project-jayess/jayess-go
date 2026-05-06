package lowering

import "jayess-go/ast"

func evaluateLogicalFunctionIdentity(operator ast.LogicalOperator, left ast.Expression, right ast.Expression, scope returnScope) (int, bool) {
	switch operator {
	case ast.OperatorAnd:
		next := scope.clone()
		if _, ok := evaluateFunctionIdentity(left, next); ok {
			identity, ok := evaluateFunctionIdentity(right, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || !leftTruthy {
			return 0, false
		}
		identity, ok := evaluateFunctionIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case ast.OperatorOr:
		next := scope.clone()
		if identity, ok := evaluateFunctionIdentity(left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || leftTruthy {
			return 0, false
		}
		identity, ok := evaluateFunctionIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	default:
		return 0, false
	}
}

func evaluateLogicalFunctionReferenceExpression(operator ast.LogicalOperator, left ast.Expression, right ast.Expression, scope returnScope) bool {
	switch operator {
	case ast.OperatorAnd:
		next := scope.clone()
		if evaluateFunctionReferenceExpression(left, next) {
			if !evaluateFunctionReferenceExpression(right, next) {
				return false
			}
			replaceReturnScopeBindings(scope, next)
			return true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || !leftTruthy || !evaluateFunctionReferenceExpression(right, next) {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	case ast.OperatorOr:
		next := scope.clone()
		if evaluateFunctionReferenceExpression(left, next) {
			replaceReturnScopeBindings(scope, next)
			return true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || leftTruthy || !evaluateFunctionReferenceExpression(right, next) {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	default:
		return false
	}
}

func materializeLogicalFunctionIdentity(operator ast.LogicalOperator, left ast.Expression, right ast.Expression, scope returnScope) (int, bool) {
	switch operator {
	case ast.OperatorAnd:
		next := scope.clone()
		if _, ok := materializeFunctionIdentity(left, next); ok {
			identity, ok := materializeFunctionIdentity(right, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || !leftTruthy {
			return 0, false
		}
		identity, ok := materializeFunctionIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case ast.OperatorOr:
		next := scope.clone()
		if identity, ok := materializeFunctionIdentity(left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || leftTruthy {
			return 0, false
		}
		identity, ok := materializeFunctionIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	default:
		return 0, false
	}
}
