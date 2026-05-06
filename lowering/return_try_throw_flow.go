package lowering

import "jayess-go/ast"

func applyTryBodyUntilThrow(statements []ast.Statement, scope returnScope) bool {
	return tryBodyUntilThrowFlow(statements, scope) == tryThrowFlowThrow
}

func tryBodyUntilThrowFlow(statements []ast.Statement, scope returnScope) tryThrowFlow {
	for _, statement := range statements {
		flow := applyTryStatementUntilThrow(statement, scope)
		if flow != tryThrowFlowNone {
			return flow
		}
	}
	return tryThrowFlowNone
}

func applyTryStatementUntilThrow(statement ast.Statement, scope returnScope) tryThrowFlow {
	switch stmt := statement.(type) {
	case *ast.ThrowStatement:
		return tryThrowFlowThrow
	case *ast.BlockStatement:
		return tryBodyUntilThrowFlow(stmt.Statements, scope)
	case *ast.IfStatement:
		condition, ok := evaluateBoolExpression(stmt.Condition, scope)
		if !ok {
			return tryThrowFlowStop
		}
		if condition {
			return tryBodyUntilThrowFlow(stmt.Consequence, scope)
		}
		return tryBodyUntilThrowFlow(stmt.Alternative, scope)
	case *ast.SwitchStatement:
		consequent, ok := matchingSwitchConsequent(stmt, scope)
		if !ok {
			return tryThrowFlowStop
		}
		return tryBodyUntilThrowFlow(consequent, scope)
	case *ast.WhileStatement:
		condition, ok := evaluateBoolExpression(stmt.Condition, scope)
		if !ok {
			return tryThrowFlowStop
		}
		if !condition {
			return tryThrowFlowNone
		}
		return tryBodyUntilThrowFlow(stmt.Body, scope)
	case *ast.ForStatement:
		next := scope.clone()
		applyNonReturnStatement(stmt.Init, next)
		if stmt.Condition != nil {
			condition, ok := evaluateBoolExpression(stmt.Condition, next)
			if !ok || !condition {
				if !ok {
					return tryThrowFlowStop
				}
				replaceReturnScopeBindings(scope, next)
				return tryThrowFlowNone
			}
		}
		if tryBodyUntilThrowFlow(stmt.Body, next) != tryThrowFlowThrow {
			return tryThrowFlowStop
		}
		replaceReturnScopeBindings(scope, next)
		return tryThrowFlowThrow
	case *ast.DoWhileStatement:
		return tryBodyUntilThrowFlow(stmt.Body, scope)
	case *ast.LabeledStatement:
		if stmt.Statement == nil {
			return tryThrowFlowNone
		}
		return applyTryStatementUntilThrow(stmt.Statement, scope)
	case *ast.TryStatement:
		next := scope.clone()
		if !applyNestedTryUntilThrow(stmt, next) {
			return tryThrowFlowStop
		}
		replaceReturnScopeBindings(scope, next)
		return tryThrowFlowThrow
	}
	if statementContainsReturn(statement) {
		return tryThrowFlowStop
	}
	if flow := applyNonReturnStatement(statement, scope); flow != nonReturnFlowNone {
		return tryThrowFlowStop
	}
	return tryThrowFlowNone
}

func applyNestedTryUntilThrow(statement *ast.TryStatement, scope returnScope) bool {
	tryFlow := tryBodyUntilThrowFlow(statement.TryBody, scope)
	if tryFlow == tryThrowFlowStop {
		return false
	}
	if tryFlow == tryThrowFlowThrow && tryHasCatch(statement) {
		catchFlow := tryCatchBodyUntilThrowFlow(statement, scope)
		if catchFlow == tryThrowFlowStop {
			return false
		}
		tryFlow = catchFlow
	}
	finallyFlow := tryBodyUntilThrowFlow(statement.FinallyBody, scope)
	if finallyFlow == tryThrowFlowStop {
		return false
	}
	if finallyFlow == tryThrowFlowThrow {
		return true
	}
	return tryFlow == tryThrowFlowThrow
}
