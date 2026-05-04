package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/e2e"
	"jayess-go/llvmbackend"
)

func TestE2EPlansNativeBindingExecutableScenario(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := e2e.PlanScenario(e2e.Scenario{
		Name:       "native-binding-smoke",
		Kind:       e2e.NativeBindingExecutable,
		SourceFile: "main.js",
		IR:         "define i32 @main() { ret i32 0 }",
		Target:     target,
		Bindings: []binding.Module{{
			Path: "./native/math.bind.js",
			Manifest: binding.Manifest{
				Sources: []string{"./math.c"},
				Exports: []binding.Export{
					{Name: "add", Symbol: "math_add", Kind: binding.FunctionExport},
				},
			},
		}},
	}, "./runtime", "./temp/native-binding-smoke")

	if !plan.Ready() {
		t.Fatalf("expected e2e native binding executable plan to be ready: %#v", plan)
	}
	if len(plan.BindingPlan.CompileUnits) != 1 {
		t.Fatalf("expected native binding compile unit, got %#v", plan.BindingPlan.CompileUnits)
	}
}

func TestE2EPlansAudioBindingExecutableScenario(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := e2e.PlanScenario(e2e.Scenario{
		Name:       "audio-binding-smoke",
		Kind:       e2e.AudioBindingExecutable,
		SourceFile: "audio.js",
		IR:         "define i32 @main() { ret i32 0 }",
		Target:     target,
		Bindings: []binding.Module{{
			Path: "./native/audio.bind.js",
			Manifest: binding.Manifest{
				Sources: []string{"./audio.c"},
				Exports: []binding.Export{
					{Name: "play", Symbol: "audio_play", Kind: binding.FunctionExport},
				},
			},
		}},
	}, "./runtime", "./temp/audio-binding-smoke")

	if !plan.Ready() {
		t.Fatalf("expected e2e audio binding executable plan to be ready: %#v", plan)
	}
	if plan.Scenario.Kind != e2e.AudioBindingExecutable {
		t.Fatalf("unexpected scenario kind %s", plan.Scenario.Kind)
	}
}

func TestE2EPlansLLVMExecutableScenario(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := e2e.PlanScenario(e2e.Scenario{
		Name:       "llvm-executable-smoke",
		Kind:       e2e.LLVMExecutable,
		SourceFile: "main.js",
		IR:         "define i32 @main() { ret i32 0 }",
		Target:     target,
	}, "./runtime", "./temp/llvm-executable-smoke")

	if !plan.Ready() {
		t.Fatalf("expected e2e LLVM executable plan to be ready: %#v", plan)
	}
	if plan.RequiredOutputs[0] != "./temp/llvm-executable-smoke" {
		t.Fatalf("unexpected required output %#v", plan.RequiredOutputs)
	}
}

func TestE2EPlanRejectsIncompleteScenario(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := e2e.PlanScenario(e2e.Scenario{
		IR:     "define i32 @main() { ret i32 0 }",
		Target: target,
	}, "./runtime", "./temp/missing")

	if plan.Ready() {
		t.Fatalf("expected incomplete e2e scenario not to be ready: %#v", plan)
	}
	if len(plan.Diagnostics) != 3 {
		t.Fatalf("expected missing name, kind, and source diagnostics, got %#v", plan.Diagnostics)
	}
}

func TestE2EPlanRequiresTempOutputPath(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := e2e.PlanScenario(e2e.Scenario{
		Name:       "bad-output",
		Kind:       e2e.LLVMExecutable,
		SourceFile: "main.js",
		IR:         "define i32 @main() { ret i32 0 }",
		Target:     target,
	}, "./runtime", "./build/bad-output")

	if plan.Ready() {
		t.Fatalf("expected non-temp output plan not to be ready: %#v", plan)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0] != "e2e executable outputs must be placed in ./temp" {
		t.Fatalf("unexpected output path diagnostics: %#v", plan.Diagnostics)
	}
}

func TestE2EPlanRejectsUnknownScenarioKind(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := e2e.PlanScenario(e2e.Scenario{
		Name:       "unknown-kind",
		Kind:       e2e.ScenarioKind("custom"),
		SourceFile: "main.js",
		IR:         "define i32 @main() { ret i32 0 }",
		Target:     target,
	}, "./runtime", "./temp/unknown-kind")

	if plan.Ready() {
		t.Fatalf("expected unknown scenario kind not to be ready: %#v", plan)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0] != "unknown e2e scenario kind" {
		t.Fatalf("unexpected scenario kind diagnostics: %#v", plan.Diagnostics)
	}
}

func TestE2EPlanRequiresJayessSourceExtension(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := e2e.PlanScenario(e2e.Scenario{
		Name:       "bad-source",
		Kind:       e2e.LLVMExecutable,
		SourceFile: "main.ts",
		IR:         "define i32 @main() { ret i32 0 }",
		Target:     target,
	}, "./runtime", "./temp/bad-source")

	if plan.Ready() {
		t.Fatalf("expected non-js source scenario not to be ready: %#v", plan)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0] != "Jayess source file must use .js extension" {
		t.Fatalf("unexpected source diagnostics: %#v", plan.Diagnostics)
	}
}
