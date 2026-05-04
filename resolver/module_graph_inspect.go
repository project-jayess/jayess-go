package resolver

import "sort"

func (g *ModuleGraph) HasModule(module string) bool {
	if g.imports == nil {
		return false
	}
	_, ok := g.imports[module]
	return ok
}

func (g *ModuleGraph) Modules() []string {
	if g.imports == nil {
		return nil
	}
	modules := make([]string, 0, len(g.imports))
	for module := range g.imports {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	return modules
}

func (g *ModuleGraph) RootModules() []string {
	if g.imports == nil {
		return nil
	}
	var roots []string
	for _, module := range g.Modules() {
		if len(g.Dependents(module)) == 0 {
			roots = append(roots, module)
		}
	}
	return roots
}

func (g *ModuleGraph) LeafModules() []string {
	if g.imports == nil {
		return nil
	}
	var leaves []string
	for _, module := range g.Modules() {
		if len(g.imports[module]) == 0 {
			leaves = append(leaves, module)
		}
	}
	return leaves
}

func (g *ModuleGraph) Dependencies(module string) []string {
	if g.imports == nil {
		return nil
	}
	imports, ok := g.imports[module]
	if !ok {
		return nil
	}
	return append([]string(nil), imports...)
}

func (g *ModuleGraph) DependsOn(module string, dependency string) bool {
	if g.imports == nil {
		return false
	}
	for _, imported := range g.imports[module] {
		if imported == dependency {
			return true
		}
	}
	return false
}

func (g *ModuleGraph) Dependents(module string) []string {
	if g.imports == nil {
		return nil
	}
	var dependents []string
	for importer, imports := range g.imports {
		for _, imported := range imports {
			if imported == module {
				dependents = append(dependents, importer)
				break
			}
		}
	}
	sort.Strings(dependents)
	return dependents
}
