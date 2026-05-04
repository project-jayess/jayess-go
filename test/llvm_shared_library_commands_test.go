package test

import (
	"strings"
	"testing"

	"jayess-go/llvmbackend"
)

func TestSharedLibraryToolchainCommandsUseTargetAndTempPaths(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanSharedLibraryFromIR("define i32 @main() { ret i32 0 }", "build/libmath.so", target)
	commands := llvmbackend.SharedLibraryToolchainCommands(plan, "temp/jayess-build")
	if len(commands) != 1 {
		t.Fatalf("expected one clang IR compile/link command, got %#v", commands)
	}
	if commands[0].Program != "clang" || commands[0].Step != llvmbackend.ClangLinkStep {
		t.Fatalf("unexpected clang command: %#v", commands[0])
	}
	link := commands[0].String()
	for _, want := range []string{
		"clang -target x86_64-pc-linux-gnu",
		"temp/jayess-build/libmath.ll",
		"-shared",
		"-o build/libmath.so",
	} {
		if !strings.Contains(link, want) {
			t.Fatalf("expected link command to contain %q, got %q", want, link)
		}
	}
}

func TestSharedLibraryToolchainCommandsUseWindowsObjectExtension(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("windows-x64")
	if !ok {
		t.Fatal("expected windows target config")
	}
	plan := llvmbackend.PlanSharedLibraryFromIR("define i32 @main() { ret i32 0 }", "build/math.dll", target)
	command := llvmbackend.SharedLibraryLinkCommand(plan, "temp/jayess-build")
	if !strings.Contains(command.String(), "temp/jayess-build/math.obj") {
		t.Fatalf("expected windows object extension in link command, got %q", command.String())
	}
}

func TestSharedLibraryLinkCommandCanBeUsedWithoutLLCCommand(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanSharedLibraryFromIR("define i32 @main() { ret i32 0 }", "build/libmath.so", target)
	command := llvmbackend.SharedLibraryLinkCommand(plan, "temp/jayess-build")
	if command.Program != "clang" || command.Step != llvmbackend.ClangLinkStep {
		t.Fatalf("unexpected link command: %#v", command)
	}
	if strings.Contains(command.String(), "llc") || !strings.Contains(command.String(), "temp/jayess-build/libmath.o") {
		t.Fatalf("unexpected link command string: %q", command.String())
	}
}

func TestSharedLibraryLinkCommandIncludesExtraBindingObjects(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanSharedLibraryFromIR("define i32 @main() { ret i32 0 }", "build/libmath.so", target)
	plan.ExtraObjectFiles = []string{"temp/jayess-bindings/0-math-math.o"}
	plan.LinkFlags = append(plan.LinkFlags, "-lm")
	command := llvmbackend.SharedLibraryLinkCommand(plan, "temp/jayess-build")
	link := command.String()
	objectIndex := strings.Index(link, "temp/jayess-build/libmath.o")
	bindingIndex := strings.Index(link, "temp/jayess-bindings/0-math-math.o")
	libraryIndex := strings.Index(link, "-lm")
	if objectIndex < 0 || bindingIndex < 0 || libraryIndex < 0 {
		t.Fatalf("expected primary object, binding object, and library flag in %q", link)
	}
	if !(objectIndex < bindingIndex && bindingIndex < libraryIndex) {
		t.Fatalf("expected binding object before library flags in %q", link)
	}
}

func TestSharedLibraryIRCommandCanCompileAndLinkTextIR(t *testing.T) {
	target, ok := llvmbackend.TargetConfigFor("linux-x64")
	if !ok {
		t.Fatal("expected linux target config")
	}
	plan := llvmbackend.PlanSharedLibraryFromIR("define i32 @main() { ret i32 0 }", "build/libmath.so", target)
	command := llvmbackend.SharedLibraryIRCommand(plan, "temp/jayess-build")
	if command.Program != "clang" || command.Step != llvmbackend.ClangLinkStep {
		t.Fatalf("unexpected IR command: %#v", command)
	}
	if strings.Contains(command.String(), "llc") || strings.Contains(command.String(), "llvm-as") || !strings.Contains(command.String(), "temp/jayess-build/libmath.ll") {
		t.Fatalf("unexpected IR command string: %q", command.String())
	}
}
