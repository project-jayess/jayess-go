package lowering

import "jayess-go/ast"

const functionSequenceKey = "next"

func evaluateFunctionExpression(expression ast.Expression) bool {
	_, ok := expression.(*ast.FunctionExpression)
	return ok
}

func evaluateFunctionIdentity(expression ast.Expression, scope returnScope) (int, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		identity, ok := scope.funcs[expr.Name]
		return identity, ok
	case *ast.MemberExpression:
		return evaluateFunctionMemberIdentity(expr, scope)
	case *ast.IndexExpression:
		return evaluateFunctionIndexIdentity(expr, scope)
	case *ast.InvokeExpression:
		return evaluateFunctionIIFEExpression(expr, scope)
	case *ast.LogicalExpression:
		return evaluateLogicalFunctionIdentity(expr.Operator, expr.Left, expr.Right, scope)
	case *ast.ConditionalExpression:
		next := scope.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return 0, false
		}
		var identity int
		if condition {
			identity, ok = evaluateFunctionIdentity(expr.Consequent, next)
		} else {
			identity, ok = evaluateFunctionIdentity(expr.Alternative, next)
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
		identity, ok := evaluateFunctionIdentity(expr.Right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case *ast.NullishCoalesceExpression:
		next := scope.clone()
		if identity, ok := evaluateFunctionIdentity(expr.Left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			identity, ok := evaluateFunctionIdentity(expr.Right, next)
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

func allocateFunctionIdentity(scope returnScope) int {
	scope.funcSeq[functionSequenceKey]++
	return scope.funcSeq[functionSequenceKey]
}

func assignFunctionIdentity(scope returnScope, name string, identity int) {
	clearReturnScopeBinding(scope, name)
	scope.funcs[name] = identity
}

func materializeFunctionIdentity(expression ast.Expression, scope returnScope) (int, bool) {
	switch expr := expression.(type) {
	case *ast.CommaExpression:
		next := scope.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return 0, false
		}
		identity, ok := materializeFunctionIdentity(expr.Right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case *ast.FunctionExpression:
		return allocateFunctionIdentity(scope), true
	case *ast.InvokeExpression:
		return evaluateFunctionIIFEExpression(expr, scope)
	case *ast.LogicalExpression:
		return materializeLogicalFunctionIdentity(expr.Operator, expr.Left, expr.Right, scope)
	case *ast.ConditionalExpression:
		next := scope.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return 0, false
		}
		var identity int
		if condition {
			identity, ok = materializeFunctionIdentity(expr.Consequent, next)
		} else {
			identity, ok = materializeFunctionIdentity(expr.Alternative, next)
		}
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(scope, next)
		return identity, true
	case *ast.NullishCoalesceExpression:
		next := scope.clone()
		if identity, ok := materializeFunctionIdentity(expr.Left, next); ok {
			replaceReturnScopeBindings(scope, next)
			return identity, true
		}
		next = scope.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			identity, ok := materializeFunctionIdentity(expr.Right, next)
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
		return evaluateFunctionIdentity(expression, scope)
	}
}

func evaluateFunctionReferenceExpression(expression ast.Expression, scope returnScope) bool {
	if _, ok := evaluateFunctionIdentity(expression, scope); ok {
		return true
	}
	switch expr := expression.(type) {
	case *ast.FunctionExpression:
		return true
	case *ast.LogicalExpression:
		return evaluateLogicalFunctionReferenceExpression(expr.Operator, expr.Left, expr.Right, scope)
	case *ast.ConditionalExpression:
		next := scope.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return false
		}
		var matched bool
		if condition {
			matched = evaluateFunctionReferenceExpression(expr.Consequent, next)
		} else {
			matched = evaluateFunctionReferenceExpression(expr.Alternative, next)
		}
		if !matched {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	case *ast.CommaExpression:
		next := scope.clone()
		if !evaluateDiscardExpression(expr.Left, next) || !evaluateFunctionReferenceExpression(expr.Right, next) {
			return false
		}
		replaceReturnScopeBindings(scope, next)
		return true
	case *ast.NullishCoalesceExpression:
		next := scope.clone()
		if evaluateFunctionReferenceExpression(expr.Left, next) {
			replaceReturnScopeBindings(scope, next)
			return true
		}
		next = scope.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			if !evaluateFunctionReferenceExpression(expr.Right, next) {
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
