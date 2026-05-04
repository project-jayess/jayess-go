package resolver

func (g *ModuleGraph) LongestDependencyPathMap() (map[string][]string, error) {
	if g.imports == nil {
		return nil, nil
	}
	paths := make(map[string][]string, len(g.imports))
	for _, module := range g.Modules() {
		path, err := g.LongestDependencyPath(module)
		if err != nil {
			return nil, err
		}
		paths[module] = path
	}
	return paths, nil
}
