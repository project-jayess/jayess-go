package test

import "testing"

func TestSemanticAllowsNewExpressionForDeclaredConstructor(t *testing.T) {
	err := analyzeSource(t, `
		function Widget(value) {
			return value;
		}
		const value = 1;
		const instance = new Widget(value);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesNewExpressionSpreadArguments(t *testing.T) {
	err := analyzeSource(t, `
		function Widget(first, second) {
			return first;
		}
		const rest = [2];
		const instance = new Widget(1, ...rest);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsNewExpressionBeforeConstructorDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		const instance = new Widget();
		function Widget() {
			return 1;
		}
	`)
	requireSemanticError(t, err, "use of Widget before declaration")
}

func TestSemanticAnalyzesNewExpressionArguments(t *testing.T) {
	err := analyzeSource(t, `
		function Widget(value) {
			return value;
		}
		const instance = new Widget(missing);
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticRejectsUnknownNewExpressionSpreadArgument(t *testing.T) {
	err := analyzeSource(t, `
		function Widget(value) {
			return value;
		}
		const instance = new Widget(...missing);
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticAllowsNewExpressionForClass(t *testing.T) {
	err := analyzeSource(t, `
		class Widget {
			constructor(value) {
				this.value = value;
			}
		}
		const instance = new Widget(1);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsNewExpressionForVariable(t *testing.T) {
	err := analyzeSource(t, `
		const Widget = 1;
		const instance = new Widget();
	`)
	requireSemanticError(t, err, "new target Widget is not constructable")
}

func TestSemanticAllowsNewExpressionForImportedName(t *testing.T) {
	err := analyzeSource(t, `
		import { Widget } from "./widget.js";
		const instance = new Widget();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsNewTargetInFunction(t *testing.T) {
	err := analyzeSource(t, `
		function Widget() {
			return new.target;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsNewTargetOutsideFunction(t *testing.T) {
	err := analyzeSource(t, `const value = new.target;`)
	requireSemanticError(t, err, "new.target outside function")
}

func TestSemanticRejectsNewTargetInTopLevelArrow(t *testing.T) {
	err := analyzeSource(t, `const getTarget = () => new.target;`)
	requireSemanticError(t, err, "new.target outside function")
}
