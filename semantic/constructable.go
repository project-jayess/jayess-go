package semantic

import "jayess-go/ast"

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
