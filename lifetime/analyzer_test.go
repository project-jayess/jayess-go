package lifetime

import (
	"testing"

	"jayess-go/lexer"
	"jayess-go/parser"
)

func analyzeLifetimeSource(t *testing.T, source string) Report {
	t.Helper()

	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	return New().Analyze(program)
}

func hasEligibleLocal(report Report, name string) bool {
	for _, item := range report.Eligible {
		if item.Name == name {
			return true
		}
	}
	return false
}

func findEligibleLocal(report Report, name string) (LocalClassification, bool) {
	for _, item := range report.Eligible {
		if item.Name == name {
			return item, true
		}
	}
	return LocalClassification{}, false
}

func TestAnalyzeClassifiesNonEscapingLocalsAsEligible(t *testing.T) {
	report := analyzeLifetimeSource(t, `
function main(args) {
  var count = 1;
  var box = { total: count + 1 };
  print(box.total);
  return 0;
}
`)

	for _, name := range []string{"count", "box"} {
		if !hasEligibleLocal(report, name) {
			t.Fatalf("expected %q to be classified as eligible, got %#v", name, report.Eligible)
		}
	}
}

func TestAnalyzeExcludesEscapingLocalsFromEligibleClassification(t *testing.T) {
	report := analyzeLifetimeSource(t, `
extern function retain(value);
var sink = null;

function main(args) {
  var returned = 1;
  var stored = 3;
  var external = 4;
  sink = { value: stored };
  retain(external);
  return returned;
}
`)

	for _, name := range []string{"returned", "stored", "external"} {
		if hasEligibleLocal(report, name) {
			t.Fatalf("expected %q to be excluded from eligible locals, got %#v", name, report.Eligible)
		}
	}
}

func TestAnalyzeMarksLoopDeclaredEligibleLocals(t *testing.T) {
	report := analyzeLifetimeSource(t, `
function main(args) {
  var scoped = 1;
  for (var i = 0; i < 2; i = i + 1) {
    print(i);
  }
  return 0;
}
`)

	scoped, ok := findEligibleLocal(report, "scoped")
	if !ok {
		t.Fatalf("expected scoped to be classified as eligible, got %#v", report.Eligible)
	}
	if scoped.InLoop {
		t.Fatalf("expected scoped to be marked outside loops, got %#v", scoped)
	}
	if scoped.Kind != "var" {
		t.Fatalf("expected scoped kind to be var, got %#v", scoped)
	}

	loopVar, ok := findEligibleLocal(report, "i")
	if !ok {
		t.Fatalf("expected i to be classified as eligible, got %#v", report.Eligible)
	}
	if !loopVar.InLoop {
		t.Fatalf("expected i to be marked as loop-declared, got %#v", loopVar)
	}
	if loopVar.Kind != "var" {
		t.Fatalf("expected i kind to be var, got %#v", loopVar)
	}
}

func TestAnalyzeClassifiesNonEscapingParametersAsEligible(t *testing.T) {
	program, err := parser.New(lexer.New(`
extern function retain(value);
extern function borrow(value);

function keep(label) {
  borrow(label);
  return 0;
}

function escape(label) {
  retain(label);
  return 0;
}
`)).ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	for _, ext := range program.ExternFunctions {
		if ext.Name == "borrow" {
			ext.BorrowsArgs = true
		}
	}
	report := New().Analyze(program)

	foundKeep := false
	foundEscape := false
	for _, item := range report.EligibleParams {
		if item.Function == "keep" && item.Name == "label" {
			foundKeep = true
		}
		if item.Function == "escape" && item.Name == "label" {
			foundEscape = true
		}
	}
	if !foundKeep {
		t.Fatalf("expected keep(label) to be eligible, got %#v", report.EligibleParams)
	}
	if foundEscape {
		t.Fatalf("expected escape(label) to be excluded, got %#v", report.EligibleParams)
	}
}
