package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitPrimitiveWhileLoop(prefix string, condition ast.Expression, body capturedStatements) error {
	conditionLabel := emitter.expressions.nextBlockLabel(prefix + ".cond")
	bodyLabel := emitter.expressions.nextBlockLabel(prefix + ".body")
	endLabel := emitter.expressions.nextBlockLabel(prefix + ".end")
	return emitter.emitPrimitiveLoop(primitiveLoop{
		ConditionLabel: conditionLabel,
		BodyLabel:      bodyLabel,
		ContinueLabel:  conditionLabel,
		EndLabel:       endLabel,
		Condition:      condition,
		Body:           body,
	})
}

type primitiveLoop struct {
	ConditionLabel string
	BodyLabel      string
	ContinueLabel  string
	EndLabel       string
	Condition      ast.Expression
	Body           capturedStatements
	Continue       capturedStatements
}

func (emitter *StatementEmitter) emitPrimitiveLoop(loop primitiveLoop) error {
	emitter.expressions.body = append(emitter.expressions.body,
		"br label %"+loop.ConditionLabel,
		loop.ConditionLabel+":",
	)
	conditionValue := "1"
	if loop.Condition != nil {
		emitted, err := emitter.expressions.EmitExpression(loop.Condition)
		if err != nil {
			return err
		}
		truthy, err := emitter.expressions.EmitTruthiness(emitted)
		if err != nil {
			return err
		}
		conditionValue = truthy
	}
	emitter.expressions.body = append(emitter.expressions.body,
		"br i1 "+conditionValue+", label %"+loop.BodyLabel+", label %"+loop.EndLabel,
		loop.BodyLabel+":",
	)
	emitter.expressions.body = append(emitter.expressions.body, loop.Body.lines...)
	if !loop.Body.returns {
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+loop.ContinueLabel)
	}
	if len(loop.Continue.lines) != 0 || loop.ContinueLabel != loop.ConditionLabel {
		emitter.expressions.body = append(emitter.expressions.body, loop.ContinueLabel+":")
		emitter.expressions.body = append(emitter.expressions.body, loop.Continue.lines...)
		if !loop.Continue.returns {
			emitter.expressions.body = append(emitter.expressions.body, "br label %"+loop.ConditionLabel)
		}
	}
	emitter.expressions.body = append(emitter.expressions.body, loop.EndLabel+":")
	return nil
}
