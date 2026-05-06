package test

import (
	"strings"
	"testing"

	"jayess-go/ast"
	"jayess-go/llvmbackend"
)

func TestLLVMLowersJayessProgramToIRModule(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	module := llvmbackend.LowerJayessProgram(llvmbackend.JayessProgram{
		Name:       "app",
		Target:     target,
		ReturnCode: 7,
	})
	ir := llvmbackend.EmitLLVMIR(module)
	if !strings.Contains(ir, "; ModuleID = 'app'") || !strings.Contains(ir, "ret i32 7") {
		t.Fatalf("expected lowered Jayess program IR, got:\n%s", ir)
	}
}

func TestLLVMLowersJayessStatementsThroughRuntimeMainWrapper(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	module, err := llvmbackend.LowerJayessStatementProgram(llvmbackend.JayessStatementProgram{
		Name:   "app",
		Target: target,
		Statements: []ast.Statement{
			&ast.ReturnStatement{Value: &ast.NumberLiteral{Value: "7"}},
		},
	})
	if err != nil {
		t.Fatalf("lower statement program: %v", err)
	}
	ir := llvmbackend.EmitLLVMIR(module)
	for _, want := range []string{
		"define i32 @main()",
		"call %jayess.value @__jayess_user_main()",
		"call i32 @jayess_value_to_exit_code",
		"define %jayess.value @__jayess_user_main()",
		"ret %jayess.value",
	} {
		if !strings.Contains(ir, want) {
			t.Fatalf("expected statement program IR to contain %q:\n%s", want, ir)
		}
	}
}

func TestLLVMPlansNativeExecutableFromIROutput(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanExecutableFromIR("define i32 @main() { ret i32 0 }", "app", target)
	if !plan.CanBuildExecutable() {
		t.Fatalf("expected executable build plan to be buildable: %#v", plan)
	}
	if len(plan.Steps) != 3 || plan.Steps[0] != llvmbackend.LLVMVerifyStep || plan.Steps[2] != llvmbackend.ClangLinkStep {
		t.Fatalf("unexpected executable steps: %#v", plan.Steps)
	}
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected no active diagnostics for complete executable plan, got %#v", plan.Diagnostics)
	}
	if len(plan.ToolchainDiagnostics) == 0 {
		t.Fatal("expected possible toolchain diagnostics to be recorded")
	}
}

func TestLLVMExecutablePlanReportsMissingInputs(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanExecutableFromIR("", "", target)
	if plan.CanBuildExecutable() {
		t.Fatalf("expected missing-input executable plan not to be buildable: %#v", plan)
	}
	if len(plan.Diagnostics) != 2 {
		t.Fatalf("expected missing IR and output diagnostics, got %#v", plan.Diagnostics)
	}
}

func TestLLVMExecutablePlanReportsMissingTargetTriple(t *testing.T) {
	plan := llvmbackend.PlanExecutableFromIR("define i32 @main() { ret i32 0 }", "app", llvmbackend.TargetConfig{})
	if plan.CanBuildExecutable() {
		t.Fatalf("expected missing target triple plan not to be buildable: %#v", plan)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0] != "missing LLVM target triple" {
		t.Fatalf("expected target triple diagnostic, got %#v", plan.Diagnostics)
	}
}
