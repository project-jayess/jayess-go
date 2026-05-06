package semantic

import "jayess-go/ast"

func analyzeExpression(scope *scope, expr ast.Expression) error {
	return analyzeExpressionWithContext(scope, rootContext(), expr)
}

func analyzeExpressionWithContext(scope *scope, context controlContext, expr ast.Expression) error {
	switch expr := expr.(type) {
	case *ast.Identifier:
		if !scope.lookup(expr.Name) {
			return errorAt(expr, "use of %s before declaration", expr.Name)
		}
	case *ast.ThisExpression:
		if !context.allowThis {
			return errorAt(expr, "this outside function or method")
		}
		return nil
	case *ast.SuperExpression:
		if !context.inClassMethod && !context.inClassStaticBlock {
			return errorAt(expr, "super outside class method")
		}
		if !context.hasSuperClass {
			return errorAt(expr, "super in class without extends")
		}
		return nil
	case *ast.ImportMetaExpression:
		return nil
	case *ast.BinaryExpression:
		return analyzeExpressionsWithContext(scope, context, expr.Left, expr.Right)
	case *ast.ComparisonExpression:
		return analyzeExpressionsWithContext(scope, context, expr.Left, expr.Right)
	case *ast.LogicalExpression:
		return analyzeExpressionsWithContext(scope, context, expr.Left, expr.Right)
	case *ast.NullishCoalesceExpression:
		return analyzeExpressionsWithContext(scope, context, expr.Left, expr.Right)
	case *ast.ConditionalExpression:
		return analyzeExpressionsWithContext(scope, context, expr.Condition, expr.Consequent, expr.Alternative)
	case *ast.CommaExpression:
		return analyzeExpressionsWithContext(scope, context, expr.Left, expr.Right)
	case *ast.UnaryExpression:
		return analyzeUnaryExpressionWithContext(scope, context, expr)
	case *ast.UpdateExpression:
		return analyzeAssignmentTarget(scope, context, expr.Target)
	case *ast.TypeofExpression:
		if _, ok := expr.Value.(*ast.Identifier); ok {
			return nil
		}
		return analyzeExpressionWithContext(scope, context, expr.Value)
	case *ast.AwaitExpression:
		if !context.inAsyncFunction {
			return errorAt(expr, "await outside async function")
		}
		return analyzeExpressionWithContext(scope, context, expr.Value)
	case *ast.YieldExpression:
		if !context.inGenerator {
			return errorAt(expr, "yield outside generator function")
		}
		return analyzeExpressionWithContext(scope, context, expr.Value)
	case *ast.InstanceofExpression:
		return analyzeInstanceofExpression(scope, context, expr)
	case *ast.ArrayLiteral:
		return analyzeExpressionsWithContext(scope, context, expr.Elements...)
	case *ast.TemplateLiteral:
		return analyzeExpressionsWithContext(scope, context, expr.Expressions...)
	case *ast.ObjectLiteral:
		return analyzeObjectLiteral(scope, context, expr)
	case *ast.SpreadExpression:
		return analyzeExpressionWithContext(scope, context, expr.Value)
	case *ast.IndexExpression:
		return analyzeExpressionsWithContext(scope, context, expr.Target, expr.Index)
	case *ast.MemberExpression:
		if err := analyzeExpressionWithContext(scope, context, expr.Target); err != nil {
			return err
		}
		if expr.Private {
			return analyzePrivateMemberAccess(context, expr)
		}
		return nil
	case *ast.CallExpression:
		if !scope.lookup(expr.Callee) {
			return errorAt(expr, "use of %s before declaration", expr.Callee)
		}
		if err := analyzeEventLoopCall(expr); err != nil {
			return err
		}
		return analyzeExpressionsWithContext(scope, context, expr.Arguments...)
	case *ast.InvokeExpression:
		if _, ok := expr.Callee.(*ast.SuperExpression); ok && !context.inConstructor {
			return errorAt(expr, "super call outside constructor")
		}
		if err := analyzeExpressionWithContext(scope, context, expr.Callee); err != nil {
			return err
		}
		return analyzeExpressionsWithContext(scope, context, expr.Arguments...)
	case *ast.FunctionExpression:
		return analyzeFunctionExpressionInContext(scope, context, expr)
	case *ast.NewExpression:
		if err := analyzeNewExpression(scope, context, expr); err != nil {
			return err
		}
		return analyzeExpressionsWithContext(scope, context, expr.Arguments...)
	case *ast.NewTargetExpression:
		if !context.allowNewTarget {
			return errorAt(expr, "new.target outside function")
		}
		return nil
	}
	return nil
}

func analyzeExpressions(scope *scope, expressions ...ast.Expression) error {
	return analyzeExpressionsWithContext(scope, rootContext(), expressions...)
}

func analyzeExpressionsWithContext(scope *scope, context controlContext, expressions ...ast.Expression) error {
	for _, expr := range expressions {
		if expr == nil {
			continue
		}
		if err := analyzeExpressionWithContext(scope, context, expr); err != nil {
			return err
		}
	}
	return nil
}
