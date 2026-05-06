package lowering

import "jayess-go/ast"

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
