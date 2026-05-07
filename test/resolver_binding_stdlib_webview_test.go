package test

import (
	"path/filepath"
	"testing"

	"jayess-go/resolver"
)

func TestResolverBindingModulesDiscoverVendoredWebviewBinding(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `import { createWindow } from "@jayess/webview";`)

	modules, diagnostics, err := resolver.ResolveBindingModules(mainPath, program)
	if err != nil {
		t.Fatalf("ResolveBindingModules returned error: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected clean binding module resolution, got %#v", diagnostics)
	}
	if len(modules) != 1 {
		t.Fatalf("expected one binding module, got %#v", modules)
	}
	requireResolvedSuffix(t, modules[0].Path, filepath.Join("stdlib", "@jayess", "webview", "native", "webview.bind.js"))
	requireStringSlice(t, modules[0].Manifest.Sources, []string{"./webview.cpp"})
}

func TestResolverBindingBuildPlanUsesVendoredWebviewBinding(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
	})
	mainPath := filepath.Join(root, "src", "main.js")
	program := parseProgram(t, `import { createWindow } from "@jayess/webview";`)

	plan, err := resolver.ResolveBindingBuildPlan(mainPath, program, "windows", "./runtime")
	if err != nil {
		t.Fatalf("ResolveBindingBuildPlan returned error: %v", err)
	}
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected one compile unit, got %#v", plan.CompileUnits)
	}
	requireResolvedSuffix(t, plan.CompileUnits[0].ModulePath, filepath.Join("stdlib", "@jayess", "webview", "native", "webview.bind.js"))
	if !hasLDFlag(plan.LDFlags, "-lole32") {
		t.Fatalf("expected Windows webview link flags in %#v", plan.LDFlags)
	}
}

func hasLDFlag(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
