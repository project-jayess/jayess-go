package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationOrderPositionFor(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)

	position, ok, err := graph.InitializationOrderPositionFor([]string{"main.js", "worker.js"}, "worker_dep.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositionFor returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected worker_dep.js to be in combined initialization order")
	}
	if position != 2 {
		t.Fatalf("expected worker_dep.js position 2, got %d", position)
	}
}

func TestResolverModuleGraphInitializationOrderPositionForReportsMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("shared.js", nil)

	position, ok, err := graph.InitializationOrderPositionFor([]string{"main.js", "worker.js"}, "missing.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositionFor returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected missing module lookup to fail, got position %d", position)
	}
}

func TestResolverModuleGraphInitializationOrderPositionForReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.InitializationOrderPositionFor([]string{"main.js", "a.js"}, "a.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphMultiInitializationOrderPositionForEmptyGraphIsMissing(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	position, ok, err := graph.InitializationOrderPositionFor([]string{"main.js"}, "main.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositionFor returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected empty graph lookup to fail, got position %d", position)
	}
}
