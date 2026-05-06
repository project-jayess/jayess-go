package semantic

import "jayess-go/ast"

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
