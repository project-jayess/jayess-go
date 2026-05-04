package lowering

import "jayess-go/ast"

func applyLabeledBreakWhile(statement *ast.WhileStatement, label string, scope returnScope) bool {
	condition, ok := evaluateBoolExpression(statement.Condition, scope)
	if !ok || !condition {
		return false
	}
	return applyLabeledBreakLoopBody(statement.Body, label, scope)
}

func applyLabeledBreakFor(statement *ast.ForStatement, label string, scope returnScope) bool {
	next := scope.clone()
	applyNonReturnStatement(statement.Init, next)
	if statement.Condition != nil {
		condition, ok := evaluateBoolExpression(statement.Condition, next)
		if !ok || !condition {
			return false
		}
	}
	if !applyLabeledBreakStatements(statement.Body, label, next) {
		return false
	}
	replaceReturnScopeBindings(scope, next)
	return true
}

func applyLabeledBreakDoWhile(statement *ast.DoWhileStatement, label string, scope returnScope) bool {
	return applyLabeledBreakLoopBody(statement.Body, label, scope)
}

func applyLabeledBreakLoopBody(statements []ast.Statement, label string, scope returnScope) bool {
	next := scope.clone()
	if !applyLabeledBreakStatements(statements, label, next) {
		return false
	}
	replaceReturnScopeBindings(scope, next)
	return true
}
