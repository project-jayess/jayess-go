package test

import (
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphClearsImports(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js", "config.js"})

	if !graph.ClearImports("main.js") {
		t.Fatalf("expected imports to be cleared")
	}
	if dependencies := graph.Dependencies("main.js"); len(dependencies) != 0 {
		t.Fatalf("expected no dependencies, got %#v", dependencies)
	}
	if !graph.HasModule("main.js") {
		t.Fatalf("expected main.js to remain in graph")
	}
}

func TestResolverModuleGraphClearImportsReportsMissingModule(t *testing.T) {
	graph := resolver.NewModuleGraph()

	if graph.ClearImports("missing.js") {
		t.Fatalf("did not expect missing module imports to be cleared")
	}
}
