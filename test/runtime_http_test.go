package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeHTTPCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"createServer",
		"request",
		"requestObject",
		"responseObject",
		"headers",
		"status",
		"readBody",
		"writeBody",
		"streamBody",
		"keepAlive",
		"withTimeout",
	}
	for _, name := range expected {
		if !jayessruntime.HasHTTPCapability(name) {
			t.Fatalf("expected HTTP runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsHTTPSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(url) {
			const server = http.createServer((req, res) => {
				const request = http.requestObject(req);
				const response = http.responseObject(res);
				const headers = http.headers(request);
				const body = http.readBody(request);
				http.status(response, 200);
				http.writeBody(response, body);
				return headers;
			});
			const client = http.request(url);
			const timed = http.withTimeout(client, 1000);
			const kept = http.keepAlive(timed);
			const stream = http.streamBody(kept);
			return server || stream;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeHTTPCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.HTTPCapabilities() {
		if capability.Name == "" {
			t.Fatalf("HTTP capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("HTTP capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("HTTP capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelHTTPRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var http = {};`)
	requireSemanticError(t, err, "duplicate declaration http")
}
