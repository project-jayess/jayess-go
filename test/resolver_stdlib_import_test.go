package test

import (
	"path/filepath"
	"testing"

	"jayess-go/resolver"
)

func TestResolverRoutesStdlibImportsBeforeNodeModules(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":              ``,
		"node_modules/fs/index.js": `export const shadow = 1;`,
	})

	resolved, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), "fs")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	if resolved != "jayess:stdlib/fs" {
		t.Fatalf("expected Jayess stdlib fs import, got %s", resolved)
	}
}

func TestResolverRoutesEditorFriendlyStdlibImports(t *testing.T) {
	for _, importPath := range []string{"fs", "path", "util"} {
		t.Run(importPath, func(t *testing.T) {
			resolved, err := resolver.ResolveImport("/project/src/main.js", importPath)
			if err != nil {
				t.Fatalf("ResolveImport returned error: %v", err)
			}
			if resolved != "jayess:stdlib/"+importPath {
				t.Fatalf("expected Jayess stdlib import for %q, got %s", importPath, resolved)
			}
		})
	}
}

func TestResolverKeepsUnknownBareImportsAsPackages(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":                    ``,
		"node_modules/math/package.json": `{"jayess":"index.js"}`,
		"node_modules/math/index.js":     `export const value = 1;`,
	})

	resolved, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), "math")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("node_modules", "math", "index.js"))
}
