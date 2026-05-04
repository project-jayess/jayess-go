package libcurl

import "jayess-go/binding"

type APIKind string

const (
	EasyAPI     APIKind = "easy"
	TransferAPI APIKind = "transfer"
	HeadersAPI  APIKind = "headers"
	MultiAPI    APIKind = "multi"
	StreamAPI   APIKind = "stream"
)

type BindingModule struct {
	Path           string
	Manifest       binding.Manifest
	APIs           []APIKind
	Handles        []HandleKind
	VendoredSource bool
}

func ValidateBindingModule(module BindingModule) []binding.Diagnostic {
	if err := binding.ValidateBindingTarget(module.Path); err != nil {
		return []binding.Diagnostic{{Field: "libcurl.binding", Message: err.Error()}}
	}
	diagnostics := binding.ValidateManifest(module.Manifest)
	if len(module.Handles) == 0 {
		diagnostics = append(diagnostics, binding.Diagnostic{
			Field:   "libcurl.handles",
			Message: "libcurl binding must declare safe handle kinds",
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

func PlanBuild(modules []BindingModule, platform string, runtimeHeaderDir string) binding.BuildPlan {
	bindingModules := make([]binding.Module, 0, len(modules))
	for _, module := range modules {
		manifest := module.Manifest
		if module.VendoredSource && len(manifest.Sources) == 0 {
			manifest.Sources = []string{"./lib/easy.c", "./lib/transfer.c"}
		}
		bindingModules = append(bindingModules, binding.Module{
			Path:     module.Path,
			Manifest: manifest,
		})
	}
	return binding.PlanBuild(bindingModules, platform, runtimeHeaderDir)
}
