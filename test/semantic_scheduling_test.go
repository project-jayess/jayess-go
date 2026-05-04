package test

import "testing"

func TestSemanticAnalyzesMicrotaskCallbackShape(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		queueMicrotask(() => {
			print(value);
		});
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesTimerQueueShapes(t *testing.T) {
	err := analyzeSource(t, `
		const timeout = setTimeout(() => print("once"), 10);
		const interval = setInterval(() => print("again"), 20);
		clearTimeout(timeout);
		clearInterval(interval);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownScheduledCallbackValue(t *testing.T) {
	err := analyzeSource(t, `
		setTimeout(() => print(missing), 10);
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
