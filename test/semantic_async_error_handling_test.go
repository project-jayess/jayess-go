package test

import "testing"

func TestSemanticAnalyzesAwaitInAsyncTryCatchFinally(t *testing.T) {
	err := analyzeSource(t, `
		async function load(read, recover, cleanup) {
			try {
				return await read();
			} catch (error) {
				throw await recover(error);
			} finally {
				await cleanup();
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownAsyncCatchRecovery(t *testing.T) {
	err := analyzeSource(t, `
		async function load() {
			try {
				return 1;
			} catch (error) {
				return await recover(error);
			}
		}
	`)
	requireSemanticError(t, err, "use of recover before declaration")
}
