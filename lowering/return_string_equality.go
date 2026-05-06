package lowering

import "jayess-go/ast"

func evaluateStringEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateStringExpression(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateStringExpression(rightExpression, bindings)
	if !ok {
		return false, false
	}
	return compareEquality(operator, left == right), true
}

func evaluateStringComparison(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateStringExpression(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateStringExpression(rightExpression, bindings)
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
