package resolver

// TransitivelyDependedOnByFor reports whether any module is transitively imported by a dependent.
func (g *ModuleGraph) TransitivelyDependedOnByFor(modules []string, dependent string) bool {
	for _, current := range g.TransitiveDependentsFor(modules) {
		if current == dependent {
			return true
		}
	}
	return false
}
