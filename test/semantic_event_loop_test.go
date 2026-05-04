package test

import "testing"

func TestSemanticEventLoopAllowsScheduledCallbacks(t *testing.T) {
	err := analyzeSource(t, `
		const timeout = setTimeout(() => print("once"), 10);
		const interval = setInterval(() => print("again"), 20);
		queueMicrotask(() => print("soon"));
		clearTimeout(timeout);
		clearInterval(interval);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticEventLoopRequiresScheduledCallbacks(t *testing.T) {
	err := analyzeSource(t, `queueMicrotask();`)
	requireSemanticError(t, err, "queueMicrotask requires callback")

	err = analyzeSource(t, `setTimeout();`)
	requireSemanticError(t, err, "setTimeout requires callback")

	err = analyzeSource(t, `setInterval();`)
	requireSemanticError(t, err, "setInterval requires callback")
}

func TestSemanticEventLoopRequiresCancellationHandles(t *testing.T) {
	err := analyzeSource(t, `clearTimeout();`)
	requireSemanticError(t, err, "clearTimeout requires scheduled handle")

	err = analyzeSource(t, `clearInterval();`)
	requireSemanticError(t, err, "clearInterval requires scheduled handle")
}
