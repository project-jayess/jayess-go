package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphReachableModuleSetForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	set, err := graph.ReachableModuleSetFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("ReachableModuleSetFor returned error: %v", err)
	}
	expected := map[string]bool{
		"app.js":          true,
		"config.js":       true,
		"main.js":         true,
		"model.js":        true,
		"worker-model.js": true,
		"worker.js":       true,
	}
	if !reflect.DeepEqual(set, expected) {
		t.Fatalf("expected reachable module set %#v, got %#v", expected, set)
	}
}

func TestResolverModuleGraphReachableModuleSetForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	set, err := graph.ReachableModuleSetFor(nil)
	if err != nil {
		t.Fatalf("ReachableModuleSetFor returned error: %v", err)
	}
	if set != nil {
		t.Fatalf("expected nil set for no entries, got %#v", set)
	}
}

func TestResolverModuleGraphReachableModuleSetForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleSetFor([]string{"main.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
