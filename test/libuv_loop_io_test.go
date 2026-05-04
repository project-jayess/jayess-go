package test

import (
	"testing"

	"jayess-go/libuv"
)

func TestLibUVEventLoopFeatures(t *testing.T) {
	features := libuv.LoopFeatures()
	for _, want := range []libuv.LoopFeature{
		libuv.CreateLoop,
		libuv.RunLoop,
		libuv.StopLoop,
		libuv.CloseLoop,
		libuv.TimerCoexistence,
		libuv.MicrotaskPolicy,
	} {
		if !hasLibUVLoopFeature(features, want) {
			t.Fatalf("expected libuv loop feature %s in %#v", want, features)
		}
	}
}

func TestLibUVAsyncIOFeatures(t *testing.T) {
	features := libuv.IOFeatures()
	for _, want := range []libuv.IOFeature{
		libuv.TCPIntegration,
		libuv.UDPIntegration,
		libuv.FilesystemAsyncOps,
		libuv.PollingWatcher,
		libuv.ProcessSpawn,
		libuv.SignalWatcher,
	} {
		if !hasLibUVIOFeature(features, want) {
			t.Fatalf("expected libuv I/O feature %s in %#v", want, features)
		}
	}
}

func hasLibUVLoopFeature(features []libuv.LoopFeature, want libuv.LoopFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}

func hasLibUVIOFeature(features []libuv.IOFeature, want libuv.IOFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
