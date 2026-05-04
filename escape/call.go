package escape

import "jayess-go/ast"

func markUnknownCallArgumentsEscaping(report *Report, scope *scope, expr *ast.CallExpression) {
	if isKnownDirectCall(scope, expr.Callee) {
		return
	}
	markExpressionListIdentifiersEscaping(report, expr.Arguments)
}

func markInvokeArgumentsEscaping(report *Report, expr *ast.InvokeExpression) {
	markExpressionListIdentifiersEscaping(report, expr.Arguments)
}

func markNewArgumentsEscaping(report *Report, expr *ast.NewExpression) {
	markExpressionListIdentifiersEscaping(report, expr.Arguments)
}

func isKnownDirectCall(scope *scope, name string) bool {
	return scope.hasLocal(name) || scope.hasOuter(name)
}
