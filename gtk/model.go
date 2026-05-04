package gtk

import "jayess-go/binding"

type APIKind string

const (
	ApplicationAPI APIKind = "application"
	WindowAPI      APIKind = "window"
	WidgetAPI      APIKind = "widget"
	LayoutAPI      APIKind = "layout"
	EventAPI       APIKind = "event"
	DrawingAPI     APIKind = "drawing"
)

type BindingModule struct {
	Path     string
	Manifest binding.Manifest
	APIs     []APIKind
	Handles  []HandleKind
}

func ValidateBindingModule(module BindingModule) []binding.Diagnostic {
	if err := binding.ValidateBindingTarget(module.Path); err != nil {
		return []binding.Diagnostic{{Field: "gtk.binding", Message: err.Error()}}
	}
	diagnostics := binding.ValidateManifest(module.Manifest)
	if len(module.Handles) == 0 {
		diagnostics = append(diagnostics, binding.Diagnostic{
			Field:   "gtk.handles",
			Message: "GTK binding must declare safe handle kinds",
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
		bindingModules = append(bindingModules, binding.Module{
			Path:     module.Path,
			Manifest: module.Manifest,
		})
	}
	return binding.PlanBuild(bindingModules, platform, runtimeHeaderDir)
}
