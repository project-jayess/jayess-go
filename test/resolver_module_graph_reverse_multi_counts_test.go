package test

import (
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsTransitiveDependentsForModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	count := graph.TransitiveDependentCountFor([]string{"model.js", "config.js"})
	if count != 3 {
		t.Fatalf("expected 3 transitive dependents, got %d", count)
	}
}

func TestResolverModuleGraphCountsTransitiveDependentsForNoModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	if count := graph.TransitiveDependentCountFor(nil); count != 0 {
		t.Fatalf("expected 0 transitive dependents for no modules, got %d", count)
	}
}

func TestResolverModuleGraphCountsTransitiveDependentsForModulesSkipsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})
	graph.AddModule("main.js", []string{"a.js"})

	count := graph.TransitiveDependentCountFor([]string{"a.js", "b.js"})
	if count != 3 {
		t.Fatalf("expected 3 transitive dependents, got %d", count)
	}
}
