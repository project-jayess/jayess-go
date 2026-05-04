package lowering

import "jayess-go/ast"

const objectSequenceKey = "next"

func evaluateObjectExpression(expression ast.Expression) bool {
	switch expression.(type) {
	case *ast.ObjectLiteral, *ast.ArrayLiteral:
		return true
	default:
		return false
	}
}

func evaluateObjectIdentity(expression ast.Expression, scope returnScope) (int, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		identity, ok := scope.objects[expr.Name]
		return identity, ok
	case *ast.MemberExpression:
		return evaluateObjectMemberIdentity(expr, scope)
	case *ast.IndexExpression:
		return evaluateObjectIndexIdentity(expr, scope)
	case *ast.InvokeExpression:
		return evaluateObjectIIFEExpression(expr, scope)
	case *ast.LogicalExpression:
		return evaluateLogicalObjectIdentity(expr.Operator, expr.Left, expr.Right, scope)
	case *ast.ConditionalExpression:
		next := scope.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return 0, false
		}
		var identity int
		if condition {
			identity, ok = evaluateObjectIdentity(expr.Consequent, next)
		} else {
			identity, ok = evaluateObjectIdentity(expr.Alternative, next)
		}
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case *ast.CommaExpression:
		next := scope.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return 0, false
		}
		identity, ok := evaluateObjectIdentity(expr.Right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case *ast.NullishCoalesceExpression:
		next := scope.clone()
		if identity, ok := evaluateObjectIdentity(expr.Left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			identity, ok := evaluateObjectIdentity(expr.Right, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(scope, next)
		}
		return 0, false
	default:
		return 0, false
	}
}

func allocateObjectIdentity(scope returnScope) int {
	scope.objectSeq[objectSequenceKey]++
	return scope.objectSeq[objectSequenceKey]
}

func assignObjectIdentity(scope returnScope, name string, identity int) {
	clearReturnScopeBinding(scope, name)
	scope.objects[name] = identity
}

func materializeObjectIdentity(expression ast.Expression, scope returnScope) (int, bool) {
	switch expr := expression.(type) {
	case *ast.CommaExpression:
		next := scope.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return 0, false
		}
		identity, ok := materializeObjectIdentity(expr.Right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case *ast.ObjectLiteral, *ast.ArrayLiteral:
		return allocateObjectIdentity(scope), true
	case *ast.InvokeExpression:
		return evaluateObjectIIFEExpression(expr, scope)
	case *ast.LogicalExpression:
		return materializeLogicalObjectIdentity(expr.Operator, expr.Left, expr.Right, scope)
	case *ast.ConditionalExpression:
		next := scope.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return 0, false
		}
		var identity int
		if condition {
			identity, ok = materializeObjectIdentity(expr.Consequent, next)
		} else {
			identity, ok = materializeObjectIdentity(expr.Alternative, next)
		}
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case *ast.NullishCoalesceExpression:
		next := scope.clone()
		if identity, ok := materializeObjectIdentity(expr.Left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			identity, ok := materializeObjectIdentity(expr.Right, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(scope, next)
		}
		return 0, false
	case *ast.NewExpression:
		if !evaluateNewObjectExpression(expr, scope) {
			return 0, false
		}
		return allocateObjectIdentity(scope), true
	default:
		return evaluateObjectIdentity(expression, scope)
	}
}

func evaluateObjectReferenceExpression(expression ast.Expression, scope returnScope) bool {
	if _, ok := evaluateObjectIdentity(expression, scope); ok {
		return true
	}
	switch expr := expression.(type) {
	case *ast.ObjectLiteral, *ast.ArrayLiteral:
		return true
	case *ast.NewExpression:
		return evaluateNewObjectExpression(expr, scope)
	case *ast.LogicalExpression:
		return evaluateLogicalObjectReferenceExpression(expr.Operator, expr.Left, expr.Right, scope)
	case *ast.ConditionalExpression:
		next := scope.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return false
		}
		var matched bool
		if condition {
			matched = evaluateObjectReferenceExpression(expr.Consequent, next)
		} else {
			matched = evaluateObjectReferenceExpression(expr.Alternative, next)
		}
		if !matched {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	case *ast.CommaExpression:
		next := scope.clone()
		if !evaluateDiscardExpression(expr.Left, next) || !evaluateObjectReferenceExpression(expr.Right, next) {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	case *ast.NullishCoalesceExpression:
		next := scope.clone()
		if evaluateObjectReferenceExpression(expr.Left, next) {
			replaceReturnScopeBindings(scope, next)
			return true
		}
		next = scope.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			if !evaluateObjectReferenceExpression(expr.Right, next) {
				return false
			}
			replaceReturnScopeBindings(scope, next)
			return true
		}
		next = scope.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(scope, next)
		}
		return false
	default:
		return false
	}
}

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
