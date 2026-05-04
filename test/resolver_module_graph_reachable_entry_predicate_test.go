package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksReachableModuleForEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", nil)

	reaches, err := graph.ReachesModule("main.js", "model.js")
	if err != nil {
		t.Fatalf("ReachesModule returned error: %v", err)
	}
	if !reaches {
		t.Fatalf("expected main.js to reach model.js")
	}

	reaches, err = graph.ReachesModule("main.js", "worker.js")
	if err != nil {
		t.Fatalf("ReachesModule returned error: %v", err)
	}
	if reaches {
		t.Fatalf("did not expect main.js to reach worker.js")
	}
}

func TestResolverModuleGraphChecksReachableModuleForUnknownEntry(t *testing.T) {
	graph := resolver.NewModuleGraph()

	reaches, err := graph.ReachesModule("main.js", "main.js")
	if err != nil {
		t.Fatalf("ReachesModule returned error: %v", err)
	}
	if !reaches {
		t.Fatalf("expected unknown entry to reach itself")
	}
}

func TestResolverModuleGraphReachableModulePredicateReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachesModule("main.js", "b.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
