package test

import (
	"testing"

	"jayess-go/binding"
	"jayess-go/openssl"
)

func TestOpenSSLBindingModuleCanImportBindJS(t *testing.T) {
	module := openssl.BindingModule{
		Path: "./native/openssl.bind.js",
		Manifest: binding.Manifest{
			Sources: []string{"./openssl.c"},
			LDFlags: []string{"-lssl", "-lcrypto"},
			Exports: []binding.Export{
				{Name: "hash", Symbol: "jayess_openssl_hash", Kind: binding.FunctionExport},
			},
		},
		APIs:    []openssl.APIKind{openssl.CryptoAPI, openssl.TLSAPI},
		Handles: []openssl.HandleKind{openssl.SSLContextHandle, openssl.KeyHandle},
	}

	if diagnostics := openssl.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid OpenSSL binding module, got %#v", diagnostics)
	}
	if !openssl.SupportsAPI(module, openssl.TLSAPI) {
		t.Fatal("expected OpenSSL TLS API support")
	}
}

func TestOpenSSLBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := openssl.BindingModule{
		Path: "./native/openssl.c",
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "hash", Symbol: "openssl_hash", Kind: binding.FunctionExport}},
		},
		Handles: []openssl.HandleKind{openssl.DigestHandle},
	}

	diagnostics := openssl.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestOpenSSLBuildPlanUsesVendoredSourceWhenRequested(t *testing.T) {
	module := openssl.BindingModule{
		Path: "./native/openssl.bind.js",
		Manifest: binding.Manifest{
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DOPENSSL_API_COMPAT=0x10101000L"},
			Exports:     []binding.Export{{Name: "hash", Symbol: "openssl_hash", Kind: binding.FunctionExport}},
		},
		Handles:        []openssl.HandleKind{openssl.DigestHandle},
		VendoredSource: true,
	}

	plan := openssl.PlanBuild([]openssl.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean OpenSSL build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 2 {
		t.Fatalf("expected vendored OpenSSL compile units, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DOPENSSL_API_COMPAT=0x10101000L"})
}

func TestOpenSSLHandlesRepresentNativeTypesSafely(t *testing.T) {
	for _, kind := range []openssl.HandleKind{
		openssl.SSLContextHandle,
		openssl.SSLHandle,
		openssl.KeyHandle,
		openssl.CertHandle,
		openssl.CipherHandle,
		openssl.DigestHandle,
	} {
		if !openssl.SupportsHandle(kind) {
			t.Fatalf("expected OpenSSL handle support for %s", kind)
		}
	}
}
