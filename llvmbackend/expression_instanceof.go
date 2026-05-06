package llvmbackend

import "jayess-go/ast"

const runtimeInstanceofSymbol = "jayess_value_instanceof"

func (emitter *ExpressionEmitter) emitInstanceofExpression(expression *ast.InstanceofExpression) (string, error) {
	left, err := emitter.EmitExpression(expression.Left)
	if err != nil {
		return "", err
	}
	right, err := emitter.EmitExpression(expression.Right)
	if err != nil {
		return "", err
	}
	return emitter.emitRuntimeBinaryValue(runtimeInstanceofSymbol, left, right)
}
