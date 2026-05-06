package lowering

import "jayess-go/ast"

func applyNonReturnStatements(statements []ast.Statement, scope returnScope) nonReturnFlow {
	for _, statement := range statements {
		if stmt, ok := statement.(*ast.BreakStatement); ok {
			return nonReturnBreakFlow(stmt.Label)
		}
		if stmt, ok := statement.(*ast.ContinueStatement); ok {
			return nonReturnContinueFlow(stmt.Label)
		}
		if flow := applyNonReturnStatement(statement, scope); flow != nonReturnFlowNone {
			return flow
		}
	}
	return nonReturnFlowNone
}

func applyNonReturnStatement(statement ast.Statement, scope returnScope) nonReturnFlow {
	switch stmt := statement.(type) {
	case *ast.VariableDecl:
		applyVariableDecl(stmt, scope)
	case *ast.AssignmentStatement:
		applyAssignmentStatement(stmt, scope)
	case *ast.ExpressionStatement:
		applyExpressionStatement(stmt, scope)
	case *ast.BlockStatement:
		return applyNonReturnStatements(stmt.Statements, scope)
	case *ast.IfStatement:
		return applyNonReturnIf(stmt, scope)
	case *ast.SwitchStatement:
		return applyNonReturnSwitch(stmt, scope)
	case *ast.WhileStatement:
		return applyNonReturnWhile(stmt, scope)
	case *ast.ForStatement:
		return applyNonReturnFor(stmt, scope)
	case *ast.DoWhileStatement:
		return applyNonReturnDoWhile(stmt, scope)
	case *ast.TryStatement:
		return applyNonReturnTry(stmt, scope)
	case *ast.LabeledStatement:
		return applyNonReturnLabeled(stmt, scope)
	case *ast.BreakStatement:
		return nonReturnBreakFlow(stmt.Label)
	case *ast.ContinueStatement:
		return nonReturnContinueFlow(stmt.Label)
	}
	return nonReturnFlowNone
}
