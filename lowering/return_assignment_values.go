package lowering

import "jayess-go/ast"

func applyIntAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	if operator == ast.AssignmentAssign {
		right, ok := evaluateIntExpression(valueExpression, scope)
		if !ok {
			return false
		}
		clearReturnScopeBinding(scope, name)
		scope.ints[name] = right
		return true
	}
	left, ok := scope.ints[name]
	if !ok {
		return false
	}
	right, ok := evaluateNumericAssignmentOperand(operator, valueExpression, scope)
	if !ok {
		return false
	}
	value, ok := evaluateIntAssignment(operator, left, right)
	if !ok {
		return false
	}
	clearReturnScopeBinding(scope, name)
	scope.ints[name] = value
	return true
}

func evaluateNumericAssignmentOperand(operator ast.AssignmentOperator, expression ast.Expression, scope returnScope) (int, bool) {
	if operator == ast.AssignmentAddAssign {
		next := scope.clone()
		if _, ok := evaluateStringExpression(expression, next); ok {
			return 0, false
		}
	}
	return evaluateNumericCoercion(expression, scope)
}

func applyBoolAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	if operator != ast.AssignmentAssign {
		return false
	}
	value, ok := evaluateBoolExpression(valueExpression, scope)
	if !ok {
		return false
	}
	clearReturnScopeBinding(scope, name)
	scope.bools[name] = value
	return true
}

func applyStringAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	if operator == ast.AssignmentAddAssign {
		left, ok := scope.strings[name]
		if !ok {
			return false
		}
		right, ok := evaluateStringCoercion(valueExpression, scope)
		if !ok {
			return false
		}
		clearReturnScopeBinding(scope, name)
		scope.strings[name] = left + right
		return true
	}
	if operator != ast.AssignmentAssign {
		return false
	}
	value, ok := evaluateStringExpression(valueExpression, scope)
	if !ok {
		return false
	}
	clearReturnScopeBinding(scope, name)
	scope.strings[name] = value
	return true
}

func applyBigIntAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	if operator == ast.AssignmentAssign {
		value, ok := evaluateBigIntValue(valueExpression, scope)
		if !ok {
			return false
		}
		clearReturnScopeBinding(scope, name)
		scope.bigints[name] = value
		return true
	}
	binaryOperator, ok := assignmentBigIntBinaryOperator(operator)
	if !ok {
		return false
	}
	left, ok := scope.bigints[name]
	if !ok {
		return false
	}
	right, ok := evaluateBigIntValue(valueExpression, scope)
	if !ok {
		return false
	}
	value, ok := evaluateBigIntBinaryValue(binaryOperator, left, right)
	if !ok {
		return false
	}
	clearReturnScopeBinding(scope, name)
	scope.bigints[name] = value
	return true
}

func applyNullishAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	if operator != ast.AssignmentAssign {
		return false
	}
	value, ok := evaluateNullishExpression(valueExpression, scope)
	if !ok {
		return false
	}
	clearReturnScopeBinding(scope, name)
	scope.nullish[name] = value
	return true
}

func applyFunctionAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	if operator != ast.AssignmentAssign {
		return false
	}
	if identity, ok := materializeFunctionIdentity(valueExpression, scope); ok {
		assignFunctionIdentity(scope, name, identity)
		return true
	}
	return false
}

func applyObjectAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	if operator != ast.AssignmentAssign {
		return false
	}
	if identity, ok := materializeObjectIdentity(valueExpression, scope); ok {
		assignObjectIdentity(scope, name, identity)
		return true
	}
	return false
}
