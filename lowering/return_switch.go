package lowering

import "jayess-go/ast"

func returnCodeFromSwitch(statement *ast.SwitchStatement, scope returnScope) (int, bool, bool) {
	consequent, ok := matchingSwitchConsequent(statement, scope)
	if !ok {
		return 0, false, false
	}
	if value, returned := returnCodeFromSwitchStatements(consequent, scope); returned {
		return value, true, true
	}
	applyNonReturnStatements(consequent, scope)
	return 0, false, true
}

func applyNonReturnSwitch(statement *ast.SwitchStatement, scope returnScope) nonReturnFlow {
	consequent, ok := matchingSwitchConsequent(statement, scope)
	if !ok {
		return nonReturnFlowNone
	}
	flow := applyNonReturnStatements(consequent, scope)
	if flow == nonReturnFlowBreak {
		return nonReturnFlowNone
	}
	return flow
}

func returnCodeFromSwitchStatements(statements []ast.Statement, scope returnScope) (int, bool) {
	switchScope := scope.clone()
	for _, statement := range statements {
		if _, ok := statement.(*ast.BreakStatement); ok {
			return 0, false
		}
		if _, ok := statement.(*ast.ContinueStatement); ok {
			return 0, false
		}
		if value, ok := returnCodeFromStatements([]ast.Statement{statement}, switchScope); ok {
			return value, true
		}
		if flow := applyNonReturnStatement(statement, switchScope); flow != nonReturnFlowNone {
			return 0, false
		}
	}
	return 0, false
}

func matchingSwitchConsequent(statement *ast.SwitchStatement, scope returnScope) ([]ast.Statement, bool) {
	if value, ok := evaluateIntExpression(statement.Discriminant, scope); ok {
		for index, clause := range statement.Cases {
			caseValue, ok := evaluateIntExpression(clause.Test, scope)
			if ok && caseValue == value {
				return switchConsequentFromCase(statement, index), true
			}
		}
		return statement.Default, len(statement.Default) > 0
	}
	if value, ok := evaluateBigIntValue(statement.Discriminant, scope); ok {
		for index, clause := range statement.Cases {
			caseValue, ok := evaluateBigIntValue(clause.Test, scope)
			if ok && caseValue == value {
				return switchConsequentFromCase(statement, index), true
			}
		}
		return statement.Default, len(statement.Default) > 0
	}
	if value, ok := evaluateNullishExpression(statement.Discriminant, scope); ok {
		for index, clause := range statement.Cases {
			caseValue, ok := evaluateNullishExpression(clause.Test, scope)
			if ok && caseValue == value {
				return switchConsequentFromCase(statement, index), true
			}
		}
		return statement.Default, len(statement.Default) > 0
	}
	if value, ok := evaluateStringExpression(statement.Discriminant, scope); ok {
		for index, clause := range statement.Cases {
			caseValue, ok := evaluateStringExpression(clause.Test, scope)
			if ok && caseValue == value {
				return switchConsequentFromCase(statement, index), true
			}
		}
		return statement.Default, len(statement.Default) > 0
	}
	if value, ok := evaluateBoolExpression(statement.Discriminant, scope); ok {
		for index, clause := range statement.Cases {
			caseValue, ok := evaluateBoolExpression(clause.Test, scope)
			if ok && caseValue == value {
				return switchConsequentFromCase(statement, index), true
			}
		}
		return statement.Default, len(statement.Default) > 0
	}
	return nil, false
}

func switchConsequentFromCase(statement *ast.SwitchStatement, index int) []ast.Statement {
	statements := []ast.Statement{}
	for ; index < len(statement.Cases); index++ {
		statements = append(statements, statement.Cases[index].Consequent...)
	}
	statements = append(statements, statement.Default...)
	return statements
}

func switchBlocksFollowing(statement *ast.SwitchStatement) bool {
	if statementsContainReturn(statement.Default) {
		return true
	}
	for _, clause := range statement.Cases {
		if statementsContainReturn(clause.Consequent) {
			return true
		}
	}
	return false
}
