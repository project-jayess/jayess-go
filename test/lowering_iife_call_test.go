package test

import "testing"

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
