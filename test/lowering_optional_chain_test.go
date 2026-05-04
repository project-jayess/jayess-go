package test

import (
	"testing"

	"jayess-go/lowering"
)

func expectOptionalChainReturnCode(t *testing.T, source string, want int, context string) {
	t.Helper()
	program := parseProgram(t, source)
	value, ok := lowering.MainReturnCode(program)
	if !ok {
		t.Fatal("expected main return code")
	}
	if value != want {
		t.Fatalf("expected %s return code %d, got %d", context, want, value)
	}
}

func TestLoweringUsesOptionalObjectMemberOnKnownObject(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { if (({ code: 231 })?.code === 231) { return 231; } return 1; }`, 231, "optional object member")
}

func TestLoweringUsesOptionalArrayIndexOnKnownArray(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { if ([230, 232]?.[1] === 232) { return 232; } return 1; }`, 232, "optional array index")
}

func TestLoweringUsesOptionalStringLengthOnKnownString(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { if ("abc"?.length === 3) { return 233; } return 1; }`, 233, "optional string length")
}

func TestLoweringUsesOptionalMemberNullishShortCircuit(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { if (null?.code === undefined) { return 234; } return 1; }`, 234, "optional member nullish short-circuit")
}

func TestLoweringUsesOptionalIndexNullishShortCircuit(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { var index = 12n; if (undefined?.[index++] === undefined && index === 12n) { return 235; } return 1; }`, 235, "optional index nullish short-circuit")
}

func TestLoweringKeepsOptionalChainTargetSideEffects(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { var value = 12n; if ((value++, null)?.code === undefined && value === 13n) { return 236; } return 1; }`, 236, "optional chain target side-effect")
}

func TestLoweringUsesOptionalCallNullishShortCircuit(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { var value = 12n; if (undefined?.(value++) === undefined && value === 12n) { return 237; } return 1; }`, 237, "optional call nullish short-circuit")
}

func TestLoweringUsesOptionalEmptyFunctionCall(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { var value = 12n; if ((function () {})?.(value++) === undefined && value === 13n) { return 238; } return 1; }`, 238, "optional empty function call")
}

func TestLoweringUsesEmptyFunctionCallAsUndefined(t *testing.T) {
	expectOptionalChainReturnCode(t, `function main() { var value = 12n; if ((function () {})(value++) === undefined && value === 13n) { return 239; } return 1; }`, 239, "empty function call")
}
