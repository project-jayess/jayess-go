package test

import (
	"testing"

	"jayess-go/libcurl"
)

func TestLibcurlAsyncAndStreamingFeatures(t *testing.T) {
	features := libcurl.AsyncFeatures()
	for _, want := range []libcurl.AsyncFeature{
		libcurl.MultiHandleSupport,
		libcurl.JayessAsyncModel,
		libcurl.StreamingBody,
	} {
		if !hasLibcurlAsyncFeature(features, want) {
			t.Fatalf("expected libcurl async feature %s in %#v", want, features)
		}
	}
}

func TestLibcurlDiagnosticKinds(t *testing.T) {
	kinds := libcurl.DiagnosticKinds()
	for _, want := range []libcurl.DiagnosticKind{
		libcurl.MissingHeaders,
		libcurl.MissingLibrary,
		libcurl.TransferError,
	} {
		if !hasLibcurlDiagnosticKind(kinds, want) {
			t.Fatalf("expected libcurl diagnostic kind %s in %#v", want, kinds)
		}
	}
}

func hasLibcurlAsyncFeature(features []libcurl.AsyncFeature, want libcurl.AsyncFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasLibcurlDiagnosticKind(kinds []libcurl.DiagnosticKind, want libcurl.DiagnosticKind) bool {
	for _, kind := range kinds {
		if kind == want {
			return true
		}
	}
	return false
}
