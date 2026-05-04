package lowering

import "jayess-go/ast"

func applyExpressionStatement(statement *ast.ExpressionStatement, scope returnScope) {
	evaluateDiscardExpression(statement.Expression, scope)
}

func applyUpdateExpression(expression *ast.UpdateExpression, scope returnScope) {
	identifier, ok := expression.Target.(*ast.Identifier)
	if !ok || identifier.Name == "" {
		return
	}
	value, ok := scope.ints[identifier.Name]
	if ok {
		next, ok := updatedIntValue(expression.Operator, value)
		if ok {
			scope.ints[identifier.Name] = next
		}
		return
	}
	bigValue, ok := scope.bigints[identifier.Name]
	if !ok {
		return
	}
	next, ok := updatedBigIntValue(expression.Operator, bigValue)
	if ok {
		scope.bigints[identifier.Name] = next
	}
}

func applyAssignmentStatement(statement *ast.AssignmentStatement, scope returnScope) {
	identifier, ok := statement.Target.(*ast.Identifier)
	if !ok {
		applyReferenceAssignmentStatement(statement, scope)
		return
	}
	if identifier.Name == "" || statement.Value == nil {
		return
	}
	if applyAssignmentProbe(applyLogicalAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
	if applyAssignmentProbe(applyIntAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
	if applyAssignmentProbe(applyStringAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
	if applyAssignmentProbe(applyBigIntAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
	if applyAssignmentProbe(applyNullishAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
	if applyAssignmentProbe(applyFunctionAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
	if applyAssignmentProbe(applyObjectAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
	if applyAssignmentProbe(applyBoolAssignment, identifier.Name, statement.Operator, statement.Value, scope) {
		return
	}
}

func applyAssignmentProbe(probe func(string, ast.AssignmentOperator, ast.Expression, returnScope) bool, name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if !probe(name, operator, valueExpression, next) {
		return false
	}
	replaceReturnScopeBindings(scope, next)
	return true
}

func applyLogicalAssignment(name string, operator ast.AssignmentOperator, valueExpression ast.Expression, scope returnScope) bool {
	switch operator {
	case ast.AssignmentNullishAssign:
		if _, ok := scope.nullish[name]; !ok {
			return bindingKnown(scope, name)
		}
		return applyScalarAssignment(name, valueExpression, scope)
	case ast.AssignmentOrAssign:
		value, ok := evaluateBoolExpression(&ast.Identifier{Name: name}, scope)
		if !ok {
			return false
		}
		if value {
			return true
		}
		return applyScalarAssignment(name, valueExpression, scope)
	case ast.AssignmentAndAssign:
		value, ok := evaluateBoolExpression(&ast.Identifier{Name: name}, scope)
		if !ok {
			return false
		}
		if !value {
			return true
		}
		return applyScalarAssignment(name, valueExpression, scope)
	default:
		return false
	}
}

func bindingKnown(scope returnScope, name string) bool {
	if _, ok := scope.ints[name]; ok {
		return true
	}
	if _, ok := scope.strings[name]; ok {
		return true
	}
	if _, ok := scope.bigints[name]; ok {
		return true
	}
	if _, ok := scope.nullish[name]; ok {
		return true
	}
	if _, ok := scope.funcs[name]; ok {
		return true
	}
	if _, ok := scope.objects[name]; ok {
		return true
	}
	if _, ok := scope.bools[name]; ok {
		return true
	}
	return false
}

func applyScalarAssignment(name string, valueExpression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if value, ok := evaluateIntExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.ints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateStringExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.strings[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBigIntValue(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bigints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateNullishExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.nullish[name] = value
		return true
	}
	next = scope.clone()
	if identity, ok := materializeFunctionIdentity(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignFunctionIdentity(scope, name, identity)
		return true
	}
	next = scope.clone()
	if identity, ok := materializeObjectIdentity(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignObjectIdentity(scope, name, identity)
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBoolExpression(valueExpression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bools[name] = value
		return true
	}
	return false
}

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
