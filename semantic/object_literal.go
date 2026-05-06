package semantic

import "jayess-go/ast"

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
