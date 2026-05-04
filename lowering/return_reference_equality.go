package lowering

import "jayess-go/ast"

func evaluateReferenceStrictPrimitiveMismatch(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, scope returnScope) (bool, bool) {
	if operator != ast.OperatorStrictEq && operator != ast.OperatorStrictNe {
		return false, false
	}
	if evaluateReferencePrimitiveMismatchProbe(leftExpression, rightExpression, scope) {
		return compareEquality(operator, false), true
	}
	if evaluateReferencePrimitiveMismatchProbe(rightExpression, leftExpression, scope) {
		return compareEquality(operator, false), true
	}
	return false, false
}

func evaluateReferencePrimitiveMismatchProbe(referenceExpression ast.Expression, primitiveExpression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if !referencePrimitiveMismatch(referenceExpression, primitiveExpression, next) {
		return false
	}
	replaceReturnScopeBindings(scope, next)
	return true
}

func referencePrimitiveMismatch(referenceExpression ast.Expression, primitiveExpression ast.Expression, scope returnScope) bool {
	if !isReferenceComparisonOperand(referenceExpression, scope) {
		return false
	}
	_, ok := evaluatePrimitiveKind(primitiveExpression, scope)
	return ok
}

func isReferenceComparisonOperand(expression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if _, ok := evaluateObjectComparisonReference(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		return true
	}
	next = scope.clone()
	if _, ok := evaluateFunctionComparisonReference(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		return true
	}
	return false
}
