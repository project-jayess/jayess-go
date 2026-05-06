package llvmbackend

import (
	"fmt"

	"jayess-go/ast"
)

func (emitter *StatementEmitter) emitTryStatement(statement *ast.TryStatement) error {
	catchLabel := ""
	if len(statement.CatchBody) != 0 {
		catchLabel = emitter.expressions.nextBlockLabel("try.catch")
	}
	finallyLabel := ""
	if len(statement.FinallyBody) != 0 {
		finallyLabel = emitter.expressions.nextBlockLabel("try.finally")
	}
	endLabel := emitter.expressions.nextBlockLabel("try.end")
	throwTarget := firstNonEmpty(catchLabel, finallyLabel)
	if throwTarget == "" {
		return fmt.Errorf("runtime try statement needs catch or finally body")
	}

	emitter.ensureThrowSlot()
	emitter.pushThrowHandler(throwTarget)
	tryBody, err := emitter.captureScopedStatements(statement.TryBody)
	emitter.popThrowHandler()
	if err != nil {
		return err
	}

	var catchBody capturedStatements
	if catchLabel != "" {
		catchBody, err = emitter.captureCatchBody(statement)
		if err != nil {
			return err
		}
	}
	var finallyBody capturedStatements
	if finallyLabel != "" {
		finallyBody, err = emitter.captureScopedStatements(statement.FinallyBody)
		if err != nil {
			return err
		}
	}

	emitter.expressions.body = append(emitter.expressions.body, tryBody.lines...)
	if !tryBody.returns {
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+firstNonEmpty(finallyLabel, endLabel))
	}
	if catchLabel != "" {
		emitter.expressions.body = append(emitter.expressions.body, catchLabel+":")
		emitter.expressions.body = append(emitter.expressions.body, catchBody.lines...)
		if !catchBody.returns {
			emitter.expressions.body = append(emitter.expressions.body, "br label %"+firstNonEmpty(finallyLabel, endLabel))
		}
	}
	if finallyLabel != "" {
		emitter.expressions.body = append(emitter.expressions.body, finallyLabel+":")
		emitter.expressions.body = append(emitter.expressions.body, finallyBody.lines...)
		if !finallyBody.returns {
			emitter.expressions.body = append(emitter.expressions.body, "br label %"+endLabel)
		}
	}
	if tryBody.termination == statementTerminationReturn &&
		(catchBody.termination == statementTerminationReturn || catchLabel == "") &&
		(finallyBody.termination == statementTerminationReturn || finallyLabel == "") {
		emitter.returned = true
		emitter.termination = statementTerminationReturn
		return nil
	}
	emitter.expressions.body = append(emitter.expressions.body, endLabel+":")
	return nil
}

func (emitter *StatementEmitter) captureCatchBody(statement *ast.TryStatement) (capturedStatements, error) {
	start := len(emitter.expressions.body)
	previousReturned := emitter.returned
	previousTermination := emitter.termination
	previousTerminationLabel := emitter.terminationLabel
	emitter.returned = false
	emitter.termination = statementTerminationNone
	emitter.terminationLabel = ""

	emitter.enterLexicalScope(lexicalScopeCatch)
	var err error
	if statement.CatchPattern != nil {
		if err := emitter.declareBindingTargets(statement.CatchPattern, "undef"); err != nil {
			_ = emitter.leaveLexicalScope(false)
			emitter.restoreCapturedState(start, previousReturned, previousTermination, previousTerminationLabel)
			return capturedStatements{}, err
		}
		caught := emitter.expressions.nextValueName()
		emitter.expressions.body = append(emitter.expressions.body, caught+" = load "+runtimeValueIRType+", "+runtimeValueIRType+"* "+emitter.throwSlot.Name)
		if err := emitter.emitDestructureToPattern(statement.CatchPattern, caught); err != nil {
			_ = emitter.leaveLexicalScope(false)
			emitter.restoreCapturedState(start, previousReturned, previousTermination, previousTerminationLabel)
			return capturedStatements{}, err
		}
		emitter.registerDeclarationLifetime(statement, statement.CatchPattern)
	}
	err = emitter.EmitStatements(statement.CatchBody)
	popErr := emitter.leaveLexicalScope(!emitter.returned)
	lines := append([]string{}, emitter.expressions.body[start:]...)
	body := capturedStatements{lines: lines, returns: emitter.returned, termination: emitter.termination}
	emitter.restoreCapturedState(start, previousReturned, previousTermination, previousTerminationLabel)
	if err != nil {
		return body, err
	}
	return body, popErr
}

func (emitter *StatementEmitter) restoreCapturedState(start int, returned bool, termination statementTermination, terminationLabel string) {
	emitter.expressions.body = emitter.expressions.body[:start]
	emitter.returned = returned
	emitter.termination = termination
	emitter.terminationLabel = terminationLabel
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
