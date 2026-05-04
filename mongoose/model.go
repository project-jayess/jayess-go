package mongoose

import "jayess-go/binding"

type APIKind string

const (
	ManagerAPI    APIKind = "manager"
	ConnectionAPI APIKind = "connection"
	HTTPAPI       APIKind = "http"
	WebSocketAPI  APIKind = "websocket"
	LoopAPI       APIKind = "loop"
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
		return []binding.Diagnostic{{Field: "mongoose.binding", Message: err.Error()}}
	}
	diagnostics := binding.ValidateManifest(module.Manifest)
	if len(module.Handles) == 0 {
		diagnostics = append(diagnostics, binding.Diagnostic{
			Field:   "mongoose.handles",
			Message: "Mongoose binding must declare safe handle kinds",
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
			manifest.Sources = []string{"./mongoose.c"}
		}
		bindingModules = append(bindingModules, binding.Module{
			Path:     module.Path,
			Manifest: manifest,
		})
	}
	return binding.PlanBuild(bindingModules, platform, runtimeHeaderDir)
}
