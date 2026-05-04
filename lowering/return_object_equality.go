package lowering

import "jayess-go/ast"

type objectComparisonReference struct {
	identity int
	fresh    bool
}

func evaluateObjectEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, scope returnScope) (bool, bool) {
	if !isEqualityOperator(operator) {
		return false, false
	}
	left, leftOK := evaluateObjectComparisonReference(leftExpression, scope)
	right, rightOK := evaluateObjectComparisonReference(rightExpression, scope)
	if leftOK && rightOK {
		return compareEquality(operator, sameObjectComparisonReference(left, right)), true
	}
	if leftOK && isNullishComparisonOperand(rightExpression, scope) {
		return compareEquality(operator, false), true
	}
	if rightOK && isNullishComparisonOperand(leftExpression, scope) {
		return compareEquality(operator, false), true
	}
	return false, false
}

func evaluateObjectComparisonReference(expression ast.Expression, scope returnScope) (objectComparisonReference, bool) {
	if identity, ok := evaluateObjectIdentity(expression, scope); ok {
		return objectComparisonReference{identity: identity}, true
	}
	if evaluateObjectExpression(expression) {
		return objectComparisonReference{fresh: true}, true
	}
	return objectComparisonReference{}, false
}

func sameObjectComparisonReference(left objectComparisonReference, right objectComparisonReference) bool {
	if left.fresh || right.fresh {
		return false
	}
	return left.identity == right.identity
}

func isNullishComparisonOperand(expression ast.Expression, scope returnScope) bool {
	_, ok := evaluateNullishExpression(expression, scope)
	return ok
}

func evaluateObjectBooleanEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, scope returnScope) (bool, bool) {
	if !isEqualityOperator(operator) {
		return false, false
	}
	if value, ok := compareObjectBoolean(leftExpression, rightExpression, scope); ok {
		return compareEquality(operator, value), true
	}
	if value, ok := compareObjectBoolean(rightExpression, leftExpression, scope); ok {
		return compareEquality(operator, value), true
	}
	return false, false
}

func compareObjectBoolean(objectExpression ast.Expression, boolExpression ast.Expression, scope returnScope) (bool, bool) {
	if _, ok := evaluateObjectComparisonReference(objectExpression, scope); !ok {
		return false, false
	}
	if _, ok := evaluateBoolValue(boolExpression, scope); !ok {
		return false, false
	}
	return false, true
}
