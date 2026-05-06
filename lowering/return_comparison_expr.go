package lowering

import "jayess-go/ast"

func evaluateComparisonExpression(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator == ast.OperatorIn {
		return evaluateInExpression(leftExpression, rightExpression, bindings)
	}
	if value, ok := evaluateComparisonProbe(evaluateIntComparison, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateStringComparison, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateBigIntRelationalComparison, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateBigIntMixedRelationalComparison, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateNumericRelationalComparison, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if !isEqualityOperator(operator) {
		return false, false
	}
	if value, ok := evaluateComparisonProbe(evaluateStringEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateNullishEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateBigIntEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateObjectEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateObjectBooleanEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateFunctionEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateFunctionBooleanEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateReferenceStrictPrimitiveMismatch, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateLooseNullishPrimitiveMismatch, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateLooseStringNumberEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateLooseStringBooleanEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateLooseBooleanNumberEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateLooseStringBigIntEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateLooseNumberBigIntEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateLooseBooleanBigIntEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateStrictPrimitiveMismatch, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	if value, ok := evaluateComparisonProbe(evaluateBoolEquality, operator, leftExpression, rightExpression, bindings); ok {
		return value, true
	}
	return false, false
}

func evaluateComparisonProbe(probe func(ast.ComparisonOperator, ast.Expression, ast.Expression, returnScope) (bool, bool), operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	next := bindings.clone()
	value, ok := probe(operator, leftExpression, rightExpression, next)
	if !ok {
		return false, false
	}
	replaceReturnScopeBindings(bindings, next)
	return value, true
}

func isEqualityOperator(operator ast.ComparisonOperator) bool {
	switch operator {
	case ast.OperatorEq, ast.OperatorStrictEq, ast.OperatorNe, ast.OperatorStrictNe:
		return true
	default:
		return false
	}
}

func compareEquality(operator ast.ComparisonOperator, equal bool) bool {
	switch operator {
	case ast.OperatorEq, ast.OperatorStrictEq:
		return equal
	case ast.OperatorNe, ast.OperatorStrictNe:
		return !equal
	default:
		return false
	}
}
