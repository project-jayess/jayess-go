package lowering

import "jayess-go/ast"

func returnCodeFromWhile(statement *ast.WhileStatement, scope returnScope) (int, bool, bool) {
	condition, ok := evaluateBoolExpression(statement.Condition, scope)
	if !ok {
		return 0, false, false
	}
	if !condition {
		return 0, false, true
	}
	value, returns := returnCodeFromStatements(statement.Body, scope)
	if returns {
		return value, true, true
	}
	return applyLoopBreakSideEffects(statement.Body, scope, scope)
}

func returnCodeFromFor(statement *ast.ForStatement, scope returnScope) (int, bool, bool) {
	next := scope.clone()
	applyNonReturnStatement(statement.Init, next)
	if statement.Condition == nil {
		value, returns := returnCodeFromStatements(statement.Body, next)
		if returns {
			return value, true, true
		}
		return applyLoopBreakSideEffects(statement.Body, next, scope)
	}
	condition, ok := evaluateBoolExpression(statement.Condition, next)
	if !ok {
		replaceReturnScopeBindings(scope, next)
		return 0, false, false
	}
	if !condition {
		replaceReturnScopeBindings(scope, next)
		return 0, false, true
	}
	value, returns := returnCodeFromStatements(statement.Body, next)
	if returns {
		return value, true, true
	}
	return applyLoopBreakSideEffects(statement.Body, next, scope)
}

func applyLoopBreakSideEffects(statements []ast.Statement, source returnScope, target returnScope) (int, bool, bool) {
	next := source.clone()
	if applyNonReturnStatements(statements, next) != nonReturnFlowBreak {
		return 0, false, false
	}
	replaceReturnScopeBindings(target, next)
	return 0, false, true
}

func applyNonReturnWhile(statement *ast.WhileStatement, scope returnScope) nonReturnFlow {
	_, _, handled := returnCodeFromWhile(statement, scope)
	if handled {
		return nonReturnFlowNone
	}
	condition, ok := evaluateBoolExpression(statement.Condition, scope)
	if !ok || !condition {
		return nonReturnFlowNone
	}
	return applyLoopLabeledFlow(statement.Body, scope)
}

func applyNonReturnFor(statement *ast.ForStatement, scope returnScope) nonReturnFlow {
	_, _, handled := returnCodeFromFor(statement, scope)
	if handled {
		return nonReturnFlowNone
	}
	next := scope.clone()
	applyNonReturnStatement(statement.Init, next)
	if statement.Condition != nil {
		condition, ok := evaluateBoolExpression(statement.Condition, next)
		if !ok || !condition {
			if !ok {
				replaceReturnScopeBindings(scope, next)
			}
			return nonReturnFlowNone
		}
	}
	flow := applyLoopLabeledFlow(statement.Body, next)
	if flow == nonReturnFlowNone {
		return nonReturnFlowNone
	}
	replaceReturnScopeBindings(scope, next)
	return flow
}

func applyLoopLabeledFlow(statements []ast.Statement, scope returnScope) nonReturnFlow {
	next := scope.clone()
	flow := applyNonReturnStatements(statements, next)
	if flow.label == "" {
		return nonReturnFlowNone
	}
	replaceReturnScopeBindings(scope, next)
	return flow
}

func whileBlocksFollowing(statement *ast.WhileStatement, scope returnScope) bool {
	condition, ok := evaluateBoolExpression(statement.Condition, scope)
	return ok && condition
}

func forBlocksFollowing(statement *ast.ForStatement, scope returnScope) bool {
	if statement.Condition == nil {
		return true
	}
	next := scope.clone()
	applyNonReturnStatement(statement.Init, next)
	condition, ok := evaluateBoolExpression(statement.Condition, next)
	return ok && condition
}

func doWhileBlocksFollowing(statement *ast.DoWhileStatement, scope returnScope) bool {
	bodyScope := scope.clone()
	flow := applyNonReturnStatements(statement.Body, bodyScope)
	if flow != nonReturnFlowNone && flow != nonReturnFlowContinue {
		return false
	}
	condition, ok := evaluateBoolExpression(statement.Condition, bodyScope)
	return ok && condition
}
