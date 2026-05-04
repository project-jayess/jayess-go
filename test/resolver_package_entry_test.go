package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"jayess-go/resolver"
)

func TestResolverUsesJayessPackageEntryBeforeModuleAndMain(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json": `{"jayess":"src/app.js","module":"module.js","main":"main.js"}`,
		"src/app.js":   `export const value = 1;`,
		"module.js":    `export const value = 2;`,
		"main.js":      `export const value = 3;`,
	})

	resolved, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err != nil {
		t.Fatalf("ResolvePackageEntry returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("src", "app.js"))
}

func TestResolverUsesModulePackageEntryWhenJayessMissing(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json":  `{"module":"dist/index.js","main":"main.js"}`,
		"dist/index.js": `export const value = 1;`,
		"main.js":       `export const value = 2;`,
	})

	resolved, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err != nil {
		t.Fatalf("ResolvePackageEntry returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("dist", "index.js"))
}

func TestResolverAllowsPackageEntryWithLeadingDotSlash(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json":  `{"main":"./dist/index.js"}`,
		"dist/index.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err != nil {
		t.Fatalf("ResolvePackageEntry returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, filepath.Join("dist", "index.js"))
}

func TestResolverFallsBackToIndexJSWithoutPackageJSON(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"index.js": `export const value = 1;`,
	})

	resolved, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err != nil {
		t.Fatalf("ResolvePackageEntry returned error: %v", err)
	}
	requireResolvedSuffix(t, resolved, "index.js")
}

func TestResolverRejectsUnsupportedPackageEntryExtension(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json": `{"jayess":"src/app.ts"}`,
		"src/app.ts":   `export const value = 1;`,
	})

	_, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err == nil {
		t.Fatalf("expected unsupported package entry extension error")
	}
	if !strings.Contains(err.Error(), `entry "src/app.ts" is not a supported Jayess .js module`) {
		t.Fatalf("expected unsupported entry extension diagnostic, got %v", err)
	}
}

func TestResolverRejectsPackageEntryOutsidePackageDirectory(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json": `{"main":"../outside.js"}`,
	})

	_, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err == nil {
		t.Fatalf("expected unsafe package entry path error")
	}
	if !strings.Contains(err.Error(), `entry "../outside.js" is not a safe package-relative path`) {
		t.Fatalf("expected unsafe package entry diagnostic, got %v", err)
	}
}

func TestResolverRejectsAbsolutePackageEntry(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json": `{"main":"/tmp/pkg/index.js"}`,
	})

	_, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err == nil {
		t.Fatalf("expected absolute package entry path error")
	}
	if !strings.Contains(err.Error(), `entry "/tmp/pkg/index.js" is not a safe package-relative path`) {
		t.Fatalf("expected unsafe package entry diagnostic, got %v", err)
	}
}

func TestResolverRejectsBackslashPackageEntry(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json": `{"main":"dist\\index.js"}`,
	})

	_, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err == nil {
		t.Fatalf("expected backslash package entry path error")
	}
	if !strings.Contains(err.Error(), `entry "dist\\index.js" is not a safe package-relative path`) {
		t.Fatalf("expected unsafe package entry diagnostic, got %v", err)
	}
}

func TestResolverRejectsSchemeLikePackageEntry(t *testing.T) {
	for _, entry := range []string{"node:fs", "https://example.com/pkg.js"} {
		t.Run(testPathNameForValue(entry), func(t *testing.T) {
			packageDir := createPackageEntryFixture(t, map[string]string{
				"package.json": `{"main":"` + entry + `"}`,
			})

			_, err := resolver.ResolvePackageEntry(packageDir, "pkg")
			if err == nil {
				t.Fatalf("expected scheme-like package entry path error")
			}
			if !strings.Contains(err.Error(), "is not a safe package-relative path") {
				t.Fatalf("expected unsafe package entry diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsPackageEntryQueryOrFragment(t *testing.T) {
	for _, entry := range []string{"dist/index.js?raw", "dist/index.js#hash"} {
		t.Run(testPathNameForValue(entry), func(t *testing.T) {
			packageDir := createPackageEntryFixture(t, map[string]string{
				"package.json": `{"main":"` + entry + `"}`,
			})

			_, err := resolver.ResolvePackageEntry(packageDir, "pkg")
			if err == nil {
				t.Fatalf("expected package entry query or fragment error")
			}
			if !strings.Contains(err.Error(), "is not a safe package-relative path") {
				t.Fatalf("expected unsafe package entry diagnostic, got %v", err)
			}
		})
	}
}

func TestResolverRejectsWhitespaceOnlyPackageEntry(t *testing.T) {
	packageDir := createPackageEntryFixture(t, map[string]string{
		"package.json": `{"main":"   "}`,
		"index.js":     `export const fallback = 1;`,
	})

	_, err := resolver.ResolvePackageEntry(packageDir, "pkg")
	if err == nil {
		t.Fatalf("expected whitespace-only package entry path error")
	}
	if !strings.Contains(err.Error(), "is not a safe package-relative path") {
		t.Fatalf("expected unsafe package entry diagnostic, got %v", err)
	}
}

func createPackageEntryFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := filepath.Join("..", "temp", "resolver-package-entry", testPathName(t))
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

func requireResolvedSuffix(t *testing.T, resolved string, suffix string) {
	t.Helper()
	if !strings.HasSuffix(filepath.ToSlash(resolved), filepath.ToSlash(suffix)) {
		t.Fatalf("expected resolved path to end with %q, got %q", suffix, resolved)
	}
}
