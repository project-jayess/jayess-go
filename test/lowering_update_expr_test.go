package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringExtractsMainReturnCodeAfterPostfixUpdate(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; code++; code++; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected folded postfix update return code 3, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeAfterPrefixUpdate(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 4; --code; return code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected folded prefix update return code 3, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromPostfixUpdateExpression(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; return code++ + code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 3 {
		t.Fatalf("expected postfix update expression return code 3, got %d", value)
	}
}

func TestLoweringExtractsMainReturnCodeFromPrefixUpdateExpression(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; return ++code + code; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 4 {
		t.Fatalf("expected prefix update expression return code 4, got %d", value)
	}
}
