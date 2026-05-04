package test

import (
	"testing"

	"jayess-go/lowering"
)

func expectIIFEReturnCode(t *testing.T, source string, want int, context string) {
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

func TestLoweringUsesFunctionIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { return 240; })(); }`, 240, "function IIFE")
}

func TestLoweringUsesArrowIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (() => 241)(); }`, 241, "arrow IIFE")
}

func TestLoweringUsesIIFEPrefixSideEffectsBeforeReturn(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; return (function () { value++; return value + 254; })(); }`, 256, "IIFE prefix side effect")
}

func TestLoweringUsesIIFEPrefixVariableBeforeReturn(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { var value = 253; return value + 4; })(); }`, 257, "IIFE prefix variable")
}

func TestLoweringUsesDuplicateIIFEPrefixVariableBeforeReturn(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { var value = 253; var value = 6; return value + 253; })(); }`, 259, "duplicate IIFE prefix variable")
}

func TestLoweringUsesUninitializedIIFEPrefixVariableAsUndefined(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { var value; return value; })() === undefined) { return 260; } return 1; }`, 260, "uninitialized IIFE prefix variable")
}

func TestLoweringClearsIIFEPrefixLocalsAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { (function () { var local = 1; return local; })(); if (typeof local === "undefined") { return 258; } return 1; }`, 258, "IIFE prefix local cleanup")
}

func TestLoweringRestoresIIFEPrefixLocalShadowAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 7; var result = (function () { var value = 250; return value + 9; })(); return result + value; }`, 266, "IIFE prefix local shadow cleanup")
}

func TestLoweringRestoresIIFEPrefixLocalShadowAfterArgumentSideEffect(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 7; var result = (function () { var value = 250; return value; })(value++); return result + value; }`, 258, "IIFE prefix local shadow after argument")
}

func TestLoweringRestoresDuplicateIIFEPrefixLocalShadowAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 7; var result = (function () { var value = 250; var value = 252; return value; })(); return result + value; }`, 259, "duplicate IIFE prefix local shadow cleanup")
}

func TestLoweringUsesUninitializedIIFEPrefixShadowAsUndefined(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 7; if ((function () { var value; return value; })() === undefined) { return value + 253; } return 1; }`, 260, "uninitialized IIFE prefix shadow")
}

func TestLoweringUsesIIFEPrefixFunctionDeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { function helper() { return 1; } return typeof helper; })() === "function") { return 263; } return 1; }`, 263, "IIFE prefix function declaration")
}

func TestLoweringClearsIIFEPrefixFunctionDeclarationAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { function helper() { return 1; } return typeof helper; })() === "function" && typeof helper === "undefined") { return 264; } return 1; }`, 264, "IIFE prefix function cleanup")
}

func TestLoweringRestoresIIFEPrefixFunctionDeclarationShadowAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var helper = 7; var result = (function () { function helper() { return 1; } return typeof helper === "function" ? 250 : 1; })(); return result + helper; }`, 257, "IIFE prefix function shadow cleanup")
}

func TestLoweringUsesIIFEPrefixFunctionVarRedeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { function helper() { return 1; } var helper; return typeof helper; })() === "function") { return 265; } return 1; }`, 265, "IIFE prefix function var redeclaration")
}

func TestLoweringUsesIIFEPrefixFunctionVarRedeclarationAssignment(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { function helper() { return 1; } var helper = 266; return helper; })(); }`, 266, "IIFE prefix function var redeclaration assignment")
}

func TestLoweringRestoresIIFEPrefixFunctionVarShadowAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var helper = 7; var result = (function () { function helper() { return 1; } var helper = 252; return helper; })(); return result + helper; }`, 259, "IIFE prefix function var shadow cleanup")
}

func TestLoweringUsesNamedFunctionIIFESelfBinding(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function helper() { return typeof helper; })() === "function") { return 273; } return 1; }`, 273, "named function IIFE self binding")
}

func TestLoweringClearsNamedFunctionIIFESelfBindingAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function helper() { return typeof helper; })() === "function" && typeof helper === "undefined") { return 274; } return 1; }`, 274, "named function IIFE self cleanup")
}

func TestLoweringRestoresNamedFunctionIIFESelfBindingShadowAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var helper = 7; var result = (function helper() { return typeof helper === "function" ? 268 : 1; })(); return result + helper; }`, 275, "named function IIFE self shadow cleanup")
}

func TestLoweringUsesNamedFunctionIIFEParameterShadow(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function helper(helper) { return helper + 268; })(8); }`, 276, "named function IIFE parameter shadow")
}

func TestLoweringUsesNamedFunctionIIFESelfInParameterDefault(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function helper(value = typeof helper) { return value; })() === "function") { return 277; } return 1; }`, 277, "named function IIFE default self binding")
}

func TestLoweringUsesNamedFunctionIIFEVarRedeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function helper() { var helper; return typeof helper; })() === "function") { return 278; } return 1; }`, 278, "named function IIFE var redeclaration")
}

func TestLoweringUsesNamedFunctionIIFEVarRedeclarationAssignment(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function helper() { var helper = 279; return helper; })(); }`, 279, "named function IIFE var redeclaration assignment")
}

func TestLoweringUsesNamedFunctionIIFESelfIdentityEquality(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function helper() { return helper === helper; })()) { return 280; } return 1; }`, 280, "named function IIFE self identity equality")
}

func TestLoweringUsesNamedFunctionIIFEPrefixFunctionRedeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function helper() { function helper() { return 1; } return typeof helper; })() === "function") { return 281; } return 1; }`, 281, "named function IIFE prefix function redeclaration")
}

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

func TestLoweringUsesIIFEArgumentSideEffects(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 12n; if ((function () { return 242; })(value++) === 242 && value === 13n) { return 242; } return 1; }`, 242, "IIFE argument side-effect")
}

func TestLoweringUsesIIFECalleeBeforeArgumentSideEffects(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; return (value++, function () { return value + 246; })(value === 2 ? value++ : value--); }`, 249, "IIFE callee-before-argument side-effects")
}

func TestLoweringUsesConditionalIIFECallee(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (true ? function () { return 243; } : function () { return 1; })(); }`, 243, "conditional IIFE callee")
}

func TestLoweringUsesStringIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return "ok"; })() === "ok") { return 244; } return 1; }`, 244, "string IIFE")
}

func TestLoweringUsesBigIntIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return 12n; })() === 12n) { return 245; } return 1; }`, 245, "BigInt IIFE")
}

func TestLoweringUsesBoolIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return true; })()) { return 246; } return 1; }`, 246, "bool IIFE")
}

func TestLoweringUsesNullishIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return null; })() === null) { return 247; } return 1; }`, 247, "nullish IIFE")
}

func TestLoweringUsesUndefinedArrowIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((() => undefined)() === undefined) { return 248; } return 1; }`, 248, "undefined arrow IIFE")
}

func TestLoweringUsesEmptyBlockIIFEAsUndefined(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () {})() === undefined) { return 267; } return 1; }`, 267, "empty block IIFE undefined")
}

func TestLoweringUsesBareReturnIIFEAsUndefined(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { return; })() === undefined) { return 268; } return 1; }`, 268, "bare return IIFE undefined")
}

func TestLoweringUsesFallthroughIIFEPrefixSideEffects(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; if ((function () { var local = 3; value += local; })() === undefined && typeof local === "undefined") { return value + 265; } return 1; }`, 269, "fallthrough IIFE prefix side effects")
}

func TestLoweringUsesArrowBlockIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (() => { return 270; })(); }`, 270, "arrow block IIFE")
}

func TestLoweringUsesArrowBareReturnIIFEAsUndefined(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((() => { return; })() === undefined) { return 271; } return 1; }`, 271, "arrow bare return IIFE undefined")
}

func TestLoweringUsesArrowFallthroughIIFEPrefixSideEffects(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; if ((() => { var local = 4; value += local; })() === undefined && typeof local === "undefined") { return value + 267; } return 1; }`, 272, "arrow fallthrough IIFE prefix side effects")
}

func TestLoweringUsesParameterizedIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function (value) { return value + 249; })(1); }`, 250, "parameterized IIFE")
}

func TestLoweringUsesParameterizedArrowIIFEReturnValue(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if (((value) => value)(12n) === 12n) { return 251; } return 1; }`, 251, "parameterized arrow IIFE")
}

func TestLoweringUsesParameterizedIIFEArgumentOrder(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; return (function (first, second) { return first * 100 + second * 10 + value; })(value++, value++); }`, 123, "parameterized IIFE argument order")
}

func TestLoweringUsesIIFEParameterVarRedeclarationBeforeReturn(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function (value) { var value; return value + 259; })(2); }`, 261, "IIFE parameter var redeclaration")
}

func TestLoweringUsesIIFEParameterVarRedeclarationAssignment(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function (value) { var value = 253; return value + 9; })(1); }`, 262, "IIFE parameter var redeclaration assignment")
}

func TestLoweringUsesMissingIIFEParameterAsUndefined(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function (missing) { return missing; })() === undefined) { return 253; } return 1; }`, 253, "missing IIFE parameter")
}

func TestLoweringUsesIIFEParameterDefaultForMissingArgument(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function (value = 254) { return value; })(); }`, 254, "missing IIFE parameter default")
}

func TestLoweringUsesIIFEParameterDefaultForUndefinedArgument(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; return (function (arg = 40) { return arg + value; })(void value++); }`, 42, "undefined IIFE parameter default")
}

func TestLoweringUsesEarlierIIFEParameterInDefault(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function (first, second = first + 3) { return second; })(252); }`, 255, "IIFE parameter default scope")
}

func TestLoweringUsesExtraIIFEArgumentSideEffectsBeforeDefault(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 1; return (function (arg = value + 40) { return arg; })(undefined, value++); }`, 42, "extra IIFE argument before default")
}

func TestLoweringClearsIIFEParametersAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { (function (temporary) { return temporary; })(12n); if (typeof temporary === "undefined") { return 252; } return 1; }`, 252, "IIFE parameter cleanup")
}

func TestLoweringClearsMissingIIFEParameterAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { (function (missing) { return missing; })(); if (typeof missing === "undefined") { return 254; } return 1; }`, 254, "missing IIFE parameter cleanup")
}
