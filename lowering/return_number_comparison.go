package lowering

import "jayess-go/ast"

func evaluateNumericRelationalComparison(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if isEqualityOperator(operator) {
		return false, false
	}
	left, ok := evaluateNumericCoercion(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateNumericCoercion(rightExpression, bindings)
	if !ok {
		return false, false
	}
	switch operator {
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
