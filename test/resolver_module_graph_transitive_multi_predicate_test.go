package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphTransitivelyDependsOnForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)

	depends, err := graph.TransitivelyDependsOnFor([]string{"main.js", "worker.js"}, "worker_dep.js")
	if err != nil {
		t.Fatalf("TransitivelyDependsOnFor returned error: %v", err)
	}
	if !depends {
		t.Fatal("expected entries to transitively depend on worker_dep.js")
	}
}

func TestResolverModuleGraphTransitivelyDependsOnForEntriesFalse(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("unused.js", nil)

	depends, err := graph.TransitivelyDependsOnFor([]string{"main.js"}, "unused.js")
	if err != nil {
		t.Fatalf("TransitivelyDependsOnFor returned error: %v", err)
	}
	if depends {
		t.Fatal("did not expect entry to transitively depend on unused.js")
	}
}

func TestResolverModuleGraphTransitivelyDependsOnForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.TransitivelyDependsOnFor([]string{"main.js", "a.js"}, "b.js")
	if err == nil {
		t.Fatal("expected import cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
