package llvmbackend

import "jayess-go/ast"

const runtimeInSymbol = "jayess_value_in"

func (emitter *ExpressionEmitter) emitInExpression(expression *ast.ComparisonExpression) (string, bool, error) {
	if expression.Operator != ast.OperatorIn {
		return "", false, nil
	}
	left, err := emitter.EmitExpression(expression.Left)
	if err != nil {
		return "", true, err
	}
	right, err := emitter.EmitExpression(expression.Right)
	if err != nil {
		return "", true, err
	}
	result, err := emitter.emitRuntimeBinaryValue(runtimeInSymbol, left, right)
	return result, true, err
}
