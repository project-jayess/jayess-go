package test

import (
	"testing"

	"jayess-go/audio"
)

func TestAudioAssetFormatsExposeDecodedBuffers(t *testing.T) {
	for _, codec := range []audio.Codec{
		audio.WAVCodec,
		audio.OGGCodec,
		audio.MP3Codec,
		audio.FLACCodec,
		audio.PCMCodec,
	} {
		if !audio.SupportsCodec(codec) {
			t.Fatalf("expected codec support for %s", codec)
		}
	}
	for _, format := range audio.AssetFormats() {
		if !format.DecodedAsJayessBuffer {
			t.Fatalf("expected decoded audio as Jayess buffer for %#v", format)
		}
	}
}

func TestAudioRealtimeFeatures(t *testing.T) {
	features := audio.RealtimeFeatures()
	for _, want := range []audio.RealtimeFeature{
		audio.LowLatencyPlayback,
		audio.UnderrunHandling,
		audio.DeviceLossHandling,
		audio.ThreadSafeCallback,
		audio.WorkerSync,
	} {
		if !hasRealtimeFeature(features, want) {
			t.Fatalf("expected realtime feature %s in %#v", want, features)
		}
	}
}

func hasRealtimeFeature(features []audio.RealtimeFeature, want audio.RealtimeFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
