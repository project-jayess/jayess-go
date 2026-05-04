package test

import (
	"testing"

	"jayess-go/binding"
)

func TestBindingRuntimeHeaderExposesLowLevelRuntimeControl(t *testing.T) {
	for _, fn := range []string{
		"jayess_value_from_number",
		"jayess_value_to_string_copy",
		"jayess_bytes_free",
		"jayess_expect_object",
		"jayess_value_from_managed_native_handle",
		"jayess_throw_type_error",
	} {
		if !binding.RuntimeHeaderHasFunction(fn) {
			t.Fatalf("expected runtime header function %s", fn)
		}
	}
}

func TestBindingNativeLibraryUseCases(t *testing.T) {
	useCases := binding.SupportedUseCases()
	for _, want := range []binding.UseCase{
		binding.EngineAPIUseCase,
		binding.PlatformAPIUseCase,
		binding.RenderingAudioInputUseCase,
		binding.ThirdPartyCUseCase,
		binding.PerformanceCriticalUseCase,
	} {
		if !hasUseCase(useCases, want) {
			t.Fatalf("expected native binding use case %s in %#v", want, useCases)
		}
	}
}

func hasUseCase(useCases []binding.UseCase, want binding.UseCase) bool {
	for _, useCase := range useCases {
		if useCase == want {
			return true
		}
	}
	return false
}
