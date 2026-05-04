package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverFindsPackageImportInNearestNodeModules(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"app/src/main.js":                    ``,
		"app/node_modules/math/package.json": `{"jayess":"src/index.js"}`,
		"app/node_modules/math/src/index.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolvePackageImport(filepath.Join(root, "app", "src"), "math")
	if err != nil {
		t.Fatalf("ResolvePackageImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("app", "node_modules", "math", "src", "index.js"))
}

func TestResolverFindsPackageImportInAncestorNodeModules(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"workspace/project/src/main.js":                 ``,
		"workspace/node_modules/math/index.js":          `export const value = 1;`,
		"workspace/project/node_modules/other/index.js": `export const other = 1;`,
	})

	resolved, err := resolver.ResolvePackageImport(filepath.Join(root, "workspace", "project", "src"), "math")
	if err != nil {
		t.Fatalf("ResolvePackageImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("workspace", "node_modules", "math", "index.js"))
}

func TestResolverFindsScopedPackageImport(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"app/src/main.js": ``,
		"app/node_modules/@jayess/httpserver/index.js":   `export const value = 1;`,
		"app/node_modules/@jayess/httpserver/package.js": `export const ignored = 1;`,
	})

	resolved, err := resolver.ResolvePackageImport(filepath.Join(root, "app", "src"), "@jayess/httpserver")
	if err != nil {
		t.Fatalf("ResolvePackageImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("app", "node_modules", "@jayess", "httpserver", "index.js"))
}

func TestResolverFindsPackageSubpathImport(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"app/src/main.js":                        ``,
		"app/node_modules/math/utils/helpers.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolvePackageImport(filepath.Join(root, "app", "src"), "math/utils/helpers")
	if err != nil {
		t.Fatalf("ResolvePackageImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("app", "node_modules", "math", "utils", "helpers.js"))
}

func TestResolverReportsMissingPackageImport(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"app/src/main.js": ``,
	})

	_, err := resolver.ResolvePackageImport(filepath.Join(root, "app", "src"), "missing")
	if err == nil {
		t.Fatalf("expected missing package error")
	}
	if !strings.Contains(err.Error(), `package "missing" was not found in node_modules`) {
		t.Fatalf("expected missing package diagnostic, got %v", err)
	}
}

func TestResolverRejectsMalformedPackageImportBeforeFilesystemLookup(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"app/src/main.js":                  ``,
		"app/node_modules/other/index.js":  `export const other = 1;`,
		"app/node_modules/math/secret.js":  `export const secret = 1;`,
		"app/node_modules/@scope/index.js": `export const invalid = 1;`,
	})

	malformed := []string{"", " ", " math", "math ", "math/", "math//utils", "math/./utils", "math/../other", "@scope", "@/math", "./math", `math\utils`}
	for _, importPath := range malformed {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolvePackageImport(filepath.Join(root, "app", "src"), importPath)
			if err == nil {
				t.Fatalf("expected malformed package import error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "malformed") && !strings.Contains(err.Error(), "must not be empty") && !strings.Contains(err.Error(), "must use /") {
				t.Fatalf("expected malformed package diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsSchemeLikePackageImportBeforeFilesystemLookup(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"app/src/main.js": ``,
	})

	schemeLike := []string{"node:fs", "https://example.com/pkg", "C:/workspace/pkg"}
	for _, importPath := range schemeLike {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolvePackageImport(filepath.Join(root, "app", "src"), importPath)
			if err == nil {
				t.Fatalf("expected scheme-like package import error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "scheme-like") && !strings.Contains(err.Error(), "bare package specifier") {
				t.Fatalf("expected scheme-like package diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsPackageImportQueryOrFragmentBeforeFilesystemLookup(t *testing.T) {
	root := createPackageImportFixture(t, map[string]string{
		"app/src/main.js": ``,
	})

	for _, importPath := range []string{"math?raw", "math#hash", "@scope/math?raw"} {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolvePackageImport(filepath.Join(root, "app", "src"), importPath)
			if err == nil {
				t.Fatalf("expected package import query or fragment error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "query strings or fragments") {
				t.Fatalf("expected package query/fragment diagnostic, got %v", err)
			}
		})
	}
}

func createPackageImportFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := filepath.Join("..", "temp", "resolver-package-import", testPathName(t))
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
