package resolver

import "sort"

func NewModuleGraphFromDependents(dependents map[string][]string) *ModuleGraph {
	graph := NewModuleGraph()
	modules := dependentMapModules(dependents)
	for _, module := range modules {
		graph.AddModule(module, nil)
	}
	for _, module := range modules {
		for _, dependent := range dependents[module] {
			graph.AddImport(dependent, module)
		}
	}
	return graph
}

func dependentMapModules(dependents map[string][]string) []string {
	modules := make([]string, 0, len(dependents))
	for module := range dependents {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	return modules
}
