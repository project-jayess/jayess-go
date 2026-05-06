package lowering

import "jayess-go/ast"

const objectSequenceKey = "next"

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
