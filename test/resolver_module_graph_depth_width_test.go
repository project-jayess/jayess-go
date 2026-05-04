package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphFindsWidestDependencyDepths(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	depths, width, err := graph.WidestDependencyDepths()
	if err != nil {
		t.Fatalf("WidestDependencyDepths returned error: %v", err)
	}
	expectedDepths := []int{0, 2}
	if !reflect.DeepEqual(depths, expectedDepths) {
		t.Fatalf("expected widest dependency depths %#v, got %#v", expectedDepths, depths)
	}
	if width != 2 {
		t.Fatalf("expected widest dependency depth width 2, got %d", width)
	}
}

func TestResolverModuleGraphWidestDependencyDepthsForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	depths, width, err := graph.WidestDependencyDepths()
	if err != nil {
		t.Fatalf("WidestDependencyDepths returned error: %v", err)
	}
	if depths != nil {
		t.Fatalf("expected nil depths for empty graph, got %#v", depths)
	}
	if width != 0 {
		t.Fatalf("expected empty graph width 0, got %d", width)
	}
}

func TestResolverModuleGraphWidestDependencyDepthsReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, _, err := graph.WidestDependencyDepths()
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
