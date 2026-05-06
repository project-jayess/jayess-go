package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitIfStatement(statement *ast.IfStatement) error {
	conditionValue, err := emitter.expressions.EmitExpression(statement.Condition)
	if err != nil {
		return err
	}
	condition, err := emitter.expressions.EmitTruthiness(conditionValue)
	if err != nil {
		return err
	}

	thenLabel := emitter.expressions.nextBlockLabel("if.then")
	elseLabel := emitter.expressions.nextBlockLabel("if.else")
	endLabel := emitter.expressions.nextBlockLabel("if.end")

	consequence, err := emitter.captureScopedStatements(statement.Consequence)
	if err != nil {
		return err
	}
	alternative, err := emitter.captureScopedStatements(statement.Alternative)
	if err != nil {
		return err
	}

	emitter.expressions.body = append(emitter.expressions.body,
		"br i1 "+condition+", label %"+thenLabel+", label %"+elseLabel,
		thenLabel+":",
	)
	emitter.expressions.body = append(emitter.expressions.body, consequence.lines...)
	if !consequence.returns {
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+endLabel)
	}
	emitter.expressions.body = append(emitter.expressions.body, elseLabel+":")
	emitter.expressions.body = append(emitter.expressions.body, alternative.lines...)
	if !alternative.returns {
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+endLabel)
	}
	if consequence.returns && alternative.returns {
		emitter.returned = true
		return nil
	}
	emitter.expressions.body = append(emitter.expressions.body, endLabel+":")
	return nil
}
