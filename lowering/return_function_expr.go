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
