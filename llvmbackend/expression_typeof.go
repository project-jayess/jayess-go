package llvmbackend

import "jayess-go/ast"

const runtimeTypeofSymbol = "jayess_value_typeof"

func (emitter *ExpressionEmitter) emitTypeofExpression(expression *ast.TypeofExpression) (string, error) {
	value, err := emitter.EmitExpression(expression.Value)
	if err != nil {
		return "", err
	}
	return emitter.emitRuntimeUnaryValue(runtimeTypeofSymbol, value)
}
