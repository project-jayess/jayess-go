package test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

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

func TestRuntimeCryptoRandomBytes(t *testing.T) {
	bytes, err := jayessruntime.CryptoRandomBytes(32)
	if err != nil {
		t.Fatalf("random bytes failed: %v", err)
	}
	if len(bytes) != 32 {
		t.Fatalf("expected 32 random bytes, got %d", len(bytes))
	}
}

func TestRuntimeCryptoHashAndHMAC(t *testing.T) {
	digest, err := jayessruntime.CryptoHashHex("sha-256", []byte("hello"))
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	expected := sha256.Sum256([]byte("hello"))
	if digest != hex.EncodeToString(expected[:]) {
		t.Fatalf("unexpected sha256 digest %q", digest)
	}

	mac, err := jayessruntime.CryptoHMACHex("sha256", []byte("key"), []byte("hello"))
	if err != nil {
		t.Fatalf("hmac failed: %v", err)
	}
	if mac == "" || mac == digest {
		t.Fatalf("expected distinct non-empty HMAC, got %q", mac)
	}
}

func TestRuntimeCryptoSecureCompare(t *testing.T) {
	left := []byte("same")
	if !jayessruntime.CryptoSecureCompare(left, bytes.Clone(left)) {
		t.Fatal("expected equal byte slices to compare securely")
	}
	if jayessruntime.CryptoSecureCompare(left, []byte("diff")) {
		t.Fatal("expected different byte slices to fail secure compare")
	}
}

func TestRuntimeCryptoRejectsUnsupportedAlgorithms(t *testing.T) {
	if _, err := jayessruntime.CryptoHash("external-openssl-only", []byte("data")); err == nil {
		t.Fatal("expected unsupported internal algorithm error")
	}
}

func TestRuntimeCryptoSymmetricEncryptDecrypt(t *testing.T) {
	key, err := jayessruntime.CryptoGenerateKey("aes-256-gcm")
	if err != nil {
		t.Fatalf("generate AES key: %v", err)
	}
	nonce := bytes.Repeat([]byte{1}, 12)
	ciphertext, err := jayessruntime.CryptoEncrypt("aes-256-gcm", key.Secret, nonce, []byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plain, err := jayessruntime.CryptoDecrypt("aes-256-gcm", key.Secret, nonce, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plain) != "secret" {
		t.Fatalf("unexpected plaintext %q", plain)
	}
}

func TestRuntimeCryptoSignVerifyAndKeyImportExport(t *testing.T) {
	key, err := jayessruntime.CryptoGenerateKey("ed25519")
	if err != nil {
		t.Fatalf("generate signing key: %v", err)
	}
	signature, err := jayessruntime.CryptoSign(key, []byte("message"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	ok, err := jayessruntime.CryptoVerify(key, []byte("message"), signature)
	if err != nil || !ok {
		t.Fatalf("verify got ok=%v err=%v", ok, err)
	}
	exported, err := jayessruntime.CryptoExportKeyPEM(key, true)
	if err != nil {
		t.Fatalf("export key: %v", err)
	}
	imported, err := jayessruntime.CryptoImportKeyPEM(exported)
	if err != nil {
		t.Fatalf("import key: %v", err)
	}
	ok, err = jayessruntime.CryptoVerify(imported, []byte("message"), signature)
	if err != nil || !ok {
		t.Fatalf("verify imported key got ok=%v err=%v", ok, err)
	}
}

func TestRuntimeCryptoPublicEncryptPrivateDecrypt(t *testing.T) {
	key, err := jayessruntime.CryptoGenerateKey("rsa-oaep")
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	sealed, err := jayessruntime.CryptoPublicEncrypt(key, []byte("secret"))
	if err != nil {
		t.Fatalf("public encrypt: %v", err)
	}
	opened, err := jayessruntime.CryptoPrivateDecrypt(key, sealed)
	if err != nil {
		t.Fatalf("private decrypt: %v", err)
	}
	if string(opened) != "secret" {
		t.Fatalf("unexpected opened message %q", opened)
	}
}

func TestRuntimeCryptoParseCertificatePEM(t *testing.T) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate cert key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject:      pkix.Name{CommonName: "jayess.local"},
		DNSNames:     []string{"jayess.local"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &private.PublicKey, private)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	info, err := jayessruntime.CryptoParseCertificatePEM(certPEM)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	if info.SerialText != "42" || len(info.DNSNames) != 1 || info.DNSNames[0] != "jayess.local" {
		t.Fatalf("unexpected certificate info: %#v", info)
	}
}

func TestSemanticRejectsTopLevelCryptoRedeclaration(t *testing.T) {
	err := analyzeSource(t, `var crypto = {};`)
	requireSemanticError(t, err, "duplicate declaration crypto")
}
