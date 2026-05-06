package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

func (emitter *ExpressionEmitter) emitIndexExpression(expression *ast.IndexExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("index expression must not be nil")
	}
	target, err := emitter.ResolveAssignmentTarget(expression)
	if err != nil {
		return "", err
	}
	return target.Load()
}
