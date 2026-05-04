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

func analyzePrivateMemberAccess(context controlContext, expr *ast.MemberExpression) error {
	if !context.inClassMethod && !context.inClassField && !context.inClassStaticBlock {
		return errorAt(expr, "private member access outside class method")
	}
	if !context.privateMembers[expr.Property] {
		return errorAt(expr, "private member #%s is not declared", expr.Property)
	}
	return nil
}

func analyzeNewExpression(scope *scope, context controlContext, expr *ast.NewExpression) error {
	if err := analyzeExpressionWithContext(scope, context, expr.Callee); err != nil {
		return err
	}
	return analyzeConstructableIdentifier(scope, expr, expr.Callee, "new target")
}

func analyzeInstanceofExpression(scope *scope, context controlContext, expr *ast.InstanceofExpression) error {
	if err := analyzeExpressionsWithContext(scope, context, expr.Left, expr.Right); err != nil {
		return err
	}
	return analyzeConstructableIdentifier(scope, expr, expr.Right, "instanceof target")
}

func analyzeConstructableIdentifier(scope *scope, source ast.Node, target ast.Expression, label string) error {
	identifier, ok := target.(*ast.Identifier)
	if !ok {
		return nil
	}
	bind, ok := scope.resolve(identifier.Name)
	if !ok {
		return errorAt(identifier, "use of %s before declaration", identifier.Name)
	}
	if !bind.constructable {
		return errorAt(source, "%s %s is not constructable", label, identifier.Name)
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

func analyzeObjectLiteral(scope *scope, context controlContext, expr *ast.ObjectLiteral) error {
	for _, property := range expr.Properties {
		if err := analyzeOptionalExpressionWithContext(scope, context, property.KeyExpr); err != nil {
			return err
		}
		if property.Method {
			method, ok := property.Value.(*ast.FunctionExpression)
			if !ok {
				return errorAt(expr, "invalid object method")
			}
			if err := analyzeFunctionExpressionWithContext(scope, method, rootContext().enterObjectMethod(method.IsAsync, method.IsGenerator)); err != nil {
				return err
			}
			continue
		}
		if err := analyzeOptionalExpressionWithContext(scope, context, property.Value); err != nil {
			return err
		}
	}
	return nil
}

func analyzeFunctionExpression(parent *scope, expr *ast.FunctionExpression) error {
	return analyzeFunctionExpressionInContext(parent, rootContext(), expr)
}

func analyzeFunctionExpressionInContext(parent *scope, context controlContext, expr *ast.FunctionExpression) error {
	if expr.IsArrowFunction {
		return analyzeFunctionExpressionWithContext(parent, expr, context.enterArrowFunction(expr.IsAsync, expr.IsGenerator))
	}
	return analyzeFunctionExpressionWithContext(parent, expr, rootContext().enterFunction(expr.IsAsync, expr.IsGenerator))
}

func analyzeFunctionExpressionWithContext(parent *scope, expr *ast.FunctionExpression, context controlContext) error {
	functionScope := newScope(parent)
	if expr.Name != "" && !functionScope.declare(expr.Name) {
		return errorAt(expr, "duplicate declaration %s", expr.Name)
	}
	if err := declareParametersWithContext(functionScope, context, expr.Params); err != nil {
		return err
	}
	declareArgumentsBinding(functionScope, context)
	if err := analyzeOptionalExpressionWithContext(functionScope, context, expr.ExpressionBody); err != nil {
		return err
	}
	return analyzeStatements(functionScope, context, expr.Body)
}

func analyzeAssignmentTarget(scope *scope, context controlContext, target ast.Expression) error {
	switch target := target.(type) {
	case *ast.Identifier:
		bind, ok := scope.resolve(target.Name)
		if !ok {
			return errorAt(target, "assignment to %s before declaration", target.Name)
		}
		if !bind.mutable {
			return errorAt(target, "assignment to const %s; use var for mutable bindings", target.Name)
		}
		return nil
	case *ast.MemberExpression:
		if err := analyzeExpressionWithContext(scope, context, target.Target); err != nil {
			return err
		}
		if target.Private {
			return analyzePrivateMemberAccess(context, target)
		}
		return nil
	case *ast.IndexExpression:
		return analyzeExpressionsWithContext(scope, context, target.Target, target.Index)
	default:
		return errorAt(target, "invalid assignment target")
	}
}
