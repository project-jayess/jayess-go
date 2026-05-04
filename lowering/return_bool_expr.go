package lowering

import "jayess-go/ast"

func evaluateBoolExpression(expression ast.Expression, bindings returnScope) (bool, bool) {
	switch expr := expression.(type) {
	case *ast.BooleanLiteral:
		return expr.Value, true
	case *ast.Identifier:
		if value, ok := bindings.bools[expr.Name]; ok {
			return value, true
		}
		if value, ok := bindings.ints[expr.Name]; ok {
			return value != 0, true
		}
		if value, ok := bindings.bigints[expr.Name]; ok {
			return isTruthyBigIntValue(value), true
		}
		if value, ok := bindings.strings[expr.Name]; ok {
			return value != "", true
		}
		if _, ok := bindings.nullish[expr.Name]; ok {
			return false, true
		}
		if _, ok := bindings.funcs[expr.Name]; ok {
			return true, true
		}
		if _, ok := bindings.objects[expr.Name]; ok {
			return true, true
		}
		return false, false
	case *ast.NullLiteral, *ast.UndefinedLiteral:
		return false, true
	case *ast.NumberLiteral:
		value, ok := evaluateIntExpression(expr, bindings)
		return value != 0, ok
	case *ast.BigIntLiteral:
		return evaluateBigIntTruthiness(expr)
	case *ast.UpdateExpression:
		if value, ok := evaluateIntUpdateExpression(expr, bindings); ok {
			return value != 0, true
		}
		value, ok := evaluateBigIntUpdateExpression(expr, bindings)
		return isTruthyBigIntValue(value), ok
	case *ast.StringLiteral:
		return expr.Value != "", true
	case *ast.FunctionExpression:
		return true, true
	case *ast.ObjectLiteral, *ast.ArrayLiteral:
		return true, true
	case *ast.NewExpression:
		if !evaluateNewObjectExpression(expr, bindings) {
			return false, false
		}
		return true, true
	case *ast.MemberExpression:
		return evaluateMemberTruthiness(expr, bindings)
	case *ast.IndexExpression:
		return evaluateIndexTruthiness(expr, bindings)
	case *ast.BinaryExpression, *ast.TemplateLiteral, *ast.TypeofExpression:
		return evaluateFoldedExpressionTruthiness(expr, bindings)
	case *ast.UnaryExpression:
		if expr.Operator == ast.OperatorDelete {
			return evaluateDeleteExpression(expr.Right, bindings)
		}
		if expr.Operator != ast.OperatorNot {
			return evaluateFoldedExpressionTruthiness(expr, bindings)
		}
		value, ok := evaluateBoolExpression(expr.Right, bindings)
		if !ok {
			return false, false
		}
		return !value, true
	case *ast.ComparisonExpression:
		return evaluateComparisonExpression(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.InstanceofExpression:
		return evaluateInstanceofExpression(expr, bindings)
	case *ast.InvokeExpression:
		return evaluateBoolIIFEExpression(expr, bindings)
	case *ast.LogicalExpression:
		return evaluateBoolLogical(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return false, false
		}
		var value bool
		if condition {
			value, ok = evaluateBoolExpression(expr.Consequent, next)
		} else {
			value, ok = evaluateBoolExpression(expr.Alternative, next)
		}
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.CommaExpression:
		next := bindings.clone()
		if !evaluateDiscardExpression(expr.Left, next) {
			return false, false
		}
		value, ok := evaluateBoolExpression(expr.Right, next)
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	case *ast.NullishCoalesceExpression:
		next := bindings.clone()
		if _, ok := evaluateNullishExpression(expr.Left, next); ok {
			value, ok := evaluateBoolExpression(expr.Right, next)
			if !ok {
				return false, false
			}
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if value, ok := evaluateBoolExpression(expr.Left, next); ok {
			replaceReturnScopeBindings(bindings, next)
			return value, true
		}
		next = bindings.clone()
		if evaluateDiscardExpression(expr.Left, next) {
			replaceReturnScopeBindings(bindings, next)
		}
		return false, false
	default:
		return false, false
	}
}

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

func evaluateBoolEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator == ast.OperatorStrictEq || operator == ast.OperatorStrictNe {
		left, leftOK := evaluateBoolValue(leftExpression, bindings)
		right, rightOK := evaluateBoolValue(rightExpression, bindings)
		if leftOK && rightOK {
			return compareEquality(operator, left == right), true
		}
		if leftOK || rightOK {
			return compareEquality(operator, false), true
		}
		return false, false
	}
	left, leftOK := evaluateBoolValue(leftExpression, bindings)
	right, rightOK := evaluateBoolValue(rightExpression, bindings)
	if !leftOK || !rightOK {
		return false, false
	}
	return compareEquality(operator, left == right), true
}

func evaluateBoolValue(expression ast.Expression, bindings returnScope) (bool, bool) {
	switch expr := expression.(type) {
	case *ast.BooleanLiteral:
		return expr.Value, true
	case *ast.Identifier:
		value, ok := bindings.bools[expr.Name]
		return value, ok
	case *ast.MemberExpression:
		return evaluateBoolMemberExpression(expr, bindings)
	case *ast.IndexExpression:
		return evaluateBoolIndexExpression(expr, bindings)
	case *ast.InstanceofExpression:
		return evaluateInstanceofExpression(expr, bindings)
	case *ast.UnaryExpression:
		if expr.Operator == ast.OperatorDelete {
			return evaluateDeleteExpression(expr.Right, bindings)
		}
		if expr.Operator != ast.OperatorNot {
			return false, false
		}
		value, ok := evaluateBoolExpression(expr.Right, bindings)
		if !ok {
			return false, false
		}
		return !value, true
	case *ast.ComparisonExpression:
		return evaluateComparisonExpression(expr.Operator, expr.Left, expr.Right, bindings)
	case *ast.ConditionalExpression:
		next := bindings.clone()
		condition, ok := evaluateBoolExpression(expr.Condition, next)
		if !ok {
			return false, false
		}
		var value bool
		if condition {
			value, ok = evaluateBoolValue(expr.Consequent, next)
		} else {
			value, ok = evaluateBoolValue(expr.Alternative, next)
		}
		if !ok {
			return false, false
		}
		replaceReturnScopeBindings(bindings, next)
		return value, true
	default:
		return false, false
	}
}

func evaluateStrictPrimitiveMismatch(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	if operator != ast.OperatorStrictEq && operator != ast.OperatorStrictNe {
		return false, false
	}
	left, ok := evaluatePrimitiveKind(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluatePrimitiveKind(rightExpression, bindings)
	if !ok || left == right {
		return false, false
	}
	return compareEquality(operator, false), true
}

func evaluatePrimitiveKind(expression ast.Expression, bindings returnScope) (string, bool) {
	next := bindings.clone()
	if _, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return "number", true
	}
	next = bindings.clone()
	if _, ok := evaluateStringExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return "string", true
	}
	next = bindings.clone()
	if _, ok := evaluateBigIntValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return "bigint", true
	}
	next = bindings.clone()
	if _, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return "nullish", true
	}
	next = bindings.clone()
	if _, ok := evaluateBoolValue(expression, next); ok {
		replaceReturnScopeBindings(bindings, next)
		return "boolean", true
	}
	return "", false
}

func evaluateNullishEquality(operator ast.ComparisonOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateNullishExpression(leftExpression, bindings)
	if !ok {
		return false, false
	}
	right, ok := evaluateNullishExpression(rightExpression, bindings)
	if !ok {
		return false, false
	}
	return compareNullishEquality(operator, left, right), true
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

func compareNullishEquality(operator ast.ComparisonOperator, left returnNullishKind, right returnNullishKind) bool {
	switch operator {
	case ast.OperatorEq:
		return true
	case ast.OperatorNe:
		return false
	case ast.OperatorStrictEq:
		return left == right
	case ast.OperatorStrictNe:
		return left != right
	default:
		return false
	}
}

func evaluateBoolLogical(operator ast.LogicalOperator, leftExpression ast.Expression, rightExpression ast.Expression, bindings returnScope) (bool, bool) {
	left, ok := evaluateBoolExpression(leftExpression, bindings)
	if !ok {
		return false, false
	}
	switch operator {
	case ast.OperatorAnd:
		if !left {
			return false, true
		}
		right, ok := evaluateBoolExpression(rightExpression, bindings)
		return right, ok
	case ast.OperatorOr:
		if left {
			return true, true
		}
		right, ok := evaluateBoolExpression(rightExpression, bindings)
		return right, ok
	default:
		return false, false
	}
}
