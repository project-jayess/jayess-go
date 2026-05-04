package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphMeasuresDependentDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", []string{"storage.js"})
	graph.AddModule("worker.js", []string{"storage.js"})
	graph.AddModule("storage.js", nil)

	depth, err := graph.DependentDepth("storage.js")
	if err != nil {
		t.Fatalf("DependentDepth returned error: %v", err)
	}
	if depth != 3 {
		t.Fatalf("expected dependent depth 3, got %d", depth)
	}
}

func TestResolverModuleGraphDependentDepthForRootAndMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	depth, err := graph.DependentDepth("main.js")
	if err != nil {
		t.Fatalf("DependentDepth returned error for root: %v", err)
	}
	if depth != 0 {
		t.Fatalf("expected root dependent depth 0, got %d", depth)
	}

	depth, err = graph.DependentDepth("missing.js")
	if err != nil {
		t.Fatalf("DependentDepth returned error for missing module: %v", err)
	}
	if depth != 0 {
		t.Fatalf("expected missing module dependent depth 0, got %d", depth)
	}
}

func TestResolverModuleGraphDependentDepthReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependentDepth("a.js")
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

func TestResolverModuleGraphExportsDependentDepthMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("worker.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	depths, err := graph.DependentDepthMap()
	if err != nil {
		t.Fatalf("DependentDepthMap returned error: %v", err)
	}
	expected := map[string]int{
		"app.js":    1,
		"config.js": 1,
		"main.js":   0,
		"model.js":  2,
		"worker.js": 0,
	}
	if !reflect.DeepEqual(depths, expected) {
		t.Fatalf("expected dependent depth map %#v, got %#v", expected, depths)
	}
}
