package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphMeasuresDependencyDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", []string{"storage.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("storage.js", nil)

	depth, err := graph.DependencyDepth("main.js")
	if err != nil {
		t.Fatalf("DependencyDepth returned error: %v", err)
	}
	if depth != 3 {
		t.Fatalf("expected dependency depth 3, got %d", depth)
	}
}

func TestResolverModuleGraphDependencyDepthForLeafAndMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	depth, err := graph.DependencyDepth("main.js")
	if err != nil {
		t.Fatalf("DependencyDepth returned error for leaf: %v", err)
	}
	if depth != 0 {
		t.Fatalf("expected leaf depth 0, got %d", depth)
	}

	depth, err = graph.DependencyDepth("missing.js")
	if err != nil {
		t.Fatalf("DependencyDepth returned error for missing module: %v", err)
	}
	if depth != 0 {
		t.Fatalf("expected missing module depth 0, got %d", depth)
	}
}

func TestResolverModuleGraphDependencyDepthReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependencyDepth("main.js")
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
	expected := []string{"a.js", "b.js", "a.js"}
	if !reflect.DeepEqual(cycleErr.Cycle, expected) {
		t.Fatalf("expected cycle %#v, got %#v", expected, cycleErr.Cycle)
	}
}

func TestResolverModuleGraphExportsDependencyDepthMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	depths, err := graph.DependencyDepthMap()
	if err != nil {
		t.Fatalf("DependencyDepthMap returned error: %v", err)
	}
	expected := map[string]int{
		"app.js":    1,
		"config.js": 0,
		"main.js":   2,
		"model.js":  0,
		"worker.js": 1,
	}
	if !reflect.DeepEqual(depths, expected) {
		t.Fatalf("expected dependency depth map %#v, got %#v", expected, depths)
	}
}
