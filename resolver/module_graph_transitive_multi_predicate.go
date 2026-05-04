package resolver

// TransitivelyDependsOnFor reports whether multiple entries reach a dependency module.
func (g *ModuleGraph) TransitivelyDependsOnFor(entries []string, dependency string) (bool, error) {
	dependencies, err := g.TransitiveDependenciesFor(entries)
	if err != nil {
		return false, err
	}
	for _, current := range dependencies {
		if current == dependency {
			return true, nil
		}
	}
	return false, nil
}
