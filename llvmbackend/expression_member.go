package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

func (emitter *ExpressionEmitter) emitMemberExpression(expression *ast.MemberExpression) (string, error) {
	if expression == nil {
		return "", fmt.Errorf("member expression must not be nil")
	}
	target, err := emitter.ResolveAssignmentTarget(expression)
	if err != nil {
		return "", err
	}
	return target.Load()
}
