package resolver

func (g *ModuleGraph) AddCompactModule(module string, imports []string) {
	g.AddModule(module, compactModuleImports(imports))
}

func compactModuleImports(imports []string) []string {
	if len(imports) == 0 {
		return nil
	}
	seen := map[string]bool{}
	compacted := make([]string, 0, len(imports))
	for _, imported := range imports {
		if seen[imported] {
			continue
		}
		seen[imported] = true
		compacted = append(compacted, imported)
	}
	return compacted
}
