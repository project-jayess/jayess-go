package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/libuv"
)

func TestLibUVBindingModuleCanImportBindJS(t *testing.T) {
	module := libuv.BindingModule{
		Path: "./native/libuv.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./libuv.c"},
			Exports: []binding.Export{
				{Name: "createLoop", Symbol: "jayess_uv_create_loop", Kind: binding.FunctionExport},
			},
		},
		APIs:    []libuv.APIKind{libuv.LoopAPI, libuv.TimerAPI, libuv.TCPAPI},
		Handles: []libuv.HandleKind{libuv.LoopHandle, libuv.TimerHandle, libuv.TCPHandle},
	}

	if diagnostics := libuv.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid libuv binding module, got %#v", diagnostics)
	}
	if !libuv.SupportsAPI(module, libuv.TCPAPI) {
		t.Fatal("expected libuv TCP API support")
	}
}

func TestLibUVBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := libuv.BindingModule{
		Path: "./native/libuv.c",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "loop", Symbol: "uv_loop", Kind: binding.FunctionExport}},
		},
		Handles: []libuv.HandleKind{libuv.LoopHandle},
	}

	diagnostics := libuv.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestLibUVBuildPlanUsesVendoredSourceWhenRequested(t *testing.T) {
	module := libuv.BindingModule{
		Path: "./native/libuv.bind.js",
		Manifest: binding.Manifest{
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DJAYESS_LIBUV=1"},
			Exports:     []binding.Export{{Name: "loop", Symbol: "uv_loop", Kind: binding.FunctionExport}},
		},
		Handles:        []libuv.HandleKind{libuv.LoopHandle},
		VendoredSource: true,
	}

	plan := libuv.PlanBuild([]libuv.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean libuv build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 2 {
		t.Fatalf("expected vendored libuv compile units, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DJAYESS_LIBUV=1"})
}

func TestLibUVHandlesRepresentNativeTypesSafely(t *testing.T) {
	for _, kind := range []libuv.HandleKind{
		libuv.LoopHandle,
		libuv.TimerHandle,
		libuv.TCPHandle,
		libuv.UDPHandle,
		libuv.FSReqHandle,
		libuv.ProcessHandle,
		libuv.SignalHandle,
	} {
		if !libuv.SupportsHandle(kind) {
			t.Fatalf("expected libuv handle support for %s", kind)
		}
	}
}
