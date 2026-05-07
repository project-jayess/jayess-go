package test

import (
	"bytes"
	"strings"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeStreamCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"readable",
		"writable",
		"duplex",
		"transform",
		"pipe",
		"awaitDrain",
	}
	for _, name := range expected {
		if !jayessruntime.HasStreamCapability(name) {
			t.Fatalf("expected stream runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsStreamSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(source, sink) {
			const input = stream.readable(source);
			const output = stream.writable(sink);
			const pair = stream.duplex(input, output);
			const upper = stream.transform((chunk) => chunk);
			stream.pipe(input, upper);
			stream.pipe(upper, output);
			const ready = stream.awaitDrain(output);
			return pair || ready;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeStreamCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.StreamCapabilities() {
		if capability.Name == "" {
			t.Fatalf("stream capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("stream capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("stream capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestRuntimeStreamPrimitivesAndBackpressure(t *testing.T) {
	source := jayessruntime.StreamReadable([]byte("hello"))
	transformed, err := jayessruntime.StreamTransform(source, func(data []byte) ([]byte, error) {
		return []byte(strings.ToUpper(string(data))), nil
	})
	if err != nil {
		t.Fatalf("transform stream: %v", err)
	}
	output, target := jayessruntime.StreamWritable()
	written, err := jayessruntime.StreamPipe(transformed, output)
	if err != nil {
		t.Fatalf("pipe stream: %v", err)
	}
	if written != 5 || target.String() != "HELLO" {
		t.Fatalf("unexpected stream pipe written=%d target=%q", written, target.String())
	}
	if !jayessruntime.StreamAwaitDrain(jayessruntime.StreamState{HighWaterMark: 8, Buffered: 4}) {
		t.Fatal("expected stream to be below high water mark")
	}
	if jayessruntime.StreamAwaitDrain(jayessruntime.StreamState{HighWaterMark: 4, Buffered: 8}) {
		t.Fatal("expected stream backpressure")
	}
}

func TestRuntimeStreamDuplex(t *testing.T) {
	duplex, sink := jayessruntime.StreamDuplex([]byte("in"))
	data, err := jayessruntime.StreamReadAll(duplex)
	if err != nil || !bytes.Equal(data, []byte("in")) {
		t.Fatalf("duplex read got data=%q err=%v", data, err)
	}
	if _, err := duplex.WriteString("out"); err != nil {
		t.Fatalf("duplex write: %v", err)
	}
	if sink.String() != "out" {
		t.Fatalf("unexpected duplex sink %q", sink.String())
	}
}

func TestSemanticRejectsTopLevelStreamRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var stream = {};`)
	requireSemanticError(t, err, "duplicate declaration stream")
}
