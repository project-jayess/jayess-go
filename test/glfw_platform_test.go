package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/glfw"
)

func TestGLFWPlatformSupport(t *testing.T) {
	for _, platform := range []string{"linux", "darwin", "windows"} {
		support, ok := glfw.PlatformSupportFor(platform)
		if !ok {
			t.Fatalf("expected GLFW platform support for %s", platform)
		}
		if !support.Supported || len(support.LibraryFlags) == 0 {
			t.Fatalf("expected GLFW library flags for %#v", support)
		}
	}
}

func TestGLFWPlatformSupportReportsMissingToolchain(t *testing.T) {
	support, ok := glfw.PlatformSupportFor("plan9")
	if ok {
		t.Fatalf("did not expect GLFW platform support for %#v", support)
	}
	if support.Diagnostic == "" {
		t.Fatal("expected missing GLFW platform diagnostic")
	}
}

func TestGLFWCrossPlatformBuildFlags(t *testing.T) {
	module := glfw.BindingModule{
		Path: "./native/glfw.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./glfw.c"},
			Platforms: map[string]binding.PlatformOptions{
				"linux":   {LDFlags: []string{"-lglfw", "-lGL", "-ldl", "-lpthread"}},
				"darwin":  {LDFlags: []string{"-lglfw", "-framework", "Cocoa", "-framework", "OpenGL"}},
				"windows": {LDFlags: []string{"-lglfw3", "-lopengl32", "-lgdi32"}},
			},
			Exports: []binding.Export{{Name: "init", Symbol: "glfw_init", Kind: binding.FunctionExport}},
		},
		Handles: []glfw.HandleKind{glfw.WindowHandle},
	}
	cases := map[string][]string{
		"linux":   {"-lglfw", "-lGL"},
		"darwin":  {"-framework", "Cocoa", "OpenGL"},
		"windows": {"-lglfw3", "-lopengl32"},
	}
	for platform, flags := range cases {
		plan := glfw.PlanBuild([]glfw.BindingModule{module}, platform, "./runtime")
		for _, flag := range flags {
			if !hasString(plan.LDFlags, flag) {
				t.Fatalf("expected GLFW %s flag %s in %#v", platform, flag, plan.LDFlags)
			}
		}
	}
}
