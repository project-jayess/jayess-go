package test

import "testing"

func TestSemanticAllowsBinaryDataConstructors(t *testing.T) {
	err := analyzeSource(t, `
		const buffer = new ArrayBuffer(8);
		const view = new DataView(buffer);
		const bytes = new Uint8Array(buffer);
		const ints = new Int32Array(buffer);
		const floats = new Float64Array(buffer);
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsAllTypedArrayConstructors(t *testing.T) {
	err := analyzeSource(t, `
		const int8 = new Int8Array();
		const int16 = new Int16Array();
		const int32 = new Int32Array();
		const uint8 = new Uint8Array();
		const clamped = new Uint8ClampedArray();
		const uint16 = new Uint16Array();
		const uint32 = new Uint32Array();
		const float32 = new Float32Array();
		const float64 = new Float64Array();
		const bigInt64 = new BigInt64Array();
		const bigUint64 = new BigUint64Array();
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsSymbolGlobal(t *testing.T) {
	err := analyzeSource(t, `const key = Symbol("key");`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsNewSymbol(t *testing.T) {
	err := analyzeSource(t, `const key = new Symbol("key");`)
	requireSemanticError(t, err, "new target Symbol is not constructable")
}

func TestSemanticRejectsBinaryDataBuiltinRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var ArrayBuffer = {};`)
	requireSemanticError(t, err, "duplicate declaration ArrayBuffer")
}
