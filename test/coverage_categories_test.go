package test

import (
	"testing"

	"jayess-go/coverage"
)

func TestCoverageCategoriesAreDeclared(t *testing.T) {
	expected := []string{
		"lexer",
		"parser",
		"ast",
		"semantic",
		"type-checking",
		"lifetime-escape",
		"codegen",
		"llvm-ir",
		"runtime",
		"filesystem",
		"network",
		"module-resolution",
		"cross-platform",
		"e2e-native",
		"regression",
	}
	for _, name := range expected {
		if !coverage.HasCategory(name) {
			t.Fatalf("expected coverage category %s", name)
		}
	}
}

func TestCoverageCategoriesHaveTestScopes(t *testing.T) {
	for _, category := range coverage.Categories() {
		if category.Name == "" {
			t.Fatalf("coverage category has empty name: %#v", category)
		}
		if category.TestScope == "" {
			t.Fatalf("coverage category %s has empty test scope", category.Name)
		}
	}
}
