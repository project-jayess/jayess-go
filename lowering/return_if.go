package lowering

import "jayess-go/ast"

func returnCodeFromIf(statement *ast.IfStatement, scope returnScope) (int, bool, bool) {
	condition, ok := evaluateBoolExpression(statement.Condition, scope)
	if !ok {
		returnCode, returns := returnCodeFromStatements(statement.Consequence, scope)
		if returns {
			return returnCode, true, true
		}
		returnCode, returns = returnCodeFromStatements(statement.Alternative, scope)
		return returnCode, returns, returns
	}
	branch := statement.Alternative
	if condition {
		branch = statement.Consequence
	}
	if returnCode, returns := returnCodeFromStatements(branch, scope); returns {
		return returnCode, true, true
	}
	applyNonReturnStatements(branch, scope)
	return 0, false, true
}

func applyNonReturnIf(statement *ast.IfStatement, scope returnScope) nonReturnFlow {
	condition, ok := evaluateBoolExpression(statement.Condition, scope)
	if !ok {
		return nonReturnFlowNone
	}
	if condition {
		return applyNonReturnStatements(statement.Consequence, scope)
	}
	return applyNonReturnStatements(statement.Alternative, scope)
}

func ifBlocksFollowing(statement *ast.IfStatement) bool {
	return statementsContainReturn(statement.Consequence) || statementsContainReturn(statement.Alternative)
}
