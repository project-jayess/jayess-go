package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsDeepestDependentModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	modules, depth, err := graph.DeepestDependentModules()
	if err != nil {
		t.Fatalf("DeepestDependentModules returned error: %v", err)
	}
	expectedModules := []string{"model.js"}
	if !reflect.DeepEqual(modules, expectedModules) {
		t.Fatalf("expected deepest dependent modules %#v, got %#v", expectedModules, modules)
	}
	if depth != 2 {
		t.Fatalf("expected deepest dependent depth 2, got %d", depth)
	}
}

func TestResolverModuleGraphDeepestDependentModulesForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	modules, depth, err := graph.DeepestDependentModules()
	if err != nil {
		t.Fatalf("DeepestDependentModules returned error: %v", err)
	}
	if modules != nil {
		t.Fatalf("expected nil modules for empty graph, got %#v", modules)
	}
	if depth != 0 {
		t.Fatalf("expected empty graph depth 0, got %d", depth)
	}
}

func TestResolverModuleGraphDeepestDependentModulesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.DeepestDependentModules()
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
