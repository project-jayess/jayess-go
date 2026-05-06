package lowering

import (
	"strconv"

	"jayess-go/ast"
)

func evaluateIntExpression(expression ast.Expression, bindings returnScope) (int, bool) {
	switch expr := expression.(type) {
	case *ast.Identifier:
		value, ok := bindings.ints[expr.Name]
		return value, ok
	case *ast.NumberLiteral:
		value, err := strconv.ParseFloat(expr.Value, 64)
		if err != nil {
			return 0, false
		}
		return int(value), true
	case *ast.MemberExpression:
		return evaluateIntMemberExpression(expr, bindings)
	case *ast.IndexExpression:
		return evaluateIntIndexExpression(expr, bindings)
	case *ast.UnaryExpression:
		value, ok := evaluateUnaryNumericOperand(expr.Operator, expr.Right, bindings)
		if !ok {
			return 0, false
		}
		switch expr.Operator {
		case ast.OperatorNegate:
			return -value, true
		case ast.OperatorPositive:
			return value, true
		case ast.OperatorBitNot:
			return ^value, true
		default:
			return 0, false
		}
	case *ast.UpdateExpression:
		return evaluateIntUpdateExpression(expr, bindings)
	case *ast.InvokeExpression:
		return evaluateIntIIFEExpression(expr, bindings)
	case *ast.BinaryExpression:
		left, ok := evaluateBinaryNumericOperand(expr.Operator, expr.Left, bindings)
		if !ok {
			return 0, false
		}
		right, ok := evaluateBinaryNumericOperand(expr.Operator, expr.Right, bindings)
		if !ok {
			return 0, false
		}
		return evaluateIntBinary(expr.Operator, left, right)
	case *ast.LogicalExpression:
		return evaluateIntLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return 0, false
		}
		var value int
		if condition {
			value, ok = evaluateIntExpression(expr.Consequent, next)
		} else {
			value, ok = evaluateIntExpression(expr.Alternative, next)
		}
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return 0, false
		}
		value, ok := evaluateIntExpression(expr.Right, next)
		if !ok {
			return 0, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateIntExpression(expr.Right, next)
			if !ok {
				return 0, false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if value, ok := evaluateIntExpression(expr.Left, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return 0, false
	default:
		return 0, false
	}
}
