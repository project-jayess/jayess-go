package lowering

import "jayess-go/ast"

func evaluateBoolExpression(expression ast.Expression, bindings returnScope) (bool, bool) {
	switch expr := expression.(type) {
	case *ast.BooleanLiteral:
		return expr.Value, true
	case *ast.Identifier:
		if value, ok := bindings.bools[expr.Name]; ok {
			return value, true
		}
		if value, ok := bindings.ints[expr.Name]; ok {
			return value != 0, true
		}
		if value, ok := bindings.bigints[expr.Name]; ok {
			return isTruthyBigIntValue(value), true
		}
		if value, ok := bindings.strings[expr.Name]; ok {
			return value != "", true
		}
		if _, ok := bindings.nullish[expr.Name]; ok {
			return false, true
		}
		if _, ok := bindings.funcs[expr.Name]; ok {
			return true, true
		}
		if _, ok := bindings.objects[expr.Name]; ok {
			return true, true
		}
		return false, false
	case *ast.NullLiteral, *ast.UndefinedLiteral:
		return false, true
	case *ast.NumberLiteral:
		value, ok := evaluateIntExpression(expr, bindings)
		return value != 0, ok
	case *ast.BigIntLiteral:
		return evaluateBigIntTruthiness(expr)
	case *ast.UpdateExpression:
		if value, ok := evaluateIntUpdateExpression(expr, bindings); ok {
			return value != 0, true
		}
		value, ok := evaluateBigIntUpdateExpression(expr, bindings)
		return isTruthyBigIntValue(value), ok
	case *ast.StringLiteral:
		return expr.Value != "", true
	case *ast.FunctionExpression:
		return true, true
	case *ast.ObjectLiteral, *ast.ArrayLiteral:
		return true, true
	case *ast.NewExpression:
		if !evaluateNewObjectExpression(expr, bindings) {
			return false, false
		}
		return true, true
	case *ast.MemberExpression:
		return evaluateMemberTruthiness(expr, bindings)
	case *ast.IndexExpression:
		return evaluateIndexTruthiness(expr, bindings)
	case *ast.BinaryExpression, *ast.TemplateLiteral, *ast.TypeofExpression:
		return evaluateFoldedExpressionTruthiness(expr, bindings)
	case *ast.UnaryExpression:
		if expr.Operator == ast.OperatorDelete {
			return evaluateDeleteExpression(expr.Right, bindings)
		}
		if expr.Operator != ast.OperatorNot {
			return evaluateFoldedExpressionTruthiness(expr, bindings)
		}
		value, ok := evaluateBoolExpression(expr.Right, bindings)
		if !ok {
			return false, false
		}
		return !value, true
	case *ast.ComparisonExpression:
		return evaluateComparisonExpression(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.InstanceofExpression:
		return evaluateInstanceofExpression(expr, bindings)
	case *ast.InvokeExpression:
		return evaluateBoolIIFEExpression(expr, bindings)
	case *ast.LogicalExpression:
		return evaluateBoolLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return false, false
		}
		var value bool
		if condition {
			value, ok = evaluateBoolExpression(expr.Consequent, next)
		} else {
			value, ok = evaluateBoolExpression(expr.Alternative, next)
		}
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return false, false
		}
		value, ok := evaluateBoolExpression(expr.Right, next)
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateBoolExpression(expr.Right, next)
			if !ok {
				return false, false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if value, ok := evaluateBoolExpression(expr.Left, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return false, false
	default:
		return false, false
	}
}
