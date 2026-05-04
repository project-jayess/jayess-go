package test

import "testing"

func TestSemanticAnalyzesDefaultExportExpression(t *testing.T) {
	err := analyzeSource(t, `
		const value = 1;
		export default value;
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownDefaultExportExpression(t *testing.T) {
	err := analyzeSource(t, `export default missing;`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticAnalyzesDefaultExportDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		export default function main() {
			return 0;
		}
		const value = main();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAnonymousDefaultExportFunction(t *testing.T) {
	err := analyzeSource(t, `
		export default function() {
			return 0;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesDefaultExportAsyncFunctionDeclaration(t *testing.T) {
	err := analyzeSource(t, `
		export default async function main() {
			await 1;
		}
		const value = main();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAnonymousDefaultExportAsyncFunction(t *testing.T) {
	err := analyzeSource(t, `
		const ready = 1;
		export default async function() {
			await ready;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesDefaultExportAsyncGeneratorFunction(t *testing.T) {
	err := analyzeSource(t, `
		async function next() {
			return 1;
		}
		export default async function* ids() {
			yield await next();
		}
		const value = ids();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAnonymousDefaultExportAsyncGeneratorFunction(t *testing.T) {
	err := analyzeSource(t, `
		async function next() {
			return 1;
		}
		export default async function*() {
			yield await next();
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesAnonymousDefaultExportClass(t *testing.T) {
	err := analyzeSource(t, `
		export default class {
			constructor(value) {
				this.value = value;
			}
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
