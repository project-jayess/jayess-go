package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsDependencyDepthLayerWidths(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	widths, err := graph.DependencyDepthLayerWidths()
	if err != nil {
		t.Fatalf("DependencyDepthLayerWidths returned error: %v", err)
	}

	expected := []int{2, 1, 2}
	if !reflect.DeepEqual(widths, expected) {
		t.Fatalf("expected dependency depth layer widths %#v, got %#v", expected, widths)
	}
}

func TestResolverModuleGraphDependencyDepthLayerWidthsForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	widths, err := graph.DependencyDepthLayerWidths()
	if err != nil {
		t.Fatalf("DependencyDepthLayerWidths returned error: %v", err)
	}
	if widths != nil {
		t.Fatalf("expected nil widths for empty graph, got %#v", widths)
	}
}

func TestResolverModuleGraphDependencyDepthLayerWidthsReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependencyDepthLayerWidths()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
