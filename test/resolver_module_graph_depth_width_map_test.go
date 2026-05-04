package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphExportsDependencyDepthWidthMap(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	widths, err := graph.DependencyDepthWidthMap()
	if err != nil {
		t.Fatalf("unexpected depth width map error: %v", err)
	}

	expected := map[int]int{
		0: 2,
		1: 1,
		2: 2,
	}
	if !reflect.DeepEqual(widths, expected) {
		t.Fatalf("unexpected dependency depth widths: got %#v want %#v", widths, expected)
	}
}

func TestResolverModuleGraphDependencyDepthWidthMapEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	widths, err := graph.DependencyDepthWidthMap()
	if err != nil {
		t.Fatalf("unexpected empty depth width map error: %v", err)
	}
	if widths != nil {
		t.Fatalf("expected nil depth width map for empty graph, got %#v", widths)
	}
}

func TestResolverModuleGraphDependencyDepthWidthMapReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependencyDepthWidthMap()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
