package test

import "testing"

func TestSemanticAllowsNamedReExportList(t *testing.T) {
	err := analyzeSource(t, `export { add as sum } from "./math.js";`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsExportAll(t *testing.T) {
	err := analyzeSource(t, `export * from "./more.js";`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsNamespaceReExport(t *testing.T) {
	err := analyzeSource(t, `export * as math from "./more.js";`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsParentDirectoryReExport(t *testing.T) {
	err := analyzeSource(t, `export { add } from "../lib/math.js";`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticAllowsBarePackageReExport(t *testing.T) {
	err := analyzeSource(t, `export * from "math";`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestSemanticRejectsURLLikeReExportSource(t *testing.T) {
	err := analyzeSource(t, `export * from "https://example.test/math.js";`)
	requireSemanticError(t, err, `unsupported module source "https://example.test/math.js"`)
}
