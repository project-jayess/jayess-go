package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

func (emitter *ExpressionEmitter) emitLogicalExpression(expression *ast.LogicalExpression) (string, error) {
	left, err := emitter.EmitExpression(expression.Left)
	if err != nil {
		return "", err
	}
	condition, err := emitter.EmitTruthiness(left)
	if err != nil {
		return "", err
	}
	leftBranch := capturedExpression{value: left}
	switch expression.Operator {
	case ast.OperatorAnd:
		rightBranch, err := emitter.captureExpression(expression.Right)
		if err != nil {
			return "", err
		}
		return emitter.emitCapturedBranchValue("logical.and", condition, rightBranch, leftBranch)
	case ast.OperatorOr:
		rightBranch, err := emitter.captureExpression(expression.Right)
		if err != nil {
			return "", err
		}
		return emitter.emitCapturedBranchValue("logical.or", condition, leftBranch, rightBranch)
	default:
		return "", fmt.Errorf("unsupported runtime logical operator %s", expression.Operator)
	}
}
