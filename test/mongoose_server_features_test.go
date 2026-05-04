package test

import (
	"testing"

	"jayess-go/mongoose"
)

func TestMongooseHTTPServerCoreFeatures(t *testing.T) {
	features := mongoose.ServerFeatures()
	for _, want := range []mongoose.ServerFeature{
		mongoose.CreateManager,
		mongoose.BindListen,
		mongoose.AcceptRequest,
		mongoose.ReadRequest,
		mongoose.SendResponse,
		mongoose.CleanShutdown,
	} {
		if !hasMongooseServerFeature(features, want) {
			t.Fatalf("expected Mongoose server feature %s in %#v", want, features)
		}
	}
}

func TestMongooseExtendedServerFeatures(t *testing.T) {
	features := mongoose.ExtendedFeatures()
	for _, want := range []mongoose.ExtendedFeature{
		mongoose.StaticFileServing,
		mongoose.RouteDispatch,
		mongoose.ChunkedStreaming,
		mongoose.HTTPSServing,
		mongoose.WebSocketUpgrade,
		mongoose.WebviewAppContent,
	} {
		if !hasMongooseExtendedFeature(features, want) {
			t.Fatalf("expected Mongoose extended feature %s in %#v", want, features)
		}
	}
}

func hasMongooseServerFeature(features []mongoose.ServerFeature, want mongoose.ServerFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasMongooseExtendedFeature(features []mongoose.ExtendedFeature, want mongoose.ExtendedFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
