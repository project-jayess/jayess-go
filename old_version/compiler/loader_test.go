package compiler

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsPathSuffix(values []string, wantSuffix string) bool {
	wantSuffix = filepath.ToSlash(wantSuffix)
	for _, value := range values {
		if strings.HasSuffix(filepath.ToSlash(value), wantSuffix) {
			return true
		}
	}
	return false
}

func repoRootFromCompilerTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(file))
}

func copyDirRecursiveCompilerTest(t *testing.T, srcDir, dstDir string) {
	t.Helper()
	if err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		var rel string
		var target string
		var data []byte
		if err != nil {
			return err
		}
		rel, err = filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target = filepath.Join(dstDir, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err = os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	}); err != nil {
		t.Fatalf("copyDirRecursiveCompilerTest returned error: %v", err)
	}
}

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
	if !strings.Contains(string(result.LLVMIR), "@jayess_fn_add(") {
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

func TestCompilePathSupportsNativeWrapperModuleManifests(t *testing.T) {
	t.Skip("legacy native manifests removed; use .bind.js")
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double jayess_manifest_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"
#include "native_math.h"

jayess_value *jayess_manifest_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_manifest_helper(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(`#include "native_math.h"

double jayess_manifest_helper(double left, double right) {
    return left + right + 5;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	manifestPath := filepath.Join(nativeDir, "math.native.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "sources": ["./math.c", "./helper.c"],
  "includeDirs": ["./include"],
  "cflags": ["-DJAYESS_NATIVE_WRAPPER=1"],
  "exports": {
    "add": "jayess_manifest_add"
  }
}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add as sum } from "./native/math.native.json";

function main(args) {
  return sum(1, 2);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_manifest_add(") {
		t.Fatalf("expected native manifest symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected 2 native sources, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/native/include") {
		t.Fatalf("expected native include directory to be carried through, got %#v", result.NativeIncludeDirs)
	}
	if len(result.NativeCompileFlags) != 1 || result.NativeCompileFlags[0] != "-DJAYESS_NATIVE_WRAPPER=1" {
		t.Fatalf("expected native compile flags to be carried through, got %#v", result.NativeCompileFlags)
	}
}

func TestCompilePathRejectsUnknownNativeManifestExport(t *testing.T) {
	t.Skip("legacy native manifests removed; use .bind.js")
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"
jayess_value *jayess_manifest_add(jayess_value *a, jayess_value *b) { return a; }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.native.json"), []byte(`{
  "sources": ["./math.c"],
  "exports": {
    "add": "jayess_manifest_add"
  }
}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { missing } from "./native/math.native.json";

function main(args) {
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected unknown native export error")
	}
	if !strings.Contains(err.Error(), "does not export missing") {
		t.Fatalf("expected native export diagnostic, got: %v", err)
	}
}

func TestCompilePathSupportsNativeManifestShorthandAndImplicitExports(t *testing.T) {
	t.Skip("legacy native manifests removed; use .bind.js")
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double jayess_native_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"
#include "native_math.h"
jayess_value *jayess_native_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_native_helper(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(`#include "native_math.h"
double jayess_native_helper(double left, double right) {
    return left + right + 7;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.native.json"), []byte(`{
  "source": "./math.c",
  "sources": ["./helper.c"],
  "includeDir": "./include",
  "cflag": "-DJAYESS_NATIVE_SHORTHAND=1"
}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { jayess_native_add as add } from "./native/math.native.json";

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
	if !strings.Contains(string(result.LLVMIR), "@jayess_native_add(") {
		t.Fatalf("expected implicit native manifest symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected 2 native sources, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/native/include") {
		t.Fatalf("expected native include directory to be carried through, got %#v", result.NativeIncludeDirs)
	}
	if len(result.NativeCompileFlags) != 1 || result.NativeCompileFlags[0] != "-DJAYESS_NATIVE_SHORTHAND=1" {
		t.Fatalf("expected native compile flags to be carried through, got %#v", result.NativeCompileFlags)
	}
}

func TestCompilePathSupportsPackageNativeImports(t *testing.T) {
	t.Skip("legacy direct native imports removed; use .bind.js")
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "demo-native-pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"demo-native-pkg","native":"native/math.c"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pkgDir, "native"), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "native", "math.c"), []byte(`#include "jayess_runtime.h"
jayess_value *jayess_native_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b) + 10);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { jayess_native_add as add } from "demo-native-pkg/native/math.c";

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
	if !strings.Contains(string(result.LLVMIR), "@jayess_native_add(") {
		t.Fatalf("expected package native symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeImports[0]), "/node_modules/demo-native-pkg/native/math.c") {
		t.Fatalf("expected package native import to be carried through, got %#v", result.NativeImports)
	}
}

func TestCompilePathSupportsDiscoveredNativeExportAliases(t *testing.T) {
	t.Skip("legacy direct native imports removed; use .bind.js")
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(js_arg_number(js, 0) + js_arg_number(js, 1));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "./native/math.c";

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
	if !strings.Contains(string(result.LLVMIR), "@jayess_native_add(") {
		t.Fatalf("expected discovered native alias to map to jayess_native_add, got:\n%s", string(result.LLVMIR))
	}
}

func TestCompilePathSupportsZeroManifestNativePackageDirectory(t *testing.T) {
	t.Skip("legacy zero-manifest native imports removed; use .bind.js")
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "demo-zero-native-pkg")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double jayess_native_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "index.c"), []byte(`#include "jayess_runtime.h"
#include "native_math.h"

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(jayess_native_helper(js_arg_number(js, 0), js_arg_number(js, 1)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(`#include "native_math.h"

double jayess_native_helper(double left, double right) {
    return left + right + 30;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "demo-zero-native-pkg/native";

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
	if !strings.Contains(string(result.LLVMIR), "@jayess_native_add(") {
		t.Fatalf("expected zero-manifest native package alias to map to jayess_native_add, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/node_modules/demo-zero-native-pkg/native/include") {
		t.Fatalf("expected inferred native include dir, got %#v", result.NativeIncludeDirs)
	}
}

func TestCompilePathSupportsRecursiveZeroManifestNativePackageDirectives(t *testing.T) {
	t.Skip("legacy zero-manifest native imports removed; use .bind.js")
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "demo-config-native-pkg")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	helpersDir := filepath.Join(nativeDir, "helpers")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(helpersDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double jayess_native_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "index.c"), []byte(`// jayess:include ./include
// jayess:cflag -DJAYESS_NATIVE_BONUS=13
// jayess:ldflag -ljayessdemo
#include "jayess_runtime.h"
#include "native_math.h"

JAYESS_EXPORT2(jayess_native_add) {
    return js_number(jayess_native_helper(js_arg_number(js, 0), js_arg_number(js, 1)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(helpersDir, "math_helper.c"), []byte(`#include "native_math.h"

double jayess_native_helper(double left, double right) {
    return left + right + JAYESS_NATIVE_BONUS;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "demo-config-native-pkg/native";

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
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected recursive native source discovery to include helper source, got %#v", result.NativeImports)
	}
	if len(result.NativeCompileFlags) != 1 || result.NativeCompileFlags[0] != "-DJAYESS_NATIVE_BONUS=13" {
		t.Fatalf("expected native directive compile flag, got %#v", result.NativeCompileFlags)
	}
	if len(result.NativeLinkFlags) != 1 || result.NativeLinkFlags[0] != "-ljayessdemo" {
		t.Fatalf("expected native directive link flag, got %#v", result.NativeLinkFlags)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/node_modules/demo-config-native-pkg/native/include") {
		t.Fatalf("expected native directive include dir, got %#v", result.NativeIncludeDirs)
	}
}

func TestCompilePathSupportsManualBindFiles(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "native_math.h"), []byte(`#pragma once
double mylib_helper(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"
#include "native_math.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(mylib_helper(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "helper.c"), []byte(`#include "native_math.h"

double mylib_helper(double left, double right) {
    return left + right + 5;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.bind.js"), []byte(`export default {
  sources: ["./math.c", "./helper.c"],
  includeDirs: ["./include"],
  cflags: ["-DMANUAL_BIND=1"],
  ldflags: ["-lm"],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "./native/math.bind.js";

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
	if !strings.Contains(string(result.LLVMIR), "@mylib_add(") {
		t.Fatalf("expected manual bind symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected 2 native sources from bind file, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/native/include") {
		t.Fatalf("expected bind include dir, got %#v", result.NativeIncludeDirs)
	}
	if len(result.NativeCompileFlags) != 1 || result.NativeCompileFlags[0] != "-DMANUAL_BIND=1" {
		t.Fatalf("expected bind compile flags, got %#v", result.NativeCompileFlags)
	}
	if len(result.NativeLinkFlags) != 1 || result.NativeLinkFlags[0] != "-lm" {
		t.Fatalf("expected bind link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsManualSDLAudioBindFiles(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	repoRoot := repoRootFromCompilerTest(t)
	sdlIncludeDir := filepath.ToSlash(filepath.Join(repoRoot, "refs", "SDL", "include"))
	if err := os.WriteFile(filepath.Join(nativeDir, "sdl_audio.c"), []byte(`#include "jayess_runtime.h"
#include <SDL3/SDL.h>

jayess_value *jayess_sdl_audio_driver_count(void) {
    return jayess_value_from_number((double) SDL_GetNumAudioDrivers());
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "sdl_audio_stub.c"), []byte(`#include <SDL3/SDL.h>

int SDL_GetNumAudioDrivers(void) {
    return 1;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "sdl_audio.bind.js"), []byte(`const f = () => {};
export const getNumAudioDrivers = f;

export default {
  sources: ["./sdl_audio.c", "./sdl_audio_stub.c"],
  includeDirs: [`+"\""+sdlIncludeDir+"\""+`],
  cflags: [],
  ldflags: [],
  exports: {
    getNumAudioDrivers: { symbol: "jayess_sdl_audio_driver_count", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { getNumAudioDrivers } from "./native/sdl_audio.bind.js";

function main(args) {
  return getNumAudioDrivers();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_sdl_audio_driver_count(") {
		t.Fatalf("expected SDL audio bind symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected 2 native sources from SDL audio bind file, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/refs/SDL/include") {
		t.Fatalf("expected SDL include dir from bind file, got %#v", result.NativeIncludeDirs)
	}
}

func TestCompilePathSupportsManualOpenALBindFiles(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	includeDir := filepath.Join(nativeDir, "include", "AL")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "al.h"), []byte(`#pragma once
typedef char ALchar;
typedef int ALenum;
#define AL_VENDOR 0xB001
const ALchar *alGetString(ALenum param);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "alc.h"), []byte(`#pragma once
typedef char ALCchar;
typedef int ALCenum;
typedef int ALCint;
typedef struct ALCdevice_struct ALCdevice;
typedef struct ALCcontext_struct ALCcontext;
#define ALC_DEFAULT_DEVICE_SPECIFIER 0x1004
ALCdevice *alcOpenDevice(const ALCchar *devicename);
int alcCloseDevice(ALCdevice *device);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openal.c"), []byte(`#include "jayess_runtime.h"
#include <AL/al.h>
#include <AL/alc.h>

jayess_value *jayess_openal_default_device_name(void) {
    const ALCchar *name = alcOpenDevice(NULL) != NULL ? "stub-openal-device" : NULL;
    if (name == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(name);
}

jayess_value *jayess_openal_vendor_name(void) {
    const ALchar *vendor = alGetString(AL_VENDOR);
    if (vendor == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(vendor);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openal_stub.c"), []byte(`#include <AL/al.h>
#include <AL/alc.h>

struct ALCdevice_struct { int open; };
static struct ALCdevice_struct jayess_openal_device = { 1 };

ALCdevice *alcOpenDevice(const ALCchar *devicename) {
    (void)devicename;
    return &jayess_openal_device;
}

int alcCloseDevice(ALCdevice *device) {
    return device != 0;
}

const ALchar *alGetString(ALenum param) {
    return param == AL_VENDOR ? "stub-openal" : 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "openal.bind.js"), []byte(`const f = () => {};
export const getDefaultDeviceName = f;
export const getVendorName = f;

export default {
  sources: ["./openal.c", "./openal_stub.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    getDefaultDeviceName: { symbol: "jayess_openal_default_device_name", type: "function" },
    getVendorName: { symbol: "jayess_openal_vendor_name", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { getDefaultDeviceName, getVendorName } from "./native/openal.bind.js";

function main(args) {
  return (getDefaultDeviceName() + ":" + getVendorName()).length;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_openal_default_device_name(") {
		t.Fatalf("expected OpenAL bind symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected 2 native sources from OpenAL bind file, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/native/include") {
		t.Fatalf("expected OpenAL include dir from bind file, got %#v", result.NativeIncludeDirs)
	}
}

func TestCompilePathSupportsManualMiniaudioBindFiles(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	repoRoot := repoRootFromCompilerTest(t)
	miniaudioDir := filepath.ToSlash(filepath.Join(repoRoot, "refs", "miniaudio"))
	if err := os.WriteFile(filepath.Join(nativeDir, "miniaudio_wrapper.c"), []byte(`#include "jayess_runtime.h"
#include "miniaudio.h"

jayess_value *jayess_miniaudio_backend_name(void) {
    ma_backend backends[] = { ma_backend_null };
    ma_context context;
    ma_result result = ma_context_init(backends, 1, NULL, &context);
    if (result != MA_SUCCESS) {
        return jayess_value_undefined();
    }
    jayess_value *value = jayess_value_from_string(ma_get_backend_name(context.backend));
    ma_context_uninit(&context);
    return value;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	miniaudioBind := "const f = () => {};\n" +
		"export const getBackendName = f;\n\n" +
		"export default {\n" +
		"  sources: [\"./miniaudio_wrapper.c\", " + strconv.Quote(filepath.ToSlash(filepath.Join(miniaudioDir, "miniaudio.c"))) + "],\n" +
		"  includeDirs: [" + strconv.Quote(filepath.ToSlash(miniaudioDir)) + "],\n" +
		"  cflags: [\"-DMA_ENABLE_ONLY_NULL\"],\n" +
		"  ldflags: [\"-pthread\", \"-ldl\", \"-lm\"],\n" +
		"  exports: {\n" +
		"    getBackendName: { symbol: \"jayess_miniaudio_backend_name\", type: \"function\" }\n" +
		"  }\n" +
		"};\n"
	if err := os.WriteFile(filepath.Join(nativeDir, "miniaudio.bind.js"), []byte(miniaudioBind), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { getBackendName } from "./native/miniaudio.bind.js";

function main(args) {
  return getBackendName().length;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_miniaudio_backend_name(") {
		t.Fatalf("expected miniaudio bind symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected 2 native sources from miniaudio bind file, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/refs/miniaudio") {
		t.Fatalf("expected miniaudio include dir from bind file, got %#v", result.NativeIncludeDirs)
	}
	if len(result.NativeCompileFlags) != 1 || result.NativeCompileFlags[0] != "-DMA_ENABLE_ONLY_NULL" {
		t.Fatalf("expected miniaudio compile flags from bind file, got %#v", result.NativeCompileFlags)
	}
}

func TestCompilePathSupportsManualPortAudioBindFiles(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	repoRoot := repoRootFromCompilerTest(t)
	portaudioIncludeDir := filepath.ToSlash(filepath.Join(repoRoot, "refs", "portaudio", "include"))
	if err := os.WriteFile(filepath.Join(nativeDir, "portaudio_wrapper.c"), []byte(`#include "jayess_runtime.h"
#include <portaudio.h>

jayess_value *jayess_portaudio_version_text(void) {
    const char *text = Pa_GetVersionText();
    if (text == NULL) {
        return jayess_value_undefined();
    }
    return jayess_value_from_string(text);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "portaudio_stub.c"), []byte(`#include <portaudio.h>

const char* Pa_GetVersionText(void) {
    return "stub-portaudio";
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "portaudio.bind.js"), []byte(`const f = () => {};
export const getVersionText = f;

export default {
  sources: ["./portaudio_wrapper.c", "./portaudio_stub.c"],
  includeDirs: [`+"\""+portaudioIncludeDir+"\""+`],
  cflags: [],
  ldflags: [],
  exports: {
    getVersionText: { symbol: "jayess_portaudio_version_text", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { getVersionText } from "./native/portaudio.bind.js";

function main(args) {
  return getVersionText().length;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_portaudio_version_text(") {
		t.Fatalf("expected PortAudio bind symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected 2 native sources from PortAudio bind file, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/refs/portaudio/include") {
		t.Fatalf("expected PortAudio include dir from bind file, got %#v", result.NativeIncludeDirs)
	}
}

func TestCompilePathSupportsPackageLocalBindFiles(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "demo-bind-pkg")
	nativeDir := filepath.Join(pkgDir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b) + 9);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.bind.js"), []byte(`const f = () => {};
export const add = f;

export default {
  sources: ["./native/math.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "demo-bind-pkg";

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
	if !strings.Contains(string(result.LLVMIR), "@mylib_add(") {
		t.Fatalf("expected package bind symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeImports[0]), "/node_modules/demo-bind-pkg/native/math.c") {
		t.Fatalf("expected package bind native import to be carried through, got %#v", result.NativeImports)
	}
}

func TestCompilePathAllowsPlaceholderExportsInBindFiles(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(jayess_value_to_number(a) + jayess_value_to_number(b));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.bind.js"), []byte(`const f = () => {};
export const add = f;

export default {
  sources: ["./math.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "./native/math.bind.js";

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
	if !strings.Contains(string(result.LLVMIR), "@mylib_add(") {
		t.Fatalf("expected placeholder-export bind symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
}

func TestCompilePathRejectsMalformedBindFilesClearly(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`#include "jayess_runtime.h"
jayess_value *mylib_add(jayess_value *a, jayess_value *b) { return a; }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.bind.js"), []byte(`export default {
  sources: ["./math.c"],
  includeDirs: [],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "mystery" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "./native/math.bind.js";

function main(args) {
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err == nil {
		t.Fatalf("expected malformed bind diagnostic")
	}
	if !strings.Contains(err.Error(), "unsupported type") {
		t.Fatalf("expected malformed bind diagnostic, got: %v", err)
	}
}

func TestCompilePathDistinguishesBindModulesFromJayessSourceModules(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.c"), []byte(`jayess_value *mylib_add(jayess_value *a, jayess_value *b) { return a; }`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "math.bind.js"), []byte(`const f = () => {};
export const add = f;
export default {
  sources: ["./math.c"],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	for _, tc := range []struct {
		name   string
		source string
	}{
		{
			name: "bare",
			source: `import "./native/math.bind.js";
function main(args) { return 0; }`,
		},
		{
			name: "default",
			source: `import native from "./native/math.bind.js";
function main(args) { return 0; }`,
		},
		{
			name: "namespace",
			source: `import * as native from "./native/math.bind.js";
function main(args) { return 0; }`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := filepath.Join(dir, tc.name+".js")
			if err := os.WriteFile(entry, []byte(tc.source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			_, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
			if err == nil {
				t.Fatalf("expected bind-module distinction diagnostic")
			}
			if !strings.Contains(err.Error(), "native binding modules are not Jayess source modules") {
				t.Fatalf("expected bind-module distinction diagnostic, got: %v", err)
			}
		})
	}
}

func TestCompilePathSupportsBindValueExports(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "value.c"), []byte(`#include "jayess_runtime.h"
jayess_value *mylib_version_value(void) { return jayess_value_from_number(7); }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "value.bind.js"), []byte(`const f = () => {};
export const version = 0;
export default {
  sources: ["./value.c"],
  exports: {
    version: { symbol: "mylib_version_value", type: "value" }
  }
};`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { version } from "./native/value.bind.js";

function main(args) {
  return version;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	if !strings.Contains(irText, "@mylib_version_value(") {
		t.Fatalf("expected bind value getter symbol in LLVM IR, got:\n%s", irText)
	}
	if !strings.Contains(irText, "@jayess_global_version") || !strings.Contains(irText, "call ptr @mylib_version_value()") {
		t.Fatalf("expected bind value global initialization in LLVM IR, got:\n%s", irText)
	}
}

func TestCompilePathSupportsJayessGLFWPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "glfw")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(dir, "refs", "glfw", "include", "GLFW")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "glfw3.h"), []byte(`#pragma once
typedef struct GLFWwindow GLFWwindow;
int glfwInit(void);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/glfw","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
	import { initNative, createOpenGLWindowNative, makeContextCurrentNative, isContextCurrentNative, swapIntervalNative, getProcAddressNative, hasProcAddressNative, isVulkanSupportedNative, getRequiredVulkanInstanceExtensionsNative, getVulkanInstanceProcAddressNative, createVulkanSurfaceNative } from "./native/glfw.bind.js";
import { setKeyCallbackNative } from "./native/glfw.bind.js";

export function init() {
  return initNative();
}

export function createOpenGLWindow(width, height, title) {
  return createOpenGLWindowNative(width, height, title);
}

export function makeContextCurrent(window) {
  return makeContextCurrentNative(window);
}

export function isContextCurrent(window) {
  return isContextCurrentNative(window);
}

export function swapInterval(interval) {
  return swapIntervalNative(interval);
}

export function getProcAddress(name) {
  return getProcAddressNative(name);
}

export function hasProcAddress(name) {
  return hasProcAddressNative(name);
}

export function isVulkanSupported() {
  return isVulkanSupportedNative();
}

export function getRequiredVulkanInstanceExtensions() {
  return getRequiredVulkanInstanceExtensionsNative();
}

export function getVulkanInstanceProcAddress(name) {
  return getVulkanInstanceProcAddressNative(name);
}

export function createVulkanSurface(window, instance) {
  return createVulkanSurfaceNative(window, instance);
}

export function setKeyCallback(window, callback) {
  return setKeyCallbackNative(window, callback);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "glfw.c"), []byte(`#include "jayess_runtime.h"
#include <GLFW/glfw3.h>

jayess_value *jayess_glfw_init(void) {
    return jayess_value_from_bool(glfwInit());
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "glfw.bind.js"), []byte(`const f = () => {};
export const initNative = f;
export const createOpenGLWindowNative = f;
export const makeContextCurrentNative = f;
export const isContextCurrentNative = f;
export const swapIntervalNative = f;
export const getProcAddressNative = f;
export const hasProcAddressNative = f;
export const isVulkanSupportedNative = f;
export const getRequiredVulkanInstanceExtensionsNative = f;
export const getVulkanInstanceProcAddressNative = f;
export const createVulkanSurfaceNative = f;
export const setKeyCallbackNative = f;

export default {
  sources: ["./glfw.c"],
  includeDirs: ["../../../../refs/glfw/include"],
  cflags: [],
  ldflags: ["-lglfw"],
  exports: {
    initNative: { symbol: "jayess_glfw_init", type: "function" },
    createOpenGLWindowNative: { symbol: "jayess_glfw_create_opengl_window", type: "function" },
    makeContextCurrentNative: { symbol: "jayess_glfw_make_context_current", type: "function" },
    isContextCurrentNative: { symbol: "jayess_glfw_is_context_current", type: "function" },
    swapIntervalNative: { symbol: "jayess_glfw_swap_interval", type: "function" },
    getProcAddressNative: { symbol: "jayess_glfw_get_proc_address", type: "function" },
    hasProcAddressNative: { symbol: "jayess_glfw_has_proc_address", type: "function" },
    isVulkanSupportedNative: { symbol: "jayess_glfw_is_vulkan_supported", type: "function" },
    getRequiredVulkanInstanceExtensionsNative: { symbol: "jayess_glfw_get_required_vulkan_instance_extensions", type: "function" },
    getVulkanInstanceProcAddressNative: { symbol: "jayess_glfw_get_vulkan_instance_proc_address", type: "function" },
    createVulkanSurfaceNative: { symbol: "jayess_glfw_create_vulkan_surface", type: "function" },
    setKeyCallbackNative: { symbol: "jayess_glfw_set_key_callback", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
	import { init, createOpenGLWindow, makeContextCurrent, isContextCurrent, swapInterval, getProcAddress, hasProcAddress, isVulkanSupported, getRequiredVulkanInstanceExtensions, getVulkanInstanceProcAddress, createVulkanSurface, setKeyCallback } from "@jayess/glfw";

function main(args) {
  var window = createOpenGLWindow(32, 24, "gl");
  makeContextCurrent(window);
  swapInterval(1);
  getProcAddress("glClear");
  hasProcAddress("glClear");
  isVulkanSupported();
  getRequiredVulkanInstanceExtensions();
  getVulkanInstanceProcAddress("vkCreateInstance");
  createVulkanSurface(undefined, 1n);
  isContextCurrent(window);
  setKeyCallback(undefined, function (event) { return event.key; });
  return init();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	for _, symbol := range []string{
		"@jayess_glfw_init(",
		"@jayess_glfw_create_opengl_window(",
		"@jayess_glfw_make_context_current(",
		"@jayess_glfw_is_context_current(",
		"@jayess_glfw_swap_interval(",
		"@jayess_glfw_get_proc_address(",
		"@jayess_glfw_has_proc_address(",
		"@jayess_glfw_is_vulkan_supported(",
		"@jayess_glfw_get_required_vulkan_instance_extensions(",
		"@jayess_glfw_get_vulkan_instance_proc_address(",
		"@jayess_glfw_create_vulkan_surface(",
		"@jayess_glfw_set_key_callback(",
	} {
		if !strings.Contains(string(result.LLVMIR), symbol) {
			t.Fatalf("expected GLFW package symbol %q in LLVM IR, got:\n%s", symbol, string(result.LLVMIR))
		}
	}
	if len(result.NativeImports) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeImports[0]), "/node_modules/@jayess/glfw/native/glfw.c") {
		t.Fatalf("expected GLFW native import to be carried through, got %#v", result.NativeImports)
	}
	if len(result.NativeLinkFlags) != 1 || result.NativeLinkFlags[0] != "-lglfw" {
		t.Fatalf("expected GLFW link flag, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessRaylibPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "raylib")
	nativeDir := filepath.Join(pkgDir, "native")
	refsDir := filepath.Join(dir, "refs", "raylib", "src")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "raylib.h"), []byte("#pragma once\ntypedef struct Color { unsigned char r, g, b, a; } Color;\nvoid InitWindow(int width, int height, const char *title);\nvoid CloseWindow(void);\nint IsWindowReady(void);\nint WindowShouldClose(void);\nvoid SetWindowTitle(const char *title);\nvoid BeginDrawing(void);\nvoid EndDrawing(void);\nvoid ClearBackground(Color color);\nvoid DrawCircle(int x, int y, float radius, Color color);\nvoid DrawText(const char *text, int posX, int posY, int fontSize, Color color);\nvoid SetTargetFPS(int fps);\nfloat GetFrameTime(void);\ndouble GetTime(void);\nvoid SetRandomSeed(unsigned int seed);\nint GetRandomValue(int min, int max);\nvoid SetTraceLogLevel(int logLevel);\nvoid SetConfigFlags(unsigned int flags);\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/raylib","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
	import { initWindowNative, isWindowReadyNative, drawTextNative } from "./native/raylib.bind.js";

export function initWindow(width, height, title) {
  return initWindowNative(width, height, title);
}

export function isWindowReady() {
  return isWindowReadyNative();
}

export function drawText(text, x, y, size, color) {
  return drawTextNative(text, x, y, size, color);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "raylib.c"), []byte(`#include "jayess_runtime.h"
#include <raylib.h>

jayess_value *jayess_raylib_init_window(jayess_value *width_value, jayess_value *height_value, jayess_value *title_value) {
    InitWindow((int)jayess_value_to_number(width_value), (int)jayess_value_to_number(height_value), jayess_expect_string(title_value, "jayess_raylib_init_window"));
    if (jayess_has_exception()) return jayess_value_undefined();
    return jayess_value_from_bool(IsWindowReady());
}

jayess_value *jayess_raylib_is_window_ready(void) {
    return jayess_value_from_bool(IsWindowReady());
}

jayess_value *jayess_raylib_draw_text(jayess_value *text_value, jayess_value *x_value, jayess_value *y_value, jayess_value *size_value, jayess_value *color_value) {
    DrawText(jayess_expect_string(text_value, "jayess_raylib_draw_text"), (int)jayess_value_to_number(x_value), (int)jayess_value_to_number(y_value), (int)jayess_value_to_number(size_value), (Color){255, 255, 255, 255});
    if (jayess_has_exception()) return jayess_value_undefined();
    (void)color_value;
    return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "raylib.bind.js"), []byte(`const f = () => {};
export const initWindowNative = f;
export const isWindowReadyNative = f;
export const drawTextNative = f;

export default {
  sources: ["./raylib.c"],
  includeDirs: ["../../../../refs/raylib/src"],
  cflags: ["-DPLATFORM_MEMORY", "-DGRAPHICS_API_OPENGL_SOFTWARE"],
  ldflags: ["-lm"],
  exports: {
    initWindowNative: { symbol: "jayess_raylib_init_window", type: "function" },
    isWindowReadyNative: { symbol: "jayess_raylib_is_window_ready", type: "function" },
    drawTextNative: { symbol: "jayess_raylib_draw_text", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { initWindow, isWindowReady, drawText } from "@jayess/raylib";

function main(args) {
  initWindow(64, 64, "compile");
  drawText("ok", 1, 1, 12, { r: 255, g: 255, b: 255, a: 255 });
  return isWindowReady();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if len(result.NativeImports) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeImports[0]), "/node_modules/@jayess/raylib/native/raylib.c") {
		t.Fatalf("expected raylib native import, got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/refs/raylib/src") {
		t.Fatalf("expected raylib include dir, got %#v", result.NativeIncludeDirs)
	}
	if len(result.NativeLinkFlags) != 1 || result.NativeLinkFlags[0] != "-lm" {
		t.Fatalf("expected raylib link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessAudioPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "audio")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(dir, "refs", "cubeb", "include", "cubeb")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "cubeb.h"), []byte(`#pragma once
typedef struct cubeb cubeb;
typedef unsigned int uint32_t;
#define CUBEB_DEVICE_TYPE_INPUT 1
#define CUBEB_DEVICE_TYPE_OUTPUT 2
#define CUBEB_DEVICE_STATE_DISABLED 0
#define CUBEB_DEVICE_STATE_UNPLUGGED 1
#define CUBEB_DEVICE_STATE_ENABLED 2
#define CUBEB_DEVICE_PREF_NONE 0x00
#define CUBEB_DEVICE_PREF_MULTIMEDIA 0x01
#define CUBEB_DEVICE_PREF_VOICE 0x02
#define CUBEB_DEVICE_PREF_NOTIFICATION 0x04
#define CUBEB_DEVICE_FMT_S16LE 0x0010
#define CUBEB_DEVICE_FMT_S16BE 0x0020
#define CUBEB_DEVICE_FMT_F32LE 0x1000
#define CUBEB_DEVICE_FMT_F32BE 0x2000
typedef int cubeb_device_type;
typedef int cubeb_device_state;
typedef int cubeb_device_pref;
typedef int cubeb_device_fmt;
typedef struct {
  void * devid;
  char const * device_id;
  char const * friendly_name;
  char const * group_id;
  char const * vendor_name;
  cubeb_device_type type;
  cubeb_device_state state;
  cubeb_device_pref preferred;
  cubeb_device_fmt format;
  cubeb_device_fmt default_format;
  uint32_t max_channels;
  uint32_t default_rate;
  uint32_t max_rate;
  uint32_t min_rate;
  uint32_t latency_lo;
  uint32_t latency_hi;
} cubeb_device_info;
typedef struct {
  cubeb_device_info * device;
  unsigned long count;
} cubeb_device_collection;
#define CUBEB_OK 0
int cubeb_init(cubeb ** context, char const * context_name, char const * backend_name);
char const * cubeb_get_backend_id(cubeb * context);
int cubeb_get_max_channel_count(cubeb * context, unsigned int * max_channels);
int cubeb_enumerate_devices(cubeb * context, cubeb_device_type devtype, cubeb_device_collection * collection);
int cubeb_device_collection_destroy(cubeb * context, cubeb_device_collection * collection);
void cubeb_destroy(cubeb * context);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/audio","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { createContextNative, listOutputDevicesNative, listInputDevicesNative } from "./native/audio.bind.js";
import { preferredSampleRateNative, minLatencyNative, createPlaybackStreamNative, startPlaybackStreamNative, pausePlaybackStreamNative, stopPlaybackStreamNative, submitPlaybackSamplesNative, playbackStatsNative, closePlaybackStreamNative, nextStreamEventNative, createCaptureStreamNative, startCaptureStreamNative, stopCaptureStreamNative, readCapturedSamplesNative, captureStatsNative, closeCaptureStreamNative, loadWavNative, loadOggNative, loadMp3Native, loadFlacNative } from "./native/audio.bind.js";

export function createContext(name, backendName) {
  return createContextNative(name, backendName);
}

export function listOutputDevices(context) {
  return listOutputDevicesNative(context);
}

export function listInputDevices(context) {
  return listInputDevicesNative(context);
}

export function preferredSampleRate(context) {
  return preferredSampleRateNative(context);
}

export function minLatency(context, options) {
  return minLatencyNative(context, options);
}

export function createPlaybackStream(context, options) {
  return createPlaybackStreamNative(context, options);
}

export function startPlaybackStream(stream) {
  return startPlaybackStreamNative(stream);
}

export function pausePlaybackStream(stream) {
  return pausePlaybackStreamNative(stream);
}

export function stopPlaybackStream(stream) {
  return stopPlaybackStreamNative(stream);
}

export function submitPlaybackSamples(stream, samples) {
  return submitPlaybackSamplesNative(stream, samples);
}

export function playbackStats(stream) {
  return playbackStatsNative(stream);
}

export function closePlaybackStream(stream) {
  return closePlaybackStreamNative(stream);
}

export function nextStreamEvent(stream) {
  return nextStreamEventNative(stream);
}

export function createCaptureStream(context, options) {
  return createCaptureStreamNative(context, options);
}

export function startCaptureStream(stream) {
  return startCaptureStreamNative(stream);
}

export function stopCaptureStream(stream) {
  return stopCaptureStreamNative(stream);
}

export function readCapturedSamples(stream, frames) {
  return readCapturedSamplesNative(stream, frames);
}

export function captureStats(stream) {
  return captureStatsNative(stream);
}

export function closeCaptureStream(stream) {
  return closeCaptureStreamNative(stream);
}

export function loadWav(path) {
  return loadWavNative(path);
}

export function loadOgg(path) {
  return loadOggNative(path);
}

export function loadMp3(path) {
  return loadMp3Native(path);
}

export function loadFlac(path) {
  return loadFlacNative(path);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "audio.c"), []byte(`#include "jayess_runtime.h"
#include <cubeb/cubeb.h>

jayess_value *jayess_audio_create_context(jayess_value *name_value, jayess_value *backend_value) {
    (void) name_value;
    (void) backend_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_list_output_devices(jayess_value *context_value) {
    (void) context_value;
    return jayess_value_undefined();
}

jayess_value *jayess_audio_list_input_devices(jayess_value *context_value) {
    (void) context_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_preferred_sample_rate(jayess_value *context_value) {
    (void) context_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_min_latency(jayess_value *context_value, jayess_value *options_value) {
    (void) context_value;
    (void) options_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_create_playback_stream(jayess_value *context_value, jayess_value *options_value) {
    (void) context_value;
    (void) options_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_start_playback_stream(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_pause_playback_stream(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_stop_playback_stream(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_submit_playback_samples(jayess_value *stream_value, jayess_value *samples_value) {
    (void) stream_value;
    (void) samples_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_playback_stats(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_close_playback_stream(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_next_stream_event(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_create_capture_stream(jayess_value *context_value, jayess_value *options_value) {
    (void) context_value;
    (void) options_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_start_capture_stream(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_stop_capture_stream(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_read_captured_samples(jayess_value *stream_value, jayess_value *frames_value) {
    (void) stream_value;
    (void) frames_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_capture_stats(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_close_capture_stream(jayess_value *stream_value) {
    (void) stream_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_load_wav(jayess_value *path_value) {
    (void) path_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_load_ogg(jayess_value *path_value) {
    (void) path_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_load_mp3(jayess_value *path_value) {
    (void) path_value;
    return jayess_value_undefined();
}
jayess_value *jayess_audio_load_flac(jayess_value *path_value) {
    (void) path_value;
    return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "audio.bind.js"), []byte(`const f = () => {};
export const createContextNative = f;
export const listOutputDevicesNative = f;
export const listInputDevicesNative = f;
export const preferredSampleRateNative = f;
export const minLatencyNative = f;
export const createPlaybackStreamNative = f;
export const startPlaybackStreamNative = f;
export const pausePlaybackStreamNative = f;
export const stopPlaybackStreamNative = f;
export const submitPlaybackSamplesNative = f;
export const playbackStatsNative = f;
export const closePlaybackStreamNative = f;
export const nextStreamEventNative = f;
export const createCaptureStreamNative = f;
export const startCaptureStreamNative = f;
export const stopCaptureStreamNative = f;
export const readCapturedSamplesNative = f;
export const captureStatsNative = f;
export const closeCaptureStreamNative = f;
export const loadWavNative = f;
export const loadOggNative = f;
export const loadMp3Native = f;
export const loadFlacNative = f;

export default {
  sources: ["./audio.c"],
  includeDirs: ["../../../../refs/cubeb/include"],
  cflags: [],
  ldflags: ["-lcubeb", "-pthread"],
  exports: {
    createContextNative: { symbol: "jayess_audio_create_context", type: "function" },
    listOutputDevicesNative: { symbol: "jayess_audio_list_output_devices", type: "function" },
    listInputDevicesNative: { symbol: "jayess_audio_list_input_devices", type: "function" },
    preferredSampleRateNative: { symbol: "jayess_audio_preferred_sample_rate", type: "function" },
    minLatencyNative: { symbol: "jayess_audio_min_latency", type: "function" },
    createPlaybackStreamNative: { symbol: "jayess_audio_create_playback_stream", type: "function" },
    startPlaybackStreamNative: { symbol: "jayess_audio_start_playback_stream", type: "function" },
    pausePlaybackStreamNative: { symbol: "jayess_audio_pause_playback_stream", type: "function" },
    stopPlaybackStreamNative: { symbol: "jayess_audio_stop_playback_stream", type: "function" },
    submitPlaybackSamplesNative: { symbol: "jayess_audio_submit_playback_samples", type: "function" },
    playbackStatsNative: { symbol: "jayess_audio_playback_stats", type: "function" },
    closePlaybackStreamNative: { symbol: "jayess_audio_close_playback_stream", type: "function" },
    nextStreamEventNative: { symbol: "jayess_audio_next_stream_event", type: "function" },
    createCaptureStreamNative: { symbol: "jayess_audio_create_capture_stream", type: "function" },
    startCaptureStreamNative: { symbol: "jayess_audio_start_capture_stream", type: "function" },
    stopCaptureStreamNative: { symbol: "jayess_audio_stop_capture_stream", type: "function" },
    readCapturedSamplesNative: { symbol: "jayess_audio_read_captured_samples", type: "function" },
    captureStatsNative: { symbol: "jayess_audio_capture_stats", type: "function" },
    closeCaptureStreamNative: { symbol: "jayess_audio_close_capture_stream", type: "function" },
    loadWavNative: { symbol: "jayess_audio_load_wav", type: "function" },
    loadOggNative: { symbol: "jayess_audio_load_ogg", type: "function" },
    loadMp3Native: { symbol: "jayess_audio_load_mp3", type: "function" },
    loadFlacNative: { symbol: "jayess_audio_load_flac", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createContext, listOutputDevices, listInputDevices, preferredSampleRate, minLatency, createPlaybackStream, startPlaybackStream, pausePlaybackStream, stopPlaybackStream, submitPlaybackSamples, playbackStats, closePlaybackStream, nextStreamEvent, createCaptureStream, startCaptureStream, stopCaptureStream, readCapturedSamples, captureStats, closeCaptureStream, loadWav, loadOgg, loadMp3, loadFlac } from "@jayess/audio";

function main(args) {
  var ctx = createContext("jayess-test", null);
  listOutputDevices(ctx);
  listInputDevices(ctx);
  var rate = preferredSampleRate(ctx);
  minLatency(ctx, { sampleRate: rate, channels: 2, format: "f32" });
  var stream = createPlaybackStream(ctx, { sampleRate: 48000, channels: 2, format: "f32" });
  startPlaybackStream(stream);
  pausePlaybackStream(stream);
  stopPlaybackStream(stream);
  submitPlaybackSamples(stream, [0, 0, 0, 0]);
  playbackStats(stream);
  nextStreamEvent(stream);
  closePlaybackStream(stream);
  var capture = createCaptureStream(ctx, { sampleRate: 48000, channels: 1, format: "f32" });
  startCaptureStream(capture);
  readCapturedSamples(capture, 4);
  captureStats(capture);
  stopCaptureStream(capture);
  closeCaptureStream(capture);
  loadWav("sample.wav");
  loadOgg("sample.ogg");
  loadMp3("sample.mp3");
  loadFlac("sample.flac");
  return ctx;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_audio_create_context(") {
		t.Fatalf("expected audio package symbol in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	for _, symbol := range []string{
		"@jayess_audio_list_output_devices(",
		"@jayess_audio_list_input_devices(",
		"@jayess_audio_preferred_sample_rate(",
		"@jayess_audio_min_latency(",
		"@jayess_audio_create_playback_stream(",
		"@jayess_audio_start_playback_stream(",
		"@jayess_audio_pause_playback_stream(",
		"@jayess_audio_stop_playback_stream(",
		"@jayess_audio_submit_playback_samples(",
		"@jayess_audio_playback_stats(",
		"@jayess_audio_close_playback_stream(",
		"@jayess_audio_next_stream_event(",
		"@jayess_audio_create_capture_stream(",
		"@jayess_audio_start_capture_stream(",
		"@jayess_audio_stop_capture_stream(",
		"@jayess_audio_read_captured_samples(",
		"@jayess_audio_capture_stats(",
		"@jayess_audio_close_capture_stream(",
		"@jayess_audio_load_wav(",
	} {
		if !strings.Contains(string(result.LLVMIR), symbol) {
			t.Fatalf("expected audio package symbol %q in LLVM IR, got:\n%s", symbol, string(result.LLVMIR))
		}
	}
	if len(result.NativeImports) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeImports[0]), "/node_modules/@jayess/audio/native/audio.c") {
		t.Fatalf("expected audio native import to be carried through, got %#v", result.NativeImports)
	}
	if len(result.NativeLinkFlags) != 2 || result.NativeLinkFlags[0] != "-lcubeb" || result.NativeLinkFlags[1] != "-pthread" {
		t.Fatalf("expected audio link flag, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessWebviewPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "webview")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(dir, "refs", "webview", "core", "include", "webview")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "webview.h"), []byte(`#pragma once
typedef void *webview_t;
typedef enum webview_error_t { WEBVIEW_ERROR_OK = 0 } webview_error_t;
typedef enum webview_hint_t { WEBVIEW_HINT_NONE = 0 } webview_hint_t;
webview_t webview_create(int debug, void *window);
webview_error_t webview_destroy(webview_t view);
webview_error_t webview_set_title(webview_t view, const char *title);
webview_error_t webview_set_size(webview_t view, int width, int height, webview_hint_t hint);
webview_error_t webview_set_html(webview_t view, const char *html);
webview_error_t webview_navigate(webview_t view, const char *url);
webview_error_t webview_init(webview_t view, const char *js);
webview_error_t webview_eval(webview_t view, const char *js);
webview_error_t webview_run(webview_t view);
webview_error_t webview_terminate(webview_t view);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/webview","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { createWindowNative, bindNative, nextBindingEventNative, returnBindingNative, unbindNative } from "./native/webview.bind.js";

export function createWindow(debug) {
  return createWindowNative(debug);
}

export function bind(view, name) {
  return bindNative(view, name);
}

export function nextBindingEvent(view) {
  return nextBindingEventNative(view);
}

export function returnBinding(view, id, status, result) {
  return returnBindingNative(view, id, status, result);
}

export function unbind(view, name) {
  return unbindNative(view, name);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "webview.cpp"), []byte(`#include "jayess_runtime.h"
#include <webview/webview.h>

extern "C" jayess_value *jayess_webview_create_window(jayess_value *debug_value) {
    (void) debug_value;
    return jayess_value_undefined();
}

extern "C" jayess_value *jayess_webview_bind(jayess_value *view_value, jayess_value *name_value) {
    (void) view_value;
    (void) name_value;
    return jayess_value_from_bool(1);
}

extern "C" jayess_value *jayess_webview_next_binding_event(jayess_value *view_value) {
    (void) view_value;
    return jayess_value_undefined();
}

extern "C" jayess_value *jayess_webview_return_binding(jayess_value *view_value, jayess_value *id_value, jayess_value *status_value, jayess_value *result_value) {
    (void) view_value;
    (void) id_value;
    (void) status_value;
    (void) result_value;
    return jayess_value_from_bool(1);
}

extern "C" jayess_value *jayess_webview_unbind(jayess_value *view_value, jayess_value *name_value) {
    (void) view_value;
    (void) name_value;
    return jayess_value_from_bool(1);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "webview.bind.js"), []byte(`const f = () => {};
export const createWindowNative = f;
export const bindNative = f;
export const nextBindingEventNative = f;
export const returnBindingNative = f;
export const unbindNative = f;

export default {
  sources: ["./webview.cpp"],
  includeDirs: ["../../../../refs/webview/core/include"],
  cflags: ["-std=c++14"],
  ldflags: [],
  platforms: {
    linux: {
      ldflags: ["-lstdc++", "-ldl", "-lgtk-3", "-lwebkit2gtk-4.1"]
    },
    darwin: {
      ldflags: ["-lstdc++", "-framework", "Cocoa", "-framework", "WebKit"]
    },
    windows: {
      ldflags: ["-lstdc++", "-lole32", "-lcomctl32"]
    }
  },
  exports: {
    createWindowNative: { symbol: "jayess_webview_create_window", type: "function" },
    bindNative: { symbol: "jayess_webview_bind", type: "function" },
    nextBindingEventNative: { symbol: "jayess_webview_next_binding_event", type: "function" },
    returnBindingNative: { symbol: "jayess_webview_return_binding", type: "function" },
    unbindNative: { symbol: "jayess_webview_unbind", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createWindow, bind, nextBindingEvent, returnBinding, unbind } from "@jayess/webview";

function main(args) {
  var view = createWindow(false);
  bind(view, "jayessEcho");
  var event = nextBindingEvent(view);
  if (event != undefined) {
    returnBinding(view, event.id, 0, "{}");
  }
  unbind(view, "jayessEcho");
  return view;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	for _, symbol := range []string{
		"@jayess_webview_create_window(",
		"@jayess_webview_bind(",
		"@jayess_webview_next_binding_event(",
		"@jayess_webview_return_binding(",
		"@jayess_webview_unbind(",
	} {
		if !strings.Contains(string(result.LLVMIR), symbol) {
			t.Fatalf("expected webview package symbol %q in LLVM IR, got:\n%s", symbol, string(result.LLVMIR))
		}
	}
	if len(result.NativeImports) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeImports[0]), "/node_modules/@jayess/webview/native/webview.cpp") {
		t.Fatalf("expected webview native import to be carried through, got %#v", result.NativeImports)
	}
	if !containsString(result.NativeLinkFlags, "-lstdc++") || !containsString(result.NativeLinkFlags, "-lole32") || containsString(result.NativeLinkFlags, "-lwebkit2gtk-4.1") {
		t.Fatalf("expected webview link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathAppliesPlatformSpecificBindFlags(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "binding.c"), []byte(`#include "jayess_runtime.h"

jayess_value *jayess_binding_value(void) {
    return jayess_value_from_number(1);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "binding.bind.js"), []byte(`const f = () => {};
export const value = 0;

export default {
  sources: ["./binding.c"],
  ldflags: ["-lcommon"],
  platforms: {
    linux: {
      ldflags: ["-llinux-only"]
    },
    darwin: {
      ldflags: ["-framework", "Cocoa"]
    },
    windows: {
      ldflags: ["-lole32"]
    }
  },
  exports: {
    value: { symbol: "jayess_binding_value", type: "value" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { value } from "./native/binding.bind.js";

function main(args) {
  return value;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cases := []struct {
		name        string
		triple      string
		want        []string
		notExpected []string
	}{
		{name: "linux", triple: "x86_64-unknown-linux-gnu", want: []string{"-lcommon", "-llinux-only"}, notExpected: []string{"-lole32", "Cocoa"}},
		{name: "darwin", triple: "arm64-apple-darwin", want: []string{"-lcommon", "-framework", "Cocoa"}, notExpected: []string{"-llinux-only", "-lole32"}},
		{name: "windows", triple: "x86_64-pc-windows-msvc", want: []string{"-lcommon", "-lole32"}, notExpected: []string{"-llinux-only", "Cocoa"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CompilePath(entry, Options{TargetTriple: tc.triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}
			for _, flag := range tc.want {
				if !containsString(result.NativeLinkFlags, flag) {
					t.Fatalf("expected platform link flag %q, got %#v", flag, result.NativeLinkFlags)
				}
			}
			for _, flag := range tc.notExpected {
				if containsString(result.NativeLinkFlags, flag) {
					t.Fatalf("did not expect platform link flag %q, got %#v", flag, result.NativeLinkFlags)
				}
			}
		})
	}
}

func TestCompilePathSupportsBindPkgConfigFields(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "binding.c"), []byte(`#include "jayess_runtime.h"
jayess_value *demo_pkg_config_value(void) { return jayess_value_from_number(1); }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "binding.bind.js"), []byte(`const f = () => {};
export const value = f;

export default {
  sources: ["./binding.c"],
  pkgConfig: ["demo-gtk"],
  exports: {
    value: { symbol: "demo_pkg_config_value", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	pkgConfigScript := filepath.Join(binDir, "pkg-config")
	if err := os.WriteFile(pkgConfigScript, []byte(`#!/bin/sh
if [ "$1" = "--cflags" ]; then
  echo "-I/demo/include -DDEMO_GTK"
  exit 0
fi
if [ "$1" = "--libs" ]; then
  echo "-L/demo/lib -ldemo-gtk"
  exit 0
fi
echo "unexpected args: $@" >&2
exit 1
`), 0o755); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { value } from "./native/binding.bind.js";

function main(args) {
  return value();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !containsString(result.NativeCompileFlags, "-I/demo/include") || !containsString(result.NativeCompileFlags, "-DDEMO_GTK") {
		t.Fatalf("expected pkg-config compile flags, got %#v", result.NativeCompileFlags)
	}
	if !containsString(result.NativeLinkFlags, "-L/demo/lib") || !containsString(result.NativeLinkFlags, "-ldemo-gtk") {
		t.Fatalf("expected pkg-config link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessGTKPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "gtk")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include", "gtk")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "gtk.h"), []byte(`#pragma once
typedef struct _GtkWidget GtkWidget;
typedef struct _GtkWindow GtkWindow;
typedef struct _GtkLabel GtkLabel;
typedef struct _GtkButton GtkButton;
typedef struct _GtkEntry GtkEntry;
typedef struct _GtkImage GtkImage;
typedef struct _GtkDrawingArea GtkDrawingArea;
typedef struct _GtkBox GtkBox;
typedef struct _GtkContainer GtkContainer;
typedef enum GtkWindowType { GTK_WINDOW_TOPLEVEL = 0 } GtkWindowType;
typedef enum GtkOrientation { GTK_ORIENTATION_HORIZONTAL = 0, GTK_ORIENTATION_VERTICAL = 1 } GtkOrientation;
#define GTK_WINDOW(widget) ((GtkWindow *)(widget))
#define GTK_LABEL(widget) ((GtkLabel *)(widget))
#define GTK_BUTTON(widget) ((GtkButton *)(widget))
#define GTK_ENTRY(widget) ((GtkEntry *)(widget))
#define GTK_CONTAINER(widget) ((GtkContainer *)(widget))
int jayess_test_is_label(GtkWidget *widget);
int jayess_test_is_button(GtkWidget *widget);
int jayess_test_is_entry(GtkWidget *widget);
#define GTK_IS_LABEL(widget) jayess_test_is_label(widget)
#define GTK_IS_BUTTON(widget) jayess_test_is_button(widget)
#define GTK_IS_ENTRY(widget) jayess_test_is_entry(widget)
typedef void *gpointer;
typedef unsigned long gulong;
typedef int gboolean;
typedef struct _cairo cairo_t;
typedef void (*GCallback)(void);
#define G_CALLBACK(callback) ((GCallback)(callback))
int gtk_init_check(int *argc, char ***argv);
GtkWidget *gtk_window_new(GtkWindowType type);
GtkWidget *gtk_label_new(const char *text);
GtkWidget *gtk_button_new_with_label(const char *text);
GtkWidget *gtk_entry_new(void);
GtkWidget *gtk_image_new_from_file(const char *path);
GtkWidget *gtk_drawing_area_new(void);
GtkWidget *gtk_box_new(GtkOrientation orientation, int spacing);
void gtk_window_set_title(GtkWindow *window, const char *title);
void gtk_label_set_text(GtkLabel *label, const char *text);
void gtk_button_set_label(GtkButton *button, const char *text);
void gtk_entry_set_text(GtkEntry *entry, const char *text);
void gtk_container_add(GtkContainer *container, GtkWidget *child);
gulong g_signal_connect_data(gpointer instance, const char *detailed_signal, GCallback c_handler, gpointer data, gpointer destroy_data, int connect_flags);
void g_signal_emit_by_name(gpointer instance, const char *detailed_signal);
void gtk_widget_show_all(GtkWidget *widget);
void gtk_widget_hide(GtkWidget *widget);
void gtk_widget_queue_draw(GtkWidget *widget);
void gtk_widget_destroy(GtkWidget *widget);
int gtk_events_pending(void);
void gtk_main_iteration_do(int blocking);
void gtk_main(void);
void gtk_main_quit(void);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/gtk","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { initNative, createWindowNative, createLabelNative, createButtonNative, createTextInputNative, createImageNative, createDrawingAreaNative, createBoxNative, setTextNative, addChildNative, connectSignalNative, emitSignalNative, queueDrawNative, hideNative, runMainLoopNative, quitMainLoopNative } from "./native/gtk.bind.js";

export function init() {
  return initNative();
}

export function createWindow() {
  return createWindowNative();
}

export function createLabel(text) {
  return createLabelNative(text);
}

export function createButton(text) {
  return createButtonNative(text);
}

export function createTextInput() {
  return createTextInputNative();
}

export function createImage(path) {
  return createImageNative(path);
}

export function createDrawingArea() {
  return createDrawingAreaNative();
}

export function createBox(vertical, spacing) {
  return createBoxNative(vertical, spacing);
}

export function setText(widget, text) {
  return setTextNative(widget, text);
}

export function addChild(parent, child) {
  return addChildNative(parent, child);
}

export function connectSignal(widget, signal, callback) {
  return connectSignalNative(widget, signal, callback);
}

export function emitSignal(widget, signal) {
  return emitSignalNative(widget, signal);
}

export function queueDraw(widget) {
  return queueDrawNative(widget);
}

export function hide(window) {
  return hideNative(window);
}

export function runMainLoop() {
  return runMainLoopNative();
}

export function quitMainLoop() {
  return quitMainLoopNative();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "gtk.c"), []byte(`#include "jayess_runtime.h"
#include <gtk/gtk.h>

jayess_value *jayess_gtk_init(void) {
    return jayess_value_from_bool(gtk_init_check(0, NULL));
}

jayess_value *jayess_gtk_create_window(void) {
    return jayess_value_from_managed_native_handle("GtkWidget", gtk_window_new(GTK_WINDOW_TOPLEVEL), NULL);
}

jayess_value *jayess_gtk_create_label(jayess_value *text_value) {
    return jayess_value_from_managed_native_handle("GtkWidget", gtk_label_new(jayess_expect_string(text_value, "jayess_gtk_create_label")), NULL);
}

jayess_value *jayess_gtk_create_button(jayess_value *text_value) {
    return jayess_value_from_managed_native_handle("GtkWidget", gtk_button_new_with_label(jayess_expect_string(text_value, "jayess_gtk_create_button")), NULL);
}

jayess_value *jayess_gtk_create_text_input(void) {
    return jayess_value_from_managed_native_handle("GtkWidget", gtk_entry_new(), NULL);
}

jayess_value *jayess_gtk_create_image(jayess_value *path_value) {
    return jayess_value_from_managed_native_handle("GtkWidget", gtk_image_new_from_file(jayess_expect_string(path_value, "jayess_gtk_create_image")), NULL);
}

jayess_value *jayess_gtk_create_drawing_area(void) {
    return jayess_value_from_managed_native_handle("GtkWidget", gtk_drawing_area_new(), NULL);
}

jayess_value *jayess_gtk_create_box(jayess_value *vertical_value, jayess_value *spacing_value) {
    return jayess_value_from_managed_native_handle("GtkWidget", gtk_box_new(jayess_value_is_truthy(vertical_value) ? GTK_ORIENTATION_VERTICAL : GTK_ORIENTATION_HORIZONTAL, (int)jayess_value_to_number(spacing_value)), NULL);
}

jayess_value *jayess_gtk_set_text(jayess_value *widget_value, jayess_value *text_value) {
    GtkWidget *widget = (GtkWidget *)jayess_expect_native_handle(widget_value, "GtkWidget", "jayess_gtk_set_text");
    const char *text = jayess_expect_string(text_value, "jayess_gtk_set_text");
    if (jayess_has_exception()) return jayess_value_undefined();
    if (GTK_IS_LABEL(widget)) gtk_label_set_text(GTK_LABEL(widget), text);
    else if (GTK_IS_BUTTON(widget)) gtk_button_set_label(GTK_BUTTON(widget), text);
    else if (GTK_IS_ENTRY(widget)) gtk_entry_set_text(GTK_ENTRY(widget), text);
    return jayess_value_undefined();
}

jayess_value *jayess_gtk_add_child(jayess_value *parent_value, jayess_value *child_value) {
    GtkWidget *parent = (GtkWidget *)jayess_expect_native_handle(parent_value, "GtkWidget", "jayess_gtk_add_child");
    GtkWidget *child = (GtkWidget *)jayess_expect_native_handle(child_value, "GtkWidget", "jayess_gtk_add_child");
    if (jayess_has_exception()) return jayess_value_undefined();
    gtk_container_add(GTK_CONTAINER(parent), child);
    return jayess_value_undefined();
}

jayess_value *jayess_gtk_hide(jayess_value *window_value) {
    GtkWidget *window = (GtkWidget *)jayess_expect_native_handle(window_value, "GtkWidget", "jayess_gtk_hide");
    if (jayess_has_exception()) return jayess_value_undefined();
    gtk_widget_hide(window);
    return jayess_value_undefined();
}

jayess_value *jayess_gtk_queue_draw(jayess_value *widget_value) {
    GtkWidget *widget = (GtkWidget *)jayess_expect_native_handle(widget_value, "GtkWidget", "jayess_gtk_queue_draw");
    if (jayess_has_exception()) return jayess_value_undefined();
    gtk_widget_queue_draw(widget);
    return jayess_value_undefined();
}

jayess_value *jayess_gtk_run_main_loop(void) {
    gtk_main();
    return jayess_value_undefined();
}

jayess_value *jayess_gtk_quit_main_loop(void) {
    gtk_main_quit();
    return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "gtk.bind.js"), []byte(`const f = () => {};
export const initNative = f;
export const createWindowNative = f;
export const createLabelNative = f;
export const createButtonNative = f;
export const createTextInputNative = f;
export const createImageNative = f;
export const createDrawingAreaNative = f;
export const createBoxNative = f;
export const setTextNative = f;
export const addChildNative = f;
export const connectSignalNative = f;
export const emitSignalNative = f;
export const queueDrawNative = f;
export const hideNative = f;
export const runMainLoopNative = f;
export const quitMainLoopNative = f;

export default {
  sources: ["./gtk.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: ["-lgtk-3", "-lgobject-2.0", "-lglib-2.0", "-lgio-2.0"],
  exports: {
    initNative: { symbol: "jayess_gtk_init", type: "function" },
    createWindowNative: { symbol: "jayess_gtk_create_window", type: "function" },
    createLabelNative: { symbol: "jayess_gtk_create_label", type: "function" },
    createButtonNative: { symbol: "jayess_gtk_create_button", type: "function" },
    createTextInputNative: { symbol: "jayess_gtk_create_text_input", type: "function" },
    createImageNative: { symbol: "jayess_gtk_create_image", type: "function" },
    createDrawingAreaNative: { symbol: "jayess_gtk_create_drawing_area", type: "function" },
    createBoxNative: { symbol: "jayess_gtk_create_box", type: "function" },
    setTextNative: { symbol: "jayess_gtk_set_text", type: "function" },
    addChildNative: { symbol: "jayess_gtk_add_child", type: "function" },
    connectSignalNative: { symbol: "jayess_gtk_connect_signal", type: "function" },
    emitSignalNative: { symbol: "jayess_gtk_emit_signal", type: "function" },
    queueDrawNative: { symbol: "jayess_gtk_queue_draw", type: "function" },
    hideNative: { symbol: "jayess_gtk_hide", type: "function" },
    runMainLoopNative: { symbol: "jayess_gtk_run_main_loop", type: "function" },
    quitMainLoopNative: { symbol: "jayess_gtk_quit_main_loop", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { init, createWindow, createLabel, createButton, createTextInput, createImage, createDrawingArea, createBox, setText, addChild, connectSignal, emitSignal, queueDraw, hide, runMainLoop, quitMainLoop } from "@jayess/gtk";

function main(args) {
  var window = createWindow();
  var box = createBox(true, 4);
  var label = createLabel("hello");
  var button = createButton("go");
  var entry = createTextInput();
  var image = createImage("icon.png");
  var drawingArea = createDrawingArea();
  setText(label, "kimchi");
  setText(button, "save");
  setText(entry, "jjigae");
  addChild(box, label);
  addChild(box, button);
  addChild(box, entry);
  addChild(box, image);
  addChild(box, drawingArea);
  addChild(window, box);
  connectSignal(button, "clicked", function(signal) { return signal; });
  connectSignal(entry, "changed", function(signal) { return signal; });
  connectSignal(window, "destroy", function(signal) { return signal; });
  connectSignal(drawingArea, "draw", function(signal) { return signal; });
  emitSignal(button, "clicked");
  queueDraw(drawingArea);
  hide(window);
  quitMainLoop();
  runMainLoop();
  return init();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_gtk_init(",
		"@jayess_gtk_create_window(",
		"@jayess_gtk_create_label(",
		"@jayess_gtk_create_button(",
		"@jayess_gtk_create_text_input(",
		"@jayess_gtk_create_image(",
		"@jayess_gtk_create_drawing_area(",
		"@jayess_gtk_create_box(",
		"@jayess_gtk_set_text(",
		"@jayess_gtk_add_child(",
		"@jayess_gtk_connect_signal(",
		"@jayess_gtk_emit_signal(",
		"@jayess_gtk_queue_draw(",
		"@jayess_gtk_hide(",
		"@jayess_gtk_run_main_loop(",
		"@jayess_gtk_quit_main_loop(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected GTK package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if len(result.NativeImports) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeImports[0]), "/node_modules/@jayess/gtk/native/gtk.c") {
		t.Fatalf("expected GTK native import to be carried through, got %#v", result.NativeImports)
	}
	if !containsString(result.NativeIncludeDirs, filepath.Join(pkgDir, "native", "include")) {
		t.Fatalf("expected GTK include dir, got %#v", result.NativeIncludeDirs)
	}
	if !containsString(result.NativeLinkFlags, "-lgtk-3") {
		t.Fatalf("expected GTK link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathAppliesPlatformSpecificGTKBindFlags(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "gtk")
	nativeDir := filepath.Join(pkgDir, "native")
	includeDir := filepath.Join(nativeDir, "include", "gtk")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "gtk.h"), []byte(`#pragma once
typedef struct _GtkWidget GtkWidget;
typedef struct _GtkWindow GtkWindow;
typedef enum { GTK_WINDOW_TOPLEVEL = 0 } GtkWindowType;
#define FALSE 0
#define GTK_WINDOW(widget) ((GtkWindow *)(widget))
int gtk_init_check(int *argc, char ***argv);
GtkWidget *gtk_window_new(GtkWindowType type);
void gtk_window_set_title(GtkWindow *window, const char *title);
void gtk_widget_show_all(GtkWidget *widget);
void gtk_widget_destroy(GtkWidget *widget);
int gtk_events_pending(void);
void gtk_main_iteration_do(int blocking);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/gtk","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { initNative } from "./native/gtk.bind.js";
export function init() {
  return initNative();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "gtk.c"), []byte(`#include "jayess_runtime.h"
#include <gtk/gtk.h>

jayess_value *jayess_gtk_init(void) {
  return jayess_value_from_bool(gtk_init_check(0, NULL));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "gtk.bind.js"), []byte(`const f = () => {};
export const initNative = f;

export default {
  sources: ["./gtk.c"],
  includeDirs: ["./include"],
  ldflags: [],
  platforms: {
    linux: {
      ldflags: ["-lgtk-3", "-lgobject-2.0", "-lglib-2.0", "-lgio-2.0"]
    },
    darwin: {
      ldflags: ["-lgtk-3", "-lgobject-2.0", "-lgio-2.0", "-framework", "Cocoa"]
    },
    windows: {
      ldflags: ["-lgtk-3", "-lgobject-2.0", "-lgio-2.0", "-lole32", "-lcomctl32"]
    }
  },
  exports: {
    initNative: { symbol: "jayess_gtk_init", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { init } from "@jayess/gtk";

function main(args) {
  return init();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	cases := []struct {
		name        string
		triple      string
		want        []string
		notExpected []string
	}{
		{name: "linux", triple: "x86_64-unknown-linux-gnu", want: []string{"-lgtk-3", "-lglib-2.0"}, notExpected: []string{"-lole32", "Cocoa"}},
		{name: "darwin", triple: "arm64-apple-darwin", want: []string{"-lgtk-3", "-framework", "Cocoa"}, notExpected: []string{"-lole32"}},
		{name: "windows", triple: "x86_64-pc-windows-msvc", want: []string{"-lgtk-3", "-lole32", "-lcomctl32"}, notExpected: []string{"Cocoa"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CompilePath(entry, Options{TargetTriple: tc.triple})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}
			for _, flag := range tc.want {
				if !containsString(result.NativeLinkFlags, flag) {
					t.Fatalf("expected GTK platform link flag %q, got %#v", flag, result.NativeLinkFlags)
				}
			}
			for _, flag := range tc.notExpected {
				if containsString(result.NativeLinkFlags, flag) {
					t.Fatalf("did not expect GTK platform link flag %q, got %#v", flag, result.NativeLinkFlags)
				}
			}
		})
	}
}

func TestCompilePathSupportsJayessHTTPServerPackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "httpserver")
	aliasDir := filepath.Join(dir, "node_modules", "@jayess", "http-server")
	nativeDir := filepath.Join(aliasDir, "native")
	picoDir := filepath.Join(dir, "refs", "picohttpparser")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(aliasDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(picoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(picoDir, "picohttpparser.h"), []byte(`#pragma once
struct phr_header { const char *name; size_t name_len; const char *value; size_t value_len; };
int phr_parse_request(const char *buf, size_t len, const char **method, size_t *method_len, const char **path, size_t *path_len, int *minor_version, struct phr_header *headers, size_t *num_headers, size_t last_len);
int phr_parse_response(const char *buf, size_t len, int *minor_version, int *status, const char **msg, size_t *msg_len, struct phr_header *headers, size_t *num_headers, size_t last_len);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(picoDir, "picohttpparser.c"), []byte(`int jayess_dummy_pico = 1;`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/httpserver","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aliasDir, "package.json"), []byte(`{"name":"@jayess/http-server","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
export * from "../http-server/index.js";
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(aliasDir, "index.js"), []byte(`
import { parseRequestNative, parseResponseNative, parseRequestIncrementalNative, parseResponseIncrementalNative, decodeChunkedNative } from "./native/server.bind.js";

export function parseRequest(text) {
  return parseRequestNative(text);
}

export function parseResponse(text) {
  return parseResponseNative(text);
}

export function parseRequestIncremental(text, lastLen) {
  return parseRequestIncrementalNative(text, lastLen);
}

export function parseResponseIncremental(text, lastLen) {
  return parseResponseIncrementalNative(text, lastLen);
}

export function decodeChunked(text) {
  return decodeChunkedNative(text);
}

export function formatRequest(request) {
  return http.formatRequest(request);
}

export function formatResponse(response) {
  return http.formatResponse(response);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "http_server.c"), []byte(`#include "jayess_runtime.h"
#include "picohttpparser.h"

jayess_value *jayess_http_parse_request_native(jayess_value *input) {
    (void) input;
    return jayess_value_undefined();
}

jayess_value *jayess_http_parse_response_native(jayess_value *input) {
    (void) input;
    return jayess_value_undefined();
}

jayess_value *jayess_http_parse_request_incremental_native(jayess_value *input, jayess_value *last_len) {
    (void) input;
    (void) last_len;
    return jayess_value_undefined();
}

jayess_value *jayess_http_parse_response_incremental_native(jayess_value *input, jayess_value *last_len) {
    (void) input;
    (void) last_len;
    return jayess_value_undefined();
}

jayess_value *jayess_http_decode_chunked_native(jayess_value *input) {
    (void) input;
    return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "server.bind.js"), []byte(`const f = () => {};
export const parseRequestNative = f;
export const parseResponseNative = f;
export const parseRequestIncrementalNative = f;
export const parseResponseIncrementalNative = f;
export const decodeChunkedNative = f;

export default {
  sources: ["./http_server.c", "../../../../refs/picohttpparser/picohttpparser.c"],
  includeDirs: [".", "../../../../refs/picohttpparser"],
  cflags: [],
  ldflags: [],
  exports: {
    parseRequestNative: { symbol: "jayess_http_parse_request_native", type: "function" },
    parseResponseNative: { symbol: "jayess_http_parse_response_native", type: "function" },
    parseRequestIncrementalNative: { symbol: "jayess_http_parse_request_incremental_native", type: "function" },
    parseResponseIncrementalNative: { symbol: "jayess_http_parse_response_incremental_native", type: "function" },
    decodeChunkedNative: { symbol: "jayess_http_decode_chunked_native", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { parseRequest, parseResponse, parseRequestIncremental, parseResponseIncremental, decodeChunked, formatRequest, formatResponse } from "@jayess/httpserver";

function main(args) {
  var requestPayload = {};
  requestPayload.method = "GET";
  requestPayload.path = "/";
  requestPayload.version = "HTTP/1.1";
  requestPayload.headers = {};
  requestPayload.body = "";
  var responsePayload = {};
  responsePayload.version = "HTTP/1.1";
  responsePayload.status = 200;
  responsePayload.reason = "OK";
  responsePayload.headers = {};
  responsePayload.body = "";
  var requestText = formatRequest(requestPayload);
  var responseText = formatResponse(responsePayload);
  console.log(parseRequest(requestText));
  console.log(parseResponse(responseText));
  console.log(parseRequestIncremental(requestText, 4));
  console.log(parseResponseIncremental(responseText, 4));
  console.log(decodeChunked(Uint8Array.fromString("340d0a57696b690d0a350d0a70656469610d0a300d0a0d0a", "hex")));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_http_parse_request_native(") || !strings.Contains(string(result.LLVMIR), "@jayess_http_parse_response_native(") {
		t.Fatalf("expected picohttp package symbols in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected two picohttp native imports, got %#v", result.NativeImports)
	}
	if !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/http-server/native/http_server.c") || !containsPathSuffix(result.NativeImports, "/refs/picohttpparser/picohttpparser.c") {
		t.Fatalf("expected picohttp native imports to be carried through, got %#v", result.NativeImports)
	}
}

func TestCompilePathSupportsJayessMongoosePackage(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "mongoose")
	nativeDir := filepath.Join(pkgDir, "native")
	mongooseDir := filepath.Join(dir, "refs", "mongoose")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(mongooseDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mongooseDir, "mongoose.h"), []byte(`#pragma once
struct mg_mgr;
struct mg_connection;
typedef void (*mg_event_handler_t)(struct mg_connection *, int, void *);
void mg_mgr_poll(struct mg_mgr *, int ms);
void mg_mgr_init(struct mg_mgr *);
void mg_mgr_free(struct mg_mgr *);
struct mg_connection *mg_http_listen(struct mg_mgr *, const char *url, mg_event_handler_t fn, void *fn_data);
void mg_http_reply(struct mg_connection *, int status_code, const char *headers, const char *body_fmt, ...);
enum { MG_EV_HTTP_MSG = 1 };
#define MG_MAX_HTTP_HEADERS 4
struct mg_str { const char *buf; unsigned long long len; };
struct mg_http_header { struct mg_str name; struct mg_str value; };
struct mg_http_message { struct mg_str method, uri, query, proto; struct mg_http_header headers[MG_MAX_HTTP_HEADERS]; struct mg_str body, head, message; };
struct mg_mgr { void *userdata; };
struct mg_connection { void *fn_data; unsigned is_draining:1; unsigned is_closing:1; };
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(mongooseDir, "mongoose.c"), []byte(`int jayess_dummy_mongoose = 1;`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/mongoose","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`
import { createManagerNative, listenNative, listenTlsNative, nextEventNative, upgradeWebSocketNative, sendWebSocketNative } from "./native/mongoose.bind.js";

export function createManager() {
  return createManagerNative();
}

export function listenServer(manager, url) {
  return listenNative(manager, url);
}

export function listenTlsServer(manager, url, certPath, keyPath) {
  return listenTlsNative(manager, url, certPath, keyPath);
}

export function nextEvent(manager) {
  return nextEventNative(manager);
}

export function upgradeWebSocket(event) {
  return upgradeWebSocketNative(event.connection);
}

export function sendWebSocket(connection, data) {
  return sendWebSocketNative(connection, data);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mongoose.c"), []byte(`#include "jayess_runtime.h"
#include "mongoose.h"

jayess_value *jayess_mongoose_create_manager(jayess_value *unused) {
    (void) unused;
    return jayess_value_undefined();
}

jayess_value *jayess_mongoose_listen(jayess_value *manager, jayess_value *url) {
    (void) manager;
    (void) url;
    return jayess_value_undefined();
}

jayess_value *jayess_mongoose_listen_tls(jayess_value *manager, jayess_value *url, jayess_value *cert_path, jayess_value *key_path) {
    (void) manager;
    (void) url;
    (void) cert_path;
    (void) key_path;
    return jayess_value_undefined();
}

jayess_value *jayess_mongoose_next_event(jayess_value *manager) {
    (void) manager;
    return jayess_value_undefined();
}

jayess_value *jayess_mongoose_upgrade_websocket(jayess_value *connection) {
    (void) connection;
    return jayess_value_undefined();
}

jayess_value *jayess_mongoose_send_websocket(jayess_value *connection, jayess_value *data) {
    (void) connection;
    (void) data;
    return jayess_value_undefined();
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mongoose.bind.js"), []byte(`const f = () => {};
export const createManagerNative = f;
export const listenNative = f;
export const listenTlsNative = f;
export const nextEventNative = f;
export const upgradeWebSocketNative = f;
export const sendWebSocketNative = f;

export default {
  sources: ["./mongoose.c", "../../../../refs/mongoose/mongoose.c"],
  includeDirs: [".", "../../../../refs/mongoose"],
  cflags: [],
  ldflags: [],
  exports: {
    createManagerNative: { symbol: "jayess_mongoose_create_manager", type: "function" },
    listenNative: { symbol: "jayess_mongoose_listen", type: "function" },
    listenTlsNative: { symbol: "jayess_mongoose_listen_tls", type: "function" },
    nextEventNative: { symbol: "jayess_mongoose_next_event", type: "function" },
    upgradeWebSocketNative: { symbol: "jayess_mongoose_upgrade_websocket", type: "function" },
    sendWebSocketNative: { symbol: "jayess_mongoose_send_websocket", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createManager, listenServer, listenTlsServer, nextEvent } from "@jayess/mongoose";

function main(args) {
  var manager = createManager();
  listenServer(manager, "http://127.0.0.1:8080");
  listenTlsServer(manager, "https://127.0.0.1:8443", "cert.pem", "key.pem");
  console.log(nextEvent(manager));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_create_manager(") || !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_listen(") || !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_listen_tls(") || !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_next_event(") {
		t.Fatalf("expected mongoose package symbols in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
	if len(result.NativeImports) != 2 {
		t.Fatalf("expected two mongoose native imports, got %#v", result.NativeImports)
	}
	if !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/mongoose/native/mongoose.c") || !containsPathSuffix(result.NativeImports, "/refs/mongoose/mongoose.c") {
		t.Fatalf("expected mongoose native imports to be carried through, got %#v", result.NativeImports)
	}
}

func TestCompilePathSupportsJayessMongooseRouterHelpers(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "mongoose")
	nativeDir := filepath.Join(pkgDir, "native")
	refsDir := filepath.Join(dir, "refs", "mongoose")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/mongoose","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`import { createManagerNative, listenNative, listenTlsNative, pollNative, nextEventNative, replyNative, upgradeWebSocketNative, sendWebSocketNative, closeConnectionNative, freeManagerNative, serveStaticNative, startChunkedNative, writeChunkNative, endChunkedNative } from "./native/mongoose.bind.js";

export function createManager() {
  return createManagerNative();
}

export function listenServer(manager, url) {
  return listenNative(manager, url);
}

export function listenTlsServer(manager, url, certPath, keyPath) {
  return listenTlsNative(manager, url, certPath, keyPath);
}

export function pollManager(manager, timeoutMs) {
  return pollNative(manager, timeoutMs);
}

export function nextEvent(manager) {
  return nextEventNative(manager);
}

export function reply(connection, status, headers, body) {
  return replyNative(connection, status, headers, body);
}

export function upgradeWebSocket(event) {
  return upgradeWebSocketNative(event.connection);
}

export function sendWebSocket(connection, data) {
  return sendWebSocketNative(connection, data);
}

export function closeConnection(connection) {
  return closeConnectionNative(connection);
}

export function freeManager(manager) {
  return freeManagerNative(manager);
}

export function createRouter() {
  return [];
}

export function addRoute(router, method, path, handler) {
  router.push({ method: method, path: path, handler: handler });
  return router;
}

export function get(router, path, handler) {
  return addRoute(router, "GET", path, handler);
}

export function post(router, path, handler) {
  return addRoute(router, "POST", path, handler);
}

function routeMatches(route, event) {
  if (route.method !== "*" && route.method !== event.method) {
    return false;
  }
  return route.path === "*" || route.path === event.path;
}

export function dispatch(router, event) {
  var i = 0;
  while (i < router.length) {
    var route = router[i];
    if (routeMatches(route, event)) {
      var response = route.handler(event);
      if (response === undefined || response === null) {
        return false;
      }
      reply(event.connection, response.status !== undefined ? response.status : 200, response.headers !== undefined ? response.headers : {}, response.body !== undefined ? response.body : "");
      return true;
    }
    i = i + 1;
  }
  return false;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mongoose.c"), []byte(`#include "jayess_runtime.h"
#include "mongoose.h"

jayess_value *jayess_mongoose_create_manager(jayess_value *unused) { (void) unused; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_listen(jayess_value *manager, jayess_value *url) { (void) manager; (void) url; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_listen_tls(jayess_value *manager, jayess_value *url, jayess_value *cert_path, jayess_value *key_path) { (void) manager; (void) url; (void) cert_path; (void) key_path; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_poll(jayess_value *manager, jayess_value *timeout) { (void) manager; (void) timeout; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_next_event(jayess_value *manager) { (void) manager; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_reply(jayess_value *connection, jayess_value *status, jayess_value *headers, jayess_value *body) { (void) connection; (void) status; (void) headers; (void) body; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_upgrade_websocket(jayess_value *connection) { (void) connection; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_send_websocket(jayess_value *connection, jayess_value *data) { (void) connection; (void) data; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_close_connection(jayess_value *connection) { (void) connection; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_free_manager(jayess_value *manager) { (void) manager; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_serve_static(jayess_value *connection, jayess_value *request_path, jayess_value *prefix, jayess_value *root_dir) { (void) connection; (void) request_path; (void) prefix; (void) root_dir; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_start_chunked(jayess_value *connection, jayess_value *status, jayess_value *headers) { (void) connection; (void) status; (void) headers; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_write_chunk(jayess_value *stream, jayess_value *chunk) { (void) stream; (void) chunk; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_end_chunked(jayess_value *stream, jayess_value *final_chunk) { (void) stream; (void) final_chunk; return jayess_value_undefined(); }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mongoose.bind.js"), []byte(`const f = () => {};
export const createManagerNative = f;
export const listenNative = f;
export const listenTlsNative = f;
export const pollNative = f;
export const nextEventNative = f;
export const replyNative = f;
export const upgradeWebSocketNative = f;
export const sendWebSocketNative = f;
export const closeConnectionNative = f;
export const freeManagerNative = f;
export const serveStaticNative = f;
export const startChunkedNative = f;
export const writeChunkNative = f;
export const endChunkedNative = f;

export default {
  sources: ["./mongoose.c", "../../../../refs/mongoose/mongoose.c"],
  includeDirs: [".", "../../../../refs/mongoose"],
  cflags: [],
  ldflags: [],
  exports: {
    createManagerNative: { symbol: "jayess_mongoose_create_manager", type: "function" },
    listenNative: { symbol: "jayess_mongoose_listen", type: "function" },
    listenTlsNative: { symbol: "jayess_mongoose_listen_tls", type: "function" },
    pollNative: { symbol: "jayess_mongoose_poll", type: "function" },
    nextEventNative: { symbol: "jayess_mongoose_next_event", type: "function" },
    replyNative: { symbol: "jayess_mongoose_reply", type: "function" },
    upgradeWebSocketNative: { symbol: "jayess_mongoose_upgrade_websocket", type: "function" },
    sendWebSocketNative: { symbol: "jayess_mongoose_send_websocket", type: "function" },
    closeConnectionNative: { symbol: "jayess_mongoose_close_connection", type: "function" },
    freeManagerNative: { symbol: "jayess_mongoose_free_manager", type: "function" },
    serveStaticNative: { symbol: "jayess_mongoose_serve_static", type: "function" },
    startChunkedNative: { symbol: "jayess_mongoose_start_chunked", type: "function" },
    writeChunkNative: { symbol: "jayess_mongoose_write_chunk", type: "function" },
    endChunkedNative: { symbol: "jayess_mongoose_end_chunked", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "mongoose.h"), []byte(`struct mg_mgr;
struct mg_connection;
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "mongoose.c"), []byte(`void jayess_mongoose_refs_placeholder(void) {}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createRouter, get, post, dispatch } from "@jayess/mongoose";

function main(args) {
  var router = createRouter();
  get(router, "/ready", function(event) {
    return { status: 200, headers: { "X-Test": "ok" }, body: "ready" };
  });
  post(router, "/submit", function(event) {
    return { status: 201, body: event.body };
  });
  console.log(dispatch);
  console.log(router.length);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_reply(") || !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_poll(") {
		t.Fatalf("expected mongoose router helper package symbols in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
}

func TestCompilePathSupportsJayessMongooseStaticFilesHelpers(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "node_modules", "@jayess", "mongoose")
	nativeDir := filepath.Join(pkgDir, "native")
	refsDir := filepath.Join(dir, "refs", "mongoose")
	if err := os.MkdirAll(nativeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"@jayess/mongoose","jayess":"index.js"}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte(`import { createManagerNative, listenNative, pollNative, nextEventNative, replyNative, upgradeWebSocketNative, sendWebSocketNative, closeConnectionNative, freeManagerNative, serveStaticNative, startChunkedNative, writeChunkNative, endChunkedNative } from "./native/mongoose.bind.js";

export function createManager() { return createManagerNative(); }
export function listenServer(manager, url) { return listenNative(manager, url); }
export function pollManager(manager, timeoutMs) { return pollNative(manager, timeoutMs); }
export function nextEvent(manager) { return nextEventNative(manager); }
export function reply(connection, status, headers, body) { return replyNative(connection, status, headers, body); }
export function upgradeWebSocket(event) { return upgradeWebSocketNative(event.connection); }
export function sendWebSocket(connection, data) { return sendWebSocketNative(connection, data); }
export function closeConnection(connection) { return closeConnectionNative(connection); }
export function freeManager(manager) { return freeManagerNative(manager); }
export function createRouter() { return []; }
export function serveStatic(event, urlPrefix, rootDir) {
  return serveStaticNative(event.connection, event.path, urlPrefix, rootDir);
}
function normalizeEmbeddedPath(path) {
  if (path === undefined || path === null || path === "") { return "/"; }
  if (path.slice(0, 1) === "/") { return path; }
  return "/" + path;
}
function embeddedContentType(path) {
  if (path.length >= 5 && path.slice(path.length - 5) === ".html") { return "text/html; charset=utf-8"; }
  if (path.length >= 3 && path.slice(path.length - 3) === ".js") { return "application/javascript; charset=utf-8"; }
  if (path.length >= 4 && path.slice(path.length - 4) === ".css") { return "text/css; charset=utf-8"; }
  if (path.length >= 5 && path.slice(path.length - 5) === ".json") { return "application/json; charset=utf-8"; }
  if (path.length >= 4 && path.slice(path.length - 4) === ".svg") { return "image/svg+xml"; }
  if (path.length >= 4 && path.slice(path.length - 4) === ".txt") { return "text/plain; charset=utf-8"; }
  return "application/octet-stream";
}
function cloneHeaders(headers) {
  var result = {};
  var keys = Object.keys(headers);
  var i = 0;
  while (i < keys.length) {
    var key = keys[i];
    result[key] = headers[key];
    i = i + 1;
  }
  return result;
}
function findEmbeddedAsset(assets, targetPath) {
  var i = 0;
  while (i < assets.length) {
    if (assets[i].path === targetPath) { return assets[i]; }
    i = i + 1;
  }
  return undefined;
}
export function createEmbeddedApp(files, fallbackPath) {
  return { assets: files, fallbackPath: fallbackPath !== undefined ? normalizeEmbeddedPath(fallbackPath) : undefined };
}
export function serveEmbeddedApp(event, urlPrefix, app, fallbackPathOverride) {
  var prefix = urlPrefix !== undefined && urlPrefix !== null && urlPrefix !== "" ? urlPrefix : "/";
  var assets = app.assets !== undefined ? app.assets : app;
  var fallbackPath = fallbackPathOverride !== undefined ? normalizeEmbeddedPath(fallbackPathOverride) : app.fallbackPath;
  var relativePath = "";
  var asset = undefined;
  var assetPath = "";
  var body = "";
  var headers = {};
  var status = 200;
  if (event.method !== "GET" && event.method !== "HEAD") { return false; }
  if (prefix === "/") {
    relativePath = event.path;
  } else if (event.path === prefix || event.path === prefix + "/") {
    relativePath = "/";
  } else if (event.path.slice(0, prefix.length + 1) === prefix + "/") {
    relativePath = event.path.slice(prefix.length);
  } else {
    return false;
  }
  relativePath = normalizeEmbeddedPath(relativePath);
  if (relativePath === "/") { relativePath = "/index.html"; }
  assetPath = relativePath;
  asset = findEmbeddedAsset(assets, assetPath);
  if (asset === undefined && fallbackPath !== undefined) {
    assetPath = fallbackPath;
    asset = findEmbeddedAsset(assets, assetPath);
  }
  if (asset === undefined) { return false; }
  status = asset.status !== undefined ? asset.status : 200;
  body = asset.body !== undefined ? asset.body : asset;
  headers = asset.headers !== undefined ? cloneHeaders(asset.headers) : {};
  if (headers["Content-Type"] === undefined) {
    headers["Content-Type"] = asset.contentType !== undefined ? asset.contentType : embeddedContentType(assetPath);
  }
  reply(event.connection, status, headers, event.method === "HEAD" ? "" : body);
  return true;
}
export function startChunked(event, status, headers) {
  return startChunkedNative(event.connection, status, headers);
}
export function writeChunk(stream, chunk) {
  return writeChunkNative(stream, chunk);
}
export function endChunked(stream, finalChunk) {
  return endChunkedNative(stream, finalChunk);
}
function routeMatches(route, event) {
  if (route.method !== "*" && route.method !== event.method) {
    return false;
  }
  if (route.prefix === true) {
    return event.path === route.path || event.path.slice(0, route.path.length + 1) === route.path + "/";
  }
  return route.path === event.path;
}
export function dispatch(router, event) {
  var i = 0;
  while (i < router.length) {
    var route = router[i];
    if (routeMatches(route, event)) {
      var response = route.handler(event);
      reply(event.connection, response.status, response.headers, response.body);
      return true;
    }
    i = i + 1;
  }
  return false;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mongoose.c"), []byte(`#include "jayess_runtime.h"
#include "mongoose.h"
jayess_value *jayess_mongoose_create_manager(jayess_value *unused) { (void) unused; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_listen(jayess_value *manager, jayess_value *url) { (void) manager; (void) url; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_poll(jayess_value *manager, jayess_value *timeout) { (void) manager; (void) timeout; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_next_event(jayess_value *manager) { (void) manager; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_reply(jayess_value *connection, jayess_value *status, jayess_value *headers, jayess_value *body) { (void) connection; (void) status; (void) headers; (void) body; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_upgrade_websocket(jayess_value *connection) { (void) connection; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_send_websocket(jayess_value *connection, jayess_value *data) { (void) connection; (void) data; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_close_connection(jayess_value *connection) { (void) connection; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_free_manager(jayess_value *manager) { (void) manager; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_serve_static(jayess_value *connection, jayess_value *request_path, jayess_value *prefix, jayess_value *root_dir) { (void) connection; (void) request_path; (void) prefix; (void) root_dir; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_start_chunked(jayess_value *connection, jayess_value *status, jayess_value *headers) { (void) connection; (void) status; (void) headers; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_write_chunk(jayess_value *stream, jayess_value *chunk) { (void) stream; (void) chunk; return jayess_value_undefined(); }
jayess_value *jayess_mongoose_end_chunked(jayess_value *stream, jayess_value *final_chunk) { (void) stream; (void) final_chunk; return jayess_value_undefined(); }
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mongoose.bind.js"), []byte(`const f = () => {};
export const createManagerNative = f;
export const listenNative = f;
export const pollNative = f;
export const nextEventNative = f;
export const replyNative = f;
export const upgradeWebSocketNative = f;
export const sendWebSocketNative = f;
export const closeConnectionNative = f;
export const freeManagerNative = f;
export const serveStaticNative = f;
export const startChunkedNative = f;
export const writeChunkNative = f;
export const endChunkedNative = f;

export default {
  sources: ["./mongoose.c", "../../../../refs/mongoose/mongoose.c"],
  includeDirs: [".", "../../../../refs/mongoose"],
  cflags: [],
  ldflags: [],
  exports: {
    createManagerNative: { symbol: "jayess_mongoose_create_manager", type: "function" },
    listenNative: { symbol: "jayess_mongoose_listen", type: "function" },
    pollNative: { symbol: "jayess_mongoose_poll", type: "function" },
    nextEventNative: { symbol: "jayess_mongoose_next_event", type: "function" },
    replyNative: { symbol: "jayess_mongoose_reply", type: "function" },
    upgradeWebSocketNative: { symbol: "jayess_mongoose_upgrade_websocket", type: "function" },
    sendWebSocketNative: { symbol: "jayess_mongoose_send_websocket", type: "function" },
    closeConnectionNative: { symbol: "jayess_mongoose_close_connection", type: "function" },
    freeManagerNative: { symbol: "jayess_mongoose_free_manager", type: "function" },
    serveStaticNative: { symbol: "jayess_mongoose_serve_static", type: "function" },
    startChunkedNative: { symbol: "jayess_mongoose_start_chunked", type: "function" },
    writeChunkNative: { symbol: "jayess_mongoose_write_chunk", type: "function" },
    endChunkedNative: { symbol: "jayess_mongoose_end_chunked", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "mongoose.h"), []byte(`struct mg_mgr;
struct mg_connection;
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "mongoose.c"), []byte(`void jayess_mongoose_refs_placeholder(void) {}`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createRouter, createEmbeddedApp, serveStatic, serveEmbeddedApp, dispatch } from "@jayess/mongoose";

function main(args) {
  var router = createRouter();
  var app = createEmbeddedApp([{ path: "/index.html", body: "<h1>app</h1>" }], "/index.html");
  console.log(serveStatic);
  console.log(serveEmbeddedApp);
  console.log(dispatch);
  console.log(router.length);
  console.log(app.fallbackPath);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_serve_static(") || !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_start_chunked(") || !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_write_chunk(") || !strings.Contains(string(result.LLVMIR), "@jayess_mongoose_end_chunked(") {
		t.Fatalf("expected mongoose static/stream helper symbols in LLVM IR, got:\n%s", string(result.LLVMIR))
	}
}
func TestCompilePathDeduplicatesSharedNativeSourcesAcrossBindFiles(t *testing.T) {
	dir := t.TempDir()
	nativeDir := filepath.Join(dir, "native")
	includeDir := filepath.Join(nativeDir, "include")
	if err := os.MkdirAll(includeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(includeDir, "shared_math.h"), []byte(`#pragma once
double shared_add_bonus(double left, double right);
double shared_mul_bonus(double left, double right);
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "shared.c"), []byte(`#include "shared_math.h"

double shared_add_bonus(double left, double right) {
    return left + right + 1;
}

double shared_mul_bonus(double left, double right) {
    return left * right + 1;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "add.c"), []byte(`#include "jayess_runtime.h"
#include "shared_math.h"

jayess_value *mylib_add(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(shared_add_bonus(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mul.c"), []byte(`#include "jayess_runtime.h"
#include "shared_math.h"

jayess_value *mylib_mul(jayess_value *a, jayess_value *b) {
    return jayess_value_from_number(shared_mul_bonus(jayess_value_to_number(a), jayess_value_to_number(b)));
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "add.bind.js"), []byte(`const f = () => {};
export const add = f;

export default {
  sources: ["./add.c", "./shared.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    add: { symbol: "mylib_add", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nativeDir, "mul.bind.js"), []byte(`const f = () => {};
export const mul = f;

export default {
  sources: ["./mul.c", "./shared.c"],
  includeDirs: ["./include"],
  cflags: [],
  ldflags: [],
  exports: {
    mul: { symbol: "mylib_mul", type: "function" }
  }
};
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { add } from "./native/add.bind.js";
import { mul } from "./native/mul.bind.js";

function main(args) {
  return add(1, 2) + mul(2, 3);
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	if len(result.NativeImports) != 3 {
		t.Fatalf("expected deduplicated native imports [add.c mul.c shared.c], got %#v", result.NativeImports)
	}
	if len(result.NativeIncludeDirs) != 1 || !strings.HasSuffix(filepath.ToSlash(result.NativeIncludeDirs[0]), "/native/include") {
		t.Fatalf("expected deduplicated include dir, got %#v", result.NativeIncludeDirs)
	}
}

func TestCompilePathSupportsJayessSQLitePackage(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	dir, err := os.MkdirTemp(repoRoot, "jayess-sqlite-loader-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { open, exec, prepare, bindInteger, step, finalize, close } from "@jayess/sqlite";

function main(args) {
  var db = open(":memory:");
  exec(db, "create table items(id integer primary key, name text)");
  var stmt = prepare(db, "select ? as id, 'kimchi' as name");
  bindInteger(stmt, 1, 3);
  console.log(step(stmt).name);
  finalize(stmt);
  close(db);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_sqlite_open_native(",
		"@jayess_sqlite_exec_native(",
		"@jayess_sqlite_prepare_native(",
		"@jayess_sqlite_bind_integer_native(",
		"@jayess_sqlite_step_object_native(",
		"@jayess_sqlite_finalize_native(",
		"@jayess_sqlite_close_native(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected SQLite package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/sqlite/native/sqlite.c") {
		t.Fatalf("expected SQLite native import to be carried through, got %#v", result.NativeImports)
	}
	if !containsPathSuffix(result.NativeImports, "/refs/sqlite/sqlite3.c") {
		t.Fatalf("expected vendored SQLite amalgamation import to be carried through, got %#v", result.NativeImports)
	}
	if len(result.NativeLinkFlags) != 0 {
		t.Fatalf("expected vendored SQLite build to avoid extra link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessOpenSSLPackage(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	dir, err := os.MkdirTemp(repoRoot, "jayess-openssl-loader-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { randomBytes, hash, hmac, encrypt, decrypt, generateKeyPair, publicEncrypt, privateDecrypt, sign, verify } from "@jayess/openssl";

function main(args) {
  var bytes = randomBytes(16);
  var digest = hash("sha256", "kimchi");
  var mac = hmac("sha256", "secret", "kimchi");
  var encrypted = encrypt({
    algorithm: "aes-256-gcm",
    key: Uint8Array.fromString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "hex"),
    iv: Uint8Array.fromString("1af38c2dc2b96ffdd86694092341bc04", "hex"),
    data: "jayess"
  });
  var pair = generateKeyPair({ type: "rsa", modulusLength: 2048 });
  var sealed = publicEncrypt({ algorithm: "rsa-oaep-sha256", key: pair.publicKey, data: "jjigae" });
  var opened = privateDecrypt({ algorithm: "rsa-oaep-sha256", key: pair.privateKey, data: sealed });
  var signature = sign({ algorithm: "rsa-pss-sha256", key: pair.privateKey, data: "kimchi" });
  console.log(bytes.length + digest.length + mac.length + decrypt({
    algorithm: encrypted.algorithm,
    key: Uint8Array.fromString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f", "hex"),
    iv: encrypted.iv,
    data: encrypted.ciphertext,
    tag: encrypted.tag
  }).length + opened.length + verify({ algorithm: "rsa-pss-sha256", key: pair.publicKey, data: "kimchi", signature: signature }));
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-pc-windows-msvc"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_openssl_random_bytes_native(",
		"@jayess_openssl_hash_native(",
		"@jayess_openssl_hmac_native(",
		"@jayess_openssl_encrypt_native(",
		"@jayess_openssl_decrypt_native(",
		"@jayess_openssl_generate_key_pair_native(",
		"@jayess_openssl_public_encrypt_native(",
		"@jayess_openssl_private_decrypt_native(",
		"@jayess_openssl_sign_native(",
		"@jayess_openssl_verify_native(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected OpenSSL package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/openssl/native/openssl.c") {
		t.Fatalf("expected OpenSSL native import to be carried through, got %#v", result.NativeImports)
	}
	if !containsPathSuffix(result.NativeIncludeDirs, "/refs/openssl/include") {
		t.Fatalf("expected vendored OpenSSL include dir to be carried through, got %#v", result.NativeIncludeDirs)
	}
	if !containsString(result.NativeLinkFlags, "-lssl") || !containsString(result.NativeLinkFlags, "-lcrypto") {
		t.Fatalf("expected OpenSSL link flags to be carried through, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessCurlPackage(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	dir, err := os.MkdirTemp(repoRoot, "jayess-curl-loader-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { createEasy, configure, perform, cleanup, createMulti, addHandle, performMulti, cleanupMulti, performStream, request, requestMulti, requestStream } from "@jayess/curl";

function main(args) {
  var easy = createEasy();
  configure(easy, {
    url: "http://example.invalid/",
    method: "POST",
    headers: ["X-Test: 1"],
    body: "kimchi",
    followRedirects: true,
    timeoutMs: 250
  });
  var response = perform(easy);
  var multi = createMulti();
  var easyA = createEasy();
  var easyB = createEasy();
  configure(easyA, { url: "http://example.invalid/a" });
  configure(easyB, { url: "http://example.invalid/b" });
  addHandle(multi, easyA);
  addHandle(multi, easyB);
  var multiResponse = performMulti(multi);
  var streamChunks = [];
  var streamed = performStream(easyA, function (chunk) {
    streamChunks.push(chunk);
  });
  cleanup(easyA);
  cleanup(easyB);
  cleanupMulti(multi);
  cleanup(easy);
  console.log(response.status + multiResponse.length + streamed.chunks + streamChunks.length + request({ url: "http://example.invalid/" }).status + requestMulti([{ url: "http://example.invalid/1" }, { url: "http://example.invalid/2" }]).length + requestStream({ url: "http://example.invalid/3" }, function (chunk) { streamChunks.push(chunk); }).status);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_curl_create_easy_native(",
		"@jayess_curl_configure_native(",
		"@jayess_curl_perform_native(",
		"@jayess_curl_cleanup_native(",
		"@jayess_curl_create_multi_native(",
		"@jayess_curl_add_handle_native(",
		"@jayess_curl_perform_multi_native(",
		"@jayess_curl_cleanup_multi_native(",
		"@jayess_curl_perform_stream_native(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected curl package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/curl/native/curl.c") {
		t.Fatalf("expected curl native import to be carried through, got %#v", result.NativeImports)
	}
	if !containsPathSuffix(result.NativeIncludeDirs, "/refs/curl/include") {
		t.Fatalf("expected curl vendored include dir to be carried through, got %#v", result.NativeIncludeDirs)
	}
	if !containsString(result.NativeLinkFlags, "/lib/x86_64-linux-gnu/libcurl.so.4") {
		t.Fatalf("expected curl link flag to be carried through, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessLibUVPackage(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	dir, err := os.MkdirTemp(repoRoot, "jayess-libuv-loader-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
	import { createLoop, scheduleStop, scheduleCallback, readFile, watchSignal, closeSignalWatcher, watchPath, closePathWatcher, spawnProcess, closeProcess, createUDP, bindUDP, recvUDP, sendUDP, closeUDP, createTCPServer, listenTCP, acceptTCP, closeTCPServer, createTCPClient, connectTCP, readTCP, writeTCP, closeTCPClient, run, runOnce, stop, closeLoop, now } from "@jayess/libuv";

function main(args) {
  var loop = createLoop();
  scheduleStop(loop, 5);
  scheduleCallback(loop, 0, () => {});
  readFile(loop, "./hello.txt", (result) => {});
  var watcher = watchSignal(loop, "SIGUSR1", (signal) => {});
  var pathWatcher = watchPath(loop, "./hello.txt", (result) => {});
  var process = spawnProcess(loop, "/bin/sh", ["-c", "exit 0"], (result, proc) => {});
  var udp = createUDP(loop);
  var server = createTCPServer(loop);
  var client = createTCPClient(loop);
  bindUDP(udp, "127.0.0.1", 0);
  recvUDP(udp, (result) => {});
  sendUDP(udp, "127.0.0.1", 9, "ping");
  listenTCP(server, "127.0.0.1", 0, (result) => {
    var accepted = acceptTCP(server);
    if (accepted != undefined) {
      readTCP(accepted, (packet) => {});
      writeTCP(accepted, "pong");
      closeTCPClient(accepted);
    }
  });
  connectTCP(client, "127.0.0.1", 0, (result) => {});
  readTCP(client, (packet) => {});
  writeTCP(client, "ping");
  closeSignalWatcher(watcher);
  closePathWatcher(pathWatcher);
  closeProcess(process);
  closeUDP(udp);
  closeTCPServer(server);
  closeTCPClient(client);
  console.log(run(loop) + runOnce(loop) + now(loop));
  stop(loop);
  closeLoop(loop);
  return 0;
}

`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_libuv_create_loop_native(",
		"@jayess_libuv_run_native(",
		"@jayess_libuv_run_once_native(",
		"@jayess_libuv_stop_native(",
		"@jayess_libuv_close_loop_native(",
		"@jayess_libuv_schedule_stop_native(",
		"@jayess_libuv_schedule_callback_native(",
		"@jayess_libuv_read_file_native(",
		"@jayess_libuv_watch_signal_native(",
		"@jayess_libuv_close_signal_watcher_native(",
		"@jayess_libuv_watch_path_native(",
		"@jayess_libuv_close_path_watcher_native(",
		"@jayess_libuv_spawn_process_native(",
		"@jayess_libuv_close_process_native(",
		"@jayess_libuv_create_udp_native(",
		"@jayess_libuv_bind_udp_native(",
		"@jayess_libuv_recv_udp_native(",
		"@jayess_libuv_send_udp_native(",
		"@jayess_libuv_close_udp_native(",
		"@jayess_libuv_create_tcp_server_native(",
		"@jayess_libuv_listen_tcp_native(",
		"@jayess_libuv_accept_tcp_native(",
		"@jayess_libuv_close_tcp_server_native(",
		"@jayess_libuv_create_tcp_client_native(",
		"@jayess_libuv_connect_tcp_native(",
		"@jayess_libuv_read_tcp_native(",
		"@jayess_libuv_write_tcp_native(",
		"@jayess_libuv_close_tcp_client_native(",
		"@jayess_libuv_now_native(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected libuv package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/libuv/native/libuv.c") {
		t.Fatalf("expected libuv native import to be carried through, got %#v", result.NativeImports)
	}
	if !containsPathSuffix(result.NativeImports, "/refs/libuv/src/unix/linux.c") {
		t.Fatalf("expected vendored libuv linux source to be carried through, got %#v", result.NativeImports)
	}
	for _, flag := range []string{"-pthread", "-ldl", "-lrt"} {
		if !containsString(result.NativeLinkFlags, flag) {
			t.Fatalf("expected vendored libuv link flag %q to be carried through, got %#v", flag, result.NativeLinkFlags)
		}
	}
}

func TestCompilePathSupportsJayessHTMLPackage(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	dir, err := os.MkdirTemp(repoRoot, "jayess-html-loader-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	copyDirRecursiveCompilerTest(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "html"),
		filepath.Join(dir, "node_modules", "@jayess", "html"),
	)

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { tokenizeHtml, parseHtml, parseHtmlFragment, serializeHtml, createElement, createText, createComment, setAttribute, removeAttribute, appendChild, removeChild, replaceChild, cloneNode, walkDepthFirst, findByTag, matchesSelector, querySelectorAll } from "@jayess/html";

function main(args) {
  var tokens = tokenizeHtml("<!DOCTYPE html><div id=a disabled><span>hi</span><!--note--><br/></div>");
  var doc = parseHtml("<div id=a disabled><span>hi</span><!--note--><br/></div>");
  var fragment = parseHtmlFragment("<p class=x>ok</p>tail");
  var built = createElement("section", undefined, undefined);
  appendChild(built, createText("body"));
  appendChild(built, createComment("note"));
  setAttribute(built, "id", "s1");
  setAttribute(built, "class", "hero");
  removeAttribute(built, "id");
  var inner = createElement("span", undefined, undefined);
  appendChild(inner, createText("x"));
  appendChild(built, inner);
  replaceChild(built, 0, createText("lead"));
  removeChild(built, 1);
  var clone = cloneNode(built);
  console.log(tokens.length + doc.children.length + fragment.children.length + serializeHtml(doc).length + walkDepthFirst(clone).length + findByTag(doc, "span").length + querySelectorAll(doc, "div > span").length + matchesSelector(inner, "#s1") + doc.span.start.line + doc.children[0].span.end.column);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_html_tokenize(",
		"@jayess_html_parse_document(",
		"@jayess_html_parse_fragment(",
		"@jayess_html_serialize(",
		"@jayess_html_create_element(",
		"@jayess_html_create_text(",
		"@jayess_html_create_comment(",
		"@jayess_html_set_attribute(",
		"@jayess_html_remove_attribute(",
		"@jayess_html_append_child(",
		"@jayess_html_remove_child(",
		"@jayess_html_replace_child(",
		"@jayess_html_clone_node(",
		"@jayess_html_walk_depth_first(",
		"@jayess_html_find_by_tag(",
		"@jayess_html_matches_selector(",
		"@jayess_html_query_selector_all(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected HTML package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if len(result.NativeImports) != 1 || !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/html/native/html.c") {
		t.Fatalf("expected HTML native import to be carried through, got %#v", result.NativeImports)
	}
	if len(result.NativeLinkFlags) != 0 {
		t.Fatalf("expected HTML package to avoid extra link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessXMLPackage(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	dir, err := os.MkdirTemp(repoRoot, "jayess-xml-loader-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	copyDirRecursiveCompilerTest(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "xml"),
		filepath.Join(dir, "node_modules", "@jayess", "xml"),
	)
	if err := os.WriteFile(filepath.Join(dir, "sample.xml"), []byte(`<?xml version="1.0"?><root id="a"><child>hi</child><![CDATA[<raw>]]></root>`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ns.xml"), []byte(`<ns:root xmlns:ns="urn:test" xmlns="urn:default"><ns:item ns:id="7"/><child/></ns:root>`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { tokenizeXml, parseXml, serializeXml } from "@jayess/xml";

function main(args) {
  var source = fs.readFile("./sample.xml", "utf8");
  var tokens = tokenizeXml(source);
  var doc = parseXml(source);
  var nsDoc = parseXml(fs.readFile("./ns.xml", "utf8"));
  console.log(tokens.length + doc.children.length + serializeXml(doc).length + nsDoc.children[0].localName.length + nsDoc.children[0].children[0].namespaceURI.length + nsDoc.children[0].attributeDetails["xmlns:ns"].namespaceURI.length);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_xml_tokenize(",
		"@jayess_xml_parse_document(",
		"@jayess_xml_serialize(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected XML package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if len(result.NativeImports) != 1 || !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/xml/native/xml.c") {
		t.Fatalf("expected XML native import to be carried through, got %#v", result.NativeImports)
	}
	if len(result.NativeLinkFlags) != 0 {
		t.Fatalf("expected XML package to avoid extra link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsJayessCSSPackage(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	dir, err := os.MkdirTemp(repoRoot, "jayess-css-loader-*")
	if err != nil {
		t.Fatalf("MkdirTemp returned error: %v", err)
	}
	defer os.RemoveAll(dir)

	copyDirRecursiveCompilerTest(
		t,
		filepath.Join(repoRoot, "node_modules", "@jayess", "css"),
		filepath.Join(dir, "node_modules", "@jayess", "css"),
	)
	if err := os.WriteFile(filepath.Join(dir, "sample.css"), []byte(`@import "theme.css";
/*lead*/
@media screen and (min-width: 600px) { .card { padding: 8px; } }
.btn.primary, #app > .item { color: red; margin: 1.5rem; content: "hi"; }
#footer { padding: 8px; }`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	entry := filepath.Join(dir, "main.js")
	if err := os.WriteFile(entry, []byte(`
import { tokenizeCss, parseCss, serializeCss } from "@jayess/css";

function main(args) {
  var source = fs.readFile("./sample.css", "utf8");
  var tokens = tokenizeCss(source);
  var sheet = parseCss(source);
  console.log(tokens.length + sheet.rules.length + serializeCss(sheet).length);
  return 0;
}
`), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	result, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
	if err != nil {
		t.Fatalf("CompilePath returned error: %v", err)
	}
	irText := string(result.LLVMIR)
	for _, symbol := range []string{
		"@jayess_css_tokenize(",
		"@jayess_css_parse_stylesheet(",
		"@jayess_css_serialize(",
	} {
		if !strings.Contains(irText, symbol) {
			t.Fatalf("expected CSS package symbol %q in LLVM IR, got:\n%s", symbol, irText)
		}
	}
	if len(result.NativeImports) != 1 || !containsPathSuffix(result.NativeImports, "/node_modules/@jayess/css/native/css.c") {
		t.Fatalf("expected CSS native import to be carried through, got %#v", result.NativeImports)
	}
	if len(result.NativeLinkFlags) != 0 {
		t.Fatalf("expected CSS package to avoid extra link flags, got %#v", result.NativeLinkFlags)
	}
}

func TestCompilePathSupportsParserPackagesThroughLocalModules(t *testing.T) {
	repoRoot := repoRootFromCompilerTest(t)
	cases := []struct {
		name         string
		pkg          string
		moduleSource string
		entrySource  string
		symbols      []string
		importSuffix string
	}{
		{
			name: "html",
			pkg:  "html",
			moduleSource: `
import { parseHtmlFragment, querySelectorAll } from "@jayess/html";

export function htmlSpanCount(source) {
  var frag = parseHtmlFragment(source);
  return querySelectorAll(frag, "section > span").length + frag.children.length;
}
`,
			entrySource: `
import { htmlSpanCount } from "./lib/parsers.js";

function main(args) {
  console.log(htmlSpanCount("<section><span>a</span><span>b</span></section>"));
  return 0;
}
`,
			symbols:      []string{"@jayess_html_parse_fragment(", "@jayess_html_query_selector_all("},
			importSuffix: "/node_modules/@jayess/html/native/html.c",
		},
		{
			name: "xml",
			pkg:  "xml",
			moduleSource: `
import { parseXml } from "@jayess/xml";

export function xmlRootTag(source) {
  return parseXml(source).children[0].tagName;
}
`,
			entrySource: `
import { xmlRootTag } from "./lib/parsers.js";

function main(args) {
  console.log(xmlRootTag("<root><child/></root>"));
  return 0;
}
`,
			symbols:      []string{"@jayess_xml_parse_document("},
			importSuffix: "/node_modules/@jayess/xml/native/xml.c",
		},
		{
			name: "css",
			pkg:  "css",
			moduleSource: `
import { parseCss } from "@jayess/css";

export function cssRuleCount(source) {
  return parseCss(source).rules.length;
}
`,
			entrySource: `
import { cssRuleCount } from "./lib/parsers.js";

function main(args) {
  console.log(cssRuleCount(".a { color: red; } .b { color: blue; }"));
  return 0;
}
`,
			symbols:      []string{"@jayess_css_parse_stylesheet("},
			importSuffix: "/node_modules/@jayess/css/native/css.c",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := os.MkdirTemp(repoRoot, "jayess-parser-modules-loader-*")
			if err != nil {
				t.Fatalf("MkdirTemp returned error: %v", err)
			}
			defer os.RemoveAll(dir)

			copyDirRecursiveCompilerTest(
				t,
				filepath.Join(repoRoot, "node_modules", "@jayess", tc.pkg),
				filepath.Join(dir, "node_modules", "@jayess", tc.pkg),
			)

			libDir := filepath.Join(dir, "lib")
			if err := os.MkdirAll(libDir, 0o755); err != nil {
				t.Fatalf("MkdirAll returned error: %v", err)
			}
			if err := os.WriteFile(filepath.Join(libDir, "parsers.js"), []byte(tc.moduleSource), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			entry := filepath.Join(dir, "main.js")
			if err := os.WriteFile(entry, []byte(tc.entrySource), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}

			result, err := CompilePath(entry, Options{TargetTriple: "x86_64-unknown-linux-gnu"})
			if err != nil {
				t.Fatalf("CompilePath returned error: %v", err)
			}
			irText := string(result.LLVMIR)
			for _, symbol := range tc.symbols {
				if !strings.Contains(irText, symbol) {
					t.Fatalf("expected parser package symbol %q in LLVM IR, got:\n%s", symbol, irText)
				}
			}
			if !containsPathSuffix(result.NativeImports, tc.importSuffix) {
				t.Fatalf("expected parser native import %q to be carried through, got %#v", tc.importSuffix, result.NativeImports)
			}
			if len(result.NativeLinkFlags) != 0 {
				t.Fatalf("expected parser package to avoid extra link flags, got %#v", result.NativeLinkFlags)
			}
		})
	}
}
