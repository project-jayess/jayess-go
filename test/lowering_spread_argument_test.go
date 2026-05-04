package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringDiscardsEmptyCallSpreadArrayArguments(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; (function () {})(...[code++, code++]); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected empty call spread arguments return code 31, got %d", value)
	}
}

func TestLoweringDiscardsEmptyNewSpreadArrayArguments(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; new (function () {})(...[code++, code++]); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 31 {
		t.Fatalf("expected empty new spread arguments return code 31, got %d", value)
	}
}

func TestLoweringDiscardsMixedSpreadArgumentsInOrder(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; (function () {})(code++, ...[code++], code++); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 41 {
		t.Fatalf("expected mixed spread arguments return code 41, got %d", value)
	}
}

func TestLoweringKeepsUnknownSpreadArgumentUnresolved(t *testing.T) {
	program := parseProgram(t, `function main() { var code = 1; (function () {})(...items); return code * 10 + 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 11 {
		t.Fatalf("expected unknown spread argument return code 11, got %d", value)
	}
}
