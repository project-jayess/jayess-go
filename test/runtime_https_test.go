package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeHTTPSCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"createServer",
		"request",
		"loadCertificate",
		"loadPrivateKey",
		"trustStore",
		"verifyCertificate",
		"secureDefaults",
	}
	for _, name := range expected {
		if !jayessruntime.HasHTTPSCapability(name) {
			t.Fatalf("expected HTTPS runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsHTTPSSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(url, certPath, keyPath, caPath) {
			const cert = https.loadCertificate(certPath);
			const key = https.loadPrivateKey(keyPath);
			const trust = https.trustStore(caPath);
			const defaults = https.secureDefaults();
			const server = https.createServer({ cert: cert, key: key }, (req, res) => res);
			const client = https.request(url, defaults);
			const verified = https.verifyCertificate(client, trust);
			return server || verified;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeHTTPSCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.HTTPSCapabilities() {
		if capability.Name == "" {
			t.Fatalf("HTTPS capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("HTTPS capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("HTTPS capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelHTTPSRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var https = {};`)
	requireSemanticError(t, err, "duplicate declaration https")
}
