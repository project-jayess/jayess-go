package test

import (
	"testing"

	"jayess-go/audio"
	"jayess-go/binding"
)

func TestAudioLibraryTargets(t *testing.T) {
	for _, target := range []audio.LibraryTarget{
		audio.SDLAudioTarget,
		audio.OpenALTarget,
		audio.MiniaudioTarget,
		audio.PortAudioTarget,
		audio.PlatformNativeTarget,
	} {
		if !audio.SupportsLibraryTarget(target) {
			t.Fatalf("expected audio library target %s", target)
		}
	}
}

func TestAudioCrossPlatformBuildFlags(t *testing.T) {
	module := audio.BindingModule{
		Path:     "./native/audio.bind.js",
		Library:  audio.PlatformNativeTarget,
		Manifest: audioPlatformManifest(),
	}
	cases := map[string][]string{
		"linux":   {"-lasound"},
		"darwin":  {"-framework", "CoreAudio"},
		"windows": {"-lwinmm"},
	}
	for platform, flags := range cases {
		plan := audio.PlanBuild([]audio.BindingModule{module}, platform, "./runtime")
		for _, flag := range flags {
			if !hasString(plan.LDFlags, flag) {
				t.Fatalf("expected %s flag %s in %#v", platform, flag, plan.LDFlags)
			}
		}
	}
}

func audioPlatformManifest() binding.Manifest {
	return binding.Manifest{
		Sources: []string{"./audio.c"},
		Platforms: map[string]binding.PlatformOptions{
			"linux":   {LDFlags: []string{"-lasound"}},
			"darwin":  {LDFlags: []string{"-framework", "CoreAudio"}},
			"windows": {LDFlags: []string{"-lwinmm"}},
		},
		Exports: []binding.Export{{Name: "open", Symbol: "audio_open", Kind: binding.FunctionExport}},
	}
}
