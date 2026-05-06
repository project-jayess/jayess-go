package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

type capturedExpression struct {
	lines []string
	value string
}

func (emitter *ExpressionEmitter) captureExpression(expression ast.Expression) (capturedExpression, error) {
	start := len(emitter.body)
	value, err := emitter.EmitExpression(expression)
	lines := append([]string{}, emitter.body[start:]...)
	emitter.body = emitter.body[:start]
	return capturedExpression{lines: lines, value: value}, err
}

func (emitter *ExpressionEmitter) emitBranchValue(prefix string, condition string, trueExpression ast.Expression, falseExpression ast.Expression) (string, error) {
	if condition == "" {
		return "", fmt.Errorf("branch expression condition must not be empty")
	}
	trueBranch, err := emitter.captureExpression(trueExpression)
	if err != nil {
		return "", err
	}
	falseBranch, err := emitter.captureExpression(falseExpression)
	if err != nil {
		return "", err
	}

	return emitter.emitCapturedBranchValue(prefix, condition, trueBranch, falseBranch)
}

func (emitter *ExpressionEmitter) emitCapturedBranchValue(prefix string, condition string, trueBranch capturedExpression, falseBranch capturedExpression) (string, error) {
	trueLabel := emitter.nextBlockLabel(prefix + ".true")
	falseLabel := emitter.nextBlockLabel(prefix + ".false")
	endLabel := emitter.nextBlockLabel(prefix + ".end")
	result := emitter.nextValueName()

	emitter.body = append(emitter.body,
		"br i1 "+condition+", label %"+trueLabel+", label %"+falseLabel,
		trueLabel+":",
	)
	emitter.body = append(emitter.body, trueBranch.lines...)
	emitter.body = append(emitter.body, "br label %"+endLabel, falseLabel+":")
	emitter.body = append(emitter.body, falseBranch.lines...)
	emitter.body = append(emitter.body,
		"br label %"+endLabel,
		endLabel+":",
		result+" = phi "+runtimeValueIRType+" [ "+trueBranch.value+", %"+trueLabel+" ], [ "+falseBranch.value+", %"+falseLabel+" ]",
	)
	return result, nil
}
