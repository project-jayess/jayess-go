package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeMVPGlobalsHaveImplementations(t *testing.T) {
	expected := []string{"console", "print", "sleep", "readLine", "readKey"}
	for _, name := range expected {
		if !jayessruntime.HasMVPGlobalImplementation(name) {
			t.Fatalf("expected runtime implementation for MVP global %s", name)
		}
	}
}

func TestRuntimeMVPGlobalsMatchSemanticRecognition(t *testing.T) {
	err := analyzeSource(t, `
		function main(delay) {
			console.log("ready");
			print("ready");
			sleep(delay);
			const line = readLine("name?");
			const key = readKey("continue?");
			return line || key;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeMVPGlobalImplementationsDeclareEntrypoints(t *testing.T) {
	for _, implementation := range jayessruntime.MVPGlobalImplementations() {
		if implementation.Name == "" {
			t.Fatalf("runtime MVP global has empty name: %#v", implementation)
		}
		if implementation.RuntimeSymbol == "" {
			t.Fatalf("runtime MVP global %s has empty runtime symbol", implementation.Name)
		}
		if implementation.Name == "console" && !hasRuntimeMethod(implementation.Methods, "log") {
			t.Fatalf("console runtime implementation must expose log method")
		}
	}
}

func hasRuntimeMethod(methods []string, expected string) bool {
	for _, method := range methods {
		if method == expected {
			return true
		}
	}
	return false
}
