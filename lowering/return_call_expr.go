package lowering

import "jayess-go/ast"

func evaluateEmptyCallExpression(expression ast.Expression, bindings returnScope) bool {
	call, ok := expression.(*ast.InvokeExpression)
	if !ok {
		return false
	}
	if _, ok := evaluateNullishCallExpression(call, bindings); ok {
		return true
	}
	return false
}

func evaluateNullishCallExpression(call *ast.InvokeExpression, bindings returnScope) (returnNullishKind, bool) {
	next := bindings.clone()
	if call.Optional {
		if _, ok := evaluateNullishExpression(call.Callee, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return returnUndefinedKind, true
		}
	}
	next = bindings.clone()
	if !evaluateEmptyCallableExpression(call.Callee, next) {
		return "", false
	}
	if !evaluateArgumentList(call.Arguments, next) {
		return "", false
	}
	replaceReturnScopeBindings(bindings, next)
	return returnUndefinedKind, true
}

func evaluateEmptyCallableExpression(expression ast.Expression, bindings returnScope) bool {
	switch expr := expression.(type) {
	case *ast.FunctionExpression:
		return len(expr.Body) == 0 && expr.ExpressionBody == nil
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return false
		}
		if !evaluateEmptyCallableExpression(expr.Right, next) {
			return false
		}
		replaceReturnScopeBindings(bindings, next)
		return true
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return false
		}
		var matched bool
		if condition {
			matched = evaluateEmptyCallableExpression(expr.Consequent, next)
		} else {
			matched = evaluateEmptyCallableExpression(expr.Alternative, next)
		}
		if !matched {
			return false
		}
		replaceReturnScopeBindings(bindings, next)
		return true
	default:
		return false
	}
}
