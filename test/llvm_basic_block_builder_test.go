package test

import (
	"reflect"
	"strings"
	"testing"

	"jayess-go/llvmbackend"
)

func TestLLVMBackendBasicBlockBuilderEmitsBranchesPhiAndReturn(t *testing.T) {
	var builder llvmbackend.BasicBlockBuilder
	entry := builder.NewLabel("entry")
	thenLabel := builder.NewLabel("then")
	elseLabel := builder.NewLabel("else")
	join := builder.NewLabel("join")

	must(t, builder.Begin(entry))
	must(t, builder.ConditionalBranch("%cond", thenLabel, elseLabel))
	must(t, builder.Begin(thenLabel))
	must(t, builder.Branch(join))
	must(t, builder.Begin(elseLabel))
	must(t, builder.Branch(join))
	must(t, builder.Begin(join))
	must(t, builder.Phi("%selected", "%jayess.value", []llvmbackend.PhiIncoming{
		{Value: "%left", Label: thenLabel},
		{Value: "%right", Label: elseLabel},
	}))
	must(t, builder.Return("%jayess.value", "%selected"))

	lines, err := builder.Lines()
	if err != nil {
		t.Fatalf("lines: %v", err)
	}
	want := []string{
		"entry.0:",
		"br i1 %cond, label %then.1, label %else.2",
		"then.1:",
		"br label %join.3",
		"else.2:",
		"br label %join.3",
		"join.3:",
		"%selected = phi %jayess.value [ %left, %then.1 ], [ %right, %else.2 ]",
		"ret %jayess.value %selected",
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("unexpected basic block lines:\nwant: %v\n got: %v", want, lines)
	}
}

func TestLLVMBackendBasicBlockBuilderRejectsUnterminatedBlock(t *testing.T) {
	var builder llvmbackend.BasicBlockBuilder
	must(t, builder.Begin("entry"))
	if err := builder.Emit("%v0 = call %jayess.value @make()"); err != nil {
		t.Fatalf("emit: %v", err)
	}
	_, err := builder.Lines()
	if err == nil || !strings.Contains(err.Error(), "has no terminator") {
		t.Fatalf("expected unterminated block error, got %v", err)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
