package resolver

// TransitiveDependentCountFor returns the number of modules that import any module.
func (g *ModuleGraph) TransitiveDependentCountFor(modules []string) int {
	return len(g.TransitiveDependentsFor(modules))
}
