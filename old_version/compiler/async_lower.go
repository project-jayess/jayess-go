package compiler

import "jayess-go/ast"

type asyncLowerer struct{}

func lowerAsyncFunctions(program *ast.Program) (*ast.Program, error) {
	l := &asyncLowerer{}
	for _, fn := range program.Functions {
		if fn.IsAsync {
			fn.Body = l.lowerStatements(fn.Body)
			fn.IsAsync = false
		}
	}
	return program, nil
}

func lowerAsyncFunctionExpressions(program *ast.Program) (*ast.Program, error) {
	l := &asyncLowerer{}
	for _, fn := range program.Functions {
		fn.Body = l.lowerFunctionExpressionsInStatements(fn.Body)
	}
	return program, nil
}

func (l *asyncLowerer) lowerStatements(statements []ast.Statement) []ast.Statement {
	out := make([]ast.Statement, 0, len(statements))
	for _, stmt := range statements {
		out = append(out, l.lowerStatement(stmt))
	}
	return out
}

func (l *asyncLowerer) lowerStatement(stmt ast.Statement) ast.Statement {
	switch stmt := stmt.(type) {
	case *ast.ReturnStatement:
		return &ast.ReturnStatement{BaseNode: stmt.BaseNode, Value: promiseResolveCall(stmt.Value)}
	case *ast.ThrowStatement:
		return &ast.ReturnStatement{BaseNode: stmt.BaseNode, Value: promiseRejectCall(stmt.Value)}
	case *ast.IfStatement:
		return &ast.IfStatement{
			BaseNode:    stmt.BaseNode,
			Condition:   stmt.Condition,
			Consequence: l.lowerStatements(stmt.Consequence),
			Alternative: l.lowerStatements(stmt.Alternative),
		}
	case *ast.WhileStatement:
		return &ast.WhileStatement{BaseNode: stmt.BaseNode, Condition: stmt.Condition, Body: l.lowerStatements(stmt.Body)}
	case *ast.ForStatement:
		return &ast.ForStatement{BaseNode: stmt.BaseNode, Init: stmt.Init, Condition: stmt.Condition, Update: stmt.Update, Body: l.lowerStatements(stmt.Body)}
	case *ast.ForOfStatement:
		return &ast.ForOfStatement{BaseNode: stmt.BaseNode, Kind: stmt.Kind, Name: stmt.Name, Iterable: stmt.Iterable, Body: l.lowerStatements(stmt.Body)}
	case *ast.ForInStatement:
		return &ast.ForInStatement{BaseNode: stmt.BaseNode, Kind: stmt.Kind, Name: stmt.Name, Iterable: stmt.Iterable, Body: l.lowerStatements(stmt.Body)}
	case *ast.TryStatement:
		return &ast.TryStatement{
			BaseNode:    stmt.BaseNode,
			TryBody:     l.lowerStatements(stmt.TryBody),
			CatchName:   stmt.CatchName,
			CatchBody:   l.lowerStatements(stmt.CatchBody),
			FinallyBody: l.lowerStatements(stmt.FinallyBody),
		}
	default:
		return stmt
	}
}

func promiseResolveCall(value ast.Expression) ast.Expression {
	if value == nil {
		value = &ast.UndefinedLiteral{}
	}
	return &ast.CallExpression{Callee: "__jayess_std_promise_resolve", Arguments: []ast.Expression{value}}
}

func promiseRejectCall(value ast.Expression) ast.Expression {
	if value == nil {
		value = &ast.UndefinedLiteral{}
	}
	return &ast.CallExpression{Callee: "__jayess_std_promise_reject", Arguments: []ast.Expression{value}}
}

func (l *asyncLowerer) lowerFunctionExpressionsInStatements(statements []ast.Statement) []ast.Statement {
	out := make([]ast.Statement, 0, len(statements))
	for _, stmt := range statements {
		out = append(out, l.lowerFunctionExpressionsInStatement(stmt))
	}
	return out
}

func (l *asyncLowerer) lowerFunctionExpressionsInStatement(stmt ast.Statement) ast.Statement {
	switch stmt := stmt.(type) {
	case *ast.VariableDecl:
		return &ast.VariableDecl{BaseNode: stmt.BaseNode, Visibility: stmt.Visibility, Kind: stmt.Kind, Name: stmt.Name, TypeAnnotation: stmt.TypeAnnotation, Value: l.lowerFunctionExpressionsInExpression(stmt.Value)}
	case *ast.AssignmentStatement:
		return &ast.AssignmentStatement{BaseNode: stmt.BaseNode, Target: l.lowerFunctionExpressionsInExpression(stmt.Target), Operator: stmt.Operator, Value: l.lowerFunctionExpressionsInExpression(stmt.Value)}
	case *ast.ReturnStatement:
		return &ast.ReturnStatement{BaseNode: stmt.BaseNode, Value: l.lowerFunctionExpressionsInExpression(stmt.Value)}
	case *ast.ExpressionStatement:
		return &ast.ExpressionStatement{BaseNode: stmt.BaseNode, Expression: l.lowerFunctionExpressionsInExpression(stmt.Expression)}
	case *ast.ThrowStatement:
		return &ast.ThrowStatement{BaseNode: stmt.BaseNode, Value: l.lowerFunctionExpressionsInExpression(stmt.Value)}
	case *ast.IfStatement:
		return &ast.IfStatement{BaseNode: stmt.BaseNode, Condition: l.lowerFunctionExpressionsInExpression(stmt.Condition), Consequence: l.lowerFunctionExpressionsInStatements(stmt.Consequence), Alternative: l.lowerFunctionExpressionsInStatements(stmt.Alternative)}
	case *ast.WhileStatement:
		return &ast.WhileStatement{BaseNode: stmt.BaseNode, Condition: l.lowerFunctionExpressionsInExpression(stmt.Condition), Body: l.lowerFunctionExpressionsInStatements(stmt.Body)}
	case *ast.ForStatement:
		return &ast.ForStatement{BaseNode: stmt.BaseNode, Init: stmt.Init, Condition: stmt.Condition, Update: stmt.Update, Body: l.lowerFunctionExpressionsInStatements(stmt.Body)}
	case *ast.TryStatement:
		return &ast.TryStatement{BaseNode: stmt.BaseNode, TryBody: l.lowerFunctionExpressionsInStatements(stmt.TryBody), CatchName: stmt.CatchName, CatchBody: l.lowerFunctionExpressionsInStatements(stmt.CatchBody), FinallyBody: l.lowerFunctionExpressionsInStatements(stmt.FinallyBody)}
	default:
		return stmt
	}
}

func (l *asyncLowerer) lowerFunctionExpressionsInExpression(expr ast.Expression) ast.Expression {
	switch expr := expr.(type) {
	case *ast.FunctionExpression:
		if expr.IsAsync {
			expr.Body = l.lowerStatements(expr.Body)
			if expr.ExpressionBody != nil {
				expr.Body = []ast.Statement{&ast.ReturnStatement{Value: promiseResolveCall(l.lowerFunctionExpressionsInExpression(expr.ExpressionBody))}}
				expr.ExpressionBody = nil
			}
			expr.IsAsync = false
		}
		return expr
	case *ast.InvokeExpression:
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			args = append(args, l.lowerFunctionExpressionsInExpression(arg))
		}
		return &ast.InvokeExpression{BaseNode: expr.BaseNode, Callee: l.lowerFunctionExpressionsInExpression(expr.Callee), Arguments: args, Optional: expr.Optional}
	case *ast.CallExpression:
		args := make([]ast.Expression, 0, len(expr.Arguments))
		for _, arg := range expr.Arguments {
			args = append(args, l.lowerFunctionExpressionsInExpression(arg))
		}
		return &ast.CallExpression{BaseNode: expr.BaseNode, Callee: expr.Callee, Arguments: args}
	case *ast.BinaryExpression:
		return &ast.BinaryExpression{BaseNode: expr.BaseNode, Left: l.lowerFunctionExpressionsInExpression(expr.Left), Operator: expr.Operator, Right: l.lowerFunctionExpressionsInExpression(expr.Right)}
	case *ast.CastExpression:
		return &ast.CastExpression{BaseNode: expr.BaseNode, Value: l.lowerFunctionExpressionsInExpression(expr.Value), TypeAnnotation: expr.TypeAnnotation}
	case *ast.AwaitExpression:
		return &ast.AwaitExpression{BaseNode: expr.BaseNode, Value: l.lowerFunctionExpressionsInExpression(expr.Value)}
	default:
		return expr
	}
}
