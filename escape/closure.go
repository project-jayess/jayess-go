package escape

import "jayess-go/ast"

func markFunctionCapturesEscaping(report *Report, fn *ast.FunctionExpression) {
	locals := map[string]bool{}
	if fn.Name != "" {
		locals[fn.Name] = true
	}
	declareParameters(locals, fn.Params)
	declareStatementLocals(locals, fn.Body)
	markCapturedExpressionIdentifiersEscaping(report, locals, fn.ExpressionBody)
	markCapturedStatementIdentifiersEscaping(report, locals, fn.Body)
}

func declareStatementLocals(locals map[string]bool, statements []ast.Statement) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			declarePattern(locals, stmt.Pattern)
		case *ast.FunctionDecl:
			locals[stmt.Name] = true
		case *ast.BlockStatement:
			declareStatementLocals(locals, stmt.Statements)
		case *ast.IfStatement:
			declareStatementLocals(locals, stmt.Consequence)
			declareStatementLocals(locals, stmt.Alternative)
		case *ast.ForStatement:
			declareStatementLocal(locals, stmt.Init)
			declareStatementLocals(locals, stmt.Body)
		case *ast.ForOfStatement:
			declarePattern(locals, stmt.Pattern)
			declareStatementLocals(locals, stmt.Body)
		case *ast.ForInStatement:
			declarePattern(locals, stmt.Pattern)
			declareStatementLocals(locals, stmt.Body)
		case *ast.SwitchStatement:
			for _, switchCase := range stmt.Cases {
				declareStatementLocals(locals, switchCase.Consequent)
			}
			declareStatementLocals(locals, stmt.Default)
		case *ast.TryStatement:
			declareStatementLocals(locals, stmt.TryBody)
			declarePattern(locals, stmt.CatchPattern)
			declareStatementLocals(locals, stmt.CatchBody)
			declareStatementLocals(locals, stmt.FinallyBody)
		}
	}
}

func declareStatementLocal(locals map[string]bool, statement ast.Statement) {
	if statement == nil {
		return
	}
	declareStatementLocals(locals, []ast.Statement{statement})
}

func markCapturedStatementIdentifiersEscaping(report *Report, locals map[string]bool, statements []ast.Statement) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Value)
		case *ast.BlockStatement:
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Statements)
		case *ast.ExpressionStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Expression)
		case *ast.AssignmentStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Target)
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Value)
		case *ast.ReturnStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Value)
		case *ast.IfStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Condition)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Consequence)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Alternative)
		case *ast.WhileStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Condition)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Body)
		case *ast.DoWhileStatement:
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Body)
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Condition)
		case *ast.ForStatement:
			markCapturedStatementIdentifiersEscaping(report, locals, []ast.Statement{stmt.Init, stmt.Update})
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Condition)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Body)
		case *ast.ForOfStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Target)
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Iterable)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Body)
		case *ast.ForInStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Target)
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Object)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Body)
		case *ast.LabeledStatement:
			markCapturedStatementIdentifiersEscaping(report, locals, []ast.Statement{stmt.Statement})
		case *ast.SwitchStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Discriminant)
			for _, switchCase := range stmt.Cases {
				markCapturedExpressionIdentifiersEscaping(report, locals, switchCase.Test)
				markCapturedStatementIdentifiersEscaping(report, locals, switchCase.Consequent)
			}
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.Default)
		case *ast.ThrowStatement:
			markCapturedExpressionIdentifiersEscaping(report, locals, stmt.Value)
		case *ast.TryStatement:
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.TryBody)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.CatchBody)
			markCapturedStatementIdentifiersEscaping(report, locals, stmt.FinallyBody)
		}
	}
}

func markCapturedExpressionIdentifiersEscaping(report *Report, locals map[string]bool, expr ast.Expression) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		if !locals[expr.Name] {
			report.markEscaping(expr.Name)
		}
	case *ast.FunctionExpression:
		return
	default:
		markExpressionIdentifiersEscaping(report, expr)
	}
}
