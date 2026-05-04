package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsReachableModulesForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	modules, err := graph.ReachableModulesFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("ReachableModulesFor returned error: %v", err)
	}
	expected := []string{"app.js", "config.js", "main.js", "model.js", "worker-model.js", "worker.js"}
	if !reflect.DeepEqual(modules, expected) {
		t.Fatalf("expected reachable modules %#v, got %#v", expected, modules)
	}
}

func TestResolverModuleGraphReachableModulesForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	modules, err := graph.ReachableModulesFor(nil)
	if err != nil {
		t.Fatalf("ReachableModulesFor returned error: %v", err)
	}
	if len(modules) != 0 {
		t.Fatalf("expected no modules for no entries, got %#v", modules)
	}
}

func TestResolverModuleGraphReachableModulesForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModulesFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
