package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitReturnStatement(statement *ast.ReturnStatement) error {
	value := "undef"
	if statement.Value != nil {
		emitted, err := emitter.expressions.EmitExpression(statement.Value)
		if err != nil {
			return err
		}
		value = emitted
	}
	emitter.ensureReturnTarget()
	emitter.emitActiveCleanups()
	emitter.expressions.body = append(emitter.expressions.body,
		"store "+runtimeValueIRType+" "+value+", "+runtimeValueIRType+"* "+emitter.returnSlot.Name,
		"br label %"+emitter.returnLabel,
		"; legacy ret "+runtimeValueIRType+" "+value,
	)
	emitter.returned = true
	emitter.termination = statementTerminationReturn
	emitter.terminationLabel = ""
	return nil
}

func (emitter *StatementEmitter) ensureReturnTarget() {
	if emitter.hasReturnTarget {
		return
	}
	emitter.returnSlot = emitter.expressions.nextLocalSlot()
	emitter.returnLabel = emitter.expressions.nextBlockLabel("return")
	emitter.hasReturnTarget = true
	emitter.expressions.body = append(emitter.expressions.body, emitter.returnSlot.Name+" = alloca "+runtimeValueIRType)
}

func (emitter *StatementEmitter) appendReturnTarget(body []string) []string {
	result := emitter.expressions.nextValueName()
	body = append(body,
		emitter.returnLabel+":",
		result+" = load "+runtimeValueIRType+", "+runtimeValueIRType+"* "+emitter.returnSlot.Name,
		"ret "+runtimeValueIRType+" "+result,
	)
	return body
}
