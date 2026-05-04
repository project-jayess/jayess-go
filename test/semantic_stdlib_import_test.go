package test

import "testing"

func TestSemanticAllowsEditorFriendlyStdlibImports(t *testing.T) {
	err := analyzeSource(t, `
		import { readFile } from "fs";
		import { createModule } from "llvm";
		import * as path from "path";
		import { inspect } from "util";

		const filename = path.join("tmp", "input.txt");
		const text = readFile(filename);
		const mod = createModule("main");
		inspect(text);
		inspect(mod);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
