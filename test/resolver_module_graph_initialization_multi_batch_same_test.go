package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksMultiInitializationSameBatch(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js", "config.js"})
	graph.AddModule("worker.js", []string{"app.js", "worker_config.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("config.js", nil)
	graph.AddModule("model.js", nil)
	graph.AddModule("worker_config.js", nil)

	same, err := graph.InitializesInSameBatchFor([]string{"main.js", "worker.js"}, "config.js", "model.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchFor returned error: %v", err)
	}
	if !same {
		t.Fatal("expected config.js and model.js to initialize in the same batch")
	}

	same, err = graph.InitializesInSameBatchFor([]string{"main.js", "worker.js"}, "app.js", "worker.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchFor returned error: %v", err)
	}
	if same {
		t.Fatal("did not expect app.js and worker.js to initialize in the same batch")
	}
}

func TestResolverModuleGraphMultiInitializationSameBatchMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	same, err := graph.InitializesInSameBatchFor([]string{"main.js"}, "app.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchFor returned error: %v", err)
	}
	if same {
		t.Fatal("did not expect missing module to participate in an initialization batch")
	}
}

func TestResolverModuleGraphMultiInitializationSameBatchSameModule(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	same, err := graph.InitializesInSameBatchFor([]string{"main.js"}, "main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchFor returned error: %v", err)
	}
	if !same {
		t.Fatal("expected a module to share an initialization batch with itself")
	}
}

func TestResolverModuleGraphMultiInitializationSameBatchReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesInSameBatchFor([]string{"main.js", "a.js"}, "a.js", "b.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
