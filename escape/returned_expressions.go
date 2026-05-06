package escape

import "jayess-go/ast"

func markReturnedExpressionEscaping(report *Report, expr ast.Expression) {
	markExpressionIdentifiersEscaping(report, expr)
}

func markExpressionIdentifiersEscaping(report *Report, expr ast.Expression) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		report.markEscaping(expr.Name)
	case *ast.BinaryExpression:
		markExpressionIdentifiersEscaping(report, expr.Left)
		markExpressionIdentifiersEscaping(report, expr.Right)
	case *ast.ComparisonExpression:
		markExpressionIdentifiersEscaping(report, expr.Left)
		markExpressionIdentifiersEscaping(report, expr.Right)
	case *ast.LogicalExpression:
		markExpressionIdentifiersEscaping(report, expr.Left)
		markExpressionIdentifiersEscaping(report, expr.Right)
	case *ast.NullishCoalesceExpression:
		markExpressionIdentifiersEscaping(report, expr.Left)
		markExpressionIdentifiersEscaping(report, expr.Right)
	case *ast.ConditionalExpression:
		markExpressionIdentifiersEscaping(report, expr.Condition)
		markExpressionIdentifiersEscaping(report, expr.Consequent)
		markExpressionIdentifiersEscaping(report, expr.Alternative)
	case *ast.CommaExpression:
		markExpressionIdentifiersEscaping(report, expr.Left)
		markExpressionIdentifiersEscaping(report, expr.Right)
	case *ast.UnaryExpression:
		markExpressionIdentifiersEscaping(report, expr.Right)
	case *ast.UpdateExpression:
		markExpressionIdentifiersEscaping(report, expr.Target)
	case *ast.TypeofExpression:
		markExpressionIdentifiersEscaping(report, expr.Value)
	case *ast.AwaitExpression:
		markExpressionIdentifiersEscaping(report, expr.Value)
	case *ast.YieldExpression:
		markExpressionIdentifiersEscaping(report, expr.Value)
	case *ast.InstanceofExpression:
		markExpressionIdentifiersEscaping(report, expr.Left)
		markExpressionIdentifiersEscaping(report, expr.Right)
	case *ast.TemplateLiteral:
		markExpressionListIdentifiersEscaping(report, expr.Expressions)
	case *ast.ArrayLiteral:
		markExpressionListIdentifiersEscaping(report, expr.Elements)
	case *ast.ObjectLiteral:
		for _, property := range expr.Properties {
			markExpressionIdentifiersEscaping(report, property.KeyExpr)
			markExpressionIdentifiersEscaping(report, property.Value)
		}
	case *ast.SpreadExpression:
		markExpressionIdentifiersEscaping(report, expr.Value)
	case *ast.IndexExpression:
		markExpressionIdentifiersEscaping(report, expr.Target)
		markExpressionIdentifiersEscaping(report, expr.Index)
	case *ast.MemberExpression:
		markExpressionIdentifiersEscaping(report, expr.Target)
	case *ast.CallExpression:
		markExpressionListIdentifiersEscaping(report, expr.Arguments)
	case *ast.InvokeExpression:
		markExpressionIdentifiersEscaping(report, expr.Callee)
		markExpressionListIdentifiersEscaping(report, expr.Arguments)
	case *ast.NewExpression:
		markExpressionIdentifiersEscaping(report, expr.Callee)
		markExpressionListIdentifiersEscaping(report, expr.Arguments)
	case *ast.FunctionExpression:
		markFunctionCapturesEscaping(report, expr)
		analyzeExpression(report, newScope(nil), expr.ExpressionBody)
		analyzeStatements(report, newScope(nil), expr.Body)
	}
}

func markExpressionListIdentifiersEscaping(report *Report, expressions []ast.Expression) {
	for _, expr := range expressions {
		markExpressionIdentifiersEscaping(report, expr)
	}
}
