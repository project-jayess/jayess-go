package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeBufferCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"create",
		"fromString",
		"toString",
		"slice",
		"copy",
		"readUInt16LE",
		"writeUInt16LE",
		"typedArrayView",
		"createReadStream",
		"createWriteStream",
	}
	for _, name := range expected {
		if !jayessruntime.HasBufferCapability(name) {
			t.Fatalf("expected buffer runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsBufferBinarySurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(text) {
			const raw = Buffer.create(16);
			const encoded = Buffer.fromString(text, "utf8");
			const decoded = Buffer.toString(encoded, "utf8");
			const part = Buffer.slice(encoded, 0, 4);
			Buffer.copy(part, raw, 0);
			const value = Buffer.readUInt16LE(raw, 0);
			Buffer.writeUInt16LE(raw, value, 2);
			const view = Buffer.typedArrayView(raw, "Uint8Array");
			const input = Buffer.createReadStream(raw);
			const output = Buffer.createWriteStream(raw);
			return decoded || view || input || output;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeBufferCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.BufferCapabilities() {
		if capability.Name == "" {
			t.Fatalf("buffer capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("buffer capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("buffer capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelBufferConstructorRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var Buffer = {};`)
	requireSemanticError(t, err, "duplicate declaration Buffer")
}
