package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsDependentDepthLevels(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	levels, err := graph.DependentDepthLevels()
	if err != nil {
		t.Fatalf("DependentDepthLevels returned error: %v", err)
	}

	expected := []int{0, 1, 2}
	if !reflect.DeepEqual(levels, expected) {
		t.Fatalf("expected dependent depth levels %#v, got %#v", expected, levels)
	}
}

func TestResolverModuleGraphDependentDepthLevelsForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	levels, err := graph.DependentDepthLevels()
	if err != nil {
		t.Fatalf("DependentDepthLevels returned error: %v", err)
	}
	if levels != nil {
		t.Fatalf("expected nil levels for empty graph, got %#v", levels)
	}
}

func TestResolverModuleGraphDependentDepthLevelsReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependentDepthLevels()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
