package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphCountsTransitiveDependencies(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	count, err := graph.TransitiveDependencyCount("main.js")
	if err != nil {
		t.Fatalf("TransitiveDependencyCount returned error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 transitive dependencies, got %d", count)
	}
}

func TestResolverModuleGraphTransitiveDependencyCountReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencyCount("main.js")
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphCountsTransitiveDependents(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("model.js", nil)

	if count := graph.TransitiveDependentCount("model.js"); count != 3 {
		t.Fatalf("expected 3 transitive dependents, got %d", count)
	}
}
