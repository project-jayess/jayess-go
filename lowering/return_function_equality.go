package lowering

import "jayess-go/ast"

type functionComparisonReference struct {
	identity int
	fresh    bool
}

func evaluateFunctionEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, scope returnScope) (bool, bool) {
	if !isEqualityOperator(operator) {
		return false, false
	}
	left, leftOK := evaluateFunctionComparisonReference(leftExpression, scope)
	right, rightOK := evaluateFunctionComparisonReference(rightExpression, scope)
	if leftOK && rightOK {
		return compareEquality(operator, sameFunctionComparisonReference(left, right)), true
	}
	if leftOK && isNullishComparisonOperand(rightExpression, scope) {
		return compareEquality(operator, false), true
	}
	if rightOK && isNullishComparisonOperand(leftExpression, scope) {
		return compareEquality(operator, false), true
	}
	return false, false
}

func evaluateFunctionComparisonReference(expression ast.Expression, scope returnScope) (functionComparisonReference, bool) {
	if identity, ok := evaluateFunctionIdentity(expression, scope); ok {
		return functionComparisonReference{identity: identity}, true
	}
	if evaluateFunctionExpression(expression) {
		return functionComparisonReference{fresh: true}, true
	}
	return functionComparisonReference{}, false
}

func sameFunctionComparisonReference(left functionComparisonReference, right functionComparisonReference) bool {
	if left.fresh || right.fresh {
		return false
	}
	return left.identity == right.identity
}

func evaluateFunctionBooleanEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, scope returnScope) (bool, bool) {
	if !isEqualityOperator(operator) {
		return false, false
	}
	if value, ok := compareFunctionBoolean(leftExpression, rightExpression, scope); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := compareFunctionBoolean(rightExpression, leftExpression, scope); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func compareFunctionBoolean(functionExpression ast.Expression, boolExpression ast.Expression, scope returnScope) (bool, bool) {
	if _, ok := evaluateFunctionComparisonReference(functionExpression, scope); !ok {
		return false, false
	}
	if _, ok := evaluateBoolValue(boolExpression, scope); !ok {
		return false, false
	}
	return false, true
}
