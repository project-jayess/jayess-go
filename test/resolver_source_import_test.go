package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverFindsRelativeSourceImportWithExplicitJSExtension(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
		"src/math.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), "./math.js")
	if err != nil {
		t.Fatalf("ResolveSourceImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("src", "math.js"))
}

func TestResolverFindsRelativeSourceImportWithoutExtension(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
		"src/math.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), "./math")
	if err != nil {
		t.Fatalf("ResolveSourceImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("src", "math.js"))
}

func TestResolverFindsRelativeDirectoryIndexSourceImport(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js":        ``,
		"src/math/index.js":  `export const value = 1;`,
		"src/math/unused.js": `export const unused = 1;`,
	})

	resolved, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), "./math")
	if err != nil {
		t.Fatalf("ResolveSourceImport returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("src", "math", "index.js"))
}

func TestResolverRejectsNonRelativeSourceImport(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), "math")
	if err == nil {
		t.Fatalf("expected non-relative source import error")
	}
	if !strings.Contains(err.Error(), `source import "math" must be relative`) {
		t.Fatalf("expected non-relative import diagnostic, got %v", err)
	}
}

func TestResolverRejectsEmptySourceImport(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), "")
	if err == nil {
		t.Fatalf("expected empty source import error")
	}
	if !strings.Contains(err.Error(), "source import must not be empty") {
		t.Fatalf("expected empty source import diagnostic, got %v", err)
	}
}

func TestResolverRejectsWhitespacePaddedSourceImport(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	for _, importPath := range []string{" ./math", "./math ", " ./math "} {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), importPath)
			if err == nil {
				t.Fatalf("expected whitespace-padded source import error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "is malformed") {
				t.Fatalf("expected malformed source import diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsDotOnlySourceImportForms(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	for _, importPath := range []string{".", "..", ".hidden"} {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), importPath)
			if err == nil {
				t.Fatalf("expected invalid relative source import error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "must start with ./ or ../") {
				t.Fatalf("expected relative source import shape diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsMalformedRelativeSourceImportSegments(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	for _, importPath := range []string{"./math//utils", "./math/./utils", "./math/"} {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), importPath)
			if err == nil {
				t.Fatalf("expected malformed relative source import error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "is malformed") {
				t.Fatalf("expected malformed relative source diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsBackslashSourceImport(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), `.\\math`)
	if err == nil {
		t.Fatalf("expected backslash source import error")
	}
	if !strings.Contains(err.Error(), `must use / as the path separator`) {
		t.Fatalf("expected path separator diagnostic, got %v", err)
	}
}

func TestResolverRejectsSchemeLikeSourceImport(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	schemeLike := []string{"node:fs", "https://example.com/pkg.js", "C:/workspace/pkg.js"}
	for _, importPath := range schemeLike {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), importPath)
			if err == nil {
				t.Fatalf("expected scheme-like source import error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "scheme-like") && !strings.Contains(err.Error(), "must be relative") {
				t.Fatalf("expected scheme-like source diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsSourceImportQueryOrFragment(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
	})

	for _, importPath := range []string{"./math.js?raw", "./math.js#hash"} {
		t.Run(testPathNameForValue(importPath), func(t *testing.T) {
			_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), importPath)
			if err == nil {
				t.Fatalf("expected source import query or fragment error for %q", importPath)
			}
			if !strings.Contains(err.Error(), "query strings or fragments") {
				t.Fatalf("expected source query/fragment diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsNonJSSourceImportExtension(t *testing.T) {
	root := createSourceImportFixture(t, map[string]string{
		"src/main.js": ``,
		"src/math.ts": `export const value = 1;`,
	})

	_, err := resolver.ResolveSourceImport(filepath.Join(root, "src", "main.js"), "./math.ts")
	if err == nil {
		t.Fatalf("expected unsupported source extension error")
	}
	if !strings.Contains(err.Error(), `is not a supported Jayess .js module`) {
		t.Fatalf("expected unsupported source extension diagnostic, got %v", err)
	}
}

func createSourceImportFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := filepath.Join("..", "temp", "resolver-source-import", testPathName(t))
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
