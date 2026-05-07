package test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
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

func TestRuntimeTLSCertificateAndConfig(t *testing.T) {
	certPEM, keyPEM := selfSignedTLSPEM(t, "localhost")
	certificate, err := jayessruntime.NewTLSCertificate(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("load certificate: %v", err)
	}
	config := jayessruntime.TLSWithALPN(jayessruntime.TLSRuntimeConfig{
		ServerName:   "localhost",
		Certificates: []jayessruntime.TLSCertificate{certificate},
	}, []string{"h2", "http/1.1"})

	serverConfig := jayessruntime.TLSServerConfig(config)
	if len(serverConfig.Certificates) != 1 {
		t.Fatalf("expected server certificate")
	}
	if serverConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected secure TLS minimum version, got %#x", serverConfig.MinVersion)
	}
	if got := serverConfig.NextProtos; len(got) != 2 || got[0] != "h2" {
		t.Fatalf("unexpected ALPN protocols %#v", got)
	}

	clientConfig := jayessruntime.TLSClientConfig(config)
	if clientConfig.ServerName != "localhost" {
		t.Fatalf("unexpected client server name %q", clientConfig.ServerName)
	}
}

func TestRuntimeTLSTrustStoreAndHostnameVerification(t *testing.T) {
	certPEM, _ := selfSignedTLSPEM(t, "localhost")
	store, err := jayessruntime.NewTLSTrustStore(certPEM)
	if err != nil {
		t.Fatalf("load trust store: %v", err)
	}
	if store.Subjects != 1 {
		t.Fatalf("expected one trust store subject, got %d", store.Subjects)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("expected PEM certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	if err := jayessruntime.TLSVerifyHostname(certificate, "localhost"); err != nil {
		t.Fatalf("verify hostname: %v", err)
	}
	if err := jayessruntime.TLSVerifyHostname(certificate, "example.com"); err == nil {
		t.Fatal("expected hostname mismatch")
	}
}

func TestSemanticRejectsTopLevelTLSRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var tls = {};`)
	requireSemanticError(t, err, "duplicate declaration tls")
}
