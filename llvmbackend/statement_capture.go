package llvmbackend

import "jayess-go/ast"

type capturedStatements struct {
	lines       []string
	returns     bool
	termination statementTermination
}

func (emitter *StatementEmitter) captureScopedStatements(statements []ast.Statement) (capturedStatements, error) {
	return emitter.captureStatements(statements, true)
}

func (emitter *StatementEmitter) captureStatements(statements []ast.Statement, scoped bool) (capturedStatements, error) {
	start := len(emitter.expressions.body)
	previousReturned := emitter.returned
	previousTermination := emitter.termination
	previousTerminationLabel := emitter.terminationLabel
	emitter.returned = false
	emitter.termination = statementTerminationNone
	emitter.terminationLabel = ""
	var err error
	if scoped {
		err = emitter.emitScopedStatements(lexicalScopeBlock, statements)
	} else {
		err = emitter.EmitStatements(statements)
	}
	lines := append([]string{}, emitter.expressions.body[start:]...)
	emitter.expressions.body = emitter.expressions.body[:start]
	result := capturedStatements{lines: lines, returns: emitter.returned, termination: emitter.termination}
	emitter.returned = previousReturned
	emitter.termination = previousTermination
	emitter.terminationLabel = previousTerminationLabel
	return result, err
}
