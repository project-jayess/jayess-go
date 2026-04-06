package compiler

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompilePathSupportsExportStarFrom(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "math.js"), []byte(`
export function add(a, b) {
  return a + b;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "index.js"), []byte(`
export * from "./math.js";
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "./lib/index.js";

function main(args) {
  return add(1, 2);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@add(") {
		t.Fatalf("expected exported symbol to be imported through export *, got:\n%s", string(result.LLVMIR))
	}
}

func TestCompilePathSupportsExportStarAsNamespaceFrom(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "math.js"), []byte(`
export function add(a, b) {
  return a + b;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "index.js"), []byte(`
export * as math from "./math.js";
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { math } from "./lib/index.js";

function main(args) {
  return math.add(1, 2);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_value_get_member") {
		t.Fatalf("expected namespace export to lower through object member access, got:\n%s", string(result.LLVMIR))
	}
}

func TestCompilePathRejectsDuplicateImportBindings(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "a.js"), []byte(`export const value = 1;`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "b.js"), []byte(`export const value = 2;`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { value } from "./lib/a.js";
import { value } from "./lib/b.js";

function main(args) {
  return value;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil || !strings.Contains(err.Error(), "duplicate import binding value") {
		t.Fatalf("expected duplicate import diagnostic, got: %v", err)
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected structured compile error, got: %T", err)
	}
	if compileErr.Diagnostic.Line != 3 || compileErr.Diagnostic.Column != 1 {
		t.Fatalf("expected duplicate import span 3:1, got %d:%d", compileErr.Diagnostic.Line, compileErr.Diagnostic.Column)
	}
}

func TestCompilePathRejectsDuplicateExports(t *testing.T) {
	dir := t.TempDir()
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
export const value = 1;
export { value };
export { value };

function main(args) {
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil || !strings.Contains(err.Error(), "duplicate export value") {
		t.Fatalf("expected duplicate export diagnostic, got: %v", err)
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected structured compile error, got: %T", err)
	}
	if compileErr.Diagnostic.Line != 3 || compileErr.Diagnostic.Column != 1 {
		t.Fatalf("expected duplicate export span 3:1, got %d:%d", compileErr.Diagnostic.Line, compileErr.Diagnostic.Column)
	}
}

func TestCompilePathReportsUnsupportedPackageEntrypointClearly(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "demo-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"main":"index.mjs"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import "demo-pkg";

function main(args) {
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected unsupported package entrypoint error")
	}
	if !strings.Contains(err.Error(), "supported Jayess .js module") {
		t.Fatalf("expected clear unsupported-package diagnostic, got: %v", err)
	}
}

func TestCompilePathReportsInvalidPackageJSONClearly(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "broken-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"main":`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import "broken-pkg";

function main(args) {
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil || !strings.Contains(err.Error(), "invalid package.json") {
		t.Fatalf("expected invalid package.json diagnostic, got: %v", err)
	}
}

func TestCompilePathReportsImportCycleWithPathChain(t *testing.T) {
	dir := t.TempDir()
	libDir := filepath.Join(dir, "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "a.js"), []byte("import \"./b.js\";\nexport const a = 1;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(libDir, "b.js"), []byte("import \"./a.js\";\nexport const b = 2;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import "./lib/a.js";

function main(args) {
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var compileErr *CompileError
	if !errors.As(err, &compileErr) {
		t.Fatalf("expected structured compile error, got: %T", err)
	}
	if compileErr.Diagnostic.Category != "loader" {
		t.Fatalf("expected loader category, got %q", compileErr.Diagnostic.Category)
	}
	if len(compileErr.Diagnostic.Notes) == 0 || !strings.Contains(compileErr.Diagnostic.Notes[0], "import cycle:") {
		t.Fatalf("expected import cycle note, got: %#v", compileErr.Diagnostic.Notes)
	}
}
