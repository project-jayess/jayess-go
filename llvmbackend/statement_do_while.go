package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitDoWhileStatement(statement *ast.DoWhileStatement) error {
	conditionLabel := emitter.expressions.nextBlockLabel("do.while.cond")
	bodyLabel := emitter.expressions.nextBlockLabel("do.while.body")
	endLabel := emitter.expressions.nextBlockLabel("do.while.end")

	emitter.pushStructuredExit(structuredExit{kind: structuredExitLoop, breakLabel: endLabel, continueLabel: conditionLabel})
	if err := emitter.emitScopedStatements(lexicalScopeLoop, statement.Body); err != nil {
		emitter.popStructuredExit()
		return err
	}
	emitter.popStructuredExit()
	if emitter.Returned() {
		if emitter.termination == statementTerminationReturn {
			return nil
		}
		emitter.returned = false
		emitter.termination = statementTerminationNone
	}
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
