package test

import (
	"bytes"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestIOStreamReadWriteAndPipe(t *testing.T) {
	source := jayessruntime.NewReadableStream("source", bytes.NewBufferString("hello"))
	var target bytes.Buffer
	sink := jayessruntime.NewWritableStream("sink", &target)

	if !source.CanRead() || source.CanWrite() {
		t.Fatalf("source stream read/write flags are wrong")
	}
	if sink.CanRead() || !sink.CanWrite() {
		t.Fatalf("sink stream read/write flags are wrong")
	}

	written, err := source.PipeTo(sink)
	if err != nil {
		t.Fatalf("PipeTo returned error: %v", err)
	}
	if written != 5 || target.String() != "hello" {
		t.Fatalf("unexpected pipe result written=%d target=%q", written, target.String())
	}
}

func TestIOStreamRejectsUnsupportedDirections(t *testing.T) {
	stream := jayessruntime.NewReadableStream("input", bytes.NewBufferString("x"))

	if _, err := stream.WriteString("nope"); err == nil {
		t.Fatalf("expected write to readable-only stream to fail")
	}
	if _, err := jayessruntime.NewWritableStream("output", &bytes.Buffer{}).ReadAll(); err == nil {
		t.Fatalf("expected read from writable-only stream to fail")
	}
}
