package test

import (
	"testing"

	"jayess-go/audio"
	"jayess-go/binding"
)

func TestAudioBindingModuleCanImportBindJS(t *testing.T) {
	module := audio.BindingModule{
		Path:    "./native/audio.bind.js",
		Library: audio.MiniaudioTarget,
		Manifest: binding.Manifest{
			Sources: []string{"./audio.c"},
			Exports: []binding.Export{
				{Name: "openPlayback", Symbol: "jayess_audio_open_playback", Kind: binding.FunctionExport},
			},
		},
		APIs: []audio.APIKind{audio.PlaybackAPI, audio.DeviceAPI},
	}

	if diagnostics := audio.ValidateBindingModule(module); len(diagnostics) != 0 {
		t.Fatalf("expected valid audio binding module, got %#v", diagnostics)
	}
	if !audio.SupportsAPI(module, audio.PlaybackAPI) {
		t.Fatal("expected audio playback API support")
	}
}

func TestAudioBindingModuleRejectsMalformedTarget(t *testing.T) {
	module := audio.BindingModule{
		Path:    "./native/audio.c",
		Library: audio.MiniaudioTarget,
		Manifest: binding.Manifest{
			Exports: []binding.Export{{Name: "open", Symbol: "audio_open", Kind: binding.FunctionExport}},
		},
	}

	diagnostics := audio.ValidateBindingModule(module)
	requireDiagnostic(t, diagnostics, ".js")
}

func TestAudioBindingBuildPlanLinksNativeSources(t *testing.T) {
	module := audio.BindingModule{
		Path:    "./native/audio.bind.js",
		Library: audio.MiniaudioTarget,
		Manifest: binding.Manifest{
			Sources:     []string{"./audio.c"},
			IncludeDirs: []string{"./include"},
			CFlags:      []string{"-DJAYESS_AUDIO=1"},
			LDFlags:     []string{"-lm"},
			Exports: []binding.Export{
				{Name: "openPlayback", Symbol: "jayess_audio_open_playback", Kind: binding.FunctionExport},
			},
		},
	}

	plan := audio.PlanBuild([]audio.BindingModule{module}, "linux", "./runtime")
	if len(plan.Diagnostics) != 0 {
		t.Fatalf("expected clean audio build plan, got %#v", plan.Diagnostics)
	}
	if len(plan.CompileUnits) != 1 {
		t.Fatalf("expected one audio compile unit, got %#v", plan.CompileUnits)
	}
	requireStringSlice(t, plan.CompileUnits[0].IncludeDirs, []string{"native/include", "./runtime"})
	requireStringSlice(t, plan.CompileUnits[0].CFlags, []string{"-DJAYESS_AUDIO=1"})
	requireStringSlice(t, plan.LDFlags, []string{"-lm"})
}
