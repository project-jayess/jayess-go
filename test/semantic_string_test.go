package test

import "testing"

func TestSemanticAnalyzesStringConcatenation(t *testing.T) {
	err := analyzeSource(t, `
		const name = "Jayess";
		const greeting = "hello, " + name + "!";
		print(greeting);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesStringConcatenationOperands(t *testing.T) {
	err := analyzeSource(t, `const greeting = "hello, " + missing;`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticAnalyzesStringIndexing(t *testing.T) {
	err := analyzeSource(t, `
		const greeting = "hello";
		const first = greeting[0];
		print(first);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesStringIndexOperand(t *testing.T) {
	err := analyzeSource(t, `const first = "hello"[missing];`)
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticAnalyzesStringLengthMember(t *testing.T) {
	err := analyzeSource(t, `
		const greeting = "hello";
		const size = greeting.length;
		print(size);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAnalyzesTemplateStringInterpolation(t *testing.T) {
	err := analyzeSource(t, "\nconst name = \"Jayess\";\nconst greeting = `hello ${name}`;\nprint(greeting);\n")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsUnknownTemplateStringInterpolation(t *testing.T) {
	err := analyzeSource(t, "const greeting = `hello ${missing}`;")
	requireSemanticError(t, err, "use of missing before declaration")
}

func TestSemanticAnalyzesUnicodeStringAndIdentifier(t *testing.T) {
	err := analyzeSource(t, `
		const 名前 = "世界";
		const greeting = "こんにちは, " + 名前;
		print(greeting);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}
