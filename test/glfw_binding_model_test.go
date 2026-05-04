package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/glfw"
)

func TestGLFWBindingModuleCanImportBindJS(t *testing.T) {
	module := glfw.BindingModule{
		Path: "./native/glfw.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./glfw.c"},
			Exports: []binding.Export{
				{Name: "createWindow", Symbol: "jayess_glfw_create_window", Kind: binding.FunctionExport},
			},
		},
		APIs:    []glfw.APIKind{glfw.WindowAPI, glfw.ContextAPI, glfw.InputAPI},
		Handles: []glfw.HandleKind{glfw.WindowHandle, glfw.MonitorHandle},
	}

	if diagnostics := glfw.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid GLFW binding module, got %#v", diagnostics)
	}
	if !glfw.SupportsAPI(module, glfw.ContextAPI) {
		t.Fatal("expected GLFW context API support")
	}
}

func TestGLFWBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := glfw.BindingModule{
		Path: "./native/glfw.c",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "init", Symbol: "glfw_init", Kind: binding.FunctionExport}},
		},
		Handles: []glfw.HandleKind{glfw.WindowHandle},
	}

	diagnostics := glfw.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestGLFWBindingBuildPlanLinksNativeSources(t *testing.T) {
	module := glfw.BindingModule{
		Path: "./native/glfw.bind.js",
		Manifest: binding.Manifest{
			Sources:     []string{"./glfw.c"},
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DJAYESS_GLFW=1"},
			LDFlags:     []string{"-lglfw"},
			Exports: []binding.Export{
				{Name: "init", Symbol: "jayess_glfw_init", Kind: binding.FunctionExport},
			},
		},
		Handles: []glfw.HandleKind{glfw.WindowHandle},
	}

	plan := glfw.PlanBuild([]glfw.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean GLFW build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected one GLFW compile unit, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DJAYESS_GLFW=1"})
	requireStringSlice(t, plan.LDFlags, []string{"-lglfw"})
}

func TestGLFWHandleRulesDefineLifecycleOwnership(t *testing.T) {
	for _, kind := range []glfw.HandleKind{
		glfw.WindowHandle,
		glfw.MonitorHandle,
		glfw.CursorHandle,
		glfw.JoystickHandle,
		glfw.VulkanSurface,
	} {
		if !glfw.SupportsHandle(kind) {
			t.Fatalf("expected GLFW handle support for %s", kind)
		}
	}
}
