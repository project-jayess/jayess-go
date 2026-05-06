package lowering

import "jayess-go/ast"

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
