package lowering

import "jayess-go/ast"

func returnCodeFromLabeled(statement *ast.LabeledStatement, scope returnScope) (int, bool, bool) {
	if statement.Statement == nil {
		return 0, false, true
	}
	if value, ok := returnCodeFromStatements([]ast.Statement{statement.Statement}, scope); ok {
		return value, true, true
	}
	if statementContainsReturn(statement.Statement) {
		return 0, false, false
	}
	if applyLabeledBreakStatement(statement.Statement, statement.Label, scope) {
		return 0, false, true
	}
	if applyNonReturnLabeled(statement, scope) == nonReturnFlowNone {
		return 0, false, true
	}
	return 0, false, false
}

func applyNonReturnLabeled(statement *ast.LabeledStatement, scope returnScope) nonReturnFlow {
	if statement.Statement == nil {
		return nonReturnFlowNone
	}
	flow := applyNonReturnStatement(statement.Statement, scope)
	if flow.label == statement.Label {
		return nonReturnFlowNone
	}
	return flow
}

func applyLabeledBreakStatements(statements []ast.Statement, label string, scope returnScope) bool {
	for _, statement := range statements {
		if applyLabeledBreakStatement(statement, label, scope) {
			return true
		}
		if flow := applyNonReturnStatement(statement, scope); flow != nonReturnFlowNone {
			return false
		}
	}
	return false
}

func applyLabeledBreakStatement(statement ast.Statement, label string, scope returnScope) bool {
	switch stmt := statement.(type) {
	case *ast.BreakStatement:
		return stmt.Label == label
	case *ast.BlockStatement:
		return applyLabeledBreakScopedStatements(stmt.Statements, label, scope)
	case *ast.IfStatement:
		return applyLabeledBreakIf(stmt, label, scope)
	case *ast.SwitchStatement:
		return applyLabeledBreakSwitch(stmt, label, scope)
	case *ast.WhileStatement:
		return applyLabeledBreakWhile(stmt, label, scope)
	case *ast.ForStatement:
		return applyLabeledBreakFor(stmt, label, scope)
	case *ast.DoWhileStatement:
		return applyLabeledBreakDoWhile(stmt, label, scope)
	case *ast.LabeledStatement:
		if stmt.Label == label {
			return false
		}
		return applyLabeledBreakNestedStatement(stmt.Statement, label, scope)
	default:
		return false
	}
}

func applyLabeledBreakIf(statement *ast.IfStatement, label string, scope returnScope) bool {
	condition, ok := evaluateBoolExpression(statement.Condition, scope)
	if !ok {
		return false
	}
	if condition {
		return applyLabeledBreakScopedStatements(statement.Consequence, label, scope)
	}
	return applyLabeledBreakScopedStatements(statement.Alternative, label, scope)
}

func applyLabeledBreakSwitch(statement *ast.SwitchStatement, label string, scope returnScope) bool {
	consequent, ok := matchingSwitchConsequent(statement, scope)
	if !ok {
		return false
	}
	return applyLabeledBreakScopedStatements(consequent, label, scope)
}

func applyLabeledBreakNestedStatement(statement ast.Statement, label string, scope returnScope) bool {
	next := scope.clone()
	if !applyLabeledBreakStatement(statement, label, next) {
		return false
	}
	replaceReturnScopeBindings(scope, next)
	return true
}

func applyLabeledBreakScopedStatements(statements []ast.Statement, label string, scope returnScope) bool {
	next := scope.clone()
	if !applyLabeledBreakStatements(statements, label, next) {
		return false
	}
	replaceReturnScopeBindings(scope, next)
	return true
}
