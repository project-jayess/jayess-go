package test

import (
	"errors"
	"reflect"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphOrdersMultipleEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"shared.js"})
	graph.AddModule("worker.js", []string{"shared.js", "worker_dep.js"})
	graph.AddModule("shared.js", nil)
	graph.AddModule("worker_dep.js", nil)

	order, err := graph.InitializationOrderFor([]string{"main.js", "worker.js"})
	if err != nil {
		t.Fatalf("InitializationOrderFor returned error: %v", err)
	}
	expected := []string{"shared.js", "main.js", "worker_dep.js", "worker.js"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("expected initialization order %#v, got %#v", expected, order)
	}
}

func TestResolverModuleGraphReportsCycleAcrossMultipleEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializationOrderFor([]string{"main.js", "a.js"})
	if err == nil {
		t.Fatalf("expected import cycle error")
	}
	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
	expected := []string{"a.js", "b.js", "a.js"}
	if !reflect.DeepEqual(cycleErr.Cycle, expected) {
		t.Fatalf("expected cycle %#v, got %#v", expected, cycleErr.Cycle)
	}
}
