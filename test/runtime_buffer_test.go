package test

import (
	"bytes"
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

func TestRuntimeBufferBinaryHelpers(t *testing.T) {
	raw := jayessruntime.BufferCreate(8)
	encoded, err := jayessruntime.BufferFromString("hello", "utf8")
	if err != nil {
		t.Fatalf("from string: %v", err)
	}
	text, err := jayessruntime.BufferToString(encoded, "utf8")
	if err != nil || text != "hello" {
		t.Fatalf("to string got text=%q err=%v", text, err)
	}
	part := jayessruntime.BufferSlice(encoded, 1, 4)
	if string(part.Data) != "ell" {
		t.Fatalf("unexpected slice %q", part.Data)
	}
	if copied := jayessruntime.BufferCopy(part, raw, 2); copied != 3 {
		t.Fatalf("expected 3 copied bytes, got %d", copied)
	}
	if err := jayessruntime.BufferWriteUInt16LE(raw, 0x1234, 0); err != nil {
		t.Fatalf("write uint16: %v", err)
	}
	value, err := jayessruntime.BufferReadUInt16LE(raw, 0)
	if err != nil || value != 0x1234 {
		t.Fatalf("read uint16 got value=%#x err=%v", value, err)
	}
	view, err := jayessruntime.BufferTypedArrayView(raw, "Uint8Array")
	if err != nil || !bytes.Equal(view, raw.Data) {
		t.Fatalf("typed view mismatch view=%#v err=%v", view, err)
	}
}

func TestRuntimeBufferStreams(t *testing.T) {
	buffer, err := jayessruntime.BufferFromString("in", "utf8")
	if err != nil {
		t.Fatalf("from string: %v", err)
	}
	input := jayessruntime.BufferCreateReadStream(buffer)
	data, err := input.ReadAll()
	if err != nil || string(data) != "in" {
		t.Fatalf("read stream got data=%q err=%v", data, err)
	}
	output := jayessruntime.BufferCreateWriteStream(buffer)
	if _, err := output.WriteString("-out"); err != nil {
		t.Fatalf("write stream: %v", err)
	}
	if string(buffer.Data) != "in-out" {
		t.Fatalf("unexpected buffer after write stream %q", buffer.Data)
	}
}

func TestSemanticRejectsTopLevelBufferConstructorRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var Buffer = {};`)
	requireSemanticError(t, err, "duplicate declaration Buffer")
}
