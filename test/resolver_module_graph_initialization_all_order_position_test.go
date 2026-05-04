package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationOrderPositionAll(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("worker.js", []string{"shared.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("shared.js", nil)

	position, ok, err := graph.InitializationOrderPositionAll("shared.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositionAll returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected shared.js to be in full initialization order")
	}
	if position != 2 {
		t.Fatalf("expected shared.js position 2, got %d", position)
	}
}

func TestResolverModuleGraphInitializationOrderPositionAllReportsMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"config.js"})
	graph.AddModule("config.js", nil)

	position, ok, err := graph.InitializationOrderPositionAll("missing.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositionAll returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected missing module lookup to fail, got position %d", position)
	}
}

func TestResolverModuleGraphInitializationOrderPositionAllReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.InitializationOrderPositionAll("a.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationOrderPositionAllForEmptyGraphIsMissing(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	position, ok, err := graph.InitializationOrderPositionAll("main.js")
	if err != nil {
		t.Fatalf("InitializationOrderPositionAll returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected empty graph lookup to fail, got position %d", position)
	}
}
