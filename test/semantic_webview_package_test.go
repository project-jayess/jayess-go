package test

import "testing"

func TestSemanticAllowsJayessWebviewPackageImport(t *testing.T) {
	err := analyzeSource(t, `
		import { createWindow, onDrop } from "@jayess/webview";
		function main() {
			return createWindow || onDrop;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
