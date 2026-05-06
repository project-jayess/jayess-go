package llvmbackend

import "fmt"

type structuredExitKind string

const (
	structuredExitBlock  structuredExitKind = "block"
	structuredExitLoop   structuredExitKind = "loop"
	structuredExitSwitch structuredExitKind = "switch"
)

type structuredExit struct {
	kind          structuredExitKind
	labels        []string
	breakLabel    string
	continueLabel string
	cleanupDepth  int
}

func (emitter *StatementEmitter) pushStructuredExit(exit structuredExit) {
	exit.cleanupDepth = len(emitter.cleanupScopes)
	if len(exit.labels) == 0 && len(emitter.pendingLabels) != 0 {
		exit.labels = append([]string{}, emitter.pendingLabels...)
	}
	emitter.exits = append(emitter.exits, exit)
}

func (emitter *StatementEmitter) popStructuredExit() {
	if len(emitter.exits) == 0 {
		return
	}
	emitter.exits = emitter.exits[:len(emitter.exits)-1]
}

func (emitter *StatementEmitter) emitBreak(label string) error {
	for index := len(emitter.exits) - 1; index >= 0; index-- {
		exit := emitter.exits[index]
		if label != "" && !exit.hasLabel(label) {
			continue
		}
		if label == "" && exit.kind == structuredExitBlock {
			continue
		}
		if exit.breakLabel == "" {
			continue
		}
		emitter.emitCleanupsUntil(exit.cleanupDepth)
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+exit.breakLabel)
		emitter.returned = true
		emitter.termination = statementTerminationBreak
		emitter.terminationLabel = label
		return nil
	}
	if label != "" {
		return fmt.Errorf("runtime break has no enclosing label %s", label)
	}
	return fmt.Errorf("runtime break has no enclosing loop or switch")
}

func (emitter *StatementEmitter) emitContinue(label string) error {
	for index := len(emitter.exits) - 1; index >= 0; index-- {
		exit := emitter.exits[index]
		if label != "" && !exit.hasLabel(label) {
			continue
		}
		if exit.kind != structuredExitLoop || exit.continueLabel == "" {
			continue
		}
		emitter.emitCleanupsUntil(exit.cleanupDepth)
		emitter.expressions.body = append(emitter.expressions.body, "br label %"+exit.continueLabel)
		emitter.returned = true
		emitter.termination = statementTerminationContinue
		emitter.terminationLabel = label
		return nil
	}
	if label != "" {
		return fmt.Errorf("runtime continue has no enclosing loop label %s", label)
	}
	return fmt.Errorf("runtime continue has no enclosing loop")
}

func (exit structuredExit) hasLabel(label string) bool {
	for _, candidate := range exit.labels {
		if candidate == label {
			return true
		}
	}
	return false
}
