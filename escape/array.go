package escape

import "jayess-go/ast"

func markArrayStoredValuesEscaping(report *Report, expr *ast.ArrayLiteral) {
	markExpressionListIdentifiersEscaping(report, expr.Elements)
}
