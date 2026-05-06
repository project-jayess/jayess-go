package llvmbackend

import "jayess-go/ast"

func (emitter *ExpressionEmitter) emitNullishCoalesceExpression(expression *ast.NullishCoalesceExpression) (string, error) {
	left, err := emitter.EmitExpression(expression.Left)
	if err != nil {
		return "", err
	}
	condition, err := emitter.EmitNullishCheck(left)
	if err != nil {
		return "", err
	}
	rightBranch, err := emitter.captureExpression(expression.Right)
	if err != nil {
		return "", err
	}
	return emitter.emitCapturedBranchValue("nullish", condition, rightBranch, capturedExpression{value: left})
}
