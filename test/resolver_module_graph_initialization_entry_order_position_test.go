package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsInitializationOrderPosition(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)

	position, ok, err := graph.InitializationOrderPosition("main.js", "config.js")
	if err != nil {
		t.Fatalf("InitializationOrderPosition returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected config.js to be in main.js initialization order")
	}
	if position != 2 {
		t.Fatalf("expected config.js position 2, got %d", position)
	}
}

func TestResolverModuleGraphInitializationOrderPositionReportsMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	position, ok, err := graph.InitializationOrderPosition("main.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializationOrderPosition returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected missing module lookup to fail, got position %d", position)
	}
}

func TestResolverModuleGraphInitializationOrderPositionReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.InitializationOrderPosition("main.js", "a.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}

func TestResolverModuleGraphInitializationOrderPositionForEmptyGraphIsMissing(t *testing.T) {
	graph := &resolver.ModuleGraph{}

	position, ok, err := graph.InitializationOrderPosition("main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializationOrderPosition returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected empty graph lookup to fail, got position %d", position)
	}
}
