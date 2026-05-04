package test

import (
	"testing"

	"jayess-go/openssl"
)

func TestOpenSSLCryptoFeatures(t *testing.T) {
	features := openssl.CryptoFeatures()
	for _, want := range []openssl.CryptoFeature{
		openssl.Hashing,
		openssl.HMAC,
		openssl.SymmetricEncryption,
		openssl.AsymmetricEncryption,
		openssl.DigitalSignatures,
		openssl.KeyGeneration,
		openssl.RandomBytes,
	} {
		if !hasOpenSSLCryptoFeature(features, want) {
			t.Fatalf("expected OpenSSL crypto feature %s in %#v", want, features)
		}
	}
}

func TestOpenSSLTLSFeatures(t *testing.T) {
	features := openssl.TLSFeatures()
	for _, want := range []openssl.TLSFeature{
		openssl.TLSClient,
		openssl.TLSServer,
		openssl.CertificateLoading,
		openssl.TrustStoreConfig,
		openssl.HostnameVerification,
		openssl.ALPN,
	} {
		if !hasOpenSSLTLSFeature(features, want) {
			t.Fatalf("expected OpenSSL TLS feature %s in %#v", want, features)
		}
	}
}

func hasOpenSSLCryptoFeature(features []openssl.CryptoFeature, want openssl.CryptoFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasOpenSSLTLSFeature(features []openssl.TLSFeature, want openssl.TLSFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
