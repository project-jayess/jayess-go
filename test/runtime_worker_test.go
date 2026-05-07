package test

import (
	"testing"
	"time"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeWorkerCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"thread",
		"postMessage",
		"onMessage",
		"sharedMemory",
		"atomicLoad",
		"atomicStore",
	}
	for _, name := range expected {
		if !jayessruntime.HasWorkerCapability(name) {
			t.Fatalf("expected worker runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsWorkerSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(source) {
			const thread = worker.thread(source);
			worker.postMessage(thread, { ready: true });
			worker.onMessage(thread, (message) => message);
			const memory = worker.sharedMemory(1024);
			worker.atomicStore(memory, 0, 1);
			const value = worker.atomicLoad(memory, 0);
			return thread || value;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeWorkerCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.WorkerCapabilities() {
		if capability.Name == "" {
			t.Fatalf("worker capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("worker capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("worker capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestRuntimeWorkerMessageFlowAndCleanup(t *testing.T) {
	worker := jayessruntime.NewWorker(func(worker *jayessruntime.Worker) {
		message, ok := worker.Receive()
		if !ok {
			return
		}
		_ = worker.PostToParent("echo:" + message.(string))
	})
	defer worker.Close()

	received := make(chan any, 1)
	worker.OnMessage(func(message any) {
		received <- message
	})
	if err := worker.PostMessage("ready"); err != nil {
		t.Fatalf("post worker message: %v", err)
	}
	select {
	case message := <-received:
		if message != "echo:ready" {
			t.Fatalf("unexpected worker message %v", message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for worker message")
	}

	worker.Close()
	if err := worker.PostMessage("late"); err == nil {
		t.Fatal("expected closed worker post to fail")
	}
}

func TestRuntimeWorkerSharedMemoryAtomics(t *testing.T) {
	memory := jayessruntime.NewSharedMemory(2)
	if memory.Length() != 2 {
		t.Fatalf("expected shared memory length 2, got %d", memory.Length())
	}
	if err := memory.AtomicStore(1, 42); err != nil {
		t.Fatalf("atomic store: %v", err)
	}
	value, err := memory.AtomicLoad(1)
	if err != nil {
		t.Fatalf("atomic load: %v", err)
	}
	if value != 42 {
		t.Fatalf("unexpected atomic value %d", value)
	}
	if err := memory.AtomicStore(3, 1); err == nil {
		t.Fatal("expected out of range atomic store error")
	}
}

func TestRuntimeWorkerCapturesHandlerError(t *testing.T) {
	worker := jayessruntime.NewWorker(func(worker *jayessruntime.Worker) {
		panic("boom")
	})
	defer worker.Close()
	deadline := time.After(time.Second)
	for worker.LastError() == nil {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for worker error")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestSemanticRejectsTopLevelWorkerRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var worker = {};`)
	requireSemanticError(t, err, "duplicate declaration worker")
}
