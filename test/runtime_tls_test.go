package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeTLSCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"client",
		"server",
		"certificate",
		"withALPN",
		"verifyHostname",
	}
	for _, name := range expected {
		if !jayessruntime.HasTLSCapability(name) {
			t.Fatalf("expected TLS runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsTLSSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(socket, host, certPath, keyPath) {
			const cert = tls.certificate(certPath, keyPath);
			const client = tls.client(socket, { host: host });
			const server = tls.server(socket, cert);
			const negotiated = tls.withALPN(client, ["h2", "http/1.1"]);
			const verified = tls.verifyHostname(negotiated, host);
			return server || verified;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeTLSCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.TLSCapabilities() {
		if capability.Name == "" {
			t.Fatalf("TLS capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("TLS capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("TLS capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelTLSRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var tls = {};`)
	requireSemanticError(t, err, "duplicate declaration tls")
}
