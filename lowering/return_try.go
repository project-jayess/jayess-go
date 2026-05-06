package lowering

import "jayess-go/ast"

type tryThrowFlow int

const (
	tryThrowFlowNone tryThrowFlow = iota
	tryThrowFlowThrow
	tryThrowFlowStop
)

func applyNonReturnTry(statement *ast.TryStatement, scope returnScope) nonReturnFlow {
	if statementContainsReturn(statement) {
		return nonReturnFlowNone
	}
	tryFlow := applyNonReturnStatements(statement.TryBody, scope)
	finallyFlow := applyNonReturnStatements(statement.FinallyBody, scope)
	if finallyFlow != nonReturnFlowNone {
		return finallyFlow
	}
	return tryFlow
}

func applyTrySideEffects(statement *ast.TryStatement, scope returnScope) (nonReturnFlow, bool) {
	if !statementContainsReturn(statement) {
		return applyNonReturnTry(statement, scope), true
	}
	next := scope.clone()
	if !applyTryBodyUntilThrow(statement.TryBody, next) {
		return nonReturnFlowNone, false
	}
	if !tryHasCatch(statement) {
		return nonReturnFlowNone, false
	}
	if statementsContainReturn(statement.CatchBody) || statementsContainReturn(statement.FinallyBody) {
		return nonReturnFlowNone, false
	}
	catchFlow := applyCatchBodySideEffects(statement, next)
	finallyFlow := applyNonReturnStatements(statement.FinallyBody, next)
	replaceReturnScopeBindings(scope, next)
	if finallyFlow != nonReturnFlowNone {
		return finallyFlow, true
	}
	return catchFlow, true
}

func tryHasCatch(statement *ast.TryStatement) bool {
	return len(statement.CatchBody) > 0 || statement.CatchName != "" || statement.CatchPattern != nil
}
