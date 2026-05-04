package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverDispatchesRelativeImportToSourceResolver(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
		"src/math.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), "./math")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("src", "math.js"))
}

func TestResolverDispatchesPackageImportToPackageResolver(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":                       ``,
		"node_modules/math/package.json":    `{"jayess":"index.js"}`,
		"node_modules/math/index.js":        `export const value = 1;`,
		"node_modules/math/other/index.js":  `export const ignored = 1;`,
		"node_modules/other/math/index.js":  `export const ignored = 2;`,
		"node_modules/@scope/math/index.js": `export const ignored = 3;`,
	})

	resolved, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), "math")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("node_modules", "math", "index.js"))
}

func TestResolverDispatchesScopedPackageImportToPackageResolver(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js":                       ``,
		"node_modules/@scope/math/index.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), "@scope/math")
	if err != nil {
		t.Fatalf("ResolveImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("node_modules", "@scope", "math", "index.js"))
}

func TestResolverRejectsImportWithoutImporterPath(t *testing.T) {
	_, err := resolver.ResolveImport("", "math")
	if err == nil {
		t.Fatalf("expected missing importer path error")
	}
	if !strings.Contains(err.Error(), `import "math" has no importer path`) {
		t.Fatalf("expected missing importer path diagnostic, got %v", err)
	}
}

func TestResolverDispatchesDotPrefixedMalformedImportsToSourceResolver(t *testing.T) {
	root := createImportResolverFixture(t, map[string]string{
		"src/main.js": ``,
	})

	for _, importPath := range []string{".", "..", ".hidden"} {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolveImport(filepath.Join(root, "src", "main.js"), importPath)
			if err == nil {
				t.Fatalf("expected malformed source import error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "source import") {
				t.Fatalf("expected source import diagnostic, got %v", err)
			}
		})
	}
}

func createImportResolverFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := filepath.Join("..", "temp", "resolver-import", testPathName(t))
	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("remove fixture root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create fixture dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}
	}
	return root
}
