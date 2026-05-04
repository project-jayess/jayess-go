package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsDependentDepthWidthMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	widths, err := graph.DependentDepthWidthMap()
	if err != nil {
		t.Fatalf("unexpected dependent depth width map error: %v", err)
	}

	expected := map[int]int{
		0: 3,
		1: 1,
		2: 1,
	}
	if !reflect.DeepEqual(widths, expected) {
		t.Fatalf("unexpected dependent depth widths: got %#v want %#v", widths, expected)
	}
}

func TestResolverModuleGraphDependentDepthWidthMapEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	widths, err := graph.DependentDepthWidthMap()
	if err != nil {
		t.Fatalf("unexpected empty dependent depth width map error: %v", err)
	}
	if widths != nil {
		t.Fatalf("expected nil dependent depth width map for empty graph, got %#v", widths)
	}
}

func TestResolverModuleGraphDependentDepthWidthMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependentDepthWidthMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
