package picohttpparser

import "jayess-go/binding"

type APIKind string

const (
	RequestAPI     APIKind = "request"
	ResponseAPI    APIKind = "response"
	HeadersAPI     APIKind = "headers"
	IncrementalAPI APIKind = "incremental"
	ChunkedAPI     APIKind = "chunked"
)

type BindingModule struct {
	Path           string
	Manifest       binding.Manifest
	APIs           []APIKind
	VendoredSource bool
}

func ValidateBindingModule(module BindingModule) []binding.Diagnostic {
	if err := binding.ValidateBindingTarget(module.Path); err != nil {
		return []binding.Diagnostic{{Field: "picohttpparser.binding", Message: err.Error()}}
	}
	diagnostics := binding.ValidateManifest(module.Manifest)
	if len(module.APIs) == 0 {
		diagnostics = append(diagnostics, binding.Diagnostic{
			Field:   "picohttpparser.apis",
			Message: "picohttpparser binding must declare supported parser APIs",
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
			manifest.Sources = []string{"./picohttpparser.c"}
		}
		bindingModules = append(bindingModules, binding.Module{
			Path:     module.Path,
			Manifest: manifest,
		})
	}
	return binding.PlanBuild(bindingModules, platform, runtimeHeaderDir)
}
