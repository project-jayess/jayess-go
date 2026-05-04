package test

import "testing"

func TestSemanticAllowsLabeledBreakToBlock(t *testing.T) {
	err := analyzeSource(t, `
		done: {
			break done;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsLabeledContinueToLoop(t *testing.T) {
	err := analyzeSource(t, `
		outer: while (true) {
			continue outer;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownBreakLabel(t *testing.T) {
	err := analyzeSource(t, `
		while (true) {
			break missing;
		}
	`)
	requireSemanticError(t, err, "unknown label missing")
}

func TestSemanticRejectsUnknownContinueLabel(t *testing.T) {
	err := analyzeSource(t, `
		while (true) {
			continue missing;
		}
	`)
	requireSemanticError(t, err, "unknown label missing")
}

func TestSemanticRejectsContinueToNonLoopLabel(t *testing.T) {
	err := analyzeSource(t, `
		retry: {
			continue retry;
		}
	`)
	requireSemanticError(t, err, "continue target retry is not a loop")
}

func TestSemanticRejectsDuplicateActiveLabel(t *testing.T) {
	err := analyzeSource(t, `
		again: {
			again: {
				break again;
			}
		}
	`)
	requireSemanticError(t, err, "duplicate label again")
}
