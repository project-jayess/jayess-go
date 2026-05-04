package resolver

func (g *ModuleGraph) TransitivelyDependedOnBy(module string, dependent string) bool {
	for _, current := range g.TransitiveDependents(module) {
		if current == dependent {
			return true
		}
	}
	return false
}
