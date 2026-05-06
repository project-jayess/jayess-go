package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

func (emitter *ExpressionEmitter) emitCommaExpression(expression *ast.CommaExpression) (string, error) {
	return emitter.EmitExpressionSequence(expression.Left, expression.Right)
}

func (emitter *ExpressionEmitter) EmitExpressionSequence(expressions ...ast.Expression) (string, error) {
	if len(expressions) == 0 {
		return "", fmt.Errorf("runtime expression sequence must not be empty")
	}
	var result string
	for _, expression := range expressions {
		value, err := emitter.EmitExpression(expression)
		if err != nil {
			return "", err
		}
		result = value
	}
	return result, nil
}
