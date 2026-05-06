package lowering

import "jayess-go/ast"

func evaluateBinaryNumericOperand(operator ast.BinaryOperator, expression ast.Expression, bindings returnScope) (int, bool) {
	if operator == ast.OperatorAdd {
		next := bindings.clone()
		if _, ok := evaluateStringExpression(expression, next); ok {
			return 0, false
		}
	}
	return evaluateNumericCoercion(expression, bindings)
}

func evaluateUnaryNumericOperand(operator ast.UnaryOperator, expression ast.Expression, bindings returnScope) (int, bool) {
	switch operator {
	case ast.OperatorNegate, ast.OperatorPositive, ast.OperatorBitNot:
		return evaluateNumericCoercion(expression, bindings)
	default:
		return 0, false
	}
}

func evaluateIntComparison(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateIntExpression(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateIntExpression(rightExpression, bindings)
	if !ok {
		return false, false
	}
	switch operator {
	case ast.OperatorEq, ast.OperatorStrictEq:
		return left == right, true
	case ast.OperatorNe, ast.OperatorStrictNe:
		return left != right, true
	case ast.OperatorLt:
		return left < right, true
	case ast.OperatorLte:
		return left <= right, true
	case ast.OperatorGt:
		return left > right, true
	case ast.OperatorGte:
		return left >= right, true
	default:
		return false, false
	}
}
