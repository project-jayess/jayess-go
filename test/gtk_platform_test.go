package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/gtk"
)

func TestGTKPlatformSupportUsesPkgConfig(t *testing.T) {
	for _, platform := range []string{"linux", "darwin", "windows"} {
		support, ok := gtk.PlatformSupportFor(platform)
		if !ok {
			t.Fatalf("expected GTK platform support for %s", platform)
		}
		if !support.PkgConfig || !support.Supported {
			t.Fatalf("expected pkg-config supported GTK platform metadata for %#v", support)
		}
		if len(support.IncludeFlags) == 0 || len(support.LibraryFlags) == 0 {
			t.Fatalf("expected include/library flags for %#v", support)
		}
	}
}

func TestGTKPlatformSupportReportsMissingToolchain(t *testing.T) {
	support, ok := gtk.PlatformSupportFor("plan9")
	if ok {
		t.Fatalf("did not expect GTK platform support for %#v", support)
	}
	if support.Diagnostic == "" {
		t.Fatalf("expected missing GTK platform diagnostic")
	}
}

func TestGTKCrossPlatformBuildFlags(t *testing.T) {
	module := gtk.BindingModule{
		Path: "./native/gtk.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./gtk.c"},
			Platforms: map[string]binding.PlatformOptions{
				"linux":   {LDFlags: []string{"-lgtk-3"}},
				"darwin":  {LDFlags: []string{"-lgtk-3"}},
				"windows": {LDFlags: []string{"-lgtk-3"}},
			},
			Exports: []binding.Export{{Name: "init", Symbol: "gtk_init", Kind: binding.FunctionExport}},
		},
		Handles: []gtk.HandleKind{gtk.ApplicationHandle},
	}
	for _, platform := range []string{"linux", "darwin", "windows"} {
		plan := gtk.PlanBuild([]gtk.BindingModule{module}, platform, "./runtime")
		if !hasString(plan.LDFlags, "-lgtk-3") {
			t.Fatalf("expected GTK link flag for %s in %#v", platform, plan.LDFlags)
		}
	}
}
