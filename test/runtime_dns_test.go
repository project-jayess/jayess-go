package test

import (
	"strings"
	"testing"
	"time"

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

func TestRuntimeDNSParseAndClassifyIP(t *testing.T) {
	parsed, ok := jayessruntime.DNSParseIP("127.0.0.1")
	if !ok {
		t.Fatal("expected IPv4 address to parse")
	}
	if parsed.Family != 4 || !parsed.Loopback {
		t.Fatalf("unexpected parsed IPv4 metadata: %#v", parsed)
	}
	if family := jayessruntime.DNSIsIP("2001:db8::1"); family != 6 {
		t.Fatalf("expected IPv6 family, got %d", family)
	}
	if family := jayessruntime.DNSIsIP("not an ip"); family != 0 {
		t.Fatalf("expected invalid IP family 0, got %d", family)
	}
}

func TestRuntimeDNSResolverConfigNormalizesServers(t *testing.T) {
	config := jayessruntime.NewDNSResolverConfig([]string{"1.1.1.1", "8.8.8.8:53", ""})
	if len(config.Servers) != 2 {
		t.Fatalf("expected two DNS servers, got %#v", config.Servers)
	}
	if config.Servers[0] != "1.1.1.1:53" {
		t.Fatalf("expected default DNS port, got %#v", config.Servers)
	}
	if !config.PreferGo || config.Timeout <= 0 {
		t.Fatalf("expected internal Go resolver defaults, got %#v", config)
	}
}

func TestRuntimeDNSLookupLocalhost(t *testing.T) {
	records, err := jayessruntime.DNSLookup("localhost", jayessruntime.DNSResolverConfig{Timeout: time.Second})
	if err != nil {
		t.Fatalf("lookup localhost: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("expected localhost records")
	}
	for _, record := range records {
		if record.Address == "" || !strings.HasPrefix(record.Network, "ipv") {
			t.Fatalf("unexpected DNS record: %#v", record)
		}
	}
}

func TestRuntimeDNSRejectsInvalidReverseIP(t *testing.T) {
	if _, err := jayessruntime.DNSReverse("not an ip", jayessruntime.DNSResolverConfig{}); err == nil {
		t.Fatal("expected invalid reverse address error")
	}
}

func TestSemanticRejectsTopLevelDNSRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var dns = {};`)
	requireSemanticError(t, err, "duplicate declaration dns")
}
