package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsDependencyDepthLayers(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	layers, err := graph.DependencyDepthLayers()
	if err != nil {
		t.Fatalf("DependencyDepthLayers returned error: %v", err)
	}

	expected := [][]string{
		{"config.js", "model.js"},
		{"app.js"},
		{"main.js", "worker.js"},
	}
	if !reflect.DeepEqual(layers, expected) {
		t.Fatalf("expected dependency depth layers %#v, got %#v", expected, layers)
	}
}

func TestResolverModuleGraphDependencyDepthLayersForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	layers, err := graph.DependencyDepthLayers()
	if err != nil {
		t.Fatalf("DependencyDepthLayers returned error: %v", err)
	}
	if layers != nil {
		t.Fatalf("expected nil layers for empty graph, got %#v", layers)
	}
}

func TestResolverModuleGraphDependencyDepthLayersReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependencyDepthLayers()
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
