package escape

import "jayess-go/ast"

func markObjectStoredValuesEscaping(report *Report, expr *ast.ObjectLiteral) {
	for _, property := range expr.Properties {
		markExpressionIdentifiersEscaping(report, property.Value)
	}
}
