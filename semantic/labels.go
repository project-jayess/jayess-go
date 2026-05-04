package semantic

import "jayess-go/ast"

func isLoopStatement(statement any) bool {
	switch statement.(type) {
	case *ast.WhileStatement, *ast.DoWhileStatement, *ast.ForStatement, *ast.ForOfStatement, *ast.ForInStatement:
		return true
	default:
		return false
	}
}
