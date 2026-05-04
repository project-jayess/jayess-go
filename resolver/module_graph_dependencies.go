package resolver

func (g *ModuleGraph) AddResolvedModule(module string, dependencies []ResolvedModuleDependency) {
	imports := make([]string, 0, len(dependencies))
	for _, dependency := range dependencies {
		imports = append(imports, dependency.Path)
	}
	g.AddModule(module, imports)
}

func (g *ModuleGraph) AddCompactResolvedModule(module string, dependencies []ResolvedModuleDependency) {
	compacted := CompactResolvedModuleDependencies(dependencies)
	imports := make([]string, 0, len(compacted))
	for _, dependency := range compacted {
		imports = append(imports, dependency.Path)
	}
	g.AddModule(module, imports)
}
