package resolver

func (g *ModuleGraph) ImportMap() map[string][]string {
	if g.imports == nil {
		return nil
	}
	imports := make(map[string][]string, len(g.imports))
	for module, moduleImports := range g.imports {
		imports[module] = append([]string(nil), moduleImports...)
	}
	return imports
}
