package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphListsReachableModuleOrderForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)
	graph.AddModule("unused.js", nil)

	order, err := graph.ReachableModuleOrderFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("ReachableModuleOrderFor returned error: %v", err)
	}
	expected := []string{"shared.js", "main.js", "worker_dep.js", "worker.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected reachable order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphReachableModuleOrderForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	order, err := graph.ReachableModuleOrderFor(nil)
	if err != nil {
		t.Fatalf("ReachableModuleOrderFor returned error: %v", err)
	}
	if order != nil {
		t.Fatalf("expected nil order for no entries, got %#v", order)
	}
}

func TestResolverModuleGraphReachableModuleOrderForEntriesReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachableModuleOrderFor([]string{"main.js", "a.js"})
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
