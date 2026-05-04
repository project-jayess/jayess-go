package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/mongoose"
)

func TestMongooseBindingModuleCanImportBindJS(t *testing.T) {
	module := mongoose.BindingModule{
		Path: "./native/mongoose.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./mongoose_wrapper.c"},
			Exports: []binding.Export{
				{Name: "createManager", Symbol: "jayess_mongoose_create_manager", Kind: binding.FunctionExport},
			},
		},
		APIs:    []mongoose.APIKind{mongoose.ManagerAPI, mongoose.HTTPAPI},
		Handles: []mongoose.HandleKind{mongoose.ManagerHandle, mongoose.ConnectionHandle},
	}

	if diagnostics := mongoose.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid Mongoose binding module, got %#v", diagnostics)
	}
	if !mongoose.SupportsAPI(module, mongoose.HTTPAPI) {
		t.Fatal("expected Mongoose HTTP API support")
	}
}

func TestMongooseBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := mongoose.BindingModule{
		Path: "./native/mongoose.c",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "create", Symbol: "mongoose_create", Kind: binding.FunctionExport}},
		},
		Handles: []mongoose.HandleKind{mongoose.ManagerHandle},
	}

	diagnostics := mongoose.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestMongooseBuildPlanUsesVendoredSourceWhenRequested(t *testing.T) {
	module := mongoose.BindingModule{
		Path: "./native/mongoose.bind.js",
		Manifest: binding.Manifest{
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DMG_ENABLE_LINES=1"},
			Exports:     []binding.Export{{Name: "serve", Symbol: "mongoose_serve", Kind: binding.FunctionExport}},
		},
		Handles:        []mongoose.HandleKind{mongoose.ManagerHandle},
		VendoredSource: true,
	}

	plan := mongoose.PlanBuild([]mongoose.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean Mongoose build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected vendored Mongoose compile unit, got %#v", plan.CompileUnits)
	}
	if plan.CompileUnits[0].Source != "./mongoose.c" {
		t.Fatalf("expected vendored mongoose.c source, got %s", plan.CompileUnits[0].Source)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
}

func TestMongooseHandlesRepresentNativeTypesSafely(t *testing.T) {
	for _, kind := range []mongoose.HandleKind{
		mongoose.ManagerHandle,
		mongoose.ConnectionHandle,
		mongoose.RequestHandle,
		mongoose.WebSocketHandle,
	} {
		if !mongoose.SupportsHandle(kind) {
			t.Fatalf("expected Mongoose handle support for %s", kind)
		}
	}
}
