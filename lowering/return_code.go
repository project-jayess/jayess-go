package lowering

import "jayess-go/ast"

func MainReturnCode(program *ast.Program) (int, bool) {
	if program == nil {
		return 0, false
	}
	for _, statement := range program.Statements {
		function, ok := statement.(*ast.FunctionDecl)
		if !ok || function.Name != "main" {
			continue
		}
		return returnCodeFromStatements(function.Body, returnScope{})
	}
	return returnCodeFromStatements(program.Statements, returnScope{})
}

func returnCodeFromStatements(statements []ast.Statement, bindings returnScope) (int, bool) {
	scope := bindings.clone()
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			applyVariableDecl(stmt, scope)
		case *ast.AssignmentStatement:
			applyAssignmentStatement(stmt, scope)
		case *ast.ExpressionStatement:
			applyExpressionStatement(stmt, scope)
		case *ast.ReturnStatement:
			if stmt.Value == nil {
				return 0, true
			}
			return evaluateIntExpression(stmt.Value, scope)
		case *ast.ThrowStatement:
			return 0, false
		case *ast.BlockStatement:
			if value, ok := returnCodeFromStatements(stmt.Statements, scope); ok {
				return value, true
			}
			if statementsContainReturn(stmt.Statements) {
				return 0, false
			}
			applyNonReturnStatements(stmt.Statements, scope)
		case *ast.LabeledStatement:
			if value, ok, handled := returnCodeFromLabeled(stmt, scope); ok {
				return value, true
			} else if handled {
				continue
			} else if statementContainsReturn(stmt) {
				return 0, false
			}
		case *ast.IfStatement:
			if value, ok, handled := returnCodeFromIf(stmt, scope); ok {
				return value, true
			} else if handled {
				continue
			} else if ifBlocksFollowing(stmt) {
				return 0, false
			}
		case *ast.WhileStatement:
			if value, ok, handled := returnCodeFromWhile(stmt, scope); ok {
				return value, true
			} else if handled {
				continue
			} else if whileBlocksFollowing(stmt, scope) || statementsContainReturn(stmt.Body) {
				return 0, false
			}
		case *ast.ForStatement:
			if value, ok, handled := returnCodeFromFor(stmt, scope); ok {
				return value, true
			} else if handled {
				continue
			} else if forBlocksFollowing(stmt, scope) || statementsContainReturn(stmt.Body) {
				return 0, false
			}
		case *ast.DoWhileStatement:
			if value, ok, handled := returnCodeFromDoWhile(stmt, scope); ok {
				return value, true
			} else if handled {
				continue
			} else if doWhileBlocksFollowing(stmt, scope) || statementsContainReturn(stmt.Body) {
				return 0, false
			}
		case *ast.SwitchStatement:
			if value, ok, handled := returnCodeFromSwitch(stmt, scope); ok {
				return value, true
			} else if handled {
				continue
			} else if switchBlocksFollowing(stmt) {
				return 0, false
			}
		case *ast.TryStatement:
			if flow, handled := applyTrySideEffects(stmt, scope); handled {
				if flow != nonReturnFlowNone {
					return 0, false
				}
				continue
			} else if statementContainsReturn(stmt) {
				return 0, false
			}
		}
	}
	return 0, false
}
