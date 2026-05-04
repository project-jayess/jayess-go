package lowering

import (
	"strconv"

	"jayess-go/ast"
)

func evaluateBigIntMixedRelationalComparison(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if isEqualityOperator(operator) {
		return false, false
	}
	if value, ok := evaluateBigIntLeftMixedRelational(operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	return evaluateMixedLeftBigIntRelational(operator, leftExpression, rightExpression, bindings)
}

func evaluateBigIntLeftMixedRelational(operator ast.ComparisonOperator, bigIntExpression ast.Expression, mixedExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateBigIntValue(bigIntExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateMixedRelationalBigIntValue(mixedExpression, bindings)
	if !ok {
		return false, false
	}
	if right == "" {
		return false, true
	}
	return compareBigIntRelationalOrder(operator, compareBigIntLiteralValues(left, right)), true
}

func evaluateMixedLeftBigIntRelational(operator ast.ComparisonOperator, mixedExpression ast.Expression, bigIntExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateMixedRelationalBigIntValue(mixedExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateBigIntValue(bigIntExpression, bindings)
	if !ok {
		return false, false
	}
	if left == "" {
		return false, true
	}
	return compareBigIntRelationalOrder(operator, compareBigIntLiteralValues(left, right)), true
}

func evaluateMixedRelationalBigIntValue(expression ast.Expression, bindings returnScope) (string, bool) {
	next := bindings.clone()
	if text, ok := evaluateStringExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return parseMixedRelationalStringBigInt(text), true
	}
	next = bindings.clone()
	if number, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return strconv.Itoa(number), true
	}
	next = bindings.clone()
	if boolean, ok := evaluateBoolValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		if boolean {
			return "1", true
		}
		return "0", true
	}
	return "", false
}

func parseMixedRelationalStringBigInt(text string) string {
	value, ok := parseLooseStringBigInt(text)
	if !ok {
		return ""
	}
	return value
}

func compareBigIntRelationalOrder(operator ast.ComparisonOperator, order int) bool {
	switch operator {
	case ast.OperatorLt:
		return order < 0
	case ast.OperatorLte:
		return order <= 0
	case ast.OperatorGt:
		return order > 0
	case ast.OperatorGte:
		return order >= 0
	default:
		return false
	}
}
