package test

import (
	"strings"
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeSourceTextIndexesUnicodeLocations(t *testing.T) {
	source := jayessruntime.NewSourceText("alpha\nβeta\nlast")
	offset := strings.Index(source.Text(), "t")
	location, ok := source.Location(offset)
	if !ok {
		t.Fatal("expected location")
	}
	if location.Line != 2 || location.Column != 3 {
		t.Fatalf("expected line 2 column 3, got %#v", location)
	}
	if source.ByteLen() <= source.RuneLen() {
		t.Fatalf("expected byte length to account for unicode")
	}
}

func TestRuntimeSourceTextSlicesOnValidBoundaries(t *testing.T) {
	source := jayessruntime.NewSourceText("aβc")
	text, ok := source.Slice(1, 3)
	if !ok || text != "β" {
		t.Fatalf("expected unicode slice, got %q %v", text, ok)
	}
	if _, ok := source.Slice(1, 2); ok {
		t.Fatal("expected invalid unicode boundary to fail")
	}
}

func TestRuntimeSourceTextConcatenatesAndTracksLines(t *testing.T) {
	source := jayessruntime.ConcatSourceText(
		jayessruntime.NewSourceText("one\n"),
		jayessruntime.NewSourceText("two"),
	)
	location, ok := source.Location(len("one\n"))
	if !ok {
		t.Fatal("expected location")
	}
	if location.Line != 2 || location.Column != 1 {
		t.Fatalf("expected second line start, got %#v", location)
	}
}

func TestRuntimeSourceTextHandlesLexerSizedInput(t *testing.T) {
	var builder strings.Builder
	for i := 0; i < 2048; i++ {
		builder.WriteString("const value = \"text\";\n")
	}
	source := jayessruntime.NewSourceText(builder.String())
	location, ok := source.Location(source.ByteLen())
	if !ok {
		t.Fatal("expected end location")
	}
	if location.Line != 2049 {
		t.Fatalf("expected final line after generated fixture, got %#v", location)
	}
}
