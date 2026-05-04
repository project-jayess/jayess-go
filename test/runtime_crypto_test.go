package test

import (
	"testing"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeCryptoCapabilitiesAreDeclared(t *testing.T) {
	expected := []string{
		"randomBytes",
		"hash",
		"hmac",
		"encrypt",
		"decrypt",
		"publicEncrypt",
		"privateDecrypt",
		"sign",
		"verify",
		"generateKey",
		"secureCompare",
	}
	for _, name := range expected {
		if !jayessruntime.HasCryptoCapability(name) {
			t.Fatalf("expected crypto runtime capability %s", name)
		}
	}
}

func TestSemanticAllowsCryptoSurface(t *testing.T) {
	err := analyzeSource(t, `
		function main(message, key, publicKey, privateKey) {
			const nonce = crypto.randomBytes(16);
			const digest = crypto.hash("sha256", message);
			const mac = crypto.hmac("sha256", key, message);
			const encrypted = crypto.encrypt("aes-256-gcm", key, nonce, message);
			const decrypted = crypto.decrypt("aes-256-gcm", key, nonce, encrypted);
			const sealed = crypto.publicEncrypt(publicKey, message);
			const opened = crypto.privateDecrypt(privateKey, sealed);
			const signingKey = crypto.generateKey("ed25519");
			const sig = crypto.sign(signingKey, message);
			const ok = crypto.verify(signingKey, message, sig);
			const same = crypto.secureCompare(digest, mac);
			return decrypted || opened || ok || same;
		}
	`)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
}

func TestRuntimeCryptoCapabilitiesDeclareEntrypoints(t *testing.T) {
	for _, capability := range jayessruntime.CryptoCapabilities() {
		if capability.Name == "" {
			t.Fatalf("crypto capability has empty name: %#v", capability)
		}
		if capability.RuntimeSymbol == "" {
			t.Fatalf("crypto capability %s has empty runtime symbol", capability.Name)
		}
		if capability.Kind != "function" {
			t.Fatalf("crypto capability %s has unsupported kind %q", capability.Name, capability.Kind)
		}
	}
}

func TestSemanticRejectsTopLevelCryptoRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var crypto = {};`)
	requireSemanticError(t, err, "duplicate declaration crypto")
}
