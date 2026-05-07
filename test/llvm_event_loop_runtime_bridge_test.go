package test

import (
	"strings"
	"testing"

	"jayess-go/lexer"
	"jayess-go/llvmbackend"
	"jayess-go/parser"
)

func TestLLVMBridgeEmitsDirectEventLoopRuntimeCalls(t *testing.T) {
	source := `
		const timeout = setTimeout(() => print("once"), 10);
		const interval = setInterval(() => print("again"), 20);
		queueMicrotask(() => print("soon"));
		clearTimeout(timeout);
		clearInterval(interval);
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
		Name:       "event-loop-runtime",
		Target:     target,
		Statements: program.Statements,
	})
	if err != nil {
		t.Fatalf("lower source: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"@jayess_timer_set_timeout",
		"@jayess_timer_set_interval",
		"@jayess_queue_microtask",
		"@jayess_timer_clear_timeout",
		"@jayess_timer_clear_interval",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected event loop runtime IR to contain %q:\n%s", want, ir)
		}
	}
}
