package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsReachableModulesForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)

	modules, err := graph.ReachableModules("main.js")
	if err != nil {
		t.Fatalf("ReachableModules returned error: %v", err)
	}
	expected := []string{"app.js", "config.js", "main.js", "model.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected reachable modules %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphReachableModulesForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	modules, err := graph.ReachableModules("main.js")
	if err != nil {
		t.Fatalf("ReachableModules returned error: %v", err)
	}
	expected := []string{"main.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected unknown entry modules %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphReachableModulesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModules("main.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
