package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitLabeledStatement(statement *ast.LabeledStatement) error {
	if statement.Label == "" {
		return emitter.EmitStatement(statement.Statement)
	}
	if block, ok := statement.Statement.(*ast.BlockStatement); ok {
		return emitter.emitLabeledBlock(statement.Label, block)
	}
	emitter.pendingLabels = append(emitter.pendingLabels, statement.Label)
	err := emitter.EmitStatement(statement.Statement)
	emitter.pendingLabels = emitter.pendingLabels[:len(emitter.pendingLabels)-1]
	return err
}

func (emitter *StatementEmitter) emitLabeledBlock(label string, block *ast.BlockStatement) error {
	endLabel := emitter.expressions.nextBlockLabel("label." + label + ".end")
	emitter.pendingLabels = append(emitter.pendingLabels, label)
	emitter.pushStructuredExit(structuredExit{kind: structuredExitBlock, breakLabel: endLabel})
	err := emitter.emitBlockStatement(block)
	emitter.popStructuredExit()
	emitter.pendingLabels = emitter.pendingLabels[:len(emitter.pendingLabels)-1]
	if err != nil {
		return err
	}
	if emitter.returned && emitter.termination == statementTerminationBreak && emitter.terminationLabel == label {
		emitter.returned = false
		emitter.termination = statementTerminationNone
		emitter.terminationLabel = ""
		emitter.expressions.body = append(emitter.expressions.body, endLabel+":")
		return nil
	}
	if !emitter.returned {
		emitter.expressions.body = append(emitter.expressions.body, endLabel+":")
	}
	return nil
}
