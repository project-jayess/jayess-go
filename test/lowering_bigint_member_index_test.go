package test

import (
	"testing"

	"jayess-go/lowering"
)

func TestLoweringUsesBigIntSwitchCase(t *testing.T) {
	program := parseProgram(t, `function main() { const kind = 12n; switch (kind) { case 11n: return 1; case 12n: return 205; default: return 2; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 205 {
		t.Fatalf("expected BigInt switch case return code 205, got %d", value)
	}
}

func TestLoweringUsesBigIntSwitchDefault(t *testing.T) {
	program := parseProgram(t, `function main() { const kind = 12n; switch (kind) { case 10n: return 1; case 11n: return 2; default: return 206; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 206 {
		t.Fatalf("expected BigInt switch default return code 206, got %d", value)
	}
}

func TestLoweringUsesBigIntSwitchDiscriminantSideEffect(t *testing.T) {
	program := parseProgram(t, `function main() { var kind = 12n; switch (kind++) { case 12n: if (kind === 13n) { return 207; } return 1; default: return 2; } }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 207 {
		t.Fatalf("expected BigInt switch side-effect return code 207, got %d", value)
	}
}

func TestLoweringUsesBigIntArrayIndex(t *testing.T) {
	program := parseProgram(t, `function main() { const values = [11n, 12n]; if (values[1] === 12n) { return 208; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 208 {
		t.Fatalf("expected BigInt array index return code 208, got %d", value)
	}
}

func TestLoweringUsesBigIntObjectMember(t *testing.T) {
	program := parseProgram(t, `function main() { const value = { code: 12n }; if (value.code === 12n) { return 209; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 209 {
		t.Fatalf("expected BigInt object member return code 209, got %d", value)
	}
}

func TestLoweringUsesBigIntObjectIndex(t *testing.T) {
	program := parseProgram(t, `function main() { const value = { code: 12n }; if (value["code"] === 12n) { return 210; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 210 {
		t.Fatalf("expected BigInt object index return code 210, got %d", value)
	}
}

func TestLoweringUsesBigIntElementTruthiness(t *testing.T) {
	program := parseProgram(t, `function main() { const values = [0n, 12n]; if (!values[0] && values[1]) { return 211; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 211 {
		t.Fatalf("expected BigInt element truthiness return code 211, got %d", value)
	}
}

func TestLoweringUsesBigIntPrimitiveInstanceof(t *testing.T) {
	program := parseProgram(t, `function main() { if (!(12n instanceof function Value() {})) { return 212; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 212 {
		t.Fatalf("expected BigInt primitive instanceof return code 212, got %d", value)
	}
}

func TestLoweringUsesBigIntInstanceofLeftSideEffects(t *testing.T) {
	program := parseProgram(t, `function main() { var value = 12n; if (!(value++ instanceof function Value() {}) && value === 13n) { return 213; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 213 {
		t.Fatalf("expected BigInt instanceof side-effect return code 213, got %d", value)
	}
}

func TestLoweringUsesBigIntComputedObjectPropertyKey(t *testing.T) {
	program := parseProgram(t, `function main() { const value = { [12n]: 216 }; if (value[12n] === 216) { return 216; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 216 {
		t.Fatalf("expected BigInt computed object key return code 216, got %d", value)
	}
}

func TestLoweringUsesBigIntInOperatorKey(t *testing.T) {
	program := parseProgram(t, `function main() { if (12n in ({ [12n]: "value" })) { return 217; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 217 {
		t.Fatalf("expected BigInt in-operator key return code 217, got %d", value)
	}
}

func TestLoweringEvaluatesBigIntComputedObjectPropertyKeyOnce(t *testing.T) {
	program := parseProgram(t, `function main() { var key = 12n; if (({ [key++]: 218 })[12n] === 218 && key === 13n) { return 218; } return 1; }`)

	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != 218 {
		t.Fatalf("expected BigInt computed key side-effect return code 218, got %d", value)
	}
}

func TestLoweringUsesBigIntArraySpreadIndex(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const values = [10n, ...[11n, 12n]]; if (values[2] === 12n) { return 220; } return 1; }`, 220, "BigInt array spread index")
}

func TestLoweringUsesBigIntObjectSpreadMember(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const value = { ...{ code: 12n } }; if (value.code === 12n) { return 221; } return 1; }`, 221, "BigInt object spread member")
}

func TestLoweringUsesBigIntSpreadSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; const values = [...[value++]]; if (values[0] === 12n && value === 13n) { return 222; } return 1; }`, 222, "BigInt spread side-effect")
}

func TestLoweringUsesBigIntVoidSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; const ignored = void value++; if (ignored === undefined && value === 13n) { return 223; } return 1; }`, 223, "BigInt void side-effect")
}

func TestLoweringUsesBigIntDeleteTargetSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var value = 12n; if (delete value++ && value === 13n) { return 224; } return 1; }`, 224, "BigInt delete target side-effect")
}

func TestLoweringUsesBigIntDeleteIndexKeySideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var key = 12n; if (delete ({ [12n]: "value" })[key++] && key === 13n) { return 225; } return 1; }`, 225, "BigInt delete index-key side-effect")
}

func TestLoweringUsesBigIntArrayIndexKey(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { const values = [10n, 11n]; if (values[1n] === 11n) { return 226; } return 1; }`, 226, "BigInt array index key")
}

func TestLoweringUsesBigIntArrayIndexSideEffects(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { var index = 1n; if ([10n, 11n][index++] === 11n && index === 2n) { return 227; } return 1; }`, 227, "BigInt array index side-effect")
}

func TestLoweringUsesBigIntStringIndexKey(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if ("ab"[1n] === "b") { return 228; } return 1; }`, 228, "BigInt string index key")
}

func TestLoweringUsesHugeBigIntArrayIndexAsUndefined(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if ([10n][999999999999999999999999999999n] === undefined) { return 229; } return 1; }`, 229, "huge BigInt array index")
}

func TestLoweringUsesHugeBigIntStringIndexAsUndefined(t *testing.T) {
	expectBigIntReturnCode(t, `function main() { if ("ab"[999999999999999999999999999999n] === undefined) { return 230; } return 1; }`, 230, "huge BigInt string index")
}
