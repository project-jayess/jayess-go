package test

import "testing"

func TestLoweringUsesFunctionReturnIIFETypeof(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((typeof (function () { return function () {}; })()) === "function") { return 282; } return 1; }`, 282, "function-return IIFE typeof")
}

func TestLoweringMaterializesFunctionReturnIIFEVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var fn = (function () { return function () {}; })(); if (typeof fn === "function" && fn === fn) { return 283; } return 1; }`, 283, "function-return IIFE variable")
}

func TestLoweringUsesNamedFunctionIIFEReturnSelfIdentity(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var fn = (function helper() { return helper; })(); if (typeof fn === "function" && fn === fn) { return 284; } return 1; }`, 284, "named function IIFE return self identity")
}

func TestLoweringUsesObjectReturnIIFETypeof(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((typeof (function () { return {}; })()) === "object") { return 285; } return 1; }`, 285, "object-return IIFE typeof")
}

func TestLoweringMaterializesObjectReturnIIFEVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var object = (function () { return {}; })(); if (typeof object === "object" && object === object) { return 286; } return 1; }`, 286, "object-return IIFE variable")
}

func TestLoweringUsesArrayReturnIIFEIdentity(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var items = (function () { return []; })(); if (typeof items === "object" && items === items) { return 287; } return 1; }`, 287, "array-return IIFE identity")
}

func TestLoweringUsesObjectReturnIIFEMember(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { return { value: 280 }; })().value + 8; }`, 288, "object-return IIFE member")
}

func TestLoweringUsesObjectReturnIIFEComputedMember(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { return { value: 281 }; })()["value"] + 8; }`, 289, "object-return IIFE computed member")
}

func TestLoweringUsesArrayReturnIIFEIndex(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { return [290]; })()[0]; }`, 290, "array-return IIFE index")
}

func TestLoweringUsesArrayReturnIIFELength(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { return [1, 2, 3]; })().length + 288; }`, 291, "array-return IIFE length")
}

func TestLoweringUsesObjectReturnIIFEInOperator(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ("value" in (function () { return { value: 1 }; })()) { return 292; } return 1; }`, 292, "object-return IIFE in operator")
}

func TestLoweringUsesObjectReturnIIFEMissingInOperator(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if (!("missing" in (function () { return { value: 1 }; })())) { return 293; } return 1; }`, 293, "object-return IIFE missing in operator")
}

func TestLoweringUsesDeleteObjectReturnIIFEMember(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; if (delete (function () { value++; return { item: 1 }; })().item) { return value + 292; } return 1; }`, 294, "delete object-return IIFE member")
}

func TestLoweringUsesDeleteObjectReturnIIFEComputedMember(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; if (delete (function () { value++; return { item: 1 }; })()["item"]) { return value + 293; } return 1; }`, 295, "delete object-return IIFE computed member")
}

func TestLoweringUsesDeleteArrayReturnIIFEIndex(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; if (delete (function () { value++; return [1]; })()[0]) { return value + 294; } return 1; }`, 296, "delete array-return IIFE index")
}

func TestLoweringUsesOptionalObjectReturnIIFEMember(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return { value: 297 }; })()?.value === 297) { return 297; } return 1; }`, 297, "optional object-return IIFE member")
}

func TestLoweringUsesOptionalArrayReturnIIFEIndex(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return [298]; })()?.[0] === 298) { return 298; } return 1; }`, 298, "optional array-return IIFE index")
}

func TestLoweringUsesOptionalNullishReturnIIFEMemberShortCircuit(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return null; })()?.value === undefined) { return 299; } return 1; }`, 299, "optional nullish-return IIFE member")
}

func TestLoweringUsesOptionalNullishReturnIIFEIndexShortCircuit(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var index = 12n; if ((function () { return undefined; })()?.[index++] === undefined && index === 12n) { return 300; } return 1; }`, 300, "optional nullish-return IIFE index")
}

func TestLoweringUsesArraySpreadFromIIFEReturn(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return [1, ...(function () { return [2, 3]; })()].length + 298; }`, 301, "array spread from IIFE return")
}

func TestLoweringUsesObjectSpreadFromIIFEReturn(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return ({ ...(function () { return { value: 296 }; })() }).value + 6; }`, 302, "object spread from IIFE return")
}

func TestLoweringUsesSpreadFromIIFEReturnSideEffects(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; return [0, ...(function () { value++; return [300]; })()][1] + value; }`, 302, "spread from IIFE return side effects")
}

func TestLoweringUsesObjectReturnIIFEIdentityEquality(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var object = (function () { return {}; })(); if (object === object) { return 303; } return 1; }`, 303, "object-return IIFE identity equality")
}

func TestLoweringUsesArrayReturnIIFEIdentityEquality(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var items = (function () { return []; })(); if (items === items) { return 304; } return 1; }`, 304, "array-return IIFE identity equality")
}

func TestLoweringUsesFreshObjectReturnIIFEInequality(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return {}; })() !== (function () { return {}; })()) { return 305; } return 1; }`, 305, "fresh object-return IIFE inequality")
}

func TestLoweringUsesFreshArrayReturnIIFEInequality(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return []; })() !== (function () { return []; })()) { return 306; } return 1; }`, 306, "fresh array-return IIFE inequality")
}
