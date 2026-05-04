package test

import "testing"

func TestSemanticRejectsURLImportSource(t *testing.T) {
	err := analyzeSource(t, `import { add } from "https://example.com/math.js";`)
	requireSemanticError(t, err, `unsupported module source "https://example.com/math.js"`)
}

func TestSemanticRejectsSchemeLikeImportSource(t *testing.T) {
	err := analyzeSource(t, `import { readFile } from "node:fs";`)
	requireSemanticError(t, err, `unsupported module source "node:fs"`)
}

func TestSemanticRejectsWindowsAbsoluteImportSource(t *testing.T) {
	err := analyzeSource(t, `import { add } from "C:/project/math.js";`)
	requireSemanticError(t, err, `unsupported module source "C:/project/math.js"`)
}

func TestSemanticRejectsSchemeLikeReExportSource(t *testing.T) {
	err := analyzeSource(t, `export { add } from "npm:math";`)
	requireSemanticError(t, err, `unsupported module source "npm:math"`)
}

func TestSemanticRejectsScopedPackageWithoutPackageName(t *testing.T) {
	err := analyzeSource(t, `import { add } from "@scope";`)
	requireSemanticError(t, err, `unsupported module source "@scope"`)
}

func TestSemanticRejectsScopedPackageWithEmptyScope(t *testing.T) {
	err := analyzeSource(t, `import { add } from "@/math";`)
	requireSemanticError(t, err, `unsupported module source "@/math"`)
}

func TestSemanticRejectsScopedPackageWithEmptyPackageName(t *testing.T) {
	err := analyzeSource(t, `import { add } from "@scope/";`)
	requireSemanticError(t, err, `unsupported module source "@scope/"`)
}

func TestSemanticRejectsPackageSourceWithTrailingSlash(t *testing.T) {
	err := analyzeSource(t, `import { add } from "math/";`)
	requireSemanticError(t, err, `unsupported module source "math/"`)
}

func TestSemanticRejectsPackageSourceWithEmptySubpathSegment(t *testing.T) {
	err := analyzeSource(t, `import { add } from "math//utils";`)
	requireSemanticError(t, err, `unsupported module source "math//utils"`)
}

func TestSemanticRejectsScopedPackageSourceWithEmptySubpathSegment(t *testing.T) {
	err := analyzeSource(t, `import { add } from "@scope/math//utils";`)
	requireSemanticError(t, err, `unsupported module source "@scope/math//utils"`)
}

func TestSemanticRejectsPackageSourceWithCurrentDirectorySegment(t *testing.T) {
	err := analyzeSource(t, `import { add } from "math/./utils";`)
	requireSemanticError(t, err, `unsupported module source "math/./utils"`)
}

func TestSemanticRejectsPackageSourceWithParentDirectorySegment(t *testing.T) {
	err := analyzeSource(t, `import { add } from "math/../utils";`)
	requireSemanticError(t, err, `unsupported module source "math/../utils"`)
}

func TestSemanticRejectsScopedPackageSourceWithParentDirectorySegment(t *testing.T) {
	err := analyzeSource(t, `import { add } from "@scope/math/../utils";`)
	requireSemanticError(t, err, `unsupported module source "@scope/math/../utils"`)
}

func TestSemanticRejectsBackslashModuleSource(t *testing.T) {
	err := analyzeSource(t, `import { add } from "math\utils";`)
	requireSemanticError(t, err, `use / as the module path separator`)
}
