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

func applyVariableDecl(statement *ast.VariableDecl, scope returnScope) {
	if statement.Name == "" || statement.Value == nil {
		return
	}
	if applyReferenceVariableDecl(statement.Name, statement.Value, scope) {
		return
	}
	if applyScalarVariableDecl(statement.Name, statement.Value, scope) {
		return
	}
	evaluateDiscardExpression(statement.Value, scope)
}

func applyScalarVariableDecl(name string, expression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if value, ok := evaluateIntExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.ints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateStringExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.strings[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBigIntValue(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bigints[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateNullishExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.nullish[name] = value
		return true
	}
	next = scope.clone()
	if value, ok := evaluateBoolExpression(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		clearReturnScopeBinding(scope, name)
		scope.bools[name] = value
		return true
	}
	return false
}

func applyReferenceVariableDecl(name string, expression ast.Expression, scope returnScope) bool {
	next := scope.clone()
	if identity, ok := materializeFunctionIdentity(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignFunctionIdentity(scope, name, identity)
		return true
	}
	next = scope.clone()
	if identity, ok := materializeObjectIdentity(expression, next); ok {
		replaceReturnScopeBindings(scope, next)
		assignObjectIdentity(scope, name, identity)
		return true
	}
	return false
}
