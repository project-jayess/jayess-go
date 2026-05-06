package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitWhileStatement(statement *ast.WhileStatement) error {
	conditionLabel := emitter.expressions.nextBlockLabel("while.cond")
	bodyLabel := emitter.expressions.nextBlockLabel("while.body")
	endLabel := emitter.expressions.nextBlockLabel("while.end")

	emitter.pushStructuredExit(structuredExit{kind: structuredExitLoop, breakLabel: endLabel, continueLabel: conditionLabel})
	body, err := emitter.captureScopedStatements(statement.Body)
	emitter.popStructuredExit()
	if err != nil {
		return err
	}
	return emitter.emitPrimitiveLoop(primitiveLoop{
		ConditionLabel: conditionLabel,
		BodyLabel:      bodyLabel,
		ContinueLabel:  conditionLabel,
		EndLabel:       endLabel,
		Condition:      statement.Condition,
		Body:           body,
	})
}
