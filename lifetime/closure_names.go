package lifetime

import "jayess-go/ast"

func declareStatementNames(names map[string]bool, statements []ast.Statement) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			for _, name := range bindingNames(stmt.Pattern) {
				names[name] = true
			}
		case *ast.FunctionDecl:
			names[stmt.Name] = true
		}
	}
}

func declareParameters(names map[string]bool, params []ast.Parameter) {
	for _, param := range params {
		for _, name := range bindingNames(param.Pattern) {
			names[name] = true
		}
	}
}

func copyNames(names map[string]bool) map[string]bool {
	copied := map[string]bool{}
	for name := range names {
		copied[name] = true
	}
	return copied
}

func collectCapturedStatementNames(scope map[string]bool, locals map[string]bool, statements []ast.Statement, captured map[string]bool) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.VariableDecl:
			collectCapturedExpressionNames(scope, locals, stmt.Value, captured)
		case *ast.ExpressionStatement:
			collectCapturedExpressionNames(scope, locals, stmt.Expression, captured)
		case *ast.AssignmentStatement:
			collectCapturedExpressionNames(scope, locals, stmt.Target, captured)
			collectCapturedExpressionNames(scope, locals, stmt.Value, captured)
		case *ast.ReturnStatement:
			collectCapturedExpressionNames(scope, locals, stmt.Value, captured)
		case *ast.BlockStatement:
			collectCapturedStatementNames(scope, locals, stmt.Statements, captured)
		}
	}
}

func collectCapturedExpressionNames(scope map[string]bool, locals map[string]bool, expr ast.Expression, captured map[string]bool) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		if scope[expr.Name] && !locals[expr.Name] {
			captured[expr.Name] = true
		}
	case *ast.BinaryExpression:
		collectCapturedExpressionNames(scope, locals, expr.Left, captured)
		collectCapturedExpressionNames(scope, locals, expr.Right, captured)
	case *ast.LogicalExpression:
		collectCapturedExpressionNames(scope, locals, expr.Left, captured)
		collectCapturedExpressionNames(scope, locals, expr.Right, captured)
	case *ast.ConditionalExpression:
		collectCapturedExpressionNames(scope, locals, expr.Condition, captured)
		collectCapturedExpressionNames(scope, locals, expr.Consequent, captured)
		collectCapturedExpressionNames(scope, locals, expr.Alternative, captured)
	case *ast.ArrayLiteral:
		for _, element := range expr.Elements {
			collectCapturedExpressionNames(scope, locals, element, captured)
		}
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			collectCapturedExpressionNames(scope, locals, property.Value, captured)
		}
	case *ast.FunctionExpression:
		return
	}
}

func collectMutatedCapturedStatementNames(scope map[string]bool, locals map[string]bool, statements []ast.Statement, mutated map[string]bool) {
	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.AssignmentStatement:
			markMutatedCapturedExpressionName(scope, locals, stmt.Target, mutated)
		case *ast.ExpressionStatement:
			markMutatedCapturedExpressionName(scope, locals, stmt.Expression, mutated)
		case *ast.BlockStatement:
			collectMutatedCapturedStatementNames(scope, locals, stmt.Statements, mutated)
		}
	}
}

func markMutatedCapturedExpressionName(scope map[string]bool, locals map[string]bool, expr ast.Expression, mutated map[string]bool) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		if scope[expr.Name] && !locals[expr.Name] {
			mutated[expr.Name] = true
		}
	case *ast.UpdateExpression:
		markMutatedCapturedExpressionName(scope, locals, expr.Target, mutated)
	}
}
