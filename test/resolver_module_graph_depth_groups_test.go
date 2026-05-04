package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphGroupsModulesByDependencyDepth(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	groups, err := graph.DependencyDepthGroups()
	if err != nil {
		t.Fatalf("DependencyDepthGroups returned error: %v", err)
	}
	expected := map[int][]string{
		0: []string{"config.js", "model.js"},
		1: []string{"app.js"},
		2: []string{"main.js", "worker.js"},
	}
	if !reflect.DeepEqual(groups, expected) {
		t.Fatalf("expected dependency depth groups %#v, got %#v", expected, groups)
	}
}

func TestResolverModuleGraphDependencyDepthGroupsForEmptyGraph(t *testing.T) {
	graph := resolver.NewModuleGraph()

	groups, err := graph.DependencyDepthGroups()
	if err != nil {
		t.Fatalf("DependencyDepthGroups returned error: %v", err)
	}
	if groups != nil {
		t.Fatalf("expected nil groups for empty graph, got %#v", groups)
	}
}

func TestResolverModuleGraphDependencyDepthGroupsReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.DependencyDepthGroups()
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
