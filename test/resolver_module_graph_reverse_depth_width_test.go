package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsWidestDependentDepths(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	depths, width, err := graph.WidestDependentDepths()
	if err != nil {
		t.Fatalf("WidestDependentDepths returned error: %v", err)
	}
	expectedDepths := []int{0}
	if !reflect.DeepEqual(depths, expectedDepths) {
		t.Fatalf("expected widest dependent depths %#v, got %#v", expectedDepths, depths)
	}
	if width != 3 {
		t.Fatalf("expected widest dependent depth width 3, got %d", width)
	}
}

func TestResolverModuleGraphWidestDependentDepthsForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	depths, width, err := graph.WidestDependentDepths()
	if err != nil {
		t.Fatalf("WidestDependentDepths returned error: %v", err)
	}
	if depths != nil {
		t.Fatalf("expected nil depths for empty graph, got %#v", depths)
	}
	if width != 0 {
		t.Fatalf("expected empty graph width 0, got %d", width)
	}
}

func TestResolverModuleGraphWidestDependentDepthsReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.WidestDependentDepths()
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
