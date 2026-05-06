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
