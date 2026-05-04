package test

import "testing"

func TestSemanticAnalyzesDateAndRegExpUsage(t *testing.T) {
	err := analyzeSource(t, `
		const today = new Date();
		const time = today.getTime();
		const matcher = new RegExp("a+");
		const ok = matcher.test("aaa");
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesSymbolUsage(t *testing.T) {
	err := analyzeSource(t, `
		const key = Symbol("key");
		const item = {};
		item[key] = "value";
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesBinaryDataUsage(t *testing.T) {
	err := analyzeSource(t, `
		const buffer = new ArrayBuffer(8);
		const view = new DataView(buffer);
		view.setInt8(0, 1);
		const bytes = new Uint8Array(buffer);
		const first = bytes[0];
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesObjectConstructorAndStaticHelpers(t *testing.T) {
	err := analyzeSource(t, `
		const base = { ready: true };
		const item = new Object();
		const child = Object.create(base);
		const names = Object.keys(child);
		item.names = names;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownBinaryDataOperand(t *testing.T) {
	err := analyzeSource(t, `
		const bytes = new Uint8Array(missing);
	`)
	requireSemanticError(t, err, "use of missing before declaration")
}
