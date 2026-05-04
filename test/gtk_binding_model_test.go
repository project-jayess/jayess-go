package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/gtk"
)

func TestGTKBindingModuleCanImportBindJS(t *testing.T) {
	module := gtk.BindingModule{
		Path: "./native/gtk.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./gtk.c"},
			Exports: []binding.Export{
				{Name: "createWindow", Symbol: "jayess_gtk_create_window", Kind: binding.FunctionExport},
			},
		},
		APIs:    []gtk.APIKind{gtk.ApplicationAPI, gtk.WindowAPI, gtk.WidgetAPI},
		Handles: []gtk.HandleKind{gtk.ApplicationHandle, gtk.WindowHandle, gtk.WidgetHandle},
	}

	if diagnostics := gtk.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid GTK binding module, got %#v", diagnostics)
	}
	if !gtk.SupportsAPI(module, gtk.WindowAPI) {
		t.Fatal("expected GTK window API support")
	}
}

func TestGTKBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := gtk.BindingModule{
		Path: "./native/gtk.c",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "init", Symbol: "gtk_init", Kind: binding.FunctionExport}},
		},
		Handles: []gtk.HandleKind{gtk.ApplicationHandle},
	}

	diagnostics := gtk.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestGTKBindingBuildPlanLinksNativeSources(t *testing.T) {
	module := gtk.BindingModule{
		Path: "./native/gtk.bind.js",
		Manifest: binding.Manifest{
			Sources:     []string{"./gtk.c"},
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DJAYESS_GTK=1"},
			LDFlags:     []string{"-lgtk-3"},
			Exports: []binding.Export{
				{Name: "init", Symbol: "jayess_gtk_init", Kind: binding.FunctionExport},
			},
		},
		Handles: []gtk.HandleKind{gtk.ApplicationHandle},
	}

	plan := gtk.PlanBuild([]gtk.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean GTK build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected one GTK compile unit, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DJAYESS_GTK=1"})
	requireStringSlice(t, plan.LDFlags, []string{"-lgtk-3"})
}

func TestGTKHandleRulesRepresentNativeTypesSafely(t *testing.T) {
	for _, kind := range []gtk.HandleKind{
		gtk.ApplicationHandle,
		gtk.WindowHandle,
		gtk.WidgetHandle,
		gtk.LayoutHandle,
		gtk.SignalHandle,
	} {
		if !gtk.SupportsHandle(kind) {
			t.Fatalf("expected managed GTK handle support for %s", kind)
		}
	}
}
