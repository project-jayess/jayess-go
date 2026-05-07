package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectWorkerRuntimeCalls(t *testing.T) {
	source := `
		const thread = worker.thread("worker.js");
		worker.postMessage(thread, "ready");
		worker.onMessage(thread, (message) => message);
		const memory = worker.sharedMemory(2);
		worker.atomicStore(memory, 0, 1);
		worker.atomicLoad(memory, 0);
	`
	program, err := parser.New(lexer.New(source)).ParseProgram()
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	module, err := llvmbackend.LowerJayessStatementProgram(llvmbackend.JayessStatementProgram{
		Name:       "worker-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_worker_thread",
		"@jayess_worker_post_message",
		"@jayess_worker_on_message",
		"@jayess_worker_shared_memory",
		"@jayess_worker_atomic_store",
		"@jayess_worker_atomic_load",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected worker runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
