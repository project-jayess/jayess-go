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

func TestLoweringUsesBlockIIFEPrefixVariableBeforeReturn(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { { var value = 298; } return value + 9; })(); }`, 307, "block IIFE prefix variable")
}

func TestLoweringClearsBlockIIFEPrefixLocalsAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { (function () { { var local = 1; } return local; })(); if (typeof local === "undefined") { return 308; } return 1; }`, 308, "block IIFE prefix local cleanup")
}

func TestLoweringRestoresBlockIIFEPrefixLocalShadowAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { var value = 7; var result = (function () { { var value = 300; } return value + 2; })(); return result + value; }`, 309, "block IIFE prefix local shadow cleanup")
}

func TestLoweringHoistsUntakenIfIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { if (false) { var value = 1; } return value; })() === undefined) { return 310; } return 1; }`, 310, "untaken if IIFE prefix variable hoist")
}

func TestLoweringUsesTakenIfIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { return (function () { if (true) { var value = 304; } return value + 7; })(); }`, 311, "taken if IIFE prefix variable")
}

func TestLoweringClearsIfIIFEPrefixLocalsAfterCall(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { (function () { if (true) { var local = 1; } return local; })(); if (typeof local === "undefined") { return 312; } return 1; }`, 312, "if IIFE prefix local cleanup")
}

func TestLoweringHoistsLoopIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { while (false) { var value = 1; } return value; })() === undefined) { return 313; } return 1; }`, 313, "loop IIFE prefix variable hoist")
}

func TestLoweringHoistsForOfIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { for (var value of []) {} return value; })() === undefined) { return 319; } return 1; }`, 319, "for-of IIFE prefix variable hoist")
}

func TestLoweringHoistsForInIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { for (var value in {}) {} return value; })() === undefined) { return 320; } return 1; }`, 320, "for-in IIFE prefix variable hoist")
}

func TestLoweringHoistsForOfDestructuredIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { for (var [value] of []) {} return value; })() === undefined) { return 321; } return 1; }`, 321, "for-of destructured IIFE prefix variable hoist")
}

func TestLoweringHoistsForInDestructuredIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { for (var {value} in {}) {} return value; })() === undefined) { return 322; } return 1; }`, 322, "for-in destructured IIFE prefix variable hoist")
}

func TestLoweringHoistsSwitchIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { switch (1) { case 2: var value = 1; } return value; })() === undefined) { return 314; } return 1; }`, 314, "switch IIFE prefix variable hoist")
}

func TestLoweringHoistsTryIIFEPrefixVariable(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { try { throw 1; var value = 1; } catch (err) {} return value; })() === undefined) { return 315; } return 1; }`, 315, "try IIFE prefix variable hoist")
}

func TestLoweringUsesIIFEPrefixFunctionDeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { function helper() { return 1; } return typeof helper; })() === "function") { return 263; } return 1; }`, 263, "IIFE prefix function declaration")
}

func TestLoweringHoistsLoopIIFEPrefixFunctionDeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { while (false) { function helper() { return 1; } } return typeof helper; })() === "function") { return 316; } return 1; }`, 316, "loop IIFE prefix function hoist")
}

func TestLoweringHoistsSwitchIIFEPrefixFunctionDeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { switch (1) { case 2: function helper() { return 1; } } return typeof helper; })() === "function") { return 317; } return 1; }`, 317, "switch IIFE prefix function hoist")
}

func TestLoweringHoistsTryIIFEPrefixFunctionDeclaration(t *testing.T) {
	expectIIFEReturnCode(t, `function main() { if ((function () { try { throw 1; function helper() { return 1; } } catch (err) {} return typeof helper; })() === "function") { return 318; } return 1; }`, 318, "try IIFE prefix function hoist")
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
