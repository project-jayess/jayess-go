package test

import (
	"testing"

	"jayess-go/libcurl"
)

func TestLibcurlCoreTransferFeatures(t *testing.T) {
	features := libcurl.CoreFeatures()
	for _, want := range []libcurl.CoreFeature{
		libcurl.CreateEasyHandle,
		libcurl.ConfigureRequest,
		libcurl.PerformTransfer,
		libcurl.ReadResponse,
		libcurl.CleanupEasyHandle,
	} {
		if !hasLibcurlCoreFeature(features, want) {
			t.Fatalf("expected libcurl core feature %s in %#v", want, features)
		}
	}
}

func TestLibcurlAdvancedTransferFeatures(t *testing.T) {
	features := libcurl.AdvancedFeatures()
	for _, want := range []libcurl.AdvancedFeature{
		libcurl.HTTPS,
		libcurl.Redirects,
		libcurl.Timeouts,
		libcurl.Upload,
		libcurl.DownloadToFile,
		libcurl.Cookies,
		libcurl.Proxy,
	} {
		if !hasLibcurlAdvancedFeature(features, want) {
			t.Fatalf("expected libcurl advanced feature %s in %#v", want, features)
		}
	}
}

func hasLibcurlCoreFeature(features []libcurl.CoreFeature, want libcurl.CoreFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasLibcurlAdvancedFeature(features []libcurl.AdvancedFeature, want libcurl.AdvancedFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
