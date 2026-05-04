package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitiveDependencyCountForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)
	graph.AddModule("unused.js", nil)

	count, err := graph.TransitiveDependencyCountFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("TransitiveDependencyCountFor returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 transitive dependencies, got %d", count)
	}
}

func TestResolverModuleGraphTransitiveDependencyCountForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	count, err := graph.TransitiveDependencyCountFor(nil)
	if err != nil {
		t.Fatalf("TransitiveDependencyCountFor returned error: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no-entry dependency count 0, got %d", count)
	}
}

func TestResolverModuleGraphTransitiveDependencyCountForEntriesDetectsCycle(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitiveDependencyCountFor([]string{"main.js", "a.js"})
	if err == nil {
		t.Fatal("expected import cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
