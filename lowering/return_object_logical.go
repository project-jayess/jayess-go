package lowering

import "jayess-go/ast"

func evaluateLogicalObjectIdentity(operator ast.LogicalOperator, left ast.Expression, right ast.Expression, scope returnScope) (int, bool) {
	switch operator {
	case ast.OperatorAnd:
		next := scope.clone()
		if _, ok := evaluateObjectIdentity(left, next); ok {
			identity, ok := evaluateObjectIdentity(right, next)
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
		identity, ok := evaluateObjectIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case ast.OperatorOr:
		next := scope.clone()
		if identity, ok := evaluateObjectIdentity(left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || leftTruthy {
			return 0, false
		}
		identity, ok := evaluateObjectIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	default:
		return 0, false
	}
}

func evaluateLogicalObjectReferenceExpression(operator ast.LogicalOperator, left ast.Expression, right ast.Expression, scope returnScope) bool {
	switch operator {
	case ast.OperatorAnd:
		next := scope.clone()
		if evaluateObjectReferenceExpression(left, next) {
			if !evaluateObjectReferenceExpression(right, next) {
				return false
			}
			replaceReturnScopeBindings(scope, next)
			return true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || !leftTruthy || !evaluateObjectReferenceExpression(right, next) {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	case ast.OperatorOr:
		next := scope.clone()
		if evaluateObjectReferenceExpression(left, next) {
			replaceReturnScopeBindings(scope, next)
			return true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || leftTruthy || !evaluateObjectReferenceExpression(right, next) {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	default:
		return false
	}
}

func materializeLogicalObjectIdentity(operator ast.LogicalOperator, left ast.Expression, right ast.Expression, scope returnScope) (int, bool) {
	switch operator {
	case ast.OperatorAnd:
		next := scope.clone()
		if _, ok := materializeObjectIdentity(left, next); ok {
			identity, ok := materializeObjectIdentity(right, next)
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
		identity, ok := materializeObjectIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case ast.OperatorOr:
		next := scope.clone()
		if identity, ok := materializeObjectIdentity(left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		leftTruthy, ok := evaluateBoolExpression(left, next)
		if !ok || leftTruthy {
			return 0, false
		}
		identity, ok := materializeObjectIdentity(right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	default:
		return 0, false
	}
}
