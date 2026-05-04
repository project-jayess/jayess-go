package resolver

func (g *ModuleGraph) Clone() *ModuleGraph {
	clone := NewModuleGraph()
	if g.imports == nil {
		return clone
	}
	for module, imports := range g.imports {
		clone.imports[module] = append([]string(nil), imports...)
	}
	return clone
}
