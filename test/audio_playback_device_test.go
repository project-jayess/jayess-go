package test

import (
	"testing"

	"jayess-go/audio"
)

func TestAudioDeviceCapabilities(t *testing.T) {
	if !audio.SupportsDeviceKind(audio.OutputDevice) {
		t.Fatal("expected enumerable/openable output device support")
	}
	if !audio.SupportsDeviceKind(audio.InputDevice) {
		t.Fatal("expected enumerable/openable input device support")
	}
	for _, capability := range audio.DefaultDeviceCapabilities() {
		if len(capability.SampleRates) == 0 || len(capability.Channels) == 0 || len(capability.Formats) == 0 {
			t.Fatalf("expected complete device capability: %#v", capability)
		}
	}
}

func TestAudioPlaybackSurface(t *testing.T) {
	features := audio.PlaybackFeatures()
	for _, want := range []audio.PlaybackFeature{
		audio.OpenPlaybackDevice,
		audio.OpenCaptureDevice,
		audio.ConfigureFormat,
		audio.StartPlayback,
		audio.StopPlayback,
		audio.PausePlayback,
		audio.SubmitAudioBuffer,
		audio.StreamingPlayback,
	} {
		if !hasPlaybackFeature(features, want) {
			t.Fatalf("expected playback feature %s in %#v", want, features)
		}
	}
}

func hasPlaybackFeature(features []audio.PlaybackFeature, want audio.PlaybackFeature) bool {
	for _, feature := range features {
		if feature == want {
			return true
		}
	}
	return false
}
