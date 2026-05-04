package lowering

import "jayess-go/ast"

func returnCodeFromDoWhile(statement *ast.DoWhileStatement, scope returnScope) (int, bool, bool) {
	if value, ok := returnCodeFromStatements(statement.Body, scope); ok {
		return value, true, true
	}
	if statementsContainReturn(statement.Body) {
		return 0, false, false
	}
	bodyScope := scope.clone()
	flow := applyNonReturnStatements(statement.Body, bodyScope)
	if flow == nonReturnFlowBreak {
		replaceReturnScopeBindings(scope, bodyScope)
		return 0, false, true
	}
	if flow != nonReturnFlowNone && flow != nonReturnFlowContinue {
		return 0, false, false
	}
	condition, ok := evaluateBoolExpression(statement.Condition, bodyScope)
	if !ok {
		return 0, false, false
	}
	if !condition {
		replaceReturnScopeBindings(scope, bodyScope)
		return 0, false, true
	}
	return 0, false, false
}

func applyNonReturnDoWhile(statement *ast.DoWhileStatement, scope returnScope) nonReturnFlow {
	_, _, handled := returnCodeFromDoWhile(statement, scope)
	if handled {
		return nonReturnFlowNone
	}
	return applyLoopLabeledFlow(statement.Body, scope)
}
