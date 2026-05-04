package test

import (
	"testing"

	"jayess-go/openssl"
)

func TestOpenSSLSafetyRules(t *testing.T) {
	rules := openssl.SafetyRules()
	for _, want := range []openssl.SafetyRule{
		openssl.KeyCertificateLifetimeSafe,
		openssl.OpenSSLErrorDiagnostics,
		openssl.VersionFeatureSafe,
	} {
		if !hasOpenSSLSafetyRule(rules, want) {
			t.Fatalf("expected OpenSSL safety rule %s in %#v", want, rules)
		}
	}
}

func TestOpenSSLVersionPolicy(t *testing.T) {
	policy := openssl.DefaultVersionPolicy()
	if policy.MinimumVersion == "" {
		t.Fatal("expected minimum OpenSSL version")
	}
	for _, feature := range []string{"tls1.3", "alpn", "evp"} {
		if !hasString(policy.FeatureGates, feature) {
			t.Fatalf("expected OpenSSL feature gate %s in %#v", feature, policy.FeatureGates)
		}
	}
}

func hasOpenSSLSafetyRule(rules []openssl.SafetyRule, want openssl.SafetyRule) bool {
	for _, rule := range rules {
		if rule == want {
			return true
		}
	}
	return false
}
