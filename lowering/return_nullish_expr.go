package lowering

import "jayess-go/ast"

func evaluateNullishExpression(expression ast.Expression, bindings returnScope) (returnNullishKind, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		value, ok := bindings.nullish[expr.Name]
		return value, ok
	case *ast.NullLiteral:
		return returnNullKind, true
	case *ast.UndefinedLiteral:
		return returnUndefinedKind, true
	case *ast.MemberExpression:
		return evaluateNullishMemberExpression(expr, bindings)
	case *ast.IndexExpression:
		if value, ok := evaluateArrayNullishIndexExpression(expr, bindings); ok {
			return value, true
		}
		return evaluateNullishIndexExpression(expr, bindings)
	case *ast.UnaryExpression:
		if expr.Operator != ast.OperatorVoid {
			return "", false
		}
		if !evaluateDiscardExpression(expr.Right, bindings) {
			return "", false
		}
		return returnUndefinedKind, true
	case *ast.InvokeExpression:
		if value, ok := evaluateNullishIIFEExpression(expr, bindings); ok {
			return value, true
		}
		return evaluateNullishCallExpression(expr, bindings)
	case *ast.LogicalExpression:
		return evaluateNullishLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return "", false
		}
		var value returnNullishKind
		if condition {
			value, ok = evaluateNullishExpression(expr.Consequent, next)
		} else {
			value, ok = evaluateNullishExpression(expr.Alternative, next)
		}
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return "", false
		}
		value, ok := evaluateNullishExpression(expr.Right, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateNullishExpression(expr.Right, next)
			if !ok {
				return "", false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return "", false
	default:
		return "", false
	}
}
