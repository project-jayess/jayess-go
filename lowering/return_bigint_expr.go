package lowering

import "jayess-go/ast"

func evaluateBigIntValue(expression ast.Expression, bindings returnScope) (string, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		value, ok := bindings.bigints[expr.Name]
		return value, ok
	case *ast.MemberExpression:
		return evaluateBigIntMemberExpression(expr, bindings)
	case *ast.IndexExpression:
		return evaluateBigIntIndexExpression(expr, bindings)
	case *ast.UnaryExpression:
		return evaluateBigIntUnary(expr, bindings)
	case *ast.UpdateExpression:
		return evaluateBigIntUpdateExpression(expr, bindings)
	case *ast.InvokeExpression:
		return evaluateBigIntIIFEExpression(expr, bindings)
	case *ast.BinaryExpression:
		return evaluateBigIntBinary(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.LogicalExpression:
		return evaluateBigIntLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return "", false
		}
		var value string
		if condition {
			value, ok = evaluateBigIntValue(expr.Consequent, next)
		} else {
			value, ok = evaluateBigIntValue(expr.Alternative, next)
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
		value, ok := evaluateBigIntValue(expr.Right, next)
		if !ok {
			return "", false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateBigIntValue(expr.Right, next)
			if !ok {
				return "", false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if value, ok := evaluateBigIntValue(expr.Left, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return "", false
	default:
		return evaluateBigIntLiteralValue(expression)
	}
}
