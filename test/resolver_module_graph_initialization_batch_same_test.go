package test

import (
	"errors"
	"testing"

	"jayess-go/resolver"
)

func TestResolverModuleGraphChecksFullInitializationSameBatch(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("worker.js", []string{"app.js"})
	graph.AddModule("app.js", []string{"model.js"})
	graph.AddModule("model.js", nil)
	graph.AddModule("config.js", nil)

	same, err := graph.InitializesInSameBatchAll("config.js", "model.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchAll returned error: %v", err)
	}
	if !same {
		t.Fatal("expected config.js and model.js to initialize in the same batch")
	}

	same, err = graph.InitializesInSameBatchAll("app.js", "worker.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchAll returned error: %v", err)
	}
	if same {
		t.Fatal("did not expect app.js and worker.js to initialize in the same batch")
	}
}

func TestResolverModuleGraphFullInitializationSameBatchMissingModules(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", []string{"app.js"})
	graph.AddModule("app.js", nil)

	same, err := graph.InitializesInSameBatchAll("app.js", "missing.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchAll returned error: %v", err)
	}
	if same {
		t.Fatal("did not expect missing module to participate in an initialization batch")
	}
}

func TestResolverModuleGraphFullInitializationSameBatchSameModule(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("main.js", nil)

	same, err := graph.InitializesInSameBatchAll("main.js", "main.js")
	if err != nil {
		t.Fatalf("InitializesInSameBatchAll returned error: %v", err)
	}
	if !same {
		t.Fatal("expected a module to share an initialization batch with itself")
	}
}

func TestResolverModuleGraphFullInitializationSameBatchReportsCycles(t *testing.T) {
	graph := resolver.NewModuleGraph()
	graph.AddModule("a.js", []string{"b.js"})
	graph.AddModule("b.js", []string{"a.js"})

	_, err := graph.InitializesInSameBatchAll("a.js", "b.js")
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var cycleErr *resolver.ImportCycleError
	if !errors.As(err, &cycleErr) {
		t.Fatalf("expected ImportCycleError, got %T: %v", err, err)
	}
}
