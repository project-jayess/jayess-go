package audio

import "jayess-go/binding"

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
