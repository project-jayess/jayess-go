package audio

import "jayess-go/binding"

type APIKind string

const (
	PlaybackAPI APIKind = "playback"
	CaptureAPI  APIKind = "capture"
	DeviceAPI   APIKind = "device"
	BufferAPI   APIKind = "buffer"
	StreamAPI   APIKind = "stream"
	CallbackAPI APIKind = "callback"
)

type BindingModule struct {
	Path     string
	Library  LibraryTarget
	Manifest binding.Manifest
	APIs     []APIKind
}

func ValidateBindingModule(module BindingModule) []binding.Diagnostic {
	if err := binding.ValidateBindingTarget(module.Path); err != nil {
		return []binding.Diagnostic{{Field: "audio.binding", Message: err.Error()}}
	}
	diagnostics := binding.ValidateManifest(module.Manifest)
	if module.Library == "" {
		diagnostics = append(diagnostics, binding.Diagnostic{
			Field:   "audio.library",
			Message: "audio binding must declare a native library target",
		})
	}
	return diagnostics
}

func SupportsAPI(module BindingModule, api APIKind) bool {
	for _, available := range module.APIs {
		if available == api {
			return true
		}
	}
	return false
}
