package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitForStatement(statement *ast.ForStatement) error {
	emitter.enterLexicalScope(lexicalScopeLoop)
	defer func() {
		_ = emitter.leaveLexicalScope(!emitter.returned)
	}()

	if statement.Init != nil {
		if err := emitter.EmitStatement(statement.Init); err != nil {
			return err
		}
	}

	conditionLabel := emitter.expressions.nextBlockLabel("for.cond")
	bodyLabel := emitter.expressions.nextBlockLabel("for.body")
	updateLabel := emitter.expressions.nextBlockLabel("for.update")
	endLabel := emitter.expressions.nextBlockLabel("for.end")

	emitter.pushStructuredExit(structuredExit{kind: structuredExitLoop, breakLabel: endLabel, continueLabel: updateLabel})
	body, err := emitter.captureScopedStatements(statement.Body)
	emitter.popStructuredExit()
	if err != nil {
		return err
	}
	update := capturedStatements{}
	if statement.Update != nil {
		var err error
		update, err = emitter.captureStatements([]ast.Statement{statement.Update}, false)
		if err != nil {
			return err
		}
	}
	return emitter.emitPrimitiveLoop(primitiveLoop{
		ConditionLabel: conditionLabel,
		BodyLabel:      bodyLabel,
		ContinueLabel:  updateLabel,
		EndLabel:       endLabel,
		Condition:      statement.Condition,
		Body:           body,
		Continue:       update,
	})
}
