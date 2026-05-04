package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeDNSCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"lookup",
		"reverse",
		"resolver",
		"isIP",
		"parseIP",
	}
	for _, name := range expected {
		if !jayessruntime.HasDNSCapability(name) {
			t.Fatalf("expected DNS runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsDNSSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(host, address) {
			const records = dns.lookup(host);
			const names = dns.reverse(address);
			const resolver = dns.resolver(["1.1.1.1", "8.8.8.8"]);
			const parsed = dns.parseIP(address);
			const ok = dns.isIP(address);
			return records || names || resolver || parsed || ok;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeDNSCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.DNSCapabilities() {
		if capability.Name == "" {
			t.Fatalf("DNS capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("DNS capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("DNS capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelDNSRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var dns = {};`)
	requireSemanticError(t, err, "duplicate declaration dns")
}
