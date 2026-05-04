package test

import "testing"

func TestSemanticRejectsReturnOutsideFunction(t *testing.T) {
	err := analyzeSource(t, `return 1;`)
	requireSemanticError(t, err, "return outside function")
}

func TestSemanticRejectsReturnInsideTopLevelBlock(t *testing.T) {
	err := analyzeSource(t, `{ return 1; }`)
	requireSemanticError(t, err, "return outside function")
}

func TestSemanticAllowsReturnInsideFunction(t *testing.T) {
	err := analyzeSource(t, `
		function main() {
			if (true) {
				return 1;
			}
			return 0;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsBreakOutsideLoopOrSwitch(t *testing.T) {
	err := analyzeSource(t, `break;`)
	requireSemanticError(t, err, "break outside loop or switch")
}

func TestSemanticRejectsContinueOutsideLoop(t *testing.T) {
	err := analyzeSource(t, `continue;`)
	requireSemanticError(t, err, "continue outside loop")
}

func TestSemanticAllowsBreakAndContinueInsideLoop(t *testing.T) {
	err := analyzeSource(t, `
		while (true) {
			continue;
			break;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsBreakInsideSwitch(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		switch (value) {
		case 1:
			break;
		default:
			break;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsContinueInsideSwitchWithoutLoop(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		switch (value) {
		case 1:
			continue;
		}
	`)
	requireSemanticError(t, err, "continue outside loop")
}

func TestSemanticRejectsDuplicateSwitchCaseDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		switch (value) {
		case 1:
			const name = "first";
			break;
		default:
			const name = "fallback";
			break;
		}
	`)
	requireSemanticError(t, err, "duplicate declaration name")
}

func TestSemanticAllowsDuplicateSwitchDeclarationInsideNestedBlocks(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		switch (value) {
		case 1:
			{
				const name = "first";
				name;
			}
			break;
		default:
			{
				const name = "fallback";
				name;
			}
			break;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
