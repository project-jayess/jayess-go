package lifetime

import "jayess-go/ast"

func collectClosureEnvironments(statements []ast.Statement, plan *Plan) {
	scope := map[string]bool{}
	slots := map[string]int{}
	declareStatementNames(scope, statements)
	collectStatementClosureEnvironments(scope, statements, plan, slots)
}

func collectStatementClosureEnvironments(scope map[string]bool, statements []ast.Statement, plan *Plan, slots map[string]int) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			collectExpressionClosureEnvironments(scope, stmt.Value, plan, slots)
		case *ast.FunctionDecl:
			nested := copyNames(scope)
			declareParameters(nested, stmt.Params)
			declareStatementNames(nested, stmt.Body)
			collectStatementClosureEnvironments(nested, stmt.Body, plan, slots)
		case *ast.BlockStatement:
			nested := copyNames(scope)
			declareStatementNames(nested, stmt.Statements)
			collectStatementClosureEnvironments(nested, stmt.Statements, plan, slots)
		case *ast.ExpressionStatement:
			collectExpressionClosureEnvironments(scope, stmt.Expression, plan, slots)
		case *ast.AssignmentStatement:
			collectExpressionClosureEnvironments(scope, stmt.Target, plan, slots)
			collectExpressionClosureEnvironments(scope, stmt.Value, plan, slots)
		case *ast.ReturnStatement:
			collectExpressionClosureEnvironments(scope, stmt.Value, plan, slots)
		case *ast.IfStatement:
			collectExpressionClosureEnvironments(scope, stmt.Condition, plan, slots)
			collectStatementClosureEnvironments(copyNames(scope), stmt.Consequence, plan, slots)
			collectStatementClosureEnvironments(copyNames(scope), stmt.Alternative, plan, slots)
		case *ast.WhileStatement:
			collectExpressionClosureEnvironments(scope, stmt.Condition, plan, slots)
			collectStatementClosureEnvironments(copyNames(scope), stmt.Body, plan, slots)
		case *ast.ForStatement:
			collectStatementClosureEnvironments(copyNames(scope), []ast.Statement{stmt.Init, stmt.Update}, plan, slots)
			collectExpressionClosureEnvironments(scope, stmt.Condition, plan, slots)
			collectStatementClosureEnvironments(copyNames(scope), stmt.Body, plan, slots)
		}
	}
}

func collectExpressionClosureEnvironments(scope map[string]bool, expr ast.Expression, plan *Plan, slots map[string]int) {
	switch expr := expr.(type) {
	case *ast.FunctionExpression:
		pos := ast.PositionOf(expr)
		captured := capturedNames(scope, expr)
		mutated := mutatedCapturedNames(scope, expr)
		if len(captured) > 0 {
			plan.ClosureEnvironments = append(plan.ClosureEnvironments, ClosureEnvironment{
				Line:       pos.Line,
				Column:     pos.Column,
				Allocation: "heap",
				Captures:   closureCaptures(captured, mutated, plan, slots),
			})
		}
	case *ast.BinaryExpression:
		collectExpressionClosureEnvironments(scope, expr.Left, plan, slots)
		collectExpressionClosureEnvironments(scope, expr.Right, plan, slots)
	case *ast.LogicalExpression:
		collectExpressionClosureEnvironments(scope, expr.Left, plan, slots)
		collectExpressionClosureEnvironments(scope, expr.Right, plan, slots)
	case *ast.ConditionalExpression:
		collectExpressionClosureEnvironments(scope, expr.Condition, plan, slots)
		collectExpressionClosureEnvironments(scope, expr.Consequent, plan, slots)
		collectExpressionClosureEnvironments(scope, expr.Alternative, plan, slots)
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			collectExpressionClosureEnvironments(scope, element, plan, slots)
		}
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			collectExpressionClosureEnvironments(scope, property.Value, plan, slots)
		}
	case *ast.CallExpression:
		for _, argument := range expr.Arguments {
			collectExpressionClosureEnvironments(scope, argument, plan, slots)
		}
	case *ast.InvokeExpression:
		collectExpressionClosureEnvironments(scope, expr.Callee, plan, slots)
		for _, argument := range expr.Arguments {
			collectExpressionClosureEnvironments(scope, argument, plan, slots)
		}
	}
}

func capturedNames(scope map[string]bool, fn *ast.FunctionExpression) []string {
	locals := map[string]bool{}
	if fn.Name != "" {
		locals[fn.Name] = true
	}
	declareParameters(locals, fn.Params)
	declareStatementNames(locals, fn.Body)
	seen := map[string]bool{}
	collectCapturedExpressionNames(scope, locals, fn.ExpressionBody, seen)
	collectCapturedStatementNames(scope, locals, fn.Body, seen)
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	return names
}
