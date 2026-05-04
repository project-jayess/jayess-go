package test

import (
	"path/filepath"
	"testing"

	"jayess-go/binding"
	"jayess-go/resolver"
)

func TestResolverResolvesRelativeBindingImport(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":             ``,
		"src/native/math.bind.js": `export default { exports: { add: { symbol: "mylib_add", type: "function" } } };`,
	})

	resolved, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), "./native/math.bind.js")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	if !binding.IsBindingModulePath(resolved) {
		t.Fatalf("expected resolved binding module path, got %s", resolved)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("src", "native", "math.bind.js"))
}

func TestResolverResolvesPackageBindingImport(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":                               ``,
		"node_modules/nativepkg/math.bind.js":       `export default { exports: { add: { symbol: "mylib_add", type: "function" } } };`,
		"node_modules/nativepkg/package.json":       `{"jayess":"index.js"}`,
		"node_modules/nativepkg/index.js":           `export const ignored = 1;`,
		"node_modules/nativepkg/other/math.bind.js": `export default { exports: { add: { symbol: "other_add", type: "function" } } };`,
	})

	resolved, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), "nativepkg/math.bind.js")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	if !binding.IsBindingModulePath(resolved) {
		t.Fatalf("expected resolved package binding module path, got %s", resolved)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("node_modules", "nativepkg", "math.bind.js"))
}
