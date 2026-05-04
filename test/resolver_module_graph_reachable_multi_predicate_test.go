package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksReachableModuleForEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)
	graph.AddModule("worker.js", []string{"worker-model.js"})
	graph.AddModule("worker-model.js", nil)
	graph.AddModule("unused.js", nil)

	reaches, err := graph.ReachesModuleFor([]string{"main.js", "worker.js"}, "worker-model.js")
	if err != nil {
		t.Fatalf("ReachesModuleFor returned error: %v", err)
	}
	if !reaches {
		t.Fatalf("expected entries to reach worker-model.js")
	}

	reaches, err = graph.ReachesModuleFor([]string{"main.js", "worker.js"}, "unused.js")
	if err != nil {
		t.Fatalf("ReachesModuleFor returned error: %v", err)
	}
	if reaches {
		t.Fatalf("did not expect entries to reach unused.js")
	}
}

func TestResolverModuleGraphChecksReachableModuleForNoEntries(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	reaches, err := graph.ReachesModuleFor(nil, "main.js")
	if err != nil {
		t.Fatalf("ReachesModuleFor returned error: %v", err)
	}
	if reaches {
		t.Fatalf("did not expect no entries to reach main.js")
	}
}

func TestResolverModuleGraphReachableModuleForPredicateReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"a.js"})
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.ReachesModuleFor([]string{"main.js"}, "b.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
