package lowering

import (
	"strings"

	"jayess-go/ast"
)

func evaluateBigIntEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if !isEqualityOperator(operator) {
		return false, false
	}
	left, ok := evaluateBigIntValue(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateBigIntValue(rightExpression, bindings)
	if !ok {
		return false, false
	}
	return compareEquality(operator, left == right), true
}

func evaluateBigIntRelationalComparison(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if isEqualityOperator(operator) {
		return false, false
	}
	left, ok := evaluateBigIntValue(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateBigIntValue(rightExpression, bindings)
	if !ok {
		return false, false
	}
	order := compareBigIntLiteralValues(left, right)
	return compareBigIntRelationalOrder(operator, order), true
}

func compareBigIntLiteralValues(left string, right string) int {
	leftNegative := strings.HasPrefix(left, "-")
	rightNegative := strings.HasPrefix(right, "-")
	if leftNegative && !rightNegative {
		return -1
	}
	if !leftNegative && rightNegative {
		return 1
	}
	if leftNegative {
		return -compareUnsignedBigIntLiteralValues(strings.TrimPrefix(left, "-"), strings.TrimPrefix(right, "-"))
	}
	return compareUnsignedBigIntLiteralValues(left, right)
}

func compareUnsignedBigIntLiteralValues(left string, right string) int {
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}
