package semantic

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/parser"
)

func TestAnalyzeAcceptsTypedProgram(t *testing.T) {
	source := `
function add(left: number, right: number): number {
  return left + right;
}

function main(args) {
  var total: number = add(1, 2);
  return 0;
}
`

	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	if err := New().Analyze(program); err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestAnalyzeRejectsTypeMismatch(t *testing.T) {
	source := `
function main(args) {
  var total: number = "kimchi";
  return 0;
}
`

	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("ParseProgram returned error: %v", err)
	}
	err = New().Analyze(program)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "cannot initialize number variable total with string") {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}
