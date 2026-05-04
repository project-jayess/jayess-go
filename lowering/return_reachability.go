package lowering

import "jayess-go/ast"

func statementsContainReturn(statements []ast.Statement) bool {
	for _, statement := range statements {
		if statementContainsReturn(statement) {
			return true
		}
	}
	return false
}

func statementContainsReturn(statement ast.Statement) bool {
	switch stmt := statement.(type) {
	case *ast.ReturnStatement:
		return true
	case *ast.ThrowStatement:
		return true
	case *ast.BlockStatement:
		return statementsContainReturn(stmt.Statements)
	case *ast.LabeledStatement:
		return stmt.Statement != nil && statementContainsReturn(stmt.Statement)
	case *ast.IfStatement:
		return statementsContainReturn(stmt.Consequence) || statementsContainReturn(stmt.Alternative)
	case *ast.SwitchStatement:
		if statementsContainReturn(stmt.Default) {
			return true
		}
		for _, clause := range stmt.Cases {
			if statementsContainReturn(clause.Consequent) {
				return true
			}
		}
	case *ast.WhileStatement:
		return statementsContainReturn(stmt.Body)
	case *ast.ForStatement:
		return statementsContainReturn(stmt.Body)
	case *ast.DoWhileStatement:
		return statementsContainReturn(stmt.Body)
	case *ast.TryStatement:
		return statementsContainReturn(stmt.TryBody) ||
			statementsContainReturn(stmt.CatchBody) ||
			statementsContainReturn(stmt.FinallyBody)
	}
	return false
}
