package test

import (
	nethttp "net/http"
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

func TestRuntimeHTTPSUsesInternalHTTPModelAndTLSConfig(t *testing.T) {
	certPEM, keyPEM := selfSignedTLSPEM(t, "localhost")
	certificate, err := jayessruntime.NewTLSCertificate(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("load certificate: %v", err)
	}
	config := jayessruntime.TLSRuntimeConfig{
		ServerName:   "localhost",
		Certificates: []jayessruntime.TLSCertificate{certificate},
		ALPN:         []string{"http/1.1"},
	}
	server := jayessruntime.CreateHTTPSServer(config, func(req *jayessruntime.HTTPRequest, res *jayessruntime.HTTPResponse) {
		jayessruntime.HTTPWriteBody(res, "ok")
	})
	if server == nil || server.HTTP == nil {
		t.Fatal("expected HTTPS server to wrap internal HTTP server")
	}
	if server.TLS.ServerName != "localhost" {
		t.Fatalf("unexpected HTTPS TLS server name %q", server.TLS.ServerName)
	}

	client := jayessruntime.HTTPSClientConfig(jayessruntime.TLSRuntimeConfig{ServerName: "localhost", InsecureSkipVerify: true})
	transport, ok := client.Transport.(*nethttp.Transport)
	if !ok {
		t.Fatalf("expected HTTPS client transport, got %T", client.Transport)
	}
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.ServerName != "localhost" {
		t.Fatalf("expected configured TLS client transport")
	}
}

func TestRuntimeHTTPSListenerCreatesTLSListener(t *testing.T) {
	certPEM, keyPEM := selfSignedTLSPEM(t, "localhost")
	certificate, err := jayessruntime.NewTLSCertificate(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("load certificate: %v", err)
	}
	listener, err := jayessruntime.HTTPSListener("127.0.0.1:0", jayessruntime.TLSRuntimeConfig{Certificates: []jayessruntime.TLSCertificate{certificate}})
	if err != nil {
		t.Fatalf("create HTTPS listener: %v", err)
	}
	defer listener.Close()
	if listener.Addr().String() == "" {
		t.Fatal("expected HTTPS listener address")
	}
}

func TestSemanticRejectsTopLevelHTTPSRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var https = {};`)
	requireSemanticError(t, err, "duplicate declaration https")
}
