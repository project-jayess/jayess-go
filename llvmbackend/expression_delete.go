package llvmbackend

import "jayess-go/ast"

const runtimeDeleteValueSymbol = "jayess_value_delete_value"

func (emitter *ExpressionEmitter) emitDeleteExpression(expression *ast.UnaryExpression) (string, error) {
	if target, err := emitter.ResolveAssignmentTarget(expression.Right); err == nil {
		return target.Delete()
	}
	value, err := emitter.EmitExpressionSequence(expression.Right)
	if err != nil {
		return "", err
	}
	return emitter.emitRuntimeUnaryValue(runtimeDeleteValueSymbol, value)
}
