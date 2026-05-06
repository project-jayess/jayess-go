package llvmbackend

import "jayess-go/ast"

const runtimeUnhandledThrowSymbol = "jayess_throw_unhandled"

func (emitter *StatementEmitter) emitThrowStatement(statement *ast.ThrowStatement) error {
	value, err := emitter.expressions.EmitExpression(statement.Value)
	if err != nil {
		return err
	}
	emitter.ensureThrowSlot()
	target := emitter.activeThrowHandler()
	if target == "" {
		emitter.ensureThrowTarget()
		target = emitter.throwLabel
	}
	emitter.emitActiveCleanups()
	emitter.expressions.body = append(emitter.expressions.body,
		"store "+runtimeValueIRType+" "+value+", "+runtimeValueIRType+"* "+emitter.throwSlot.Name,
		"br label %"+target,
	)
	emitter.returned = true
	emitter.termination = statementTerminationThrow
	emitter.terminationLabel = ""
	return nil
}

func (emitter *StatementEmitter) ensureThrowSlot() {
	if emitter.throwSlot.Name != "" {
		return
	}
	emitter.throwSlot = emitter.expressions.nextLocalSlot()
	emitter.expressions.body = append(emitter.expressions.body, emitter.throwSlot.Name+" = alloca "+runtimeValueIRType)
}

func (emitter *StatementEmitter) ensureThrowTarget() {
	if emitter.hasThrowTarget {
		return
	}
	emitter.ensureThrowSlot()
	emitter.throwLabel = emitter.expressions.nextBlockLabel("throw")
	emitter.hasThrowTarget = true
}

func (emitter *StatementEmitter) appendThrowTarget(body []string) []string {
	result := emitter.expressions.nextValueName()
	args := []RuntimeCallArg{{IRType: runtimeValueIRType, Value: result}}
	emitter.expressions.addDeclarations([]Declaration{RuntimeCallDeclaration(runtimeUnhandledThrowSymbol, "void", args)})
	body = append(body,
		emitter.throwLabel+":",
		result+" = load "+runtimeValueIRType+", "+runtimeValueIRType+"* "+emitter.throwSlot.Name,
		RuntimeVoidCall(runtimeUnhandledThrowSymbol, args),
		"ret "+runtimeValueIRType+" undef",
	)
	return body
}

func (emitter *StatementEmitter) pushThrowHandler(label string) {
	emitter.throwHandlers = append(emitter.throwHandlers, label)
}

func (emitter *StatementEmitter) popThrowHandler() {
	if len(emitter.throwHandlers) == 0 {
		return
	}
	emitter.throwHandlers = emitter.throwHandlers[:len(emitter.throwHandlers)-1]
}

func (emitter *StatementEmitter) activeThrowHandler() string {
	if len(emitter.throwHandlers) == 0 {
		return ""
	}
	return emitter.throwHandlers[len(emitter.throwHandlers)-1]
}
