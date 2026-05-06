package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestCompilerVectorStoresOrderedValues(t *testing.T) {
	vector := jayessruntime.NewCompilerVector()
	if got := vector.Push(jayessruntime.NewString("token")); got != 1 {
		t.Fatalf("expected length 1, got %d", got)
	}
	vector.Push(jayessruntime.NewNumber(2))
	value, ok := vector.Get(1)
	if !ok || value.Number() != 2 {
		t.Fatalf("expected numeric second value, got %#v %v", value, ok)
	}
	if !vector.Set(0, jayessruntime.NewString("node")) {
		t.Fatal("expected set to succeed")
	}
	snapshot := vector.Values()
	snapshot[0] = jayessruntime.NewString("mutated")
	value, _ = vector.Get(0)
	if value.Text() != "node" {
		t.Fatalf("expected vector storage to ignore snapshot mutation, got %q", value.Text())
	}
}

func TestCompilerTableKeepsDeterministicKeyOrder(t *testing.T) {
	table := jayessruntime.NewCompilerTable()
	table.Set("scope", jayessruntime.NewString("global"))
	table.Set("symbol", jayessruntime.NewString("main"))
	table.Set("scope", jayessruntime.NewString("local"))
	keys := table.Keys()
	if len(keys) != 2 || keys[0] != "scope" || keys[1] != "symbol" {
		t.Fatalf("expected stable insertion order, got %#v", keys)
	}
	value, ok := table.Get("scope")
	if !ok || value.Text() != "local" {
		t.Fatalf("expected updated scope value, got %#v %v", value, ok)
	}
}

func TestCompilerRecordRejectsUnknownFields(t *testing.T) {
	record := jayessruntime.NewCompilerRecord(jayessruntime.CompilerRecordShape{
		Name:   "Token",
		Fields: []string{"kind", "text", "line"},
	})
	if !record.Set("kind", jayessruntime.NewString("identifier")) {
		t.Fatal("expected known field write to succeed")
	}
	if record.Set("column", jayessruntime.NewNumber(1)) {
		t.Fatal("expected unknown field write to fail")
	}
	value, ok := record.Get("kind")
	if !ok || value.Text() != "identifier" {
		t.Fatalf("expected stored kind, got %#v %v", value, ok)
	}
	if _, ok := record.Get("column"); ok {
		t.Fatal("expected unknown field read to fail")
	}
}
