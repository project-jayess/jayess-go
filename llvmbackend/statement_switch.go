package llvmbackend

import "jayess-go/ast"

func (emitter *StatementEmitter) emitSwitchStatement(statement *ast.SwitchStatement) error {
	discriminant, err := emitter.expressions.EmitExpression(statement.Discriminant)
	if err != nil {
		return err
	}

	endLabel := emitter.expressions.nextBlockLabel("switch.end")
	defaultLabel := emitter.expressions.nextBlockLabel("switch.default")
	checkLabels := emitter.switchCheckLabels(len(statement.Cases))
	caseLabels := emitter.switchCaseLabels(len(statement.Cases))

	emitter.pushStructuredExit(structuredExit{kind: structuredExitSwitch, breakLabel: endLabel})
	cases, defaultCase, err := emitter.captureSwitchCases(statement)
	emitter.popStructuredExit()
	if err != nil {
		return err
	}

	emitter.expressions.body = append(emitter.expressions.body, "br label %"+switchFirstCheckLabel(checkLabels, defaultLabel))
	for index, switchCase := range cases {
		nextCheck := switchNextCheckLabel(index, checkLabels, defaultLabel)
		emitter.expressions.body = append(emitter.expressions.body, checkLabels[index]+":")
		test, err := emitter.expressions.EmitExpression(switchCase.test)
		if err != nil {
			return err
		}
		matched, err := emitter.expressions.EmitRuntimeStrictEqual(discriminant, test)
		if err != nil {
			return err
		}
		emitter.expressions.body = append(emitter.expressions.body,
			"br i1 "+matched+", label %"+caseLabels[index]+", label %"+nextCheck,
			caseLabels[index]+":",
		)
		emitter.expressions.body = append(emitter.expressions.body, switchCase.body.lines...)
		if !switchCase.body.returns {
			emitter.expressions.body = append(emitter.expressions.body, "br label %"+switchFallthroughLabel(index, caseLabels, defaultLabel, endLabel))
		}
	}

	emitter.expressions.body = append(emitter.expressions.body, defaultLabel+":")
	emitter.expressions.body = append(emitter.expressions.body, defaultCase.lines...)
	if !defaultCase.returns {
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+endLabel)
	}
	emitter.expressions.body = append(emitter.expressions.body, endLabel+":")
	return nil
}

type capturedSwitchCase struct {
	test ast.Expression
	body capturedStatements
}

func (emitter *StatementEmitter) captureSwitchCases(statement *ast.SwitchStatement) ([]capturedSwitchCase, capturedStatements, error) {
	cases := make([]capturedSwitchCase, 0, len(statement.Cases))
	for _, switchCase := range statement.Cases {
		body, err := emitter.captureScopedStatements(switchCase.Consequent)
		if err != nil {
			return nil, capturedStatements{}, err
		}
		cases = append(cases, capturedSwitchCase{test: switchCase.Test, body: body})
	}
	defaultCase, err := emitter.captureScopedStatements(statement.Default)
	if err != nil {
		return nil, capturedStatements{}, err
	}
	return cases, defaultCase, nil
}

func (emitter *StatementEmitter) switchCheckLabels(count int) []string {
	labels := make([]string, 0, count)
	for i := 0; i < count; i++ {
		labels = append(labels, emitter.expressions.nextBlockLabel("switch.check"))
	}
	return labels
}

func (emitter *StatementEmitter) switchCaseLabels(count int) []string {
	labels := make([]string, 0, count)
	for i := 0; i < count; i++ {
		labels = append(labels, emitter.expressions.nextBlockLabel("switch.case"))
	}
	return labels
}

func switchFirstCheckLabel(checkLabels []string, defaultLabel string) string {
	if len(checkLabels) == 0 {
		return defaultLabel
	}
	return checkLabels[0]
}

func switchNextCheckLabel(index int, checkLabels []string, defaultLabel string) string {
	next := index + 1
	if next >= len(checkLabels) {
		return defaultLabel
	}
	return checkLabels[next]
}

func switchFallthroughLabel(index int, caseLabels []string, defaultLabel string, endLabel string) string {
	next := index + 1
	if next < len(caseLabels) {
		return caseLabels[next]
	}
	if defaultLabel != "" {
		return defaultLabel
	}
	return endLabel
}
